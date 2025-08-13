package withdrawal

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestWithdrawalRoot(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for this test")

	secondCheck, err := dsl.CheckForChainFork(t.Ctx(), sys.L2Networks(), t.Logger())
	require.NoError(err, "error checking for chain fork")
	defer func() {
		require.NoError(secondCheck(false), "error checking for chain fork")
	}()

	bridge := sys.StandardBridge()
	// 333_333_333 gwei (1 billion wei)
	initialL1Balance := eth.OneThirdEther
	initialL2Balance := eth.OneThirdEther

	// l1User and l2User share same private key
	// op-faucet called
	l1User := sys.FunderL1.NewFundedEOA(initialL1Balance)
	l2User := l1User.AsEL(sys.L2EL) // Only receives funds via the deposit
	// op-faucet called
	sys.FunderL2.FundAtLeast(l2User, initialL2Balance)

	// 333333333 * 1000000000, 333333333 * 1000000000
	sys.Log.Info("Balance", "l1User", l1User.GetBalance(), "l2User", l2User.GetBalance())

	withdrawalAmount := eth.OneHundredthEther

	withdrawal := bridge.InitiateWithdrawal(withdrawalAmount, l2User)
	expectedL2UserBalance := initialL2Balance.Sub(withdrawalAmount).Sub(withdrawal.InitiateGasCost())

	// 333333333 * 1000000000, 10000000 * 1000000000, 128781093286213
	sys.Log.Info("Balance", "initialL2Balance", initialL2Balance, "withdrawalAmount", withdrawalAmount, "initiateGasCost", withdrawal.InitiateGasCost())

	receiptRaw, _ := json.Marshal(withdrawal.InitiateReceipt())
	sys.Log.Info("Receipt", "receipt", string(receiptRaw))

	// 128781093286213 = 0x12448 * 0x6696339c + 0x2353d65
	// cost := eth.WeiBig(new(big.Int).Mul(new(big.Int).SetUint64(rcpt.GasUsed), rcpt.EffectiveGasPrice))
	// if rcpt.L1Fee != nil {
	// 	cost = cost.Add(eth.WeiBig(rcpt.L1Fee))
	// }

	// expected: 323204551906713787
	// actual: 323204551906713365
	// expected + 422 = 323204551906713365
	// why actual did not use 422

	// 0x190 + 0x12c * 0x12448 / 1e6 = 422
	l2User.VerifyBalanceExact(expectedL2UserBalance)

	sys.L2EL.VerifyWithdrawalHashChangedIn(withdrawal.InitiateBlockHash())

	require.True(false)
}

// {
// 	"type": "0x2",
// 	"root": "0x",
// 	"status": "0x1",
// 	"cumulativeGasUsed": "0x1d894",
// 	"logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000100000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000800000000000000000008004000000000000000040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000800000000100000000000000020000000000008000000000000000",
// 	"logs": [
// 	  {
// 		"address": "0x4200000000000000000000000000000000000016",
// 		"topics": [
// 		  "0x02a52367d10742d8032712c1bb8e0144ff1ec5ffda1ed7d70bb05a2744955054",
// 		  "0x0001000000000000000000000000000000000000000000000000000000000000",
// 		  "0x000000000000000000000000b28a98ac25d20f3e3275cc69185b715dd7209e49",
// 		  "0x000000000000000000000000b28a98ac25d20f3e3275cc69185b715dd7209e49"
// 		],
// 		"data": "0x000000000000000000000000000000000000000000000000002386f26fc1000000000000000000000000000000000000000000000000000000000000000186a000000000000000000000000000000000000000000000000000000000000000807111b82b5a4123f19081f6cbdff435e32f3f53796417b114ac33ca3916dd54f10000000000000000000000000000000000000000000000000000000000000000",
// 		"blockNumber": "0x52",
// 		"transactionHash": "0x1d851a54e1ce81baa9444fdfb7c26daf3543ef2f561389915596c280b78466af",
// 		"transactionIndex": "0x1",
// 		"blockHash": "0x37faf98e7dc973328877f149c719ef2cc38119b2c0cb4cfc7ed5f57e9c64d839",
// 		"logIndex": "0x0",
// 		"removed": false
// 	  }
// 	],
// 	"transactionHash": "0x1d851a54e1ce81baa9444fdfb7c26daf3543ef2f561389915596c280b78466af",
// 	"contractAddress": "0x0000000000000000000000000000000000000000",
// 	"gasUsed": "0x12448",
// 	"effectiveGasPrice": "0x6696339c",
// 	"blockHash": "0x37faf98e7dc973328877f149c719ef2cc38119b2c0cb4cfc7ed5f57e9c64d839",
// 	"blockNumber": "0x52",
// 	"transactionIndex": "0x1",
// 	"l1GasPrice": "0x1023dc7",
// 	"l1BlobBaseFee": "0x1",
// 	"l1GasUsed": "0x640",
// 	"l1Fee": "0x2353d65",
// 	"l1BaseFeeScalar": "0x558",
// 	"l1BlobBaseFeeScalar": "0xc5fc5",
// 	"operatorFeeScalar": "0x12c",
// 	"operatorFeeConstant": "0x190"
//   }
