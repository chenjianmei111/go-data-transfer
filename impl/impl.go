package impl

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log"
	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/xerrors"

	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-data-transfer/channels"
	"github.com/filecoin-project/go-data-transfer/message"
	"github.com/filecoin-project/go-data-transfer/network"
	"github.com/filecoin-project/go-data-transfer/registry"
	"github.com/filecoin-project/go-data-transfer/transport"
	"github.com/filecoin-project/go-storedcounter"
	"github.com/hannahhoward/go-pubsub"
)

var log = logging.Logger("dt-impl")

type manager struct {
	dataTransferNetwork network.DataTransferNetwork
	validatedTypes      *registry.Registry
	resultTypes         *registry.Registry
	revalidators        *registry.Registry
	pubSub              *pubsub.PubSub
	channels            *channels.Channels
	peerID              peer.ID
	transport           transport.Transport
	storedCounter       *storedcounter.StoredCounter
}

type internalEvent struct {
	evt   datatransfer.Event
	state datatransfer.ChannelState
}

func dispatcher(evt pubsub.Event, subscriberFn pubsub.SubscriberFn) error {
	ie, ok := evt.(internalEvent)
	if !ok {
		return errors.New("wrong type of event")
	}
	cb, ok := subscriberFn.(datatransfer.Subscriber)
	if !ok {
		return errors.New("wrong type of event")
	}
	cb(ie.evt, ie.state)
	return nil
}

// NewDataTransfer initializes a new instance of a data transfer manager
func NewDataTransfer(ds datastore.Datastore, dataTransferNetwork network.DataTransferNetwork, transport transport.Transport, storedCounter *storedcounter.StoredCounter) (datatransfer.Manager, error) {
	m := &manager{
		dataTransferNetwork: dataTransferNetwork,
		validatedTypes:      registry.NewRegistry(),
		resultTypes:         registry.NewRegistry(),
		revalidators:        registry.NewRegistry(),
		pubSub:              pubsub.New(dispatcher),
		peerID:              dataTransferNetwork.ID(),
		transport:           transport,
		storedCounter:       storedCounter,
	}
	channels, err := channels.New(ds, m.notifier, m.validatedTypes.Decoder, m.resultTypes.Decoder)
	if err != nil {
		return nil, err
	}
	m.channels = channels
	return m, nil
}

func (m *manager) notifier(evt datatransfer.Event, chst datatransfer.ChannelState) {
	err := m.pubSub.Publish(internalEvent{evt, chst})
	if err != nil {
		log.Warnf("err publishing DT event: %s", err.Error())
	}
}

// Start initializes data transfer processing
func (m *manager) Start(ctx context.Context) error {
	dtReceiver := &receiver{m}
	m.dataTransferNetwork.SetDelegate(dtReceiver)
	return m.transport.SetEventHandler(m)
}

// Stop terminates all data transfers and ends processing
func (m *manager) Stop() error {
	return nil
}

// RegisterVoucherType registers a validator for the given voucher type
// returns error if:
// * voucher type does not implement voucher
// * there is a voucher type registered with an identical identifier
// * voucherType's Kind is not reflect.Ptr
func (m *manager) RegisterVoucherType(voucherType datatransfer.Voucher, validator datatransfer.RequestValidator) error {
	err := m.validatedTypes.Register(voucherType, validator)
	if err != nil {
		return xerrors.Errorf("error registering voucher type: %w", err)
	}
	return nil
}

// OpenPushDataChannel opens a data transfer that will send data to the recipient peer and
// transfer parts of the piece that match the selector
func (m *manager) OpenPushDataChannel(ctx context.Context, requestTo peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.ChannelID, error) {
	req, err := m.newRequest(ctx, selector, false, voucher, baseCid, requestTo)
	if err != nil {
		return datatransfer.ChannelID{}, err
	}

	chid, err := m.channels.CreateNew(req.TransferID(), baseCid, selector, voucher,
		m.peerID, m.peerID, requestTo) // initiator = us, sender = us, receiver = them
	if err != nil {
		return chid, err
	}
	if err := m.dataTransferNetwork.SendMessage(ctx, requestTo, req); err != nil {
		err = fmt.Errorf("Unable to send request: %w", err)
		_ = m.channels.Error(chid, err)
		return chid, err
	}
	return chid, nil
}

