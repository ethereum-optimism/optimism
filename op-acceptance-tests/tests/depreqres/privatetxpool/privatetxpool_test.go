package privatetxpool

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func TestRestrictedTxPool(gt *testing.T) {
	gt.Skip("Skipping test until we have a way to add IP addresses. For now, we only run this test locally.")

	// You need to add 2 IP addresses locally for this test:
	// sudo ifconfig lo0 alias 127.0.0.2 up
	// sudo ifconfig lo0 alias 127.0.0.3 up
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainThreeNodes(t)
	l := t.Logger()
	require := t.Require()

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	delta := uint64(2)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLC.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	txs := 10
	users := sys.FunderL2.NewFundedEOAs(txs, eth.OneHundredthEther)

	for i := 0; i < txs-1; i++ {
		tx := txplan.NewPlannedTx(
			users[i].Plan(),
			txplan.WithTo(&common.Address{}),
			txplan.WithValue(eth.ETH(eth.OneGWei)),
		)
		_, err := tx.Submitted.Eval(t.Ctx())
		require.NoError(err)
	}

	restrictedEL := sys.L2ELC.Escape().EthClient()

	tx_restricted := txplan.NewPlannedTx(
		txplan.WithChainID(restrictedEL),
		users[txs-1].Key().Plan(),
		txplan.WithAgainstLatestBlock(restrictedEL),
		txplan.WithEstimator(restrictedEL, true),
		txplan.WithRetrySubmission(restrictedEL, 5, retry.Exponential()),
		txplan.WithBlockInclusionInfo(restrictedEL),
		txplan.WithTo(&common.Address{}),
		txplan.WithValue(eth.ETH(eth.OneGWei)),
	)
	_, err := tx_restricted.Submitted.Eval(t.Ctx())
	require.NoError(err)

	tpmInternal := &TxPoolManager{
		rpcClient: sys.L2ELB.Escape().EthClient().RPC(),
	}

	tpmRestricted := &TxPoolManager{
		rpcClient: sys.L2ELC.Escape().EthClient().RPC(),
	}

	pendingTxsInternal, err := tpmInternal.GetPendingTransactions(t.Ctx())
	require.NoError(err)
	pendingTxsRestricted, err := tpmRestricted.GetPendingTransactions(t.Ctx())
	require.NoError(err)

	l.Info("Pending transactions", "count_internal", len(pendingTxsInternal), "count_restricted", len(pendingTxsRestricted))
	require.Equal(len(pendingTxsInternal), txs-1)
	require.Equal(len(pendingTxsRestricted), 1)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLC.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	dsl.CheckAll(t,
		sys.L2CL.MatchedFn(sys.L2CLB, types.LocalUnsafe, 5),
		sys.L2CLB.MatchedFn(sys.L2CLC, types.LocalUnsafe, 5),
		sys.L2CLC.MatchedFn(sys.L2CL, types.LocalUnsafe, 5),
	)

	pendingTxsInternal, err = tpmInternal.GetPendingTransactions(t.Ctx())
	require.NoError(err)
	pendingTxsRestricted, err = tpmRestricted.GetPendingTransactions(t.Ctx())
	require.NoError(err)

	l.Info("Remaining pending transactions", "count_internal", len(pendingTxsInternal), "count_restricted", len(pendingTxsRestricted))
	require.Equal(len(pendingTxsInternal), 0)
	require.Equal(len(pendingTxsRestricted), 1)
}

// TxPoolManager handles TxPool operations for L2EL
type TxPoolManager struct {
	rpcClient client.RPC
}

// TxPoolContent represents the content of the transaction pool
type TxPoolContent struct {
	Pending map[common.Address]map[uint64]*ethtypes.Transaction `json:"pending"`
	Queued  map[common.Address]map[uint64]*ethtypes.Transaction `json:"queued"`
}

// GetTxPoolContent fetches all transactions in the pool
func (tpm *TxPoolManager) GetTxPoolContent(ctx context.Context) (*TxPoolContent, error) {
	var content TxPoolContent
	err := tpm.rpcClient.CallContext(ctx, &content, "txpool_content")
	if err != nil {
		return nil, fmt.Errorf("failed to get txpool content: %w", err)
	}
	return &content, nil
}

// GetPendingTransactions gets all pending transactions
func (tpm *TxPoolManager) GetPendingTransactions(ctx context.Context) ([]*ethtypes.Transaction, error) {
	content, err := tpm.GetTxPoolContent(ctx)
	if err != nil {
		return nil, err
	}

	var pendingTxs []*ethtypes.Transaction
	for _, txs := range content.Pending {
		for _, tx := range txs {
			pendingTxs = append(pendingTxs, tx)
		}
	}

	return pendingTxs, nil
}

// GetQueuedTransactions gets all queued transactions
func (tpm *TxPoolManager) GetQueuedTransactions(ctx context.Context) ([]*ethtypes.Transaction, error) {
	content, err := tpm.GetTxPoolContent(ctx)
	if err != nil {
		return nil, err
	}

	var queuedTxs []*ethtypes.Transaction
	for _, txs := range content.Queued {
		for _, tx := range txs {
			queuedTxs = append(queuedTxs, tx)
		}
	}

	return queuedTxs, nil
}
