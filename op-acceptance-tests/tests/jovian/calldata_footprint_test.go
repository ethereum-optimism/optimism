package jovian

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func sendTx(t devtest.T, eoa *dsl.EOA, to common.Address, calldataSize int) {
	data := make([]byte, calldataSize)
	_, err := rand.Read(data)
	t.Require().NoError(err)
	eoa.Transact(
		eoa.Plan(),
		txplan.WithTo(&to),
		txplan.WithData(data),
	)
	fmt.Println("sent tx", eoa.Address())
}

// TestCalldataFootprint generates a large volume of transactions with configurable calldata size.
// It is used to test the calldata footprint response of the L2 under Jovian HF.
func TestCalldataFootprint(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// check that the chain is running on Jovian fork
	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Jovian)
	require.NoError(t, err, "Jovian fork must be active for this test")

	calldataSize := 120_000
	eoaGroupSize := 50

	// create eoas
	eoaGroup := make([]*dsl.EOA, eoaGroupSize)
	for i := range eoaGroup {
		eoaGroup[i] = sys.FunderL2.NewFundedEOA(eth.HundredEther)
	}

	// all eoas send tx to the same recipient simultaneously
	to := eoaGroup[0].Address()
	wg := sync.WaitGroup{}
	for i := range eoaGroup {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sendTx(t, eoaGroup[i], to, calldataSize)
		}()
	}
	wg.Wait()
	fmt.Println("all txs sent")

	// get blocks of the L2 and check for the txs
	ethClient := sys.L2Chain.Escape().L2ELNodes()[0].EthClient()
	lastBlock, err := ethClient.BlockRefByLabel(t.Ctx(), eth.BlockLabel(eth.Unsafe))
	t.Require().NoError(err)

	daLimitedSeen := false
	// check for the txs in the block
	for i := range lastBlock.Number {
		info, txs, err := ethClient.InfoAndTxsByNumber(t.Ctx(), uint64(i))
		t.Require().NoError(err)
		fmt.Println("txs", txs.Len())
		fmt.Println("info", info)
		blockGasUsed := info.GasUsed()
		gasUsedAccumulated := uint64(0)
		daFootprintAccumulated := big.NewInt(0)
		for _, tx := range txs {
			receipt, err := ethClient.TransactionReceipt(t.Ctx(), tx.Hash())
			t.Require().NoError(err)
			gasUsedAccumulated += receipt.GasUsed
			daFootprintAccumulated.Add(daFootprintAccumulated, tx.RollupCostData().EstimatedDASize())
		}
		fmt.Println("blockGasUsed", blockGasUsed)
		fmt.Println("gasUsedAccumulated", gasUsedAccumulated)
		fmt.Println("daFootprintAccumulated", daFootprintAccumulated)

		// if the blockGasUsed is based on actual gasUsed, nothing to validate
		if blockGasUsed == gasUsedAccumulated {
			continue
		}

		// else, the blockGasUsed is based on da footprint,
		// so we need to validate that the calldata footprint rules were followed
		daFootprintAccumulated.Mul(daFootprintAccumulated, big.NewInt(params.DAFootprintGasScalar))
		require.Equal(t, daFootprintAccumulated, big.NewInt(int64(blockGasUsed)), "blockGasUsed is not gas or da based")
		daLimitedSeen = true
	}
	require.True(t, daLimitedSeen, "no da limited blocks seen")
}
