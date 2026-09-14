//go:build atomic_integration

package atomic

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func aaArtifact(t *testing.T, name string) *foundry.Artifact {
	t.Helper()
	artifact, err := foundry.ReadArtifact(filepath.Join("../../packages/contracts-bedrock/forge-artifacts", name+".sol", name+".json"))
	require.NoError(t, err)
	return artifact
}

// aaApply provisions the fixture using actual constructor/deposit calls. The
// preinstalled EntryPoint bytecode is the exact bytecode used by L2Genesis.
func aaApply(t *testing.T, e *EVMExecutor, from common.Address, to *common.Address, data []byte, value *big.Int) common.Address {
	t.Helper()
	st := e.State.Copy()
	nonce := st.GetNonce(from)
	msg := &core.Message{From: from, To: to, Nonce: nonce, Value: value, Data: data, GasLimit: 15_000_000, GasPrice: big.NewInt(8), GasFeeCap: big.NewInt(8), GasTipCap: big.NewInt(1)}
	st.SetTxContext(crypto.Keccak256Hash(from.Bytes(), new(big.Int).SetUint64(nonce).Bytes(), data), 0)
	result, err := core.ApplyMessage(vm.NewEVM(e.Block, st, e.Config, vm.Config{}), msg, core.NewGasPool(e.Block.GasLimit))
	require.NoError(t, err)
	require.False(t, result.Failed(), "fixture execution reverted: %x", result.ReturnData)
	st.Finalise(true)
	e.State = st
	return crypto.CreateAddress(from, nonce)
}

func newSponsoredFixture(t *testing.T) (*fixture, map[eth.ChainID]*UserOperationExecutor) {
	t.Helper()
	f := newFixture(t, 2)
	source, err := os.ReadFile("../../packages/contracts-bedrock/src/libraries/Preinstalls.sol")
	require.NoError(t, err)
	match := regexp.MustCompile(`EntryPoint_v070Code\s*=\s*hex"([a-fA-F0-9]+)"`).FindSubmatch(source)
	require.Len(t, match, 2)
	bytecode := common.FromHex(string(match[1]))
	factory, paymaster, account, entry := aaArtifact(t, "AtomicAccountFactory"), aaArtifact(t, "AtomicPaymaster"), aaArtifact(t, "SimpleAccount"), aaArtifact(t, "IEntryPoint")
	owner, err := crypto.GenerateKey()
	require.NoError(t, err)
	executors := make(map[eth.ChainID]*UserOperationExecutor)
	for id, chain := range f.b.Chains {
		backend := chain.Executor.(*EVMExecutor)
		backend.State.SetCode(predeploys.EntryPoint_v070Addr, bytecode, tracing.CodeChangeUnspecified)
		backend.State.SetBalance(f.sender, uint256.MustFromBig(new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))), tracing.BalanceChangeUnspecified)
		factoryAddr := aaApply(t, backend, f.sender, nil, factory.Bytecode.Object, new(big.Int))
		data, err := factory.ABI.Pack("createAccount", crypto.PubkeyToAddress(owner.PublicKey), new(big.Int))
		require.NoError(t, err)
		aaApply(t, backend, f.sender, &factoryAddr, data, new(big.Int))
		data, err = factory.ABI.Pack("getAddress", crypto.PubkeyToAddress(owner.PublicKey), new(big.Int))
		require.NoError(t, err)
		result, err := backend.Replay(t.Context(), Transaction{From: f.sender, To: factoryAddr, Data: data, Gas: f.b.Gas})
		require.NoError(t, err)
		require.False(t, result.Reverted)
		accountAddr := common.BytesToAddress(result.Output)
		args, err := paymaster.ABI.Constructor.Inputs.Pack(f.sender, f.b.Router, big.NewInt(1e17))
		require.NoError(t, err)
		paymasterAddr := aaApply(t, backend, f.sender, nil, append(append([]byte(nil), paymaster.Bytecode.Object...), args...), new(big.Int))
		data, err = paymaster.ABI.Pack("setAccountAllowed", accountAddr, true)
		require.NoError(t, err)
		aaApply(t, backend, f.sender, &paymasterAddr, data, new(big.Int))
		data, err = paymaster.ABI.Pack("deposit")
		require.NoError(t, err)
		aaApply(t, backend, f.sender, &paymasterAddr, data, big.NewInt(1e18))
		executor := &UserOperationExecutor{Backend: backend, EntryPointABI: entry.ABI, AccountABI: account.ABI, Account: accountAddr, Bundler: f.sender, Paymaster: paymasterAddr, ChainID: id, Nonce: new(big.Int), VerificationGas: 500_000, PaymasterVerificationGas: 200_000, PreVerificationGas: 300_000, OuterGas: 15_000_000, MaxFeePerGas: big.NewInt(1e9), MaxPriorityFeePerGas: big.NewInt(1), Sign: func(_ context.Context, hash common.Hash) ([]byte, error) {
			signature, err := crypto.Sign(accounts.TextHash(hash[:]), owner)
			if err == nil {
				signature[64] += 27
			}
			return signature, err
		}}
		chain.Sender, chain.Executor = accountAddr, executor
		chain.FirstLogIndex++ // EntryPoint.BeforeExecution precedes all router logs.
		f.b.Chains[id] = chain
		executors[id] = executor
	}
	return f, executors
}

