package reorg

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op-rs/op-geth/proofs/utils"
	"github.com/stretchr/testify/require"
)

func TestReorgUsingAccountProof(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := utils.NewMixedOpProofPreset(t)
	l := sys.Log

	ia := sys.TestSequencer.Escape().ControlAPI(sys.L2Chain.ChainID())

	// stop batcher on chain A
	sys.L2Batcher.Stop()

	// two EOAs for a sample transfer tx used later in a conflicting block
	alice := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderL2.NewFundedEOA(eth.OneHundredthEther)

	user := sys.FunderL2.NewFundedEOA(eth.OneEther)
	contract, deployBlock := utils.DeploySimpleStorage(t, user)
	t.Logf("SimpleStorage deployed at %s block=%d", contract.Address().Hex(), deployBlock.BlockNumber.Uint64())

	time.Sleep(12 * time.Second)
	divergenceHead := sys.L2Chain.WaitForBlock()
	// build up some blocks that will be reorged away

	type caseEntry struct {
		Block uint64
		addr  common.Address
		slots []common.Hash
	}
	var cases []caseEntry

	// deploy another contract in the reorged blocks
	{
		rContract, rDeployBlock := utils.DeploySimpleStorage(t, user)
		t.Logf("Reorg SimpleStorage deployed at %s block=%d", rContract.Address().Hex(), rDeployBlock.BlockNumber.Uint64())

		cases = append(cases, caseEntry{
			Block: rDeployBlock.BlockNumber.Uint64(),
			addr:  rContract.Address(),
			slots: []common.Hash{common.HexToHash("0x0")},
		})
	}

	for i := 0; i < 3; i++ {
		tx := alice.Transfer(bob.Address(), eth.OneGWei)
		receipt, err := tx.Included.Eval(ctx)
		require.NoError(gt, err)
		require.Equal(gt, types.ReceiptStatusSuccessful, receipt.Status)

		cases = append(cases, caseEntry{
			Block: receipt.BlockNumber.Uint64(),
			addr:  alice.Address(),
			slots: []common.Hash{},
		})
		cases = append(cases, caseEntry{
			Block: receipt.BlockNumber.Uint64(),
			addr:  bob.Address(),
			slots: []common.Hash{},
		})

		// also include the contract account in the proofs to verify
		val := big.NewInt(int64(i * 10))
		callRes := contract.SetValue(user, val)

		cases = append(cases, caseEntry{
			Block: callRes.BlockNumber.Uint64(),
			addr:  contract.Address(),
			slots: []common.Hash{common.HexToHash("0x0")},
		})
	}

	sys.L2CLSequencer.StopSequencer()

	var divergenceBlockNumber uint64
	var originalRef eth.L2BlockRef
	// prepare and sequence a conflicting block for the L2A chain
	{
		divergenceBlockRef := sys.L2ELSequencerNode().BlockRefByNumber(divergenceHead.Number)

		l.Info("Expect to reorg the chain on block", "number", divergenceBlockRef.Number, "head", divergenceHead, "parent", divergenceBlockRef.ParentID().Hash)
		divergenceBlockNumber = divergenceBlockRef.Number
		originalRef = divergenceBlockRef

		parentOfDivergenceHead := divergenceBlockRef.ParentID()

		l.Info("Sequencing a conflicting block", "divergenceBlockRef", divergenceBlockRef, "parent", parentOfDivergenceHead)

		// sequence a conflicting block with a simple transfer tx, based on the parent of the parent of the unsafe head
		{
			err := ia.New(ctx, seqtypes.BuildOpts{
				Parent:   parentOfDivergenceHead.Hash,
				L1Origin: nil,
			})
			require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

			// include simple transfer tx in opened block
			{
				to := bob.PlanTransfer(alice.Address(), eth.OneGWei)
				opt := txplan.Combine(to)
				ptx := txplan.NewPlannedTx(opt)
				signed_tx, err := ptx.Signed.Eval(ctx)
				require.NoError(t, err, "Expected to be able to evaluate a planned transaction on op-test-sequencer, but got error")
				txdata, err := signed_tx.MarshalBinary()
				require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")

				err = ia.IncludeTx(ctx, txdata)
				require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")

				cases = append(cases, caseEntry{
					Block: divergenceHead.Number,
					addr:  alice.Address(),
					slots: []common.Hash{},
				})
				cases = append(cases, caseEntry{
					Block: divergenceHead.Number,
					addr:  bob.Address(),
					slots: []common.Hash{},
				})
			}

			err = ia.Next(ctx)
			require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		}
	}

	// start batcher on chain A
	sys.L2Batcher.Start()

	// sequence a second block with op-test-sequencer (no L1 origin override)
	{
		l.Info("Sequencing with op-test-sequencer (no L1 origin override)")
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   sys.L2ELSequencerNode().BlockRefByLabel(eth.Unsafe).Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)
	}

	// continue sequencing with consensus node (op-node)
	sys.L2CLSequencer.StartSequencer()

	for i := 0; i < 3; i++ {
		sys.L2Chain.WaitForBlock()
	}

	latestBlock := sys.L2Chain.WaitForBlock()
	sys.L2ELValidatorNode().WaitForBlockNumber(latestBlock.Number)

	// verify that the L2A validator has reorged and reached the latest block
	err := wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		blockRef, err := sys.L2ELValidatorNode().Escape().EthClient().BlockRefByNumber(ctx, latestBlock.Number)
		if err != nil {
			// this could happen if the validator is still syncing after reorg
			l.Warn("Error fetching block reference from validator", "error", err)
			return false, nil
		}
		return blockRef.Hash == latestBlock.Hash, nil
	})
	require.NoError(t, err, "Expected block hash to match latest block hash on validator")

	reorgedRef_A, err := sys.L2ELSequencerNode().Escape().EthClient().BlockRefByNumber(ctx, divergenceBlockNumber)
	require.NoError(t, err, "Expected to be able to call BlockRefByNumber API, but got error")

	l.Info("Reorged chain on divergence block number (prior the reorg)", "number", divergenceBlockNumber, "head", originalRef.Hash, "parent", originalRef.ParentID().Hash)
	l.Info("Reorged chain on divergence block number (after the reorg)", "number", divergenceBlockNumber, "head", reorgedRef_A.Hash, "parent", reorgedRef_A.ParentID().Hash)
	require.NotEqual(t, originalRef.Hash, reorgedRef_A.Hash, "Expected to get different heads on divergence block number, but got the same hash, so no reorg happened on chain A")
	require.Equal(t, originalRef.ParentID().Hash, reorgedRef_A.ParentHash, "Expected to get same parent hashes on divergence block number, but got different hashes")

	time.Sleep(10 * time.Second)

	// verify that the accounts involved in the conflicting blocks
	for i, c := range cases {
		l.Info("Verifying proof", "case", i, "addr", c.addr.Hex(), "block", c.Block)
		utils.FetchAndVerifyProofs(t, sys, c.addr, c.slots, c.Block)
	}
}
