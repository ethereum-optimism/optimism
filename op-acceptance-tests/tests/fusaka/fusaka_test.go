package fusaka

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func TestSafeHeadAdvancesAfterOsaka(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	l1Config := sys.L1Network.Escape().ChainConfig()
	t.Log("Waiting for Osaka to activate")
	t.Require().NotNil(l1Config.OsakaTime)
	sys.L1EL.WaitForTime(*l1Config.OsakaTime)
	t.Log("Osaka activated")

	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	for range time.Tick(l2BlockTime) {
		l2SafeRef := sys.L2EL.BlockRefByLabel(eth.Safe)
		if l1Config.IsOsaka(new(big.Int).SetUint64(l2SafeRef.Number), l2SafeRef.Time) {
			return
		}
	}
}

func TestBlobBaseFeeIsCorrectAfterBPOFork(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	t.Log("Waiting for BPO1 to activate")
	t.Require().NotNil(sys.L1Network.Escape().ChainConfig().BPO1Time)
	sys.L1EL.WaitForTime(*sys.L1Network.Escape().ChainConfig().BPO1Time)
	t.Log("BPO1 activated")

	sys.L1EL.WaitForBlock()
	l1BlockTime := sys.L1EL.EstimateBlockTime()
	l1ChainConfig := sys.L1Network.Escape().ChainConfig()

	spamBlobs(t, sys) // Raise the blob base fee to make blob parameter changes visible.

	// Wait for the blob base fee to rise above 1 so the blob parameter changes will be visible.
	for range time.Tick(l1BlockTime) {
		info, _, err := sys.L1EL.EthClient().InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
		t.Require().NoError(err)
		if calcBlobBaseFee(l1ChainConfig, info).Cmp(big.NewInt(1)) > 0 {
			break
		}
		t.Logf("Waiting for blob base fee to rise above 1")
	}

	l2UnsafeRef := sys.L2CL.SyncStatus().UnsafeL2

	// Get the L1 blob base fee.
	l1OriginInfo, err := sys.L1EL.EthClient().InfoByHash(t.Ctx(), l2UnsafeRef.L1Origin.Hash)
	t.Require().NoError(err)
	l1BlobBaseFee := calcBlobBaseFee(l1ChainConfig, l1OriginInfo)

	l2Info, l2Txs, err := sys.L2EL.Escape().EthClient().InfoAndTxsByHash(t.Ctx(), l2UnsafeRef.Hash)
	t.Require().NoError(err)

	// Check the L1 blob base fee in the system deposit tx.
	blockInfo, err := derive.L1BlockInfoFromBytes(sys.L2Chain.Escape().RollupConfig(), l2Info.Time(), l2Txs[0].Data())
	t.Require().NoError(err)
	l2BlobBaseFee := blockInfo.BlobBaseFee
	t.Require().Equal(l1BlobBaseFee, l2BlobBaseFee)

	// Check the L1 Blob base fee in the L1Block contract.
	l1Block := bindings.NewL1Block(bindings.WithClient(sys.L2EL.Escape().EthClient()), bindings.WithTo(predeploys.L1BlockAddr))
	l2BlobBaseFee, err = contractio.Read(l1Block.BlobBaseFee(), t.Ctx(), func(tx *txplan.PlannedTx) {
		tx.AgainstBlock.Set(l2Info)
	})
	t.Require().NoError(err)
	t.Require().Equal(l1BlobBaseFee, l2BlobBaseFee)
}

func spamBlobs(t devtest.T, sys *presets.Minimal) {
	l1BlockTime := sys.L1EL.EstimateBlockTime()
	l1ChainConfig := sys.L1Network.Escape().ChainConfig()

	eoa := sys.FunderL1.NewFundedEOA(eth.OneEther.Mul(5))
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), sys.L1Network.ChainID().ToBig())
	l1ETHClient := sys.L1EL.EthClient()
	syncEOA := loadtest.NewSyncEOA(txinclude.NewPersistent(signer, struct {
		*txinclude.Monitor
		*txinclude.Resubmitter
	}{
		txinclude.NewMonitor(l1ETHClient, l1BlockTime),
		txinclude.NewResubmitter(l1ETHClient, l1BlockTime),
	}), eoa.Plan())

	var blob eth.Blob
	_, err := rand.Read(blob[:])
	t.Require().NoError(err)
	// get the field-elements into a valid range
	for i := range 4096 {
		blob[32*i] &= 0b0011_1111
	}

	const maxBlobTxsPerAccountInMempool = 16 // Private policy param in geth.
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := syncEOA.Include(t, txplan.WithBlobs([]*eth.Blob{&blob}, l1ChainConfig), txplan.WithTo(&common.Address{}))
		return err
	})
	txsPerSlot := min(l1ChainConfig.BlobScheduleConfig.BPO1.Max*3/4, maxBlobTxsPerAccountInMempool)
	schedule := loadtest.NewConstant(l1BlockTime, loadtest.WithBaseRPS(uint64(txsPerSlot)))

	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		schedule.Run(t.WithCtx(ctx), spammer)
	}()
}

func calcBlobBaseFee(cfg *params.ChainConfig, info eth.BlockInfo) *big.Int {
	return eip4844.CalcBlobFee(cfg, &types.Header{
		// It's unfortunate that we can't build a proper header from a BlockInfo.
		// We do our best to work around deficiencies in the BlockInfo implementation here.
		Time:          info.Time(),
		ExcessBlobGas: info.ExcessBlobGas(),
	})
}
