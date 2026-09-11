package superfaultproofs

import (
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/atomic"
)

// AtomicCallScenario selects the successful call, propagated remote revert,
// or adversarial inclusion of an orphaned successful destination.
type AtomicCallScenario string

const (
	AtomicCallsSucceed     AtomicCallScenario = "TwoRoundTrips"
	AtomicRemoteReverts    AtomicCallScenario = "RemoteReverts"
	AtomicOrphanedRemote   AtomicCallScenario = "OrphanedRemote"
	AtomicSponsoredSuccess AtomicCallScenario = "SponsoredSuccess"
	AtomicSponsoredRevert  AtomicCallScenario = "SponsoredRemoteReverts"
)

const (
	AtomicSuspendedSuccess AtomicCallScenario = "SuspendedSponsoredSuccess"
	AtomicSuspendedRevert  AtomicCallScenario = "SuspendedSponsoredRemoteReverts"
)

const (
	AtomicNestedSuccess AtomicCallScenario = "NestedSponsoredSuccess"
	AtomicNestedRevert  AtomicCallScenario = "NestedSponsoredCaughtRevert"
)

func (s AtomicCallScenario) nested() bool {
	return s == AtomicNestedSuccess || s == AtomicNestedRevert
}

func (s AtomicCallScenario) suspended() bool {
	return s == AtomicSuspendedSuccess || s == AtomicSuspendedRevert || s.nested()
}

func (s AtomicCallScenario) sponsored() bool {
	return s == AtomicSponsoredSuccess || s == AtomicSponsoredRevert || s.suspended()
}

func (s AtomicCallScenario) remoteReverts() bool {
	return s == AtomicRemoteReverts || s == AtomicSponsoredRevert || s == AtomicSuspendedRevert || s == AtomicNestedRevert
}

// RunAtomicCallConsolidationTest deploys the opt-in facade and constructs two
// actual transactions with the shared builder. The application makes two typed
// calls to the remote counter, with the second depending on the first result.
// Rejecting at the end of A also invalidates B through normal consolidation.
func RunAtomicCallConsolidationTest(t devtest.T, sys *presets.SimpleInterop, scenario AtomicCallScenario, runners ...ProofRunner) {
	RunIntraBlockConsolidationTest(t, sys, atomicCallCase(t, sys, scenario), runners...)
}

// RunAtomicCallVerificationTest runs the same real transactions and supernode
// verification without provisioning a fault-proof VM. The separate proof test
// additionally checks the resulting transition in Kona and the challenger.
func RunAtomicCallVerificationTest(t devtest.T, sys *presets.TwoL2SupernodeInterop, scenario AtomicCallScenario) {
	view := &presets.SimpleInterop{SingleChainInterop: presets.SingleChainInterop{
		T: t, L1Network: sys.L1Network, L1EL: sys.L1EL, L1CL: sys.L1CL,
		SuperRoots: sys.Supernode, TestSequencer: sys.TestSequencer,
		L2ChainA: sys.L2A, L2ELA: sys.L2ELA, L2CLA: sys.L2ACL, FunderA: sys.FunderA,
	}, L2ChainB: sys.L2B, L2ELB: sys.L2ELB, L2CLB: sys.L2BCL, FunderB: sys.FunderB}
	tc := atomicCallCase(t, view, scenario)
	tc.Prepare(t, view, sys.FunderA.NewFundedEOA(eth.OneEther), sys.FunderB.NewFundedEOA(eth.OneEther))
	s := sys.ForSameTimestampTesting(t)
	if scenario.suspended() {
		// Preview prefixes must never reach L1 while we retain them for discovery.
		sys.L2BatcherA.Stop()
		sys.L2BatcherB.Stop()
	}
	parentA := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	parentB := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	t.Logger().Info("discovering atomic bundle", "timestamp", s.NextTimestamp)
	txsA, txsB := tc.BuildTxs(&intraBlockSetup{alice: s.Alice, bob: s.Bob, expectedBlockNumA: s.ExpectedBlockNumA, expectedBlockNumB: s.ExpectedBlockNumB, nextTimestamp: s.NextTimestamp})
	t.Logger().Info("including atomic bundle", "timestamp", s.NextTimestamp)
	// Propagated failure retains both reverted transactions. Only deliberate
	// inclusion of an orphaned successful B requires block replacement.
	if scenario.suspended() {
		s.IncludeAndValidateOnParents(txsA, txsB, parentA, parentB, false, false, func() {
			// Both final siblings are now canonical. Only these candidates may be batched.
			sys.L2BatcherA.Start()
			sys.L2BatcherB.Start()
		})
	} else {
		s.IncludeAndValidate(txsA, txsB, false, scenario == AtomicOrphanedRemote)
	}
	if scenario != AtomicOrphanedRemote {
		status := uint64(types.ReceiptStatusSuccessful)
		if scenario == AtomicRemoteReverts {
			status = types.ReceiptStatusFailed
		}
		for _, chain := range []struct {
			el  *dsl.L2ELNode
			txs []*txplan.PlannedTx
		}{{sys.L2ELA, txsA}, {sys.L2ELB, txsB}} {
			for _, tx := range chain.txs {
				// These transactions were inserted by the shared payload builder;
				// Included.Eval would submit them a second time through the pool.
				receipt, err := chain.el.EthClient().TransactionReceipt(t.Ctx(), tx.Signed.Value().Hash())
				t.Require().NoError(err)
				t.Require().Equal(status, receipt.Status, "both transaction receipts must match the call outcome")
			}
		}
	}
	tc.Check(t, view)
}

