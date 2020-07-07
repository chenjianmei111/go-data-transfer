package channels

import (
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-statemachine/fsm"
	logging "github.com/ipfs/go-log"
	cbg "github.com/whyrusleeping/cbor-gen"
)

var log = logging.Logger("data-transfer")

// ChannelEvents describe the events taht can
var ChannelEvents = fsm.Events{
	fsm.Event(datatransfer.Open).FromAny().To(datatransfer.Requested),
	fsm.Event(datatransfer.Accept).From(datatransfer.Requested).To(datatransfer.Ongoing),
	fsm.Event(datatransfer.Cancel).FromAny().To(datatransfer.Cancelled),
	fsm.Event(datatransfer.Progress).FromMany(
		datatransfer.Ongoing,
		datatransfer.InitiatorPaused,
		datatransfer.ResponderPaused,
		datatransfer.BothPaused,
		datatransfer.ResponderCompleted,
		datatransfer.ResponderFinalizing).ToNoChange().Action(func(chst *internalChannelState, deltaSent uint64, deltaReceived uint64) error {
		chst.Received += deltaReceived
		chst.Sent += deltaSent
		return nil
	}),
	fsm.Event(datatransfer.Error).FromAny().To(datatransfer.Failed).Action(func(chst *internalChannelState, err error) error {
		chst.Message = err.Error()
		return nil
	}),
	fsm.Event(datatransfer.NewVoucher).FromAny().ToNoChange().
		Action(func(chst *internalChannelState, vtype datatransfer.TypeIdentifier, voucherBytes []byte) error {
			chst.Vouchers = append(chst.Vouchers, encodedVoucher{Type: vtype, Voucher: &cbg.Deferred{Raw: voucherBytes}})
			return nil
		}),
	fsm.Event(datatransfer.NewVoucherResult).FromAny().ToNoChange().
		Action(func(chst *internalChannelState, vtype datatransfer.TypeIdentifier, voucherResultBytes []byte) error {
			chst.VoucherResults = append(chst.VoucherResults,
				encodedVoucherResult{Type: vtype, VoucherResult: &cbg.Deferred{Raw: voucherResultBytes}})
			return nil
		}),
	fsm.Event(datatransfer.PauseInitiator).
		FromMany(datatransfer.Requested, datatransfer.Ongoing).To(datatransfer.InitiatorPaused).
		From(datatransfer.ResponderPaused).To(datatransfer.BothPaused).
		FromAny().ToNoChange(),
	fsm.Event(datatransfer.PauseResponder).
		FromMany(datatransfer.Requested, datatransfer.Ongoing).To(datatransfer.ResponderPaused).
		From(datatransfer.InitiatorPaused).To(datatransfer.BothPaused).
		FromAny().ToNoChange(),
	fsm.Event(datatransfer.ResumeInitiator).
		From(datatransfer.InitiatorPaused).To(datatransfer.Ongoing).
		From(datatransfer.BothPaused).To(datatransfer.ResponderPaused).
		FromAny().ToNoChange(),
	fsm.Event(datatransfer.ResumeResponder).
		From(datatransfer.ResponderPaused).To(datatransfer.Ongoing).
		From(datatransfer.BothPaused).To(datatransfer.InitiatorPaused).
		From(datatransfer.Finalizing).To(datatransfer.Completed).
		From(datatransfer.ResponderFinalizing).To(datatransfer.ResponderCompleted).
		From(datatransfer.ResponderFinalizingTransferFinished).To(datatransfer.Completed).
		FromAny().ToNoChange(),
	fsm.Event(datatransfer.FinishTransfer).
		FromAny().To(datatransfer.TransferFinished).
		From(datatransfer.ResponderCompleted).To(datatransfer.Completed).
		From(datatransfer.ResponderFinalizing).To(datatransfer.ResponderFinalizingTransferFinished),
	fsm.Event(datatransfer.ResponderBeginsFinalization).
		FromAny().To(datatransfer.ResponderFinalizing).
		From(datatransfer.TransferFinished).To(datatransfer.ResponderFinalizingTransferFinished),
	fsm.Event(datatransfer.ResponderCompletes).
		FromAny().To(datatransfer.ResponderCompleted).
		From(datatransfer.ResponderPaused).To(datatransfer.ResponderFinalizing).
		From(datatransfer.TransferFinished).To(datatransfer.Completed),
	fsm.Event(datatransfer.BeginFinalizing).FromAny().To(datatransfer.Finalizing),
	fsm.Event(datatransfer.Complete).FromAny().To(datatransfer.Completed),
	fsm.Event(noopSynchronize).FromAny().ToNoChange(),
}

// ChannelStateEntryFuncs are handlers called as we enter different states
// (currently unused for this fsm)
var ChannelStateEntryFuncs = fsm.StateEntryFuncs{}

// ChannelFinalityStates are the final states for a channel
var ChannelFinalityStates = []fsm.StateKey{
	datatransfer.Cancelled,
	datatransfer.Completed,
	datatransfer.Failed,
}