// OpenPullDataChannel opens a data transfer that will request data from the sending peer and
// transfer parts of the piece that match the selector
func (m *manager) OpenPullDataChannel(ctx context.Context, requestTo peer.ID, voucher datatransfer.Voucher, baseCid cid.Cid, selector ipld.Node) (datatransfer.ChannelID, error) {
	req, err := m.newRequest(ctx, selector, true, voucher, baseCid, requestTo)
	if err != nil {
		return datatransfer.ChannelID{}, err
	}
	// initiator = us, sender = them, receiver = us
	chid, err := m.channels.CreateNew(req.TransferID(), baseCid, selector, voucher,
		m.peerID, requestTo, m.peerID)
	if err != nil {
		return chid, err
	}
	if err := m.transport.OpenChannel(ctx, requestTo, chid, cidlink.Link{Cid: baseCid}, selector, req); err != nil {
		err = fmt.Errorf("Unable to send request: %w", err)
		_ = m.channels.Error(chid, err)
		return chid, err
	}
	return chid, nil
}

// SendVoucher sends an intermediate voucher as needed when the receiver sends a request for revalidation
func (m *manager) SendVoucher(ctx context.Context, channelID datatransfer.ChannelID, voucher datatransfer.Voucher) error {
	chst, err := m.channels.GetByID(ctx, channelID)
	if err != nil {
		return err
	}
	if channelID.Initiator != m.peerID {
		return errors.New("cannot send voucher for request we did not initiate")
	}
	updateRequest, err := message.VoucherRequest(channelID.ID, voucher.Type(), voucher)
	if err != nil {
		return err
	}
	if err := m.dataTransferNetwork.SendMessage(ctx, chst.OtherParty(m.peerID), updateRequest); err != nil {
		err = fmt.Errorf("Unable to send request: %w", err)
		_ = m.channels.Error(channelID, err)
		return err
	}
	return m.channels.NewVoucher(channelID, voucher)
}

// newRequest encapsulates message creation
func (m *manager) newRequest(ctx context.Context, selector ipld.Node, isPull bool, voucher datatransfer.Voucher, baseCid cid.Cid, to peer.ID) (message.DataTransferRequest, error) {
	next, err := m.storedCounter.Next()
	if err != nil {
		return nil, err
	}
	tid := datatransfer.TransferID(next)
	return message.NewRequest(tid, isPull, voucher.Type(), voucher, baseCid, selector)
}

func (m *manager) response(isNew bool, err error, tid datatransfer.TransferID, voucherResult datatransfer.VoucherResult) (message.DataTransferResponse, error) {
	isAccepted := err == nil || err == datatransfer.ErrPause
	isPaused := err == datatransfer.ErrPause
	resultType := datatransfer.EmptyTypeIdentifier
	if voucherResult != nil {
		resultType = voucherResult.Type()
	}
	if isNew {
		return message.NewResponse(tid, isAccepted, isPaused, resultType, voucherResult)
	}
	return message.VoucherResultResponse(tid, isAccepted, isPaused, resultType, voucherResult)
}

// close an open channel (effectively a cancel)
func (m *manager) CloseDataTransferChannel(ctx context.Context, chid datatransfer.ChannelID) error {
	chst, err := m.channels.GetByID(ctx, chid)
	if err != nil {
		return err
	}
	err = m.transport.CloseChannel(ctx, chid)
	if err != nil {
		return err
	}

	if err := m.dataTransferNetwork.SendMessage(ctx, chst.OtherParty(m.peerID), m.cancelMessage(chid)); err != nil {
		err = fmt.Errorf("Unable to send cancel message: %w", err)
		_ = m.channels.Error(chid, err)
		return err
	}

	return m.channels.Cancel(chid)
}

// pause a running data transfer channel
func (m *manager) PauseDataTransferChannel(ctx context.Context, chid datatransfer.ChannelID) error {

	pausable, ok := m.transport.(transport.PauseableTransport)
	if !ok {
		return errors.New("unsupported")
	}

	err := pausable.PauseChannel(ctx, chid)
	if err != nil {
		return err
	}

	if err := m.dataTransferNetwork.SendMessage(ctx, chid.OtherParty(m.peerID), m.pauseMessage(chid)); err != nil {
		err = fmt.Errorf("Unable to send pause message: %w", err)
		_ = m.channels.Error(chid, err)
		return err
	}

	return m.pause(chid)
}

// resume a running data transfer channel
func (m *manager) ResumeDataTransferChannel(ctx context.Context, chid datatransfer.ChannelID) error {
	pausable, ok := m.transport.(transport.PauseableTransport)
	if !ok {
		return errors.New("unsupported")
	}

	err := pausable.ResumeChannel(ctx, m.resumeMessage(chid), chid)
	if err != nil {
		return err
	}

	return m.resume(chid)
}

