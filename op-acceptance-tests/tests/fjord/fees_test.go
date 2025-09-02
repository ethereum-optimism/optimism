package fjord

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	dsl "github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	txib "github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
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

	signedTx, err := dsl.FindSignedTransactionFromReceipt(ctx, l2Client, result.TransactionReceipt)
	require.NoError(err)
	require.NotNil(signedTx)

	unsignedTx, err := dsl.CreateUnsignedTransactionFromSigned(signedTx)
	require.NoError(err)

	txUnsigned, err := unsignedTx.MarshalBinary()
	require.NoError(err)

	gpoL1Fee, err := dsl.ReadGasPriceOracleL1FeeAt(ctx, l2Client, gpo, txUnsigned, result.TransactionReceipt.BlockHash)
	require.NoError(err)
	dsl.ValidateL1FeeMatches(t, result.L1Fee, gpoL1Fee)
}
