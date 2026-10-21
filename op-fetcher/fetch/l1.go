package fetch

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// l1State reads L1 contract state at a fixed block.
type l1State interface {
	// Call executes a read-only call from the zero address. An error matching
	// errExecutionReverted means the call reverted; any other error means the call
	// could not be executed.
	Call(ctx context.Context, to common.Address, data []byte) ([]byte, error)
	StorageAt(ctx context.Context, addr common.Address, slot common.Hash) (common.Hash, error)
}

var errExecutionReverted = errors.New("execution reverted")

// revertError is a node's report of a call that reverted.
type revertError struct{ error }

func (e revertError) Is(target error) bool { return target == errExecutionReverted }

// rpcL1State is an l1State served by an L1 JSON-RPC endpoint. It reads every value at one
// block hash, so a fetch never mixes state from two blocks.
type rpcL1State struct {
	client *ethclient.Client
	block  common.Hash
}

// newRPCL1State returns an l1State at the given block number, or at the latest block if
// number is nil.
func newRPCL1State(ctx context.Context, client *ethclient.Client, number *big.Int) (*rpcL1State, *types.Header, error) {
	header, err := client.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get L1 block header: %w", err)
	}
	return &rpcL1State{client: client, block: header.Hash()}, header, nil
}

func (s *rpcL1State) Call(ctx context.Context, to common.Address, data []byte) ([]byte, error) {
	ret, err := s.client.CallContractAtHash(ctx, ethereum.CallMsg{To: &to, Data: data}, s.block)
	if isRevert(err) {
		return nil, revertError{err}
	}
	return ret, err
}

func (s *rpcL1State) StorageAt(ctx context.Context, addr common.Address, slot common.Hash) (common.Hash, error) {
	val, err := s.client.StorageAtHash(ctx, addr, slot, s.block)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(val), nil
}

// isRevert reports whether an eth_call error is the node reporting that the call reverted,
// as opposed to a transport failure, a rate limit or missing state. Only an explicit revert
// counts: misreading any other error as a revert would silently take a legacy-contract
// fallback and return wrong addresses, so an unrecognized error fails the fetch instead.
// Other EVM failures, such as running out of gas or an invalid opcode, fail the fetch too,
// although a Solidity try/catch would catch them; the getters read here revert on an unknown
// selector.
func isRevert(err error) bool {
	var rpcErr rpc.Error
	if !errors.As(err, &rpcErr) {
		return false
	}
	// geth and reth report a revert with code 3, with or without revert data. Some nodes use
	// the generic server error code instead, with the same message.
	switch rpcErr.ErrorCode() {
	case 3:
		return true
	case -32000:
		return strings.HasPrefix(strings.ToLower(rpcErr.Error()), "execution reverted")
	default:
		return false
	}
}
