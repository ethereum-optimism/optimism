//go:build atomic_integration

package atomic

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type fixture struct {
	b                        *Builder
	app                      *foundry.Artifact
	root, remote             eth.ChainID
	appAddr, counter, sender common.Address
	proxies                  map[eth.ChainID]common.Address
}

func newFixture(t *testing.T, n int) *fixture {
	t.Helper()
	artifact := func(source, name string) *foundry.Artifact {
		a, err := foundry.ReadArtifact(filepath.Join("../../packages/contracts-bedrock/forge-artifacts", source+".sol", name+".json"))
		require.NoError(t, err, "build atomic contract artifacts before running with -tags atomic_integration")
		return a
	}
	router := artifact("AtomicCallRouter", "AtomicCallRouter")
	app := artifact("AtomicCallExample", "AtomicCallExample")
	counter := artifact("AtomicCounter", "AtomicCounter")
	inbox := artifact("CrossL2Inbox", "CrossL2Inbox")
	f := &fixture{b: &Builder{Router: common.HexToAddress("0x1000"), ABI: router.ABI, Chains: make(map[eth.ChainID]Chain), MaxCalls: 16, Gas: 10_000_000}, app: app, root: eth.ChainIDFromUInt64(901), remote: eth.ChainIDFromUInt64(902), appAddr: common.HexToAddress("0x2000"), counter: common.HexToAddress("0x3000"), sender: common.HexToAddress("0x4000"), proxies: make(map[eth.ChainID]common.Address)}
	for i := range n {
		id := eth.ChainIDFromUInt64(uint64(901 + i))
		st, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		require.NoError(t, err)
		st.SetCode(f.b.Router, router.DeployedBytecode.Object, tracing.CodeChangeUnspecified)
		st.SetCode(f.appAddr, app.DeployedBytecode.Object, tracing.CodeChangeUnspecified)
		st.SetCode(f.counter, counter.DeployedBytecode.Object, tracing.CodeChangeUnspecified)
		st.SetCode(predeploys.CrossL2InboxAddr, inbox.DeployedBytecode.Object, tracing.CodeChangeUnspecified)
		st.SetBalance(f.sender, uint256.NewInt(1_000_000_000_000_000_000), tracing.BalanceChangeUnspecified)
		config := *params.AllDevChainProtocolChanges
		config.ChainID = id.ToBig()
		executor := &EVMExecutor{State: st, Config: &config, Block: vm.BlockContext{CanTransfer: core.CanTransfer, Transfer: core.Transfer, GetHash: func(uint64) common.Hash { return common.Hash{} }, BlockNumber: big.NewInt(int64(20 + i)), Time: 1000, GasLimit: 30_000_000, BaseFee: big.NewInt(7), BlobBaseFee: big.NewInt(1), Difficulty: new(big.Int), Random: &common.Hash{}}}
		f.b.Chains[id] = Chain{ID: id, BlockNumber: uint64(20 + i), Timestamp: 1000, FirstLogIndex: uint32(i * 3), Sender: f.sender, Executor: executor}
	}
	for id := range f.b.Chains {
		if id == f.root {
			continue
		}
		data, err := router.ABI.Pack("proxyFor", id.ToBig(), f.counter)
		require.NoError(t, err)
		e := f.b.Chains[f.root].Executor.(*EVMExecutor)
		result, st, err := e.Execute(t.Context(), Transaction{From: f.sender, To: f.b.Router, Data: data, Gas: f.b.Gas})
		require.NoError(t, err)
		require.False(t, result.Reverted, "%x", result.Output)
		e.State = st
		f.proxies[id] = common.BytesToAddress(result.Output)
	}
	return f
}

func (f *fixture) build(t *testing.T, limit int64) (*Plan, error) {
	t.Helper()
	data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(limit))
	require.NoError(t, err)
	return f.b.Build(t.Context(), f.root, 0, f.appAddr, data)
}

