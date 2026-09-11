package enginetest

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestEstimateGasMatchesGeth runs the same estimation and call requests against both L2Engine
// backends over the same genesis and requires byte-identical answers.
//
// This is the §6.3 parity gate for eth_estimateGas: the action tests put the estimate into the
// signed transaction's gas field (BasicUser.MakeTransaction), so an estimator that deviates from
// op-geth's changes the transaction — and hence block — hashes between the two ELs even when every
// test passes on both. The reth engine deliberately reimplements op-geth's estimator control flow;
// this test pins that equivalence where it matters, on the real action-test genesis (predeploys
// included).
func TestEstimateGasMatchesGeth(gt *testing.T) {
	t := helpers.SubTest(gt)
	dp := e2eutils.MakeDeployParams(t, helpers.DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	logger := testlog.Logger(t, gethlog.LevelError)
	jwtPath := e2eutils.WriteDefaultJWT(t)

	// One engine per backend over the same genesis. The reth engine marshals the genesis at spawn
	// time, so it is created first, before geth's setup touches the object.
	gt.Setenv(helpers.ELSelectorEnv, "reth-test-engine")
	rethEng := helpers.NewL2Engine(t, logger, sd.L2Cfg, jwtPath)
	defer func() { _ = rethEng.Close() }()
	gt.Setenv(helpers.ELSelectorEnv, "geth")
	gethEng := helpers.NewL2Engine(t, logger, sd.L2Cfg, jwtPath)
	defer func() { _ = gethEng.Close() }()

	rethCl, gethCl := rethEng.EthClient(), gethEng.EthClient()

	baseFee := gethEng.LatestHeader(t).BaseFee
	feeCap := new(big.Int).Add(baseFee, big.NewInt(2*params.GWei))
	gpoABI, err := bindings.GasPriceOracleMetaData.GetAbi()
	require.NoError(t, err)
	gpoCalldata, err := gpoABI.Pack("getL1Fee", []byte("some sample calldata to price"))
	require.NoError(t, err)
	gpo := predeploys.GasPriceOracleAddr

	cases := []struct {
		name string
		msg  ethereum.CallMsg
	}{
		// The exact shape BasicUser.MakeTransaction estimates: fee-capped plain value transfer.
		{"plain transfer", ethereum.CallMsg{
			From: dp.Addresses.Alice, To: &dp.Addresses.Bob, Value: e2eutils.Ether(1),
			GasFeeCap: feeCap, GasTipCap: big.NewInt(2 * params.GWei),
		}},
		// Calldata to an EOA: intrinsic-gas-dominated, exercises the binary search rather than
		// the 21000 shortcut.
		{"calldata to EOA", ethereum.CallMsg{
			From: dp.Addresses.Alice, To: &dp.Addresses.Bob,
			Data: []byte{0xde, 0xad, 0x00, 0x00, 0xbe, 0xef, 0x01, 0x02, 0x00, 0xff},
		}},
		// Real contract execution against the GasPriceOracle predeploy.
		{"GPO getL1Fee", ethereum.CallMsg{
			From: dp.Addresses.Alice, To: &gpo, Data: gpoCalldata,
		}},
	}
	for _, tc := range cases {
		gethGas, err := gethCl.EstimateGas(t.Ctx(), tc.msg)
		require.NoError(t, err, "geth estimate (%s)", tc.name)
		rethGas, err := rethCl.EstimateGas(t.Ctx(), tc.msg)
		require.NoError(t, err, "reth estimate (%s)", tc.name)
		require.Equal(t, gethGas, rethGas, "estimates must byte-match (%s)", tc.name)
	}

	// eth_call parity through the same abigen binding path the upgrade tests use.
	gethGPO, err := bindings.NewGasPriceOracleCaller(gpo, gethCl)
	require.NoError(t, err)
	rethGPO, err := bindings.NewGasPriceOracleCaller(gpo, rethCl)
	require.NoError(t, err)
	gethFee, err := gethGPO.GetL1Fee(&bind.CallOpts{}, []byte("some sample calldata to price"))
	require.NoError(t, err, "geth call")
	rethFee, err := rethGPO.GetL1Fee(&bind.CallOpts{}, []byte("some sample calldata to price"))
	require.NoError(t, err, "reth call")
	require.Equal(t, gethFee, rethFee, "eth_call results must match")
}
