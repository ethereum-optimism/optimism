//go:build atomic_integration && atomic_suspension

package atomic

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

type memoryPrefix struct{ state *state.StateDB }

func (s memoryPrefix) Account(_ context.Context, address common.Address) (*PrefixAccount, error) {
	if !s.state.Exist(address) {
		return nil, nil
	}
	return &PrefixAccount{Balance: (*hexutil.Big)(s.state.GetBalance(address).ToBig()), Nonce: s.state.GetNonce(address), Code: s.state.GetCode(address)}, nil
}
func (s memoryPrefix) Storage(_ context.Context, address common.Address, index common.Hash) (common.Hash, error) {
	return s.state.GetState(address, index), nil
}
func (s memoryPrefix) BlockHash(context.Context, uint64) (common.Hash, error) {
	return common.Hash{}, nil
}

func TestSuspendedSponsoredWorker(t *testing.T) {
	binary := os.Getenv("ATOMIC_BUILDER_BIN")
	require.NotEmpty(t, binary, "build the Rust worker and set ATOMIC_BUILDER_BIN")
	for _, mode := range []string{"success", "remote_revert", "missing_final_access_list", "prefix_gas_exhausted", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			fail := mode == "remote_revert"
			f, executors := newSponsoredFixture(t)
			if fail {
				backend := executors[f.remote].Backend.(*EVMExecutor)
				data, err := aaArtifact(t, "AtomicCounter").ABI.Pack("setLimit", big.NewInt(5))
				require.NoError(t, err)
				aaApply(t, backend, f.sender, &f.counter, data, new(big.Int))
			}
			key, err := crypto.GenerateKey()
			require.NoError(t, err)
			bundler := crypto.PubkeyToAddress(key.PublicKey)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var chains []SuspendedChain
			prepares := make(map[uint64]int)
			for id, e := range executors {
				backend := e.Backend.(*EVMExecutor)
				backend.State.SetBalance(bundler, uint256.NewInt(1_000_000_000_000_000_000), tracing.BalanceChangeUnspecified)
				e.Bundler = bundler
				chain := f.b.Chains[id]
				n := bigs.Uint64Strict(id.ToBig())
				chains = append(chains, SuspendedChain{Chain: n, Router: f.b.Router, Sender: e.Account, ApplicationGas: 1_000_000,
					PrefixLogs: chain.FirstLogIndex - 1, Spec: "LAGOON", Number: chain.BlockNumber, Timestamp: chain.Timestamp, GasLimit: backend.Block.GasLimit, BaseFee: 7,
					Snapshot: memoryPrefix{backend.State.Copy()}, Prepare: func(ctx context.Context, data []byte, accesses types.AccessList) (*types.Transaction, error) {
						prepares[n]++
						if mode == "cancel" {
							cancel()
							return nil, ctx.Err()
						}
						if mode == "missing_final_access_list" && prepares[n] == 2 {
							accesses = nil
						}
						_, tx, err := e.Prepare(ctx, Transaction{From: e.Account, To: f.b.Router, Data: data, AccessList: accesses, Gas: 8_000_000})
						if err != nil {
							return nil, err
						}
						return types.SignNewTx(key, types.LatestSignerForChainID(id.ToBig()), &types.DynamicFeeTx{ChainID: id.ToBig(), To: &tx.To, Gas: tx.Gas, GasFeeCap: tx.GasFeeCap, GasTipCap: tx.GasTipCap, Data: tx.Data, AccessList: tx.AccessList})
					}})
			}
			if mode == "prefix_gas_exhausted" {
				for i := range chains {
					chains[i].PrefixGas = chains[i].GasLimit - 1
				}
			}
			data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(6))
			require.NoError(t, err)
			result, err := BuildSuspended(ctx, binary, chains, 901, 0, f.appAddr, data, 8)
			if mode == "missing_final_access_list" || mode == "prefix_gas_exhausted" || mode == "cancel" {
				require.Error(t, err)
				require.Nil(t, result)
				for _, count := range prepares {
					require.LessOrEqual(t, count, 2)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, fail, result.Reverted)
			require.Len(t, result.Included, 2)
			for id, inclusion := range result.Included {
				require.Equal(t, 2, prepares[id])
				require.True(t, inclusion.Success, "EntryPoint catches the operation revert")
				require.Positive(t, inclusion.GasUsed)
				require.NotEmpty(t, inclusion.Logs)
			}
		})
	}
}
