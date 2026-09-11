package atomic

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// PackedUserOperation is the ERC-4337 v0.7 on-chain encoding.
// This demo uses the EntryPoint already preinstalled on OP Stack chains.
type PackedUserOperation struct {
	Sender                      common.Address
	Nonce                       *big.Int
	InitCode, CallData          []byte
	AccountGasLimits            [32]byte
	PreVerificationGas          *big.Int
	GasFees                     [32]byte
	PaymasterAndData, Signature []byte
}

// UserOperationExecutor wraps every discovery/replay attempt in a real, signed
// EntryPoint.handleOps call. Accounts and proxies must already be deployed.
// Sign is called during discovery as well as finalization; the demo uses local
// signers. Interactive one-signature wallet flows need a separate intent design.
type UserOperationExecutor struct {
	Backend interface {
		Trace(context.Context, Transaction, bool) (callFrame, error)
	}
	EntryPointABI, AccountABI                                               abi.ABI
	Account, Bundler, Paymaster                                             common.Address
	ChainID                                                                 eth.ChainID
	Nonce                                                                   *big.Int
	VerificationGas, PaymasterVerificationGas, PreVerificationGas, OuterGas uint64
	MaxFeePerGas, MaxPriorityFeePerGas                                      *big.Int
	// Sign receives the v0.7 userOpHash. Upstream SimpleAccount expects an
	// Ethereum signed-message signature over it, including v=27 or v=28.
	Sign func(context.Context, common.Hash) ([]byte, error)
}

// SponsoredPlan carries ordinary bundler transactions and their signed UserOperations.
// Atomic executions describe the router call, not the outer handleOps receipt:
// EntryPoint catches an operation revert and settles gas through the paymaster.
// Settlement is EntryPoint's metered accounting, not a guarantee of reimbursing
// every outer-transaction cost or OP Stack L1 data fee.
type SponsoredPlan struct {
	Atomic       *Plan
	Operations   map[eth.ChainID]PackedUserOperation
	Transactions map[eth.ChainID]Transaction
}

// BuildSponsored discovers the atomic bundle through actual ERC-4337 execution.
// FirstLogIndex on each chain includes EntryPoint's one BeforeExecution log.
// There is exactly one predeployed account/UserOperation per outer transaction.
func BuildSponsored(ctx context.Context, b *Builder, root eth.ChainID, nonce uint64, target common.Address, data []byte) (*SponsoredPlan, error) {
	builder := *b
	builder.Chains = make(map[eth.ChainID]Chain, len(b.Chains))
	replays := make(map[eth.ChainID]*sponsoredReplay, len(b.Chains))
	for id, chain := range b.Chains {
		executor, ok := chain.Executor.(*UserOperationExecutor)
		if !ok || executor.ChainID != id || chain.Sender != executor.Account || chain.FirstLogIndex == 0 {
			return nil, fmt.Errorf("chain %s needs an account executor and its EntryPoint log prefix", id)
		}
		replay := &sponsoredReplay{UserOperationExecutor: executor}
		replays[id] = replay
		chain.Executor = replay
		builder.Chains[id] = chain
	}
	p, err := builder.Build(ctx, root, nonce, target, data)
	if err != nil {
		return nil, err
	}
	out := &SponsoredPlan{Atomic: p, Operations: make(map[eth.ChainID]PackedUserOperation), Transactions: make(map[eth.ChainID]Transaction)}
	for _, id := range orderedChains(p.Transactions) {
		// Return the exact signed UserOperation and outer calldata/fee envelope
		// used in the builder's single final replay. The adapter's pinned block
		// environment still must agree with subsequent full candidate execution.
		replay := replays[id]
		out.Operations[id], out.Transactions[id] = replay.operation, replay.transaction
	}
	return out, nil
}

type sponsoredReplay struct {
	*UserOperationExecutor
	operation   PackedUserOperation
	transaction Transaction
}

func (e *sponsoredReplay) Replay(ctx context.Context, call Transaction) (Execution, error) {
	operation, transaction, err := e.Prepare(ctx, call)
	if err != nil {
		return Execution{}, err
	}
	execution, err := e.executePrepared(ctx, call, operation, transaction, false)
	if err == nil {
		e.operation, e.transaction = operation, transaction
	}
	return execution, err
}

