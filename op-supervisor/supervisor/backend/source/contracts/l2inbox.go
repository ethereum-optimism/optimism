package contracts

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	eventExecutingMessage = "ExecutingMessage"
)

var (
	ErrEventNotFound = errors.New("event not found")
)

type contractIdentifier struct {
	Origin      common.Address
	BlockNumber *big.Int
	LogIndex    *big.Int
	Timestamp   *big.Int
	ChainId     *big.Int
}

type CrossL2Inbox struct {
	contract *batching.BoundContract
}

func NewCrossL2Inbox() *CrossL2Inbox {
	abi := snapshots.LoadCrossL2InboxABI()
	return &CrossL2Inbox{
		contract: batching.NewBoundContract(abi, predeploys.CrossL2InboxAddr),
	}
}

func (i *CrossL2Inbox) DecodeExecutingMessageLog(l *ethTypes.Log) (backendTypes.ExecutingMessage, error) {
	if l.Address != i.contract.Addr() {
		return backendTypes.ExecutingMessage{}, fmt.Errorf("%w: log not from CrossL2Inbox", ErrEventNotFound)
	}
	name, result, err := i.contract.DecodeEvent(l)
	if errors.Is(err, batching.ErrUnknownEvent) {
		return backendTypes.ExecutingMessage{}, fmt.Errorf("%w: %v", ErrEventNotFound, err.Error())
	} else if err != nil {
		return backendTypes.ExecutingMessage{}, fmt.Errorf("failed to decode event: %w", err)
	}
	if name != eventExecutingMessage {
		return backendTypes.ExecutingMessage{}, fmt.Errorf("%w: event %v not an ExecutingMessage event", ErrEventNotFound, name)
	}
	var ident contractIdentifier
	result.GetStruct(0, &ident)
	payload := result.GetBytes(1)
	payloadHash := crypto.Keccak256Hash(payload)

	chainID, err := types.ChainIDFromBig(ident.ChainId).ToUInt32()
	if err != nil {
		return backendTypes.ExecutingMessage{}, fmt.Errorf("failed to convert chain ID %v to uint32: %w", ident.ChainId, err)
	}
	return backendTypes.ExecutingMessage{
		Chain:     chainID,
		BlockNum:  ident.BlockNumber.Uint64(),
		LogIdx:    uint32(ident.LogIndex.Uint64()),
		Timestamp: ident.Timestamp.Uint64(),
		Hash:      backendTypes.TruncateHash(payloadHash),
	}, nil
}