// get status of a transfer
func (m *manager) TransferChannelStatus(ctx context.Context, chid datatransfer.ChannelID) datatransfer.Status {
	chst, err := m.channels.GetByID(ctx, chid)
	if err != nil {
		return datatransfer.ChannelNotFoundError
	}
	return chst.Status()
}

// get notified when certain types of events happen
func (m *manager) SubscribeToEvents(subscriber datatransfer.Subscriber) datatransfer.Unsubscribe {
	return datatransfer.Unsubscribe(m.pubSub.Subscribe(subscriber))
}

// get all in progress transfers
func (m *manager) InProgressChannels(ctx context.Context) (map[datatransfer.ChannelID]datatransfer.ChannelState, error) {
	return m.channels.InProgress(ctx)
}

// RegisterRevalidator registers a revalidator for the given voucher type
// Note: this is the voucher type used to revalidate. It can share a name
// with the initial validator type and CAN be the same type, or a different type.
// The revalidator can simply be the sampe as the original request validator,
// or a different validator that satisfies the revalidator interface.
func (m *manager) RegisterRevalidator(voucherType datatransfer.Voucher, revalidator datatransfer.Revalidator) error {
	err := m.revalidators.Register(voucherType, revalidator)
	if err != nil {
		return xerrors.Errorf("error registering revalidator type: %w", err)
	}
	return nil
}

// RegisterVoucherResultType allows deserialization of a voucher result,
// so that a listener can read the metadata
func (m *manager) RegisterVoucherResultType(resultType datatransfer.VoucherResult) error {
	err := m.resultTypes.Register(resultType, nil)
	if err != nil {
		return xerrors.Errorf("error registering voucher type: %w", err)
	}
	return nil
}

type statusList []datatransfer.Status

func (sl statusList) Contains(s datatransfer.Status) bool {
	for _, ts := range sl {
		if ts == s {
			return true
		}
	}
	return false
}

func (m *manager) resume(chid datatransfer.ChannelID) error {
	if chid.Initiator == m.peerID {
		return m.channels.ResumeInitiator(chid)
	}
	return m.channels.ResumeResponder(chid)
}

func (m *manager) pause(chid datatransfer.ChannelID) error {
	if chid.Initiator == m.peerID {
		return m.channels.PauseInitiator(chid)
	}
	return m.channels.PauseResponder(chid)
}

func (m *manager) resumeOther(chid datatransfer.ChannelID) error {
	if chid.Responder == m.peerID {
		return m.channels.ResumeInitiator(chid)
	}
	return m.channels.ResumeResponder(chid)
}

func (m *manager) pauseOther(chid datatransfer.ChannelID) error {
	if chid.Responder == m.peerID {
		return m.channels.PauseInitiator(chid)
	}
	return m.channels.PauseResponder(chid)
}

func (m *manager) resumeMessage(chid datatransfer.ChannelID) message.DataTransferMessage {
	if chid.Initiator == m.peerID {
		return message.UpdateRequest(chid.ID, false)
	}
	return message.UpdateResponse(chid.ID, false)
}

func (m *manager) pauseMessage(chid datatransfer.ChannelID) message.DataTransferMessage {
	if chid.Initiator == m.peerID {
		return message.UpdateRequest(chid.ID, true)
	}
	return message.UpdateResponse(chid.ID, true)
}

func (m *manager) cancelMessage(chid datatransfer.ChannelID) message.DataTransferMessage {
	if chid.Initiator == m.peerID {
		return message.CancelRequest(chid.ID)
	}
	return message.CancelResponse(chid.ID)
}

func (m *manager) decodeVoucherResult(response message.DataTransferResponse) (datatransfer.VoucherResult, error) {
	vtypStr := datatransfer.TypeIdentifier(response.VoucherResultType())
	decoder, has := m.resultTypes.Decoder(vtypStr)
	if !has {
		return nil, xerrors.Errorf("unknown voucher result type: %s", vtypStr)
	}
	encodable, err := response.VoucherResult(decoder)
	if err != nil {
		return nil, err
	}
	return encodable.(datatransfer.Registerable), nil
}

func (m *manager) decodeVoucher(request message.DataTransferRequest, registry *registry.Registry) (datatransfer.Voucher, error) {
	vtypStr := datatransfer.TypeIdentifier(request.VoucherType())
	decoder, has := registry.Decoder(vtypStr)
	if !has {
		return nil, xerrors.Errorf("unknown voucher type: %s", vtypStr)
	}
	encodable, err := request.Voucher(decoder)
	if err != nil {
		return nil, err
	}
	return encodable.(datatransfer.Registerable), nil
}