func (e *UserOperationExecutor) Discover(ctx context.Context, tx Transaction) (Execution, error) {
	return e.execute(ctx, tx, true)
}

func (e *UserOperationExecutor) Replay(ctx context.Context, tx Transaction) (Execution, error) {
	return e.execute(ctx, tx, false)
}

func (e *UserOperationExecutor) execute(ctx context.Context, call Transaction, discovery bool) (Execution, error) {
	op, tx, err := e.Prepare(ctx, call)
	if err != nil {
		return Execution{}, err
	}
	return e.executePrepared(ctx, call, op, tx, discovery)
}

// Prepare signs one UserOperation and encodes the bundler's ordinary transaction.
// Access-list checks remain on that outer transaction, including during final replay.
func (e *UserOperationExecutor) Prepare(ctx context.Context, call Transaction) (PackedUserOperation, Transaction, error) {
	var op PackedUserOperation
	if err := ctx.Err(); err != nil {
		return op, Transaction{}, err
	}
	if e.Backend == nil || e.Sign == nil || e.Nonce == nil || e.Nonce.Sign() < 0 || e.Nonce.BitLen() > 256 || call.From != e.Account || e.Account == (common.Address{}) || e.Bundler == (common.Address{}) || e.Paymaster == (common.Address{}) || e.VerificationGas == 0 || e.PaymasterVerificationGas == 0 || e.PreVerificationGas == 0 || e.OuterGas <= call.Gas || call.Gas == 0 {
		return op, Transaction{}, fmt.Errorf("invalid sponsored operation configuration")
	}
	for _, fee := range []*big.Int{e.MaxFeePerGas, e.MaxPriorityFeePerGas} {
		if fee == nil || fee.Sign() < 0 || fee.BitLen() > 128 {
			return op, Transaction{}, fmt.Errorf("invalid UserOperation gas fee")
		}
	}
	if e.MaxPriorityFeePerGas.Cmp(e.MaxFeePerGas) > 0 {
		return op, Transaction{}, fmt.Errorf("priority fee exceeds fee cap")
	}
	data, err := e.AccountABI.Pack("execute", call.To, new(big.Int), call.Data)
	if err != nil {
		return op, Transaction{}, err
	}
	op = PackedUserOperation{Sender: e.Account, Nonce: new(big.Int).Set(e.Nonce), CallData: data, PreVerificationGas: new(big.Int).SetUint64(e.PreVerificationGas)}
	op.AccountGasLimits = pack128(new(big.Int).SetUint64(e.VerificationGas), new(big.Int).SetUint64(call.Gas))
	op.GasFees = pack128(e.MaxPriorityFeePerGas, e.MaxFeePerGas)
	paymasterGas := pack128(new(big.Int).SetUint64(e.PaymasterVerificationGas), new(big.Int))
	op.PaymasterAndData = append(e.Paymaster.Bytes(), paymasterGas[:]...)
	hash, err := e.UserOperationHash(op)
	if err != nil {
		return op, Transaction{}, err
	}
	op.Signature, err = e.Sign(ctx, hash)
	if err != nil {
		return op, Transaction{}, err
	}
	data, err = e.EntryPointABI.Pack("handleOps", []PackedUserOperation{op}, e.Bundler)
	return op, Transaction{From: e.Bundler, To: predeploys.EntryPoint_v070Addr, Data: data, AccessList: call.AccessList, Gas: e.OuterGas, GasFeeCap: new(big.Int).Set(e.MaxFeePerGas), GasTipCap: new(big.Int).Set(e.MaxPriorityFeePerGas)}, err
}

