package withdrawal

import (
	"math/big"
	"testing"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestWithdrawal(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := sys.T.Require()

	l1Client := sys.L1EL.Escape().EthClient()
	l2Client := sys.L2EL.Escape().EthClient()

	// initialize contract bindings
	optimismPortalAddr := sys.L2Chain.Escape().RollupConfig().DepositContractAddress
	portal := bindings.NewBindings[bindings.OptimismPortal2](bindings.WithClient(l1Client), bindings.WithTo(optimismPortalAddr), bindings.WithTest(t))
	l2tol1MessagePasser := bindings.NewBindings[bindings.L2ToL1MessagePasser](bindings.WithClient(l2Client), bindings.WithTo(predeploys.L2ToL1MessagePasserAddr), bindings.WithTest(t))

	// Make sure fast game is set
	require.Equal(uint32(faultTypes.FastGameType), contract.Read(portal.RespectedGameType()))

	initialL1Balance, initialL2Balance := eth.OneThirdEther, eth.OneTenthEther

	// l1User and l2User share same private key
	l2User := sys.FunderL2.NewFundedEOA(initialL2Balance)
	l1User := l2User.AsEL(sys.L1EL)
	sys.FunderL1.Fund(l1User, initialL1Balance)
	userAddr := l1User.Address()

	// The max amount of withdrawal is limited to the total amount of deposit
	// We trigger deposit first to fund the L1 ETHLockbox to satisfy the invariant

	// Deposit 0.1 ETH
	depositAmount := eth.OneTenthEther
	l1DepositReceipt := contract.Write(l1User,
		portal.DepositTransaction(userAddr, depositAmount, 1_000_000, false, []byte{}),
		txplan.WithValue(depositAmount.ToBig()),
	)
	// L2CL will read L1, fetch TransactionDeposited() event and convert as a deposit transaction at L2
	// Construct the L2 deposit tx to check the tx is included at L2
	idx := len(l1DepositReceipt.Logs) - 1
	l2DepositTx, err := derive.UnmarshalDepositLogEvent(l1DepositReceipt.Logs[idx])
	require.NoError(err, "Could not reconstruct L2 Deposit")
	l2DepositTxHash := types.NewTx(l2DepositTx).Hash()
	// Give time for L2CL to include the L2 deposit tx
	var l2DepositReceipt *types.Receipt
	require.Eventually(func() bool {
		l2DepositReceipt, err = l2Client.TransactionReceipt(t.Ctx(), l2DepositTxHash)
		return err == nil
	}, 60*time.Second, 500*time.Millisecond, "L2 Deposit never found")
	require.Equal(types.ReceiptStatusSuccessful, l2DepositReceipt.Status)
	// Deposit tx included so L2 user balance must be updated

	l1BalanceAfterDeposit, err := l1Client.BalanceAt(t.Ctx(), userAddr, nil)
	require.NoError(err)
	l2BalanceAfterDeposit, err := l2Client.BalanceAt(t.Ctx(), userAddr, nil)
	require.NoError(err)

	require.True(l2BalanceAfterDeposit.Cmp(new(big.Int).Add(initialL2Balance.ToBig(), depositAmount.ToBig())) == 0)
	// L1 gas fee check
	{
		depositGasFee := new(big.Int).Mul(big.NewInt(int64(l1DepositReceipt.GasUsed)), l1DepositReceipt.EffectiveGasPrice)
		diff := new(big.Int).Sub(
			new(big.Int).Sub(initialL1Balance.ToBig(), depositAmount.ToBig()),
			depositGasFee,
		)
		require.True(l1BalanceAfterDeposit.Cmp(diff) == 0)
	}

	// Withdrawal 0.1 ETH
	withdrawalAmount := eth.OneTenthEther
	l1BalanceBeforeWithdrawal := l1BalanceAfterDeposit
	l2BalanceBeforeWithdrawal := l2BalanceAfterDeposit

	// Satisfy ETHLockbox invariant
	require.True(depositAmount.ToBig().Cmp(withdrawalAmount.ToBig()) != -1)

	// Withdrawal STEP 0: Trigger L2toL1 Message
	l2WithdrawalReceipt := contract.Write(l2User,
		l2tol1MessagePasser.InitiateWithdrawal(userAddr, new(big.Int).SetInt64(500_000), []byte{}),
		txplan.WithValue(withdrawalAmount.ToBig()),
	)
	// Emit MessagePassed event
	require.Equal(1, len(l2WithdrawalReceipt.Logs))

	l2BalanceAfterWithdrawal, err := l2Client.BalanceAt(t.Ctx(), userAddr, nil)
	require.NoError(err)

	computeFee := func(receipt *types.Receipt) *big.Int {
		return new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), receipt.EffectiveGasPrice)
	}

	// L2 gas fee check
	{
		fees := computeFee(l2WithdrawalReceipt)
		fees = fees.Add(fees, l2WithdrawalReceipt.L1Fee)
		diff := new(big.Int).Sub(l2BalanceBeforeWithdrawal, l2BalanceAfterWithdrawal)
		diff = diff.Sub(diff, fees)
		require.True(withdrawalAmount.ToBig().Cmp(diff) == 0)
	}

	// Withdrawal STEP 1: Prove Withdrawal at L1
	withdrawalParams, proveReceipt := ProveWithdrawal(t, &portal, optimismPortalAddr, sys, l1User, l2WithdrawalReceipt)

	// Withdrawal STEP 2: Finalize Withdrawal at L2
	finalizeReceipt, resolveClaimReceipt, resolveReceipt := FinalizeWithdrawal(t, &portal, sys, l1User, proveReceipt, withdrawalParams)

	// L1 gas fee check for proving and finalizing
	{
		receipts := []*types.Receipt{proveReceipt, finalizeReceipt, resolveClaimReceipt, resolveReceipt}
		fees := new(big.Int)
		for _, r := range receipts {
			fees.Add(fees, computeFee(r))
		}
		expectedAmount := new(big.Int).Sub(withdrawalAmount.ToBig(), fees)
		var endBalanceAfterFinalize *big.Int
		require.Eventually(func() bool {
			endBalanceAfterFinalize, err = l1Client.BalanceAt(t.Ctx(), userAddr, nil)
			if err != nil {
				return false
			}
			diff := new(big.Int).Sub(endBalanceAfterFinalize, l1BalanceBeforeWithdrawal)
			return diff.Cmp(expectedAmount) == 0
		}, time.Second*60, time.Second, "awaiting balance to be changed")
	}
}
