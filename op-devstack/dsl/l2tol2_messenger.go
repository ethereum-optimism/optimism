package dsl

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var sentMessageTopic = crypto.Keccak256Hash([]byte("SentMessage(uint256,address,uint256,address,bytes)"))

// SentMessage is an L2ToL2CrossDomainMessenger message, read from the SentMessage event that
// initiated it.
type SentMessage struct {
	Hash        common.Hash
	Source      eth.ChainID
	Destination eth.ChainID
	Nonce       *big.Int
	Sender      common.Address
	Target      common.Address
	Message     []byte
	// LogIndex is the index of the SentMessage event in its transaction's receipt.
	LogIndex int
}

// SentMessageFromReceipt decodes the first L2ToL2CrossDomainMessenger SentMessage event in a
// receipt of a transaction on the chain `source`, and computes the message hash.
func SentMessageFromReceipt(rcpt *types.Receipt, source eth.ChainID) (SentMessage, error) {
	for i, l := range rcpt.Logs {
		if l.Address != predeploys.L2toL2CrossDomainMessengerAddr || len(l.Topics) != 4 || l.Topics[0] != sentMessageTopic {
			continue
		}
		data, err := abi.Arguments{{Type: abiType("address")}, {Type: abiType("bytes")}}.Unpack(l.Data)
		if err != nil {
			return SentMessage{}, fmt.Errorf("decode SentMessage data: %w", err)
		}
		m := SentMessage{
			Source:      source,
			Destination: eth.ChainIDFromBig(l.Topics[1].Big()),
			Target:      common.BytesToAddress(l.Topics[2].Bytes()),
			Nonce:       l.Topics[3].Big(),
			Sender:      data[0].(common.Address),
			Message:     data[1].([]byte),
			LogIndex:    i,
		}
		encoded, err := abi.Arguments{
			{Type: abiType("uint256")}, {Type: abiType("uint256")}, {Type: abiType("uint256")},
			{Type: abiType("address")}, {Type: abiType("address")}, {Type: abiType("bytes")},
		}.Pack(m.Destination.ToBig(), source.ToBig(), m.Nonce, m.Sender, m.Target, m.Message)
		if err != nil {
			return SentMessage{}, fmt.Errorf("encode message hash preimage: %w", err)
		}
		m.Hash = crypto.Keccak256Hash(encoded)
		return m, nil
	}
	return SentMessage{}, fmt.Errorf("no SentMessage event in tx %s", rcpt.TxHash)
}

func abiType(name string) abi.Type {
	typ, err := abi.NewType(name, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}
