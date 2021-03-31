package extension

import (
	"bytes"
	"errors"
	"io"

	"github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/protocol"

	datatransfer "github.com/chenjianmei111/go-data-transfer"
	"github.com/chenjianmei111/go-data-transfer/message"
	"github.com/chenjianmei111/go-data-transfer/message/message1_0"
)

const (
	// ExtensionDataTransfer1_1 is the identifier for the current data transfer extension to graphsync
	ExtensionDataTransfer1_1 = graphsync.ExtensionName("fil/data-transfer/1.1")
	// ExtensionDataTransfer1_0 is the identifier for the legacy data transfer extension to graphsync
	ExtensionDataTransfer1_0 = graphsync.ExtensionName("fil/data-transfer")
)

// ProtocolMap maps graphsync extensions to their libp2p protocols
var ProtocolMap = map[graphsync.ExtensionName]protocol.ID{
	ExtensionDataTransfer1_1: datatransfer.ProtocolDataTransfer1_1,
	ExtensionDataTransfer1_0: datatransfer.ProtocolDataTransfer1_0,
}

// ToExtensionData converts a message to a graphsync extension
func ToExtensionData(msg datatransfer.Message, supportedExtensions []graphsync.ExtensionName) ([]graphsync.ExtensionData, error) {
	exts := make([]graphsync.ExtensionData, 0, len(supportedExtensions))
	for _, supportedExtension := range supportedExtensions {
		protoID, ok := ProtocolMap[supportedExtension]
		if !ok {
			return nil, errors.New("unsupported protocol")
		}
		versionedMsg, err := msg.MessageForProtocol(protoID)
		if err != nil {
			continue
		}
		buf := new(bytes.Buffer)
		err = versionedMsg.ToNet(buf)
		if err != nil {
			return nil, err
		}
		exts = append(exts, graphsync.ExtensionData{
			Name: supportedExtension,
			Data: buf.Bytes(),
		})
	}
	if len(exts) == 0 {
		return nil, errors.New("message not encodable in any supported extensions")
	}
	return exts, nil
}

// GsExtended is a small interface used by getExtensionData
type GsExtended interface {
	Extension(name graphsync.ExtensionName) ([]byte, bool)
}

// GetTransferData unmarshals extension data.
// Returns:
//    * nil + nil if the extension is not found
//    * nil + error if the extendedData fails to unmarshal
//    * unmarshaled ExtensionDataTransferData + nil if all goes well
func GetTransferData(extendedData GsExtended) (datatransfer.Message, error) {
	extName := ExtensionDataTransfer1_1
	data, ok := extendedData.Extension(extName)
	if !ok {
		extName = ExtensionDataTransfer1_0
		data, ok = extendedData.Extension(extName)
		if !ok {
			return nil, nil
		}
	}
	reader := bytes.NewReader(data)
	return decoders[extName](reader)
}

type decoder func(io.Reader) (datatransfer.Message, error)

var decoders = map[graphsync.ExtensionName]decoder{
	ExtensionDataTransfer1_1: message.FromNet,
	ExtensionDataTransfer1_0: message1_0.FromNet,
}