func TestSponsoredAtomicCalls(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "remote revert"
		}
		t.Run(name, func(t *testing.T) {
			f, executors := newSponsoredFixture(t)
			if fail {
				backend := executors[f.remote].Backend.(*EVMExecutor)
				data, err := aaArtifact(t, "AtomicCounter").ABI.Pack("setLimit", big.NewInt(5))
				require.NoError(t, err)
				aaApply(t, backend, f.sender, &f.counter, data, new(big.Int))
			}
			data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(6))
			require.NoError(t, err)
			traces := make(map[eth.ChainID]*countedSponsoredTrace)
			for id, executor := range executors {
				executor.MaxPriorityFeePerGas = big.NewInt(17)
				traces[id] = &countedSponsoredTrace{EVMExecutor: executor.Backend.(*EVMExecutor)}
				executor.Backend = traces[id]
			}
			p, err := BuildSponsored(t.Context(), f.b, f.root, 0, f.appAddr, data)
			require.NoError(t, err)
			require.Equal(t, fail, p.Atomic.Reverted)
			require.Len(t, p.Transactions, 2)
			for id, tx := range p.Transactions {
				executor := executors[id]
				require.Equal(t, 1, traces[id].replays, "exactly one canonical replay per included chain")
				require.Equal(t, tx, traces[id].final, "return the exact envelope that passed final replay")
				require.Equal(t, executor.MaxFeePerGas, tx.GasFeeCap)
				require.Equal(t, executor.MaxPriorityFeePerGas, tx.GasTipCap)
				backend := traces[id].EVMExecutor
				require.True(t, backend.State.GetBalance(executor.Account).IsZero(), "account needs no ETH")
				result, st, err := backend.Execute(t.Context(), tx)
				require.NoError(t, err)
				require.False(t, result.Reverted, "EntryPoint settles gas even when the UserOperation reverts")
				address := f.counter
				if id == f.root {
					address = f.appAddr
				}
				want := common.BigToHash(big.NewInt(6))
				if fail {
					want = common.Hash{}
				}
				require.Equal(t, want, st.GetState(address, common.Hash{}))
				require.True(t, st.GetBalance(executor.Account).IsZero())
				// The adapter checked the real UserOperationEvent and positive gas charge.
				require.Equal(t, fail, p.Atomic.Executions[id].Reverted)
				if !fail {
					withoutAccess := tx
					withoutAccess.AccessList = nil
					result, err := backend.Replay(t.Context(), withoutAccess)
					require.NoError(t, err)
					require.False(t, result.Reverted)
					event := executor.EntryPointABI.Events["UserOperationEvent"]
					seen := 0
					for _, log := range result.Logs {
						if len(log.Topics) > 0 && log.Topics[0] == event.ID {
							seen++
							values, err := event.Inputs.NonIndexed().Unpack(log.Data)
							require.NoError(t, err)
							require.False(t, values[1].(bool), "AA must not bypass inbox access-list checks")
						}
					}
					require.Equal(t, 1, seen)
				}
				// Match the local signer hash against the actual preinstalled EntryPoint.
				query, err := executor.EntryPointABI.Pack("getUserOpHash", p.Operations[id])
				require.NoError(t, err)
				hashResult, err := backend.Replay(t.Context(), Transaction{From: f.sender, To: predeploys.EntryPoint_v070Addr, Data: query, Gas: f.b.Gas})
				require.NoError(t, err)
				wantHash, err := executor.UserOperationHash(p.Operations[id])
				require.NoError(t, err)
				require.Equal(t, wantHash, common.BytesToHash(hashResult.Output))
			}
		})
	}
}

