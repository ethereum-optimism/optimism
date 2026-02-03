package throttling

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TestDABlockThrottling verifies that the execution client respects the block size limit set via
// miner_setMaxDASize. It spams transactions to saturate block space and asserts that blocks are
// filled to near capacity without exceeding the limit.
func TestDABlockThrottling(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	spamCtx, cancelSpam := context.WithCancel(t.Ctx())
	defer cancelSpam()
	spamTxs(spamCtx, sys)

	const minFullSize = blockSizeLimit * 95 / 100

	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	var consecutiveFull uint64
	for {
		select {
		case <-time.Tick(l2BlockTime):
			_, txs, err := sys.L2EL.Escape().EthClient().InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
			t.Require().NoError(err)

			var calldataSize uint64
			for _, tx := range txs {
				if tx.IsDepositTx() {
					continue
				}
				calldataSize += bigs.Uint64Strict(tx.RollupCostData().EstimatedDASize())
			}
			t.Require().LessOrEqual(calldataSize, uint64(blockSizeLimit))

			if calldataSize >= minFullSize {
				consecutiveFull++
			} else {
				consecutiveFull = 0
			}

			if consecutiveFull == 3 {
				return
			}
		case <-t.Ctx().Done():
			t.Require().Fail("Never saw three consecutive blocks near the max size")
		}
	}
}

func spamTxs(ctx context.Context, sys *presets.Minimal) {
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second

	// Fund a lot of spammer EOAs. The funder provided by the devstack isn't very reliable when
	// funding lots of different accounts. We fund one account from the faucet and then use that
	// account to fund all the others.
	const numAccounts = 50
	totalETH := eth.OneEther.Mul(numAccounts)
	spammerELClient := txinclude.NewReliableEL(sys.L2EL.Escape().EthClient(), l2BlockTime)
	funder := newSyncEOA(sys.FunderL2.NewFundedEOA(totalETH), spammerELClient)
	totalETH = totalETH.Sub(totalETH.Div(50)) // Reserve 2% of the balance for gas.
	ethPerAccount := totalETH.Div(numAccounts)
	var eoas []*loadtest.SyncEOA
	var mu sync.Mutex
	var wgEOA sync.WaitGroup
	for range numAccounts {
		wgEOA.Add(1)
		go func() {
			defer wgEOA.Done()
			eoa := sys.Wallet.NewEOA(sys.L2EL)
			addr := eoa.Address()
			_, err := funder.Include(sys.T, txplan.WithTo(&addr), txplan.WithValue(ethPerAccount))
			sys.T.Require().NoError(err)

			mu.Lock()
			defer mu.Unlock()
			eoas = append(eoas, newSyncEOA(eoa, spammerELClient))
		}()
	}
	wgEOA.Wait()

	eoasRR := loadtest.NewRoundRobin(eoas)
	spammer := loadtest.SpammerFunc(func(t devtest.T) error {
		_, err := eoasRR.Get().Include(t, txplan.WithTo(&predeploys.L1BlockAddr), txplan.WithData(make([]byte, 0)), txplan.WithGasLimit(70_000))
		return err
	})
	schedule := loadtest.NewBurst(l2BlockTime, loadtest.WithBaseRPS(50))

	var wg sync.WaitGroup
	wg.Add(1)
	sys.T.Cleanup(func() {
		wg.Wait()
	})
	go func() {
		defer wg.Done()
		schedule.Run(sys.T.WithCtx(ctx), spammer)
	}()
}

func newSyncEOA(eoa *dsl.EOA, el txinclude.EL) *loadtest.SyncEOA {
	signer := txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig())
	const maxConcurrentTxs = 16 // Reth's mempool limits the number of txs per account to 16.
	return loadtest.NewSyncEOA(txinclude.NewLimit(txinclude.NewPersistent(signer, el), maxConcurrentTxs), eoa.Plan())
}