func TestAtomicTwoRoundTrips(t *testing.T) {
	f := newFixture(t, 2)
	p, err := f.build(t, 6)
	require.NoError(t, err)
	require.Len(t, p.Transactions, 2)
	require.Len(t, p.Witnesses, 2)
	require.Equal(t, big.NewInt(3), new(big.Int).SetBytes(p.Witnesses[0].ReturnData))
	require.Equal(t, big.NewInt(6), new(big.Int).SetBytes(p.Witnesses[1].ReturnData))
	for id, address := range map[eth.ChainID]common.Address{f.root: f.appAddr, f.remote: f.counter} {
		e := f.b.Chains[id].Executor.(*EVMExecutor)
		_, st, err := e.Execute(t.Context(), p.Transactions[id])
		require.NoError(t, err)
		require.Equal(t, common.BigToHash(big.NewInt(6)), st.GetState(address, common.Hash{}))
		require.Equal(t, common.Hash{}, e.State.GetState(address, common.Hash{}), "discovery/replay must not mutate pinned state")
	}
	// Both actual bytecode executions retain their interleaved application and inbox logs.
	require.Len(t, p.Executions[f.root].Logs, 5)
	require.Len(t, p.Executions[f.remote].Logs, 5)

	t.Run("access list is mandatory", func(t *testing.T) {
		for id, tx := range p.Transactions {
			tx.AccessList = nil
			result, err := f.b.Chains[id].Executor.Replay(t.Context(), tx)
			require.NoError(t, err)
			require.True(t, result.Reverted)
			require.Empty(t, result.Logs)
		}
	})
	t.Run("completed operations cannot replay", func(t *testing.T) {
		for id, tx := range p.Transactions {
			e := f.b.Chains[id].Executor.(*EVMExecutor)
			_, st, err := e.Execute(t.Context(), tx)
			require.NoError(t, err)
			replay := *e
			replay.State = st
			result, err := replay.Replay(t.Context(), tx)
			require.NoError(t, err)
			require.True(t, result.Reverted)
			require.Empty(t, result.Logs)
		}
	})
	t.Run("proxy requires router session", func(t *testing.T) {
		data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(6))
		require.NoError(t, err)
		result, err := f.b.Chains[f.root].Executor.Replay(t.Context(), Transaction{From: f.sender, To: f.appAddr, Data: data, Gas: f.b.Gas})
		require.NoError(t, err)
		require.True(t, result.Reverted)
	})
	t.Run("missing root completion invalidates remote", func(t *testing.T) {
		root := p.Executions[f.root]
		defer func() { p.Executions[f.root] = root }()
		withoutCompletion := root
		withoutCompletion.Logs = root.Logs[:len(root.Logs)-1]
		p.Executions[f.root] = withoutCompletion
		require.Error(t, f.b.Verify(p))
	})
	t.Run("forged caller cannot authenticate against root request", func(t *testing.T) {
		calls := append([]RemoteCall(nil), p.Calls[f.remote]...)
		calls[0].Sender = common.HexToAddress("0xbad")
		tx, err := f.b.remoteTx(f.b.Chains[f.remote], p.BundleID, calls, p.RootCompletion)
		require.NoError(t, err)
		execution, err := f.b.Chains[f.remote].Executor.Discover(t.Context(), tx)
		require.NoError(t, err)
		require.False(t, execution.Reverted)
		var dependencies []messages.Message
		for _, log := range execution.Logs {
			msg, err := messages.MessageFromLog(log)
			require.NoError(t, err)
			if msg != nil {
				dependencies = append(dependencies, *msg)
			}
		}
		// Even a caller-supplied access list warming the forged request cannot
		// authenticate it against the actual root transaction's request log.
		tx.AccessList = accessList(dependencies)
		execution, err = f.b.Chains[f.remote].Executor.Replay(t.Context(), tx)
		require.NoError(t, err)
		require.False(t, execution.Reverted)
		original := p.Executions[f.remote]
		defer func() { p.Executions[f.remote] = original }()
		p.Executions[f.remote] = execution
		require.ErrorContains(t, f.b.Verify(p), "mismatched candidate message")
	})
	t.Run("changed result invalidates root", func(t *testing.T) {
		log := p.Executions[f.remote].Logs[1]
		original := append([]byte(nil), log.Data...)
		defer func() { log.Data = original }()
		log.Data = make([]byte, len(original))
		require.ErrorContains(t, f.b.Verify(p), "mismatched candidate message")
	})
	t.Run("wrong source index invalidates root", func(t *testing.T) {
		chain := f.b.Chains[f.remote]
		defer func() { f.b.Chains[f.remote] = chain }()
		changed := chain
		changed.FirstLogIndex++
		f.b.Chains[f.remote] = changed
		require.Error(t, f.b.Verify(p))
	})
}