func atomicCallCase(t devtest.T, sys *presets.SimpleInterop, scenario AtomicCallScenario) *IntraBlockTestCase {
	wd, err := os.Getwd()
	t.Require().NoError(err)
	root, err := opservice.FindMonorepoRoot(wd)
	t.Require().NoError(err)
	read := func(source, name string) *foundry.Artifact {
		a, err := foundry.ReadArtifact(filepath.Join(root, "packages/contracts-bedrock/forge-artifacts", source+".sol", name+".json"))
		t.Require().NoError(err, "build the atomic contract artifacts first")
		return a
	}
	router := read("AtomicCallRouter", "AtomicCallRouter")
	app := read("AtomicCallExample", "AtomicCallExample")
	counter := read("AtomicCounter", "AtomicCounter")
	if scenario.nested() {
		app = read("AtomicNestedFixture.s", "AtomicCallRouter_Nested_Harness")
		counter = app
	}
	var routerAddr, appAddr, counterAddr, proxy, callback, callbackTarget common.Address
	blockNumbers := make(map[eth.ChainID]uint64)
	sponsors := make(map[eth.ChainID]*sponsoredAccount)
	plannedTransactions := make(map[eth.ChainID]*txplan.PlannedTx)
	var suspendedResult *atomic.SuspendedResult
	deploy := func(eoa *dsl.EOA, artifact *foundry.Artifact) common.Address {
		tx := txplan.NewPlannedTx(eoa.Plan(), txplan.WithData(artifact.Bytecode.Object))
		receipt, err := tx.Included.Eval(t.Ctx())
		t.Require().NoError(err)
		t.Require().Equal(types.ReceiptStatusSuccessful, receipt.Status)
		return receipt.ContractAddress
	}
	return &IntraBlockTestCase{
		Name: string(scenario), ExpectReplacement: scenario == AtomicOrphanedRemote,
		Prepare: func(t devtest.T, sys *presets.SimpleInterop, alice, bob *dsl.EOA) {
			// One fresh deployer, nonce zero on both chains, yields identical router addresses.
			deployerA := sys.FunderA.NewFundedEOA(eth.OneEther)
			deployerB := deployerA.AsEL(sys.L2ELB)
			bob.Transfer(deployerB.Address(), eth.OneEther.Div(2))
			routerAddr = deploy(deployerA, router)
			t.Require().Equal(routerAddr, deploy(deployerB, router))
			appAddr = deploy(alice, app)
			counterAddr = deploy(bob, counter)
			if scenario.remoteReverts() && !scenario.nested() {
				data, err := counter.ABI.Pack("setLimit", big.NewInt(5))
				t.Require().NoError(err)
				bob.Transact(bob.Plan(), txplan.WithTo(&counterAddr), txplan.WithData(data))
			}
			input, err := router.ABI.Pack("proxyFor", bob.ChainID().ToBig(), counterAddr)
			t.Require().NoError(err)
			tx := alice.Transact(alice.Plan(), txplan.WithTo(&routerAddr), txplan.WithData(input))
			receipt, err := tx.Included.Eval(t.Ctx())
			t.Require().NoError(err)
			for _, log := range receipt.Logs {
				if log.Address == routerAddr && len(log.Topics) > 0 && log.Topics[0] == router.ABI.Events["ProxyCreated"].ID {
					values, err := router.ABI.Events["ProxyCreated"].Inputs.NonIndexed().Unpack(log.Data)
					t.Require().NoError(err)
					proxy = values[0].(common.Address)
				}
			}
			t.Require().NotEqual(common.Address{}, proxy)
			if scenario.nested() {
				artifact := read("AtomicNestedFixture.s", "AtomicCallRouter_NestedCallback_Harness")
				callbackTarget = deploy(alice, artifact)
				configure, err := artifact.ABI.Pack("configure", appAddr)
				t.Require().NoError(err)
				alice.Transact(alice.Plan(), txplan.WithTo(&callbackTarget), txplan.WithData(configure))
				input, err := router.ABI.Pack("proxyFor", alice.ChainID().ToBig(), callbackTarget)
				t.Require().NoError(err)
				tx := bob.Transact(bob.Plan(), txplan.WithTo(&routerAddr), txplan.WithData(input))
				receipt, err := tx.Included.Eval(t.Ctx())
				t.Require().NoError(err)
				for _, log := range receipt.Logs {
					if log.Address == routerAddr && len(log.Topics) > 0 && log.Topics[0] == router.ABI.Events["ProxyCreated"].ID {
						values, err := router.ABI.Events["ProxyCreated"].Inputs.NonIndexed().Unpack(log.Data)
						t.Require().NoError(err)
						callback = values[0].(common.Address)
					}
				}
				t.Require().NotEqual(common.Address{}, callback)
			}
			if scenario.sponsored() {
				sponsors[alice.ChainID()] = prepareSponsoredAccount(t, alice, sys.L2ELA, routerAddr, read, deploy)
				sponsors[bob.ChainID()] = prepareSponsoredAccount(t, bob, sys.L2ELB, routerAddr, read, deploy)
			}
		},
		BuildTxs: func(s *intraBlockSetup) ([]*txplan.PlannedTx, []*txplan.PlannedTx) {
			if scenario.suspended() {
				var data []byte
				var err error
				if scenario.nested() {
					failure := int64(0)
					if scenario.remoteReverts() {
						failure = 4
					}
					data, err = app.ABI.Pack("run", proxy, callback, big.NewInt(failure))
				} else {
					data, err = app.ABI.Pack("run", proxy, big.NewInt(3), big.NewInt(6))
				}
				t.Require().NoError(err)
				txs, result := buildSuspendedAtomic(t, sys, s, routerAddr, appAddr, data, sponsors)
				suspendedResult = result
				for id, tx := range txs {
					plannedTransactions[id] = tx
				}
				blockNumbers[s.alice.ChainID()] = s.expectedBlockNumA
				blockNumbers[s.bob.ChainID()] = s.expectedBlockNumB
				t.Require().Equal(scenario.remoteReverts(), result.Reverted)
				return []*txplan.PlannedTx{txs[s.alice.ChainID()]}, []*txplan.PlannedTx{txs[s.bob.ChainID()]}
			}
			builder := &atomic.Builder{Router: routerAddr, ABI: router.ABI, MaxCalls: 8, Gas: 8_000_000, Chains: make(map[eth.ChainID]atomic.Chain)}
			for _, item := range []struct {
				eoa    *dsl.EOA
				el     *dsl.L2ELNode
				number uint64
			}{
				{s.alice, sys.L2ELA, s.expectedBlockNumA}, {s.bob, sys.L2ELB, s.expectedBlockNumB},
			} {
				id := item.eoa.ChainID()
				blockNumbers[id] = item.number
				backend := &atomic.RPCExecutor{RPC: item.el.EthClient().RPC(), Parent: item.el.BlockRefByLabel(eth.Unsafe).Hash, Number: item.number, Timestamp: s.nextTimestamp, MaxDiscoveryPasses: 16}
				chain := atomic.Chain{ID: id, BlockNumber: item.number, Timestamp: s.nextTimestamp, Sender: item.eoa.Address(), Executor: backend}
				if scenario.sponsored() {
					chain.Executor = sponsors[id].executor(backend, item.eoa)
					chain.Sender = sponsors[id].account
					chain.FirstLogIndex = 1 // EntryPoint.BeforeExecution
				}
				builder.Chains[id] = chain
			}
			data, err := app.ABI.Pack("run", proxy, big.NewInt(3), big.NewInt(6))
			t.Require().NoError(err)
			var plan *atomic.Plan
			var transactions map[eth.ChainID]atomic.Transaction
			if scenario.sponsored() {
				sponsored, buildErr := atomic.BuildSponsored(t.Ctx(), builder, s.alice.ChainID(), 0, appAddr, data)
				t.Require().NoError(buildErr)
				plan, transactions = sponsored.Atomic, sponsored.Transactions
			} else {
				plan, err = builder.Build(t.Ctx(), s.alice.ChainID(), 0, appAddr, data)
				t.Require().NoError(err)
				transactions = plan.Transactions
			}
			t.Require().Equal(scenario.remoteReverts(), plan.Reverted)
			if scenario.remoteReverts() {
				t.Require().False(plan.Witnesses[len(plan.Witnesses)-1].Success)
				t.Require().Equal(plan.Witnesses[len(plan.Witnesses)-1].ReturnData, plan.Executions[s.alice.ChainID()].Output)
			}
			rootTx := transactions[s.alice.ChainID()]
			if scenario == AtomicOrphanedRemote {
				// Deliberately submit the speculative remote leg alongside a root
				// transaction whose final application check reverts.
				data, err = app.ABI.Pack("run", proxy, big.NewInt(3), big.NewInt(5))
				t.Require().NoError(err)
				rootTx.Data, err = router.ABI.Pack("executeRoot", new(big.Int), appAddr, data, plan.Witnesses)
				t.Require().NoError(err)
			}
			planned := func(eoa *dsl.EOA, tx atomic.Transaction) *txplan.PlannedTx {
				planned := txplan.NewPlannedTx(eoa.Plan(), txintent.WithoutInteropDependencyWait(), txplan.WithTo(&tx.To), txplan.WithData(tx.Data), txplan.WithGasLimit(tx.Gas), txplan.WithAccessList(tx.AccessList))
				if tx.GasFeeCap != nil {
					txplan.WithGasFeeCap(tx.GasFeeCap)(planned)
					txplan.WithGasTipCap(tx.GasTipCap)(planned)
				}
				return planned
			}
			plannedTransactions[s.alice.ChainID()] = planned(s.alice, rootTx)
			plannedTransactions[s.bob.ChainID()] = planned(s.bob, transactions[s.bob.ChainID()])
			return []*txplan.PlannedTx{plannedTransactions[s.alice.ChainID()]}, []*txplan.PlannedTx{plannedTransactions[s.bob.ChainID()]}
		},
		Check: func(t devtest.T, sys *presets.SimpleInterop) {
			want := common.BigToHash(big.NewInt(6))
			if scenario.remoteReverts() || scenario == AtomicOrphanedRemote {
				want = common.Hash{}
			}
			if scenario.nested() {
				count, err := sys.L2ELA.EthClient().GetStorageAt(t.Ctx(), callbackTarget, common.BigToHash(big.NewInt(1)), hexutil.EncodeUint64(blockNumbers[sys.L2ELA.ChainID()]))
				t.Require().NoError(err)
				expected := common.BigToHash(big.NewInt(1))
				if scenario.remoteReverts() {
					expected = common.Hash{}
				}
				t.Require().Equal(expected, count, "distinct contract C executes inside A's original transaction")
			}
			for _, item := range []struct {
				el   *dsl.L2ELNode
				addr common.Address
			}{{sys.L2ELA, appAddr}, {sys.L2ELB, counterAddr}} {
				// Check the candidate height after acceptance/replacement. The EL's
				// safe label may lag the supernode's completed verification round.
				number := blockNumbers[item.el.ChainID()]
				got, err := item.el.EthClient().GetStorageAt(t.Ctx(), item.addr, common.Hash{}, hexutil.EncodeUint64(number))
				t.Require().NoError(err)
				expectedValue := want
				if scenario.nested() && !scenario.remoteReverts() {
					n := int64(10)
					if item.el.ChainID() == sys.L2ELA.ChainID() {
						n = 34
					}
					expectedValue = common.BigToHash(big.NewInt(n))
				}
				t.Require().Equal(expectedValue, got, "application state after cross-chain validation")
				if scenario.sponsored() {
					id := item.el.ChainID()
					sponsors[id].check(t, item.el, plannedTransactions[id], number, !scenario.remoteReverts())
					if suspendedResult != nil {
						expected := suspendedResult.Included[bigs.Uint64Strict(id.ToBig())]
						receipt, err := item.el.EthClient().TransactionReceipt(t.Ctx(), expected.Transaction.Hash())
						t.Require().NoError(err)
						t.Require().Equal(expected.GasUsed, receipt.GasUsed, "node gas matches the exact signed canonical replay")
						t.Require().Len(receipt.Logs, len(expected.Logs))
						for i, log := range receipt.Logs {
							t.Require().Equal(expected.Logs[i].Address, log.Address)
							t.Require().Equal(expected.Logs[i].Topics, log.Topics)
							t.Require().Equal(expected.Logs[i].Data, log.Data)
						}
					}
				}
			}
		},
	}
}
