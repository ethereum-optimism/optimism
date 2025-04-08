package interop

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type txSubmitter struct {
	t       helpers.Testing
	chain   *dsl.Chain
	from    common.Address
	receipt *types.Receipt
}

func (ts *txSubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	// we need low level interaction here
	// do not submit transactions via RPC, instead directly interact with block builder
	receipt, err := ts.chain.SequencerEngine.EngineApi.IncludeTx(tx, ts.from)
	if err == nil {
		// be aware that this receipt is not finalized...
		// which means its info may be incorrect, such as block hash
		// you must call ActL2EndBlock to seal the L2 block
		ts.receipt = receipt
	}
	return err
}

type receiptGetter struct {
	t     helpers.Testing
	chain *dsl.Chain
	sc    *sources.EthClient
}

func (rg *receiptGetter) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	// close l2 block before fetching actual receipt
	rg.chain.Sequencer.ActL2EndBlock(rg.t)
	return rg.sc.TransactionReceipt(ctx, txHash)
}

func TestTxPlanDeployEventLogger(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)

	aliceA := setupUser(t, is, actors.ChainA, 0)

	l2sc := actors.ChainA.SequencerEngine.SourceClient(t, 10)

	submitter1 := &txSubmitter{t: t, chain: actors.ChainA, from: aliceA.address}
	// txplan options for only tx submission, not ensuring block inclusion
	opts1 := txplan.Combine(
		txplan.WithPrivateKey(aliceA.secret),
		txplan.WithChainID(l2sc),
		txplan.WithAgainstLatestBlock(l2sc),
		txplan.WithPendingNonce(l2sc),
		txplan.WithEstimator(l2sc, false),
		txplan.WithTransactionSubmitter(submitter1),
	)

	actors.ChainA.Sequencer.ActL2StartBlock(t)

	deployCalldata := common.FromHex(bindings.EventloggerBin)
	// tx submitted but not sealed in block
	deployTxWithoutSeal := txplan.NewPlannedTx(opts1, txplan.WithData(deployCalldata))
	_, err := deployTxWithoutSeal.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	latestBlock, err := deployTxWithoutSeal.AgainstBlock.Eval(t.Ctx())
	require.NoError(t, err)

	getter := &receiptGetter{t: t, chain: actors.ChainA, sc: l2sc}
	submitter2 := &txSubmitter{t: t, chain: actors.ChainA, from: aliceA.address}
	// txplan options for tx submission and ensuring block inclusion
	opts2 := txplan.Combine(
		txplan.WithPrivateKey(aliceA.secret),
		txplan.WithChainID(l2sc),
		txplan.WithAgainstLatestBlock(l2sc),
		// no pending nonce
		txplan.WithEstimator(l2sc, false),
		txplan.WithTransactionSubmitter(submitter2),
		txplan.WithAssumedInclusion(getter),
		txplan.WithBlockInclusionInfo(l2sc),
	)
	deployTx := txplan.NewPlannedTx(opts2, txplan.WithData(deployCalldata))
	// manually set nonce because we cannot use the pending nonce
	nonce, err := deployTxWithoutSeal.Nonce.Get()
	require.NoError(t, err)
	deployTx.Nonce.Set(nonce + 1)

	// tx submitted and sealed in block
	// now the tx is actually included in L2 block, as well as included the tx submitted before
	receipt, err := deployTx.Included.Eval(t.Ctx())
	require.NoError(t, err)

	// all intermediate receipts / finalized receipt must contain the contractAddress field
	// because they all deployed contract
	require.NotNil(t, receipt.ContractAddress)
	require.NotNil(t, submitter1.receipt.ContractAddress)
	require.NotNil(t, submitter2.receipt.ContractAddress)

	// different nonce so different contract address
	require.NotEqual(t, submitter1.receipt.ContractAddress, submitter2.receipt.ContractAddress)
	// second and the finalized contract address must be equal
	require.Equal(t, submitter2.receipt.ContractAddress, receipt.ContractAddress)

	includedBlock, err := deployTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// single block advanced
	require.Equal(t, latestBlock.NumberU64()+1, includedBlock.Number)
}
