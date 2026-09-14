package atomic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
)

// EVMExecutor runs each attempt against a copy of a pinned block-prefix state.
// It is a prototype execution adapter, not a payload producer: callers still
// sign the returned transactions and validate the complete candidate blocks.
type EVMExecutor struct {
	State  *state.StateDB
	Block  vm.BlockContext
	Config *params.ChainConfig
}

func (e *EVMExecutor) Discover(ctx context.Context, tx Transaction) (Execution, error) {
	execution, _, err := e.execute(ctx, tx, true)
	return execution, err
}

func (e *EVMExecutor) Replay(ctx context.Context, tx Transaction) (Execution, error) {
	execution, _, err := e.Execute(ctx, tx)
	return execution, err
}

// Execute returns the isolated candidate state after ordinary EVM execution.
// The pinned state is never changed, including when execution fails.
func (e *EVMExecutor) Execute(ctx context.Context, tx Transaction) (Execution, *state.StateDB, error) {
	return e.execute(ctx, tx, false)
}

var validateSelector = crypto.Keccak256([]byte("validateMessage((address,uint256,uint256,uint256,uint256),bytes32)"))[:4]

func (e *EVMExecutor) execute(ctx context.Context, tx Transaction, discovery bool) (Execution, *state.StateDB, error) {
	return e.executeWithHooks(ctx, tx, discovery, nil)
}

// Trace executes the complete transaction with the native callTracer.
func (e *EVMExecutor) Trace(ctx context.Context, tx Transaction, discovery bool) (callFrame, error) {
	tracer, err := tracers.DefaultDirectory.New("callTracer", &tracers.Context{}, json.RawMessage(`{"withLog":true}`), e.Config)
	if err != nil {
		return callFrame{}, err
	}
	if _, _, err := e.executeWithHooks(ctx, tx, discovery, tracer.Hooks); err != nil {
		return callFrame{}, err
	}
	encoded, err := tracer.GetResult()
	if err != nil {
		return callFrame{}, err
	}
	var frame callFrame
	err = json.Unmarshal(encoded, &frame)
	return frame, err
}

func (e *EVMExecutor) executeWithHooks(ctx context.Context, tx Transaction, discovery bool, hooks *tracing.Hooks) (Execution, *state.StateDB, error) {
	if err := ctx.Err(); err != nil {
		return Execution{}, nil, err
	}
	if err := tx.validateFees(); err != nil {
		return Execution{}, nil, err
	}
	st := e.State.Copy()
	config := vm.Config{Tracer: hooks}
	if discovery {
		// Canonical inbox access lists warm the inbox address before its first
		// CALL. Warming only its checksum slot inside OnEnter is too late to
		// avoid the cold account surcharge on that CALL.
		inboxWarm := false
		for _, tuple := range tx.AccessList {
			inboxWarm = inboxWarm || tuple.Address == predeploys.CrossL2InboxAddr
		}
		if !inboxWarm {
			tx.AccessList = append(append(types.AccessList(nil), tx.AccessList...), types.AccessTuple{Address: predeploys.CrossL2InboxAddr})
		}
		// Warm each requested checksum immediately before the real inbox executes.
		// All bytecode and logs are preserved. This deliberately relaxes only the
		// discovery run; final replay has no hook and requires the actual access list.
		var combined tracing.Hooks
		if hooks != nil {
			combined = *hooks
		}
		onEnter := combined.OnEnter
		combined.OnEnter = func(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
			if typ == byte(vm.CALL) {
				if checksum, ok := requestedChecksum(to, input); ok {
					st.AddSlotToAccessList(to, checksum)
				}
			}
			if onEnter != nil {
				onEnter(depth, typ, from, to, input, gas, value)
			}
		}
		config.Tracer = &combined
	}
	block := e.Block
	price := new(big.Int).Add(block.BaseFee, big.NewInt(1))
	feeCap, tipCap := price, big.NewInt(1)
	if tx.GasFeeCap != nil {
		feeCap, tipCap = new(big.Int).Set(tx.GasFeeCap), new(big.Int).Set(tx.GasTipCap)
		price = new(big.Int).Add(block.BaseFee, tipCap)
		if price.Cmp(feeCap) > 0 {
			price.Set(feeCap)
		}
	}
	var evmState vm.StateDB = st
	if config.Tracer != nil {
		evmState = state.NewHookedState(st, config.Tracer)
	}
	evm := vm.NewEVM(block, evmState, e.Config, config)
	stop := context.AfterFunc(ctx, evm.Cancel)
	defer stop()
	// This adapter evaluates unsigned plans with the sender's pinned nonce.
	// No transaction validity checks (nonce, fees, intrinsic gas) are bypassed.
	msg := &core.Message{To: &tx.To, From: tx.From, Nonce: st.GetNonce(tx.From), Value: new(big.Int), GasLimit: tx.Gas, GasPrice: price, GasFeeCap: feeCap, GasTipCap: tipCap, Data: tx.Data, AccessList: tx.AccessList}
	// Include the account nonce so identical calldata in different prefix
	// transactions cannot retrieve one another's retained StateDB logs.
	nonce := common.BigToHash(new(big.Int).SetUint64(msg.Nonce))
	txHash := crypto.Keccak256Hash(tx.From.Bytes(), nonce[:], tx.Data)
	st.SetTxContext(txHash, 0)
	result, err := core.ApplyMessage(evm, msg, core.NewGasPool(block.GasLimit))
	if err != nil {
		return Execution{}, nil, fmt.Errorf("apply transaction: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Execution{}, nil, err
	}
	logs := st.GetLogs(txHash, bigs.Uint64Strict(block.BlockNumber), common.Hash{}, block.Time)
	st.Finalise(true)
	return Execution{Output: result.ReturnData, Logs: logs, Reverted: result.Failed()}, st, nil
}
