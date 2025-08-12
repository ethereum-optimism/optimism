package jovian

import (
	"context"
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop/loadtest"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
)

type CalldataSpammer struct {
	eoa *loadtest.SyncEOA
}

func NewCalldataSpammer(eoa *loadtest.SyncEOA) *CalldataSpammer {
	return &CalldataSpammer{
		eoa: eoa,
	}
}

func (s *CalldataSpammer) Spam(t devtest.T) error {
	data := make([]byte, 50_000)
	_, err := rand.Read(data)
	t.Require().NoError(err)
	_, err = s.eoa.Include(t, txplan.WithEth(eth.OneWei.ToBig()), txplan.WithTo(&common.Address{}), txplan.WithData(data))
	return err
}

func TestDAFootprint(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	l2BlockTime := time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second
	ethClient := sys.L2EL.Escape().EthClient()

	sys.L2EL.WaitForOnline()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(t.Ctx())
	defer cancel()
	t = t.WithCtx(ctx)

	wg.Add(1)
	go func() {
		defer wg.Done()
		eoa := sys.FunderL2.NewFundedEOA(eth.OneEther.Mul(100))
		includer := txinclude.NewPersistent(txinclude.NewPkSigner(eoa.Key().Priv(), eoa.ChainID().ToBig()), struct {
			*txinclude.Resubmitter
			*txinclude.Monitor
		}{
			txinclude.NewResubmitter(ethClient, l2BlockTime),
			txinclude.NewMonitor(ethClient, l2BlockTime),
		})
		loadtest.NewBurst(l2BlockTime).Run(t, NewCalldataSpammer(loadtest.NewSyncEOA(includer, eoa.Plan())))
	}()

	for range time.Tick(l2BlockTime) {
		info, txs, err := ethClient.InfoAndTxsByLabel(t.Ctx(), eth.Unsafe)
		t.Require().NoError(err)

		blockGasUsed := info.GasUsed()

		receipt, err := ethClient.TransactionReceipt(t.Ctx(), txs[len(txs)-1].Hash())
		t.Require().NoError(err)
		if receipt.CumulativeGasUsed == blockGasUsed {
			continue
		}

		var cumulativeDAFootprint uint64
		for _, tx := range txs {
			if tx.IsDepositTx() {
				continue
			}
			cumulativeDAFootprint += tx.RollupCostData().EstimatedDASize().Uint64()
		}
		t.Require().Equal(400*cumulativeDAFootprint, blockGasUsed)
		return
	}
}
