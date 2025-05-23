package deposit

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	supervisorTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/lmittmann/w3"
)

func TestMain(m *testing.M) {
	presets.DoMain(m, presets.WithMinimal())
}

func TestL1ToL2Deposit(gt *testing.T) {
	// Create a test environment using op-devstack
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)

	// Wait for L1 node to be responsive
	sys.L1Network.WaitForOnline()

	// Fund Alice on L1
	fundingAmount := eth.ThousandEther
	alice := sys.Wallet.NewEOA(sys.L1EL)
	initialBalance := sys.FunderL1.FundAtLeast(alice, fundingAmount)

	alicel2 := alice.AsEL(sys.L2EL)
	initialL2Balance := alicel2.GetBalance()

	// Get the optimism portal address
	rollupConfig := sys.L2Chain.Escape().RollupConfig()
	portalAddr := rollupConfig.DepositContractAddress

	depositAmount := eth.OneEther

	// Define the deposit function and encode arguments
	// TODO: Redo this when the new DSL (#16079) is merged
	funcDeposit := w3.MustNewFunc(`depositTransaction(address,uint256,uint64,bool,bytes)`, "")
	args, err := funcDeposit.EncodeArgs(
		alice.Address(),       // _to
		depositAmount.ToBig(), // _value
		uint64(300_000),       // _gasLimit
		false,                 // _isCreation
		[]byte{},              // _data
	)
	require.NoError(t, err)

	// Create and confirm the transaction (awaits tx inclusion and checks success status)
	tx := alice.Transact(
		alice.Plan(),
		txplan.WithGasLimit(500_000),
		txplan.WithTo(&portalAddr),
		txplan.WithData(args),
		txplan.WithValue(depositAmount.ToBig()))

	receipt := tx.Included.Value()
	gasPrice := receipt.EffectiveGasPrice // bugfix: use the exact price that was charged

	// Verify the deposit was successful
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(receipt.GasUsed), gasPrice)
	expectedFinalL1 := new(big.Int).Sub(initialBalance.ToBig(), depositAmount.ToBig())
	expectedFinalL1.Sub(expectedFinalL1, gasCost)

	alice.VerifyBalanceExact(eth.WeiBig(expectedFinalL1))

	// Wait for the sequencer to process the deposit
	t.Require().Eventually(func() bool {
		head := sys.L2CL.HeadBlockRef(supervisorTypes.LocalUnsafe)
		return head.L1Origin.Number >= receipt.BlockNumber.Uint64()
	}, time.Second*30, time.Second, "awaiting deposit to be processed by L2")
	alicel2.VerifyBalanceExact(initialL2Balance.Add(depositAmount))
}