// UserOperationHash matches EntryPoint v0.7, binding chain, EntryPoint, account,
// nonce, call, fees and paymaster. The signature itself is excluded by ERC-4337.
func (e *UserOperationExecutor) UserOperationHash(op PackedUserOperation) (common.Hash, error) {
	address, _ := abi.NewType("address", "", nil)
	u256, _ := abi.NewType("uint256", "", nil)
	hash, _ := abi.NewType("bytes32", "", nil)
	packed, err := (abi.Arguments{{Type: address}, {Type: u256}, {Type: hash}, {Type: hash}, {Type: hash}, {Type: u256}, {Type: hash}, {Type: hash}}).Pack(op.Sender, op.Nonce, crypto.Keccak256Hash(op.InitCode), crypto.Keccak256Hash(op.CallData), op.AccountGasLimits, op.PreVerificationGas, op.GasFees, crypto.Keccak256Hash(op.PaymasterAndData))
	if err != nil {
		return common.Hash{}, err
	}
	domain, err := (abi.Arguments{{Type: hash}, {Type: address}, {Type: u256}}).Pack(crypto.Keccak256Hash(packed), predeploys.EntryPoint_v070Addr, e.ChainID.ToBig())
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(domain), nil
}

func pack128(high, low *big.Int) (out [32]byte) {
	high.FillBytes(out[:16])
	low.FillBytes(out[16:])
	return
}

func (e *UserOperationExecutor) executePrepared(ctx context.Context, call Transaction, op PackedUserOperation, tx Transaction, discovery bool) (Execution, error) {
	frame, err := e.Backend.Trace(ctx, tx, discovery)
	if err != nil {
		return Execution{}, err
	}
	if frame.Error != "" {
		return Execution{}, fmt.Errorf("EntryPoint envelope reverted: 0x%x", frame.Output)
	}
	var operation *callFrame
	count := 0
	var visit func(callFrame)
	visit = func(f callFrame) {
		if f.Type == "CALL" && f.From == e.Account && f.To == call.To && bytes.Equal(f.Input, call.Data) {
			copy := f
			operation = &copy
			count++
		}
		for _, child := range f.Calls {
			visit(child)
		}
	}
	visit(frame)
	if count != 1 {
		return Execution{}, fmt.Errorf("expected exactly one account-to-router call, got %d", count)
	}
	inner, err := executionFromFrame(*operation)
	if err != nil {
		return Execution{}, err
	}
	outer, err := executionFromFrame(frame)
	if err != nil {
		return Execution{}, err
	}
	before := e.EntryPointABI.Events["BeforeExecution"].ID
	if len(outer.Logs) < len(inner.Logs)+2 || outer.Logs[0].Address != predeploys.EntryPoint_v070Addr || len(outer.Logs[0].Topics) != 1 || outer.Logs[0].Topics[0] != before || len(outer.Logs[0].Data) != 0 || !sameLogs(outer.Logs[1:1+len(inner.Logs)], inner.Logs) {
		return Execution{}, fmt.Errorf("unsupported EntryPoint log prefix or operation layout")
	}
	opHash, err := e.UserOperationHash(op)
	if err != nil {
		return Execution{}, err
	}
	event := e.EntryPointABI.Events["UserOperationEvent"]
	matches := 0
	for _, log := range outer.Logs[1+len(inner.Logs):] {
		if log.Address != predeploys.EntryPoint_v070Addr {
			return Execution{}, fmt.Errorf("unexpected envelope log")
		}
		if len(log.Topics) == 4 && log.Topics[0] == event.ID && log.Topics[1] == opHash && common.BytesToAddress(log.Topics[2].Bytes()) == e.Account && common.BytesToAddress(log.Topics[3].Bytes()) == e.Paymaster {
			values, err := event.Inputs.NonIndexed().Unpack(log.Data)
			if err != nil {
				return Execution{}, err
			}
			if values[0].(*big.Int).Cmp(op.Nonce) != 0 || values[1].(bool) == inner.Reverted || values[2].(*big.Int).Sign() <= 0 {
				return Execution{}, fmt.Errorf("UserOperation outcome or sponsored gas does not match router execution")
			}
			matches++
		}
	}
	if matches != 1 {
		return Execution{}, fmt.Errorf("missing unique sponsored UserOperation event")
	}
	return inner, nil
}

func sameLogs(a, b []*types.Log) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Address != b[i].Address || !bytes.Equal(messages.LogToMessagePayload(a[i]), messages.LogToMessagePayload(b[i])) {
			return false
		}
	}
	return true
}