func TestAtomicRootRejection(t *testing.T) {
	f := newFixture(t, 2)
	p, err := f.build(t, 5)
	require.NoError(t, err)
	require.True(t, p.Reverted)
	require.Len(t, p.Transactions, 1)
	require.True(t, p.Executions[f.root].Reverted)
	for id, address := range map[eth.ChainID]common.Address{f.root: f.appAddr, f.remote: f.counter} {
		require.Equal(t, common.Hash{}, f.b.Chains[id].Executor.(*EVMExecutor).State.GetState(address, common.Hash{}))
	}
	// Even if a malicious builder includes the destination transaction from a
	// successful discovery, a reverting root removes all its initiating logs.
	p, err = f.build(t, 6)
	require.NoError(t, err)
	data, err := f.app.ABI.Pack("run", f.proxies[f.remote], big.NewInt(3), big.NewInt(5))
	require.NoError(t, err)
	tx, err := f.b.rootTx(f.b.Chains[f.root], 0, f.appAddr, data, p.Witnesses)
	require.NoError(t, err)
	tx.AccessList = p.Transactions[f.root].AccessList
	root, err := f.b.Chains[f.root].Executor.Replay(t.Context(), tx)
	require.NoError(t, err)
	require.True(t, root.Reverted)
	require.Empty(t, root.Logs)
	p.Executions[f.root] = root
	require.Error(t, f.b.Verify(p))
}

func TestAtomicThreeChains(t *testing.T) {
	f := newFixture(t, 3)
	third := eth.ChainIDFromUInt64(903)
	data, err := f.app.ABI.Pack("runAcross", f.proxies[f.remote], f.proxies[third], big.NewInt(9))
	require.NoError(t, err)
	p, err := f.b.Build(t.Context(), f.root, 0, f.appAddr, data)
	require.NoError(t, err)
	require.Len(t, p.Transactions, 3)
	for id, address := range map[eth.ChainID]common.Address{f.root: f.appAddr, f.remote: f.counter, third: f.counter} {
		_, st, err := f.b.Chains[id].Executor.(*EVMExecutor).Execute(t.Context(), p.Transactions[id])
		require.NoError(t, err)
		require.Equal(t, common.BigToHash(big.NewInt(9)), st.GetState(address, common.Hash{}))
	}
}