type countedSponsoredTrace struct {
	*EVMExecutor
	replays                   int
	final                     Transaction
	lastDiscovery, finalFrame callFrame
}

func (e *countedSponsoredTrace) Trace(ctx context.Context, tx Transaction, discovery bool) (callFrame, error) {
	if !discovery {
		e.replays++
		e.final = tx
	}
	frame, err := e.EVMExecutor.Trace(ctx, tx, discovery)
	if discovery {
		e.lastDiscovery = frame
	} else {
		e.finalFrame = frame
	}
	return frame, err
}

func TestSponsoredDiscoveryGasMatchesReplay(t *testing.T) {
	f, executors := newSponsoredFixture(t)
	executor := executors[f.root]
	trace := &countedSponsoredTrace{EVMExecutor: executor.Backend.(*EVMExecutor)}
	executor.Backend = trace
	data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(6))
	require.NoError(t, err)
	_, err = BuildSponsored(t.Context(), f.b, f.root, 0, f.appAddr, data)
	require.NoError(t, err)
	var frames func(callFrame) []callFrame
	frames = func(frame callFrame) (out []callFrame) {
		if frame.To == f.appAddr || frame.To == f.proxies[f.remote] {
			out = append(out, frame)
		}
		for _, child := range frame.Calls {
			out = append(out, frames(child)...)
		}
		return out
	}
	discovery, replay := frames(trace.lastDiscovery), frames(trace.finalFrame)
	require.Len(t, discovery, 3)
	require.Len(t, replay, len(discovery))
	for i := range discovery {
		t.Logf("frame %d %s: discovery gas=%d used=%d; final gas=%d used=%d", i, discovery[i].To, discovery[i].Gas, discovery[i].GasUsed, replay[i].Gas, replay[i].GasUsed)
	}
	for i := range discovery {
		require.Equal(t, discovery[i].Gas, replay[i].Gas)
		require.Equal(t, discovery[i].GasUsed, replay[i].GasUsed)
	}
}

func TestSponsoredOperationRequiresAuthorization(t *testing.T) {
	for _, invalidSignature := range []bool{false, true} {
		t.Run(map[bool]string{true: "signature", false: "sponsorship"}[invalidSignature], func(t *testing.T) {
			f, executors := newSponsoredFixture(t)
			executor := executors[f.root]
			if invalidSignature {
				signer := executor.Sign
				executor.Sign = func(ctx context.Context, hash common.Hash) ([]byte, error) {
					signature, err := signer(ctx, hash)
					if err == nil {
						signature[5] ^= 1
					}
					return signature, err
				}
			} else {
				data, err := aaArtifact(t, "AtomicPaymaster").ABI.Pack("setAccountAllowed", executor.Account, false)
				require.NoError(t, err)
				aaApply(t, executor.Backend.(*EVMExecutor), f.sender, &executor.Paymaster, data, new(big.Int))
			}
			data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(6))
			require.NoError(t, err)
			_, err = BuildSponsored(t.Context(), f.b, f.root, 0, f.appAddr, data)
			require.ErrorContains(t, err, "EntryPoint envelope reverted")
		})
	}
}
