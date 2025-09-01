package fjord

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	txib "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestFees(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()
	ctx := t.Ctx()

	err := dsl.RequiresL2Fork(ctx, sys, 0, rollup.Fjord)
	require.NoError(err)
	operatorFee := dsl.NewOperatorFee(t, sys.L2Chain, sys.L1EL)
	operatorFee.SetOperatorFee(100000000, 500)
	operatorFee.WaitForL2SyncWithCurrentL1State()

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)
	bob := sys.Wallet.NewEOA(sys.L2EL)

	fjordFees := dsl.NewFjordFees(t, sys.L2Chain)
	result := fjordFees.ValidateTransaction(alice, bob, eth.OneHundredthEther.ToBig())

	l2Client := sys.L2EL.Escape().EthClient()
	gpo := txib.NewGasPriceOracle(
		txib.WithClient(l2Client),
		txib.WithTo(predeploys.GasPriceOracleAddr),
		txib.WithTest(t),
	)

	_, txs, err := l2Client.InfoAndTxsByHash(ctx, result.TransactionReceipt.BlockHash)
	require.NoError(err)

	var signedTx *types.Transaction
	for _, tx := range txs {
		if tx.Hash() == result.TransactionReceipt.TxHash {
			signedTx = tx
			break
		}
	}
	require.NotNil(signedTx)

	unsignedTx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     signedTx.Nonce(),
		To:        signedTx.To(),
		Value:     signedTx.Value(),
		Gas:       signedTx.Gas(),
		GasFeeCap: signedTx.GasFeeCap(),
		GasTipCap: signedTx.GasTipCap(),
		Data:      signedTx.Data(),
	})

	txUnsigned, err := unsignedTx.MarshalBinary()
	require.NoError(err)

	gpoL1Fee, err := contractio.Read(gpo.GetL1Fee(txUnsigned), ctx)
	require.NoError(err)
	require.Equal(result.L1Fee, gpoL1Fee.ToBig())
}
