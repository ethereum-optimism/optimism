package flashblocks

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

func TestFlashblocksUnderLoad(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainWithFlashblocks(t)
	driveViaTestSequencer(t, sys, 3)
	spamTxs(t, sys)
	client := sources.NewFlashblockClient(sys.L2RollupBoost.FlashblocksClient(), sys.Log.With("source", "rollup-boost"), 100)
	startClient(t, client)

	lastBlockNumber := -1
	var lastCumulativeTxCount int
	var lastFlashblockHash common.Hash
	consecutiveHeavy := 0

	for {
		select {
		case <-t.Ctx().Done():
			t.Require().NoError(t.Ctx().Err())
		case flashblock, ok := <-client.Next():
			t.Require().True(ok)
			t.Require().NotNil(flashblock)

			if flashblock.Metadata.BlockNumber != lastBlockNumber && lastBlockNumber >= 0 {
				// Block transition: verify the sealed block hash matches the last flashblock.
				sealed, err := sys.L2EL.EthClient().InfoByNumber(t.Ctx(), uint64(lastBlockNumber))
				t.Require().NoError(err)

				if sealed.Hash() == lastFlashblockHash {
					if lastCumulativeTxCount > 100 {
						consecutiveHeavy++
						if consecutiveHeavy >= 3 {
							return
						}
					} else {
						consecutiveHeavy = 0
					}
				} else {
					t.Logger().Info("Flashblock reorg, retrying...")
				}

				lastCumulativeTxCount = 0
			}

			lastBlockNumber = flashblock.Metadata.BlockNumber
			lastCumulativeTxCount += len(flashblock.Diff.Transactions)
			lastFlashblockHash = common.HexToHash(flashblock.Diff.BlockHash)

			t.Logger().Info("Latest flashblock", "block", flashblock.Metadata.BlockNumber, "index", flashblock.Index, "cumulativeTxCount", lastCumulativeTxCount)
		}
	}
}

func spamTxs(t devtest.T, sys *presets.SingleChainWithFlashblocks) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	eoas := loadtest.NewRoundRobin(loadtest.FundEOAs(
		t,
		eth.ThousandEther,
		20,
		l2BlockTime,
		sys.L2EL,
		sys.Wallet,
		sys.FaucetL2,
	))
	ctxSpam, cancelSpam := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancelSpam()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(200)).Run(t.WithCtx(ctxSpam), loadtest.SpammerFunc(func(t devtest.T) error {
			_, err := eoas.Get().Include(t, txplan.WithTo(&common.Address{}), txplan.WithValue(eth.OneWei))
			return err
		}))
	}()
}
