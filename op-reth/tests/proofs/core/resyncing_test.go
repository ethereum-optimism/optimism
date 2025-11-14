package core

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op-rs/op-geth/proofs/utils"
	"github.com/stretchr/testify/require"
)

func TestResyncing(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSingleChainMultiNode(t)

	alice := sys.FunderL2.NewFundedEOA(eth.OneEther)
	bob := sys.FunderL2.NewFundedEOA(eth.OneEther)

	tx := alice.Transfer(bob.Address(), eth.OneHundredthEther)
	receipt, err := tx.Included.Eval(ctx)
	require.NoError(gt, err)
	require.Equal(gt, types.ReceiptStatusSuccessful, receipt.Status)

	t.Logf("Stopping L2 CLB and ELB to simulate downtime")
	sys.L2CLB.Stop()
	sys.L2ELB.Stop()

	var blockNumbers []uint64
	// produce some transactions while the node is down
	for i := 0; i < 5; i++ {
		tx := alice.Transfer(bob.Address(), eth.OneHundredthEther)
		receipt, err := tx.Included.Eval(ctx)
		require.NoError(gt, err)
		require.Equal(gt, types.ReceiptStatusSuccessful, receipt.Status)
		blockNumbers = append(blockNumbers, receipt.BlockNumber.Uint64())
	}

	// restart the node and ensure it can sync the missing blocks
	t.Logf("Restarting L2 CLB and ELB to resync")
	sys.L2ELB.Start()
	sys.L2CLB.Start()

	time.Sleep(3 * time.Second)

	err = wait.For(t.Ctx(), 2*time.Second, func() (bool, error) {
		status := sys.L2CLB.SyncStatus()
		return status.UnsafeL2.Number > blockNumbers[len(blockNumbers)-1], nil
	})
	require.NoError(gt, err, "L2 CLB failed to resync to latest block")

	t.Logf("Fetching and verifying proofs for transactions produced while node was down")
	// verify the proofs for the transactions produced while the node was down
	for _, blockNumber := range blockNumbers {
		utils.FetchAndVerifyProofs(t, sys, bob.Address(), []common.Hash{}, blockNumber)
		utils.FetchAndVerifyProofs(t, sys, alice.Address(), []common.Hash{}, blockNumber)
	}
}
