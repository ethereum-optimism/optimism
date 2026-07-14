package script

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BroadcastType string

const (
	BroadcastCall   BroadcastType = "call"
	BroadcastCreate BroadcastType = "create"
	// BroadcastCreate2 is to be broadcast via the Create2Deployer,
	// and not really documented much anywhere.
	BroadcastCreate2 BroadcastType = "create2"
)

func (bt BroadcastType) String() string {
	return string(bt)
}

func (bt BroadcastType) MarshalText() ([]byte, error) {
	return []byte(bt.String()), nil
}

func (bt *BroadcastType) UnmarshalText(data []byte) error {
	v := BroadcastType(data)
	switch v {
	case BroadcastCall, BroadcastCreate, BroadcastCreate2:
		*bt = v
		return nil
	default:
		return fmt.Errorf("unrecognized broadcast type bytes: %x", data)
	}
}

// Broadcast captures a transaction that was selected to be broadcast
// via vm.broadcast(). Actually submitting the transaction is left up
// to other tools.
type Broadcast struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`    // set to expected contract address, if this is a deployment
	Input   hexutil.Bytes  `json:"input"` // set to contract-creation code, if this is a deployment
	Value   *hexutil.U256  `json:"value"`
	Salt    common.Hash    `json:"salt"` // set if this is a Create2 broadcast
	GasUsed uint64         `json:"gasUsed"`
	Type    BroadcastType  `json:"type"`
	Nonce   uint64         `json:"nonce"` // pre-state nonce of From, before any increment (always 0 if create2)
}

// ID returns a hash that can be used to identify the broadcast.
// This is used instead of the transaction hash since broadcasting
// tools can change gas limits and other fields which would change
// the resulting transaction hash.
func (b Broadcast) ID() common.Hash {
	h := sha256.New()
	_, _ = h.Write(b.From[:])
	_, _ = h.Write(b.To[:])
	_, _ = h.Write(b.Input)
	_, _ = h.Write(((*uint256.Int)(b.Value)).Bytes())
	_, _ = h.Write(b.Salt[:])
	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, b.Nonce)
	_, _ = h.Write(nonce)
	sum := h.Sum(nil)
	return common.BytesToHash(sum)
}
