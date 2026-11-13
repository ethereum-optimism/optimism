package superfaultproofs

import (
	"bytes"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// The block deploys recreatedInitCode with recreatedSalt through the DeterministicDeploymentProxy
// preinstall, so the account that it destroys and re-creates has the same address on every devnet
// and in every run. The init code self-destructs to the transaction's sender.
var (
	recreatedSalt     = common.HexToHash("0x1f4082")
	recreatedInitCode = program.New().Op(vm.ORIGIN, vm.SELFDESTRUCT).Bytes()
	// recreatedSibling's state-trie key shares its first 11 nibbles (f36a7482b80) with the
	// re-created account's, so the two accounts are alone under the branch where their keys part
	// unless another account's key shares those 11 nibbles. requireBranchOfTwo checks that none does.
	//
	// recreatedSalt and recreatedSibling were found together by a birthday search, which must be
	// rerun if recreatedInitCode or the proxy changes. For each salt from 0 to 2^22-1, record the
	// salt under the first 11 nibbles of keccak256(crypto.CreateAddress2(proxy, salt,
	// keccak256(recreatedInitCode))). Then, for n = 0, 1, 2, ..., take the address with 5eed in its
	// first 2 bytes, n big-endian in its last 8 bytes and zeros between, and stop at the first whose
	// keccak256 starts with recorded nibbles: that address is recreatedSibling and the salt recorded
	// under those nibbles is recreatedSalt. The search stopped at n = 0x4961d9 after about 10
	// seconds. If requireBranchOfTwo fails with a new pair, continue from n+1. An 11-nibble match
	// takes about 2^22 tries on each side, and an account created at runtime shares the 11 nibbles
	// with probability 2^-44.
	recreatedSibling = common.HexToAddress("0x5eeD0000000000000000000000000000004961d9")
)

// RunDestroyedAndRecreatedAccountTest proves a block that destroys an account and re-creates it.
// The account's leaf shares its branch with one other account, which the block does not touch, so
// the block's execution witness holds no node of that account.
func RunDestroyedAndRecreatedAccountTest(t devtest.T, sys *presets.SimpleInterop, runners ...ProofRunner) {
	chains := replacedBlockChains(t, sys)
	proxy := predeploys.DeterministicDeploymentProxyAddr
	recreated := crypto.CreateAddress2(proxy, recreatedSalt, crypto.Keccak256(recreatedInitCode))

	// Fund before freezeChains stops the sequencers.
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	bob.Transfer(recreated, eth.WeiU64(1))
	bob.Transfer(recreatedSibling, eth.WeiU64(1))

	freezeChains(chains)
	endTimestamp := nextTimestampAfterSafeHeads(t, chains)
	parentA, parentB := advanceToReplacedBlockParents(t, sys, chains, endTimestamp)
	requireBranchOfTwo(t, sys.L2ELB.AccountProofAt(recreated, parentB.Number))

	sys.TestSequencer.SequenceBlock(t, chains[0].ID, parentA.Hash)
	// The CREATE2 takes over recreated's balance and self-destructs in the transaction that created
	// it, which deletes the account. The transfer then re-creates it.
	sys.TestSequencer.SequenceBlockWithPlannedTxs(t, parentB.Hash, bob,
		txplan.NewPlannedTx(bob.Plan(), txplan.WithTo(&proxy),
			txplan.WithData(append(recreatedSalt.Bytes(), recreatedInitCode...)), txplan.WithGasLimit(100_000)),
		txplan.NewPlannedTx(bob.PlanTransfer(recreated, eth.WeiU64(1)), txplan.WithGasLimit(21_000)))
	after := sys.L2ELB.AccountProofAt(recreated, parentB.Number+1)
	t.Require().Equal(uint64(1), bigs.Uint64Strict(after.Balance.ToInt()),
		"%s must hold only the re-creating transfer's 1 wei; more means the CREATE2 failed", recreated)
	t.Require().Zero(uint64(after.Nonce),
		"%s must be re-created with nonce 0; nonce 1 means the CREATE2's account outlived its SELFDESTRUCT", recreated)

	advanceSafeToCurrentUnsafe(t, chains[0])
	advanceSafeToCurrentUnsafe(t, chains[1])
	sys.SuperRoots.AwaitValidatedTimestamp(endTimestamp)

	queryAPI := sys.SuperRoots.QueryAPI()
	start := superRootAtTimestamp(t, chains, endTimestamp-1)
	first := optimisticBlockAtTimestamp(t, queryAPI, chains[0].ID, endTimestamp)
	second := optimisticBlockAtTimestamp(t, queryAPI, chains[1].ID, endTimestamp)
	runScenarioProofs(t, &sys.SingleChainInterop, chains, &scenarioProofData{
		fpvmTransitions: []*transitionTest{{
			Name:               "SecondChainOptimisticBlock",
			AgreedClaim:        marshalTransition(start, 1, first),
			DisputedClaim:      marshalTransition(start, 2, first, second),
			DisputedTraceIndex: 1,
			L1Head:             latestRequiredL1(sys.SuperRoots.SuperRootAtTimestamp(endTimestamp)),
			ClaimTimestamp:     endTimestamp,
			ExpectValid:        true,
		}},
		fpvmStartTimestamp: endTimestamp - 1,
		zkCheckpoint:       newZKCheckpointForRunners(t, &sys.SingleChainInterop, endTimestamp, false, runners),
	}, runners...)
}

// requireBranchOfTwo requires that the account's leaf hangs off a branch with exactly two
// children. Deleting the leaf collapses that branch onto the other child, which is on no path of
// the account's proof.
func requireBranchOfTwo(t devtest.T, account *eth.AccountResult) {
	proof := account.AccountProof
	t.Require().GreaterOrEqual(len(proof), 2, "%s's leaf must sit below a branch", account.Address)
	var branch []rlp.RawValue
	t.Require().NoError(rlp.DecodeBytes(proof[len(proof)-2], &branch))
	t.Require().Len(branch, 17, "the node above %s's leaf must be a branch", account.Address)
	children := 0
	for _, child := range branch[:16] {
		if !bytes.Equal(child, rlp.EmptyString) {
			children++
		}
	}
	t.Require().Equal(2, children, "the branch above %s's leaf must have exactly two children", account.Address)
}