func TestAtomicDiscoveryBounds(t *testing.T) {
	f := newFixture(t, 2)
	f.b.MaxCalls = 1
	_, err := f.build(t, 6)
	require.ErrorContains(t, err, "exceeds 1 calls")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = f.b.Build(ctx, f.root, 0, f.appAddr, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestAtomicNestedRouterEntry(t *testing.T) {
	f := newFixture(t, 2)
	inner, err := f.b.rootTx(f.b.Chains[f.root], 0, f.counter, nil, nil)
	require.NoError(t, err)
	outer, err := f.b.rootTx(f.b.Chains[f.root], 0, f.b.Router, inner.Data, nil)
	require.NoError(t, err)
	execution, err := f.b.Chains[f.root].Executor.Replay(t.Context(), outer)
	require.NoError(t, err)
	require.True(t, execution.Reverted)
	selector := f.b.ABI.Errors["AtomicCallRouter_AlreadyEntered"].ID
	require.Equal(t, selector[:4], execution.Output)
	require.Empty(t, execution.Logs)
}

func setRemoteLimit(t *testing.T, f *fixture, id eth.ChainID, limit int64) {
	t.Helper()
	counter, err := foundry.ReadArtifact("../../packages/contracts-bedrock/forge-artifacts/AtomicCounter.sol/AtomicCounter.json")
	require.NoError(t, err)
	data, err := counter.ABI.Pack("setLimit", big.NewInt(limit))
	require.NoError(t, err)
	executor := f.b.Chains[id].Executor.(*EVMExecutor)
	result, st, err := executor.Execute(t.Context(), Transaction{From: f.sender, To: f.counter, Data: data, Gas: f.b.Gas})
	require.NoError(t, err)
	require.False(t, result.Reverted)
	executor.State = st
}

func TestAtomicRemoteRevertPropagates(t *testing.T) {
	f := newFixture(t, 2)
	setRemoteLimit(t, f, f.remote, 5)
	p, err := f.build(t, 6)
	require.NoError(t, err)
	require.True(t, p.Reverted)
	require.Len(t, p.Transactions, 2)
	require.Len(t, p.Witnesses, 2)
	require.True(t, p.Witnesses[0].Success, "B's first operation succeeds")
	require.False(t, p.Witnesses[1].Success, "B's second operation reverts")
	require.Equal(t, p.Witnesses[1].ReturnData, p.Executions[f.root].Output, "A bubbles B's exact revert")
	for id, address := range map[eth.ChainID]common.Address{f.root: f.appAddr, f.remote: f.counter} {
		result, st, err := f.b.Chains[id].Executor.(*EVMExecutor).Execute(t.Context(), p.Transactions[id])
		require.NoError(t, err)
		require.True(t, result.Reverted)
		require.Empty(t, result.Logs)
		require.Equal(t, common.Hash{}, st.GetState(address, common.Hash{}), "earlier successful writes must roll back")
	}
	require.NoError(t, f.b.Verify(p))
	result := p.Executions[f.remote]
	result.Reverted = false
	p.Executions[f.remote] = result
	require.Error(t, f.b.Verify(p), "an abort plan cannot include successful remote work")
}

func TestAtomicAbortOmitsSuccessfulChain(t *testing.T) {
	f := newFixture(t, 3)
	third := eth.ChainIDFromUInt64(903)
	setRemoteLimit(t, f, third, 5)
	data, err := f.app.ABI.Pack("runAcross", f.proxies[f.remote], f.proxies[third], big.NewInt(9))
	require.NoError(t, err)
	p, err := f.b.Build(t.Context(), f.root, 0, f.appAddr, data)
	require.NoError(t, err)
	require.True(t, p.Reverted)
	require.Len(t, p.Transactions, 2)
	require.NotContains(t, p.Transactions, f.remote, "omit the earlier successful destination")
	for id, tx := range p.Transactions {
		result, st, err := f.b.Chains[id].Executor.(*EVMExecutor).Execute(t.Context(), tx)
		require.NoError(t, err)
		require.True(t, result.Reverted)
		require.Empty(t, result.Logs)
		address := f.counter
		if id == f.root {
			address = f.appAddr
		}
		require.Equal(t, common.Hash{}, st.GetState(address, common.Hash{}))
	}
	require.Equal(t, common.Hash{}, f.b.Chains[f.remote].Executor.(*EVMExecutor).State.GetState(f.counter, common.Hash{}))
}

func TestAtomicFailureHintCannotCommit(t *testing.T) {
	for _, caught := range []bool{true, false} {
		name := "unused"
		if caught {
			name = "caught"
		}
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, 2)
			var target common.Address
			var data []byte
			var err error
			if caught {
				target = f.appAddr
				data, err = f.app.ABI.Pack("catchRemoteFailure", f.proxies[f.remote])
				require.NoError(t, err)
			} else {
				target = f.sender // EOA ignores the unused tape and returns success.
			}
			reason := []byte{0xde, 0xad, 0xbe, 0xef}
			tx, err := f.b.rootTx(f.b.Chains[f.root], 0, target, data, []ResultWitness{{Identifier: f.b.identifier(f.b.Chains[f.remote], 0), ReturnData: reason}})
			require.NoError(t, err)
			result, st, err := f.b.Chains[f.root].Executor.(*EVMExecutor).Execute(t.Context(), tx)
			require.NoError(t, err)
			require.True(t, result.Reverted)
			require.Equal(t, reason, result.Output)
			require.Empty(t, result.Logs)
			require.Equal(t, common.Hash{}, st.GetState(f.appAddr, common.Hash{}))
		})
	}
}
