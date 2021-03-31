package testnet

import (
	gsnet "github.com/ipfs/go-graphsync/network"
	"github.com/libp2p/go-libp2p-core/peer"

	dtnet "github.com/chenjianmei111/go-data-transfer/network"
)

// Network is an interface for generating graphsync network interfaces
// based on a test network.
type Network interface {
	Adapter() (peer.ID, gsnet.GraphSyncNetwork, dtnet.DataTransferNetwork)
	HasPeer(peer.ID) bool
}
