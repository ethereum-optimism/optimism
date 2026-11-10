package fees

import (
	"context"
	"math/big"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	opfees "github.com/ethereum-optimism/optimism/op-core/fees"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	op_e2e.RunMain(m)
}

// TestFees checks that L1/L2 fees are handled.
func TestFees(t *testing.T) {
	t.Run("pre-regolith", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.RegolithSystemConfig(t, nil, e2esys.WithL2ELKind(services.ELKindOpGeth))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		testFees(t, cfg)
	})
	t.Run("regolith", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.RegolithSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		testFees(t, cfg)
	})
	t.Run("ecotone", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.EcotoneSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		testFees(t, cfg)
	})
	t.Run("fjord", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.FjordSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		testFees(t, cfg)
	})

	t.Run("isthmus without operator fee", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.IsthmusSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))

		testFees(t, cfg)
	})

	t.Run("isthmus with operator fee", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.IsthmusSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))
		cfg.DeployConfig.GasPriceOracleOperatorFeeScalar = 1439103868
		cfg.DeployConfig.GasPriceOracleOperatorFeeConstant = 12564178266093314607

		testFees(t, cfg)
	})

	t.Run("jovian", func(t *testing.T) {
		op_e2e.InitParallel(t)
		cfg := e2esys.JovianSystemConfig(t, new(hexutil.Uint64))
		cfg.DeployConfig.L1GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(7))
		cfg.DeployConfig.GasPriceOracleOperatorFeeScalar = 1439103868
		cfg.DeployConfig.GasPriceOracleOperatorFeeConstant = 12564178266093314607

		testFees(t, cfg)
	})
}

func testFees(t *testing.T, cfg e2esys.SystemConfig) {
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")
	l2SeqSrc := sys.NodeSourceClient("sequencer")
	l2Verif := sys.NodeClient("verifier")
	l1 := sys.NodeClient("l1")

	balanceAt := func(a common.Address, blockNumber *big.Int) *big.Int {
		t.Helper()
		bal, err := l2Seq.BalanceAt(context.Background(), a, blockNumber)
		require.NoError(t, err)
		return bal
	}

	// Wait for first block after genesis. The genesis block has zero L1Block values and will throw off the GPO checks
	_, err = geth.WaitForBlock(big.NewInt(1), l2Verif)
	require.NoError(t, err)

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice
	fromAddr := crypto.PubkeyToAddress(ethPrivKey.PublicKey)

	require.NotEqual(t, cfg.DeployConfig.L2OutputOracleProposer, fromAddr)
	require.NotEqual(t, cfg.DeployConfig.BatchSenderAddress, fromAddr)

	// Find gaspriceoracle contract
	gpoContract, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, l2Seq)
	require.Nil(t, err)

	if !sys.RollupConfig.IsEcotone(sys.L2GenesisCfg.Timestamp) {
		overhead, err := gpoContract.Overhead(&bind.CallOpts{})
		require.Nil(t, err, "reading gpo overhead")
		require.Equal(t, bigs.Uint64Strict(overhead), cfg.DeployConfig.GasPriceOracleOverhead, "wrong gpo overhead")

		scalar, err := gpoContract.Scalar(&bind.CallOpts{})
		require.Nil(t, err, "reading gpo scalar")
		feeScalar := cfg.DeployConfig.FeeScalar()
		require.Equal(t, scalar, new(big.Int).SetBytes(feeScalar[:]), "wrong gpo scalar")
	} else {
		_, err := gpoContract.Overhead(&bind.CallOpts{})
		require.ErrorContains(t, err, "deprecated")
		_, err = gpoContract.Scalar(&bind.CallOpts{})
		require.ErrorContains(t, err, "deprecated")
	}

	decimals, err := gpoContract.Decimals(&bind.CallOpts{})
	require.Nil(t, err, "reading gpo decimals")

	require.Equal(t, bigs.Uint64Strict(decimals), uint64(6), "wrong gpo decimals")

	baseFeeRecipientStartBalance := balanceAt(predeploys.BaseFeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	l1FeeRecipientStartBalance := balanceAt(predeploys.L1FeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	sequencerFeeVaultStartBalance := balanceAt(predeploys.SequencerFeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))
	operatorFeeVaultStartBalance := balanceAt(predeploys.OperatorFeeVaultAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))

	require.Equal(t, baseFeeRecipientStartBalance.Sign(), 0, "base fee vault should be empty")
	require.Equal(t, l1FeeRecipientStartBalance.Sign(), 0, "l1 fee vault should be empty")
	require.Equal(t, sequencerFeeVaultStartBalance.Sign(), 0, "sequencer fee vault should be empty")
	require.Equal(t, operatorFeeVaultStartBalance.Sign(), 0, "operator fee vault should be empty")

	genesisHeader, err := l2Seq.HeaderByNumber(context.Background(), big.NewInt(rpc.EarliestBlockNumber.Int64()))
	require.NoError(t, err)

	coinbaseStartBalance := balanceAt(genesisHeader.Coinbase, big.NewInt(rpc.EarliestBlockNumber.Int64()))

	// Simple transfer from signer to random account
	startBalance := balanceAt(fromAddr, big.NewInt(rpc.EarliestBlockNumber.Int64()))

	require.True(t, startBalance.Cmp(big.NewInt(params.Ether)) > 0,
		"Expected start balance (%v) to be greater than 1 ether (%v)", startBalance, params.Ether)

	transferAmount := big.NewInt(params.Ether)
	gasTip := big.NewInt(10)
	receipt := helpers.SendL2Tx(t, cfg, l2SeqSrc, ethPrivKey, func(opts *helpers.TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = transferAmount
		opts.GasTipCap = gasTip
		opts.Gas = 21000
		opts.GasFeeCap = big.NewInt(200)
		opts.VerifyOnClients(sys.NodeSourceClient("verifier"))
	})

	require.Equal(t, receipt.Status, types.ReceiptStatusSuccessful)

	header, err := l2Seq.HeaderByNumber(context.Background(), receipt.BlockNumber)
	require.Nil(t, err)

	coinbaseEndBalance := balanceAt(header.Coinbase, header.Number)
	endBalance := balanceAt(fromAddr, header.Number)
	baseFeeRecipientEndBalance := balanceAt(predeploys.BaseFeeVaultAddr, header.Number)

	l1Header, err := l1.HeaderByNumber(context.Background(), nil)
	require.Nil(t, err)

	l1FeeRecipientEndBalance := balanceAt(predeploys.L1FeeVaultAddr, header.Number)
	sequencerFeeVaultEndBalance := balanceAt(predeploys.SequencerFeeVaultAddr, header.Number)
	operatorFeeVaultEndBalance := balanceAt(predeploys.OperatorFeeVaultAddr, header.Number)

	// Diff fee recipient + coinbase balances
	baseFeeRecipientDiff := new(big.Int).Sub(baseFeeRecipientEndBalance, baseFeeRecipientStartBalance)
	l1FeeRecipientDiff := new(big.Int).Sub(l1FeeRecipientEndBalance, l1FeeRecipientStartBalance)
	sequencerFeeVaultDiff := new(big.Int).Sub(sequencerFeeVaultEndBalance, sequencerFeeVaultStartBalance)
	operatorFeeVaultDiff := new(big.Int).Sub(operatorFeeVaultEndBalance, operatorFeeVaultStartBalance)
	coinbaseDiff := new(big.Int).Sub(coinbaseEndBalance, coinbaseStartBalance)

	// Tally L2 Fee
	gasUsed := new(big.Int).SetUint64(receipt.GasUsed)
	l2Fee := gasTip.Mul(gasTip, gasUsed)
	require.Equal(t, sequencerFeeVaultDiff, coinbaseDiff, "coinbase is always sequencer fee vault")
	require.Equal(t, l2Fee, coinbaseDiff, "l2 fee mismatch")
	require.Equal(t, l2Fee, sequencerFeeVaultDiff)

	// Tally BaseFee
	baseFee := new(big.Int).Mul(header.BaseFee, gasUsed)
	require.Equal(t, baseFee, baseFeeRecipientDiff, "base fee mismatch")

	// Tally L1 Fee
	tx, _, err := l2Seq.TransactionByHash(context.Background(), receipt.TxHash) //nolint:forbidigo // L2 user tx: the tx SendL2Tx sent
	require.NoError(t, err, "Should be able to get transaction")
	bytes, err := tx.MarshalBinary()
	require.Nil(t, err)
	rollupCostData := opfees.NewRollupCostData(bytes)

	// The fee parameters are the L1 attributes of the block that includes the transaction.
	_, l1InfoDeposit, err := l2SeqSrc.InfoAndFirstDepositByHash(context.Background(), receipt.BlockHash)
	require.NoError(t, err)
	l1Info, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, header.Time, l1InfoDeposit.Data)
	require.NoError(t, err)

	l1Fee := l1Cost(sys.RollupConfig, header.Time, rollupCostData, l1Info)
	require.Equalf(t, l1Fee, l1FeeRecipientDiff, "L1 fee mismatch: start balance %v, end balance %v", l1FeeRecipientStartBalance, l1FeeRecipientEndBalance)

	operatorFee := operatorCost(sys.RollupConfig, header.Time, receipt.GasUsed, l1Info)
	require.True(t, operatorFeeVaultDiff.Cmp(operatorFee) == 0, "operator fee mismatch: start balance %v, end balance %v", operatorFeeVaultStartBalance, operatorFeeVaultEndBalance)

	gpoEcotone, err := gpoContract.IsEcotone(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsEcotone(header.Time), gpoEcotone, "GPO and chain must have same ecotone view")

	gpoFjord, err := gpoContract.IsFjord(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsFjord(header.Time), gpoFjord, "GPO and chain must have same fjord view")

	gpoIsthmus, err := gpoContract.IsIsthmus(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsIsthmus(header.Time), gpoIsthmus, "GPO and chain must have same isthmus view")

	gpoJovian, err := gpoContract.IsJovian(nil)
	require.NoError(t, err)
	require.Equal(t, sys.RollupConfig.IsJovian(header.Time), gpoJovian, "GPO and chain must have same jovian view")

	gpoL1Fee, err := gpoContract.GetL1Fee(&bind.CallOpts{}, bytes)
	require.Nil(t, err)

	adjustedGPOFee := gpoL1Fee
	if sys.RollupConfig.IsFjord(header.Time) {
		// The fastlz size of the transaction is 102 bytes
		require.Equal(t, uint64(102), rollupCostData.FastLzSize)
		// Which results in both the fjord cost function and GPO using the minimum value for the fastlz regression:
		// Geth Linear Regression: -42.5856 + 102 * 0.8365 = 42.7374
		// GPO Linear Regression: -42.5856 + 170 * 0.8365 = 99.6194
		// The additional 68 (170 vs. 102) is due to the GPO adding 68 bytes to account for the signature.
		require.Greater(t, bigs.Uint64Strict(opfees.MinTransactionSize), uint64(99))
		// Because of this, we don't need to do any adjustment as the GPO and cost func are both bounded to the minimum value.
		// However, if the fastlz regression output is ever larger than the minimum, this will require an adjustment.
	} else if sys.RollupConfig.IsRegolith(header.Time) {
		// if post-regolith, adjust the GPO fee by removing the overhead it adds because of signature data
		artificialGPOOverhead := big.NewInt(68 * 16) // it adds 68 bytes to cover signature and RLP data
		l1BaseFee := big.NewInt(7)                   // we assume the L1 basefee is the minimum, 7
		// in our case we already include that, so we subtract it, to do a 1:1 comparison
		adjustedGPOFee = new(big.Int).Sub(gpoL1Fee, new(big.Int).Mul(artificialGPOOverhead, l1BaseFee))
	}
	require.Equal(t, l1Fee, adjustedGPOFee, "GPO reports L1 fee mismatch")

	require.Equal(t, receipt.L1Fee, l1Fee, "l1 fee in receipt is correct")
	if !sys.RollupConfig.IsEcotone(header.Time) { // FeeScalar receipt attribute is removed as of Ecotone
		require.Equal(t,
			new(big.Float).Mul(
				new(big.Float).SetInt(l1Header.BaseFee),
				new(big.Float).Mul(new(big.Float).SetInt(receipt.L1GasUsed), receipt.FeeScalar),
			),
			new(big.Float).SetInt(receipt.L1Fee), "fee field in receipt matches gas used times scalar times base fee")
	}

	if sys.RollupConfig.IsIsthmus(header.Time) {
		baseOperatorFee := new(big.Int).Mul(
			gasUsed,
			new(big.Int).SetUint64(uint64(cfg.DeployConfig.GasPriceOracleOperatorFeeScalar)),
		)

		scaledOperatorFee := new(big.Int)
		if sys.RollupConfig.IsJovian(header.Time) {
			// For Jovian, multiply by 100.
			scaledOperatorFee.Mul(baseOperatorFee, new(big.Int).SetUint64(uint64(100)))
		} else {
			// For Isthmus, divide by 1e6.
			scaledOperatorFee.Div(baseOperatorFee, new(big.Int).SetUint64(uint64(1e6)))
		}

		expectedOperatorFee := new(big.Int).Add(
			scaledOperatorFee,
			new(big.Int).SetUint64(cfg.DeployConfig.GasPriceOracleOperatorFeeConstant),
		)

		require.True(t, expectedOperatorFee.Cmp(operatorFee) == 0,
			"operator fee is correct",
		)

		if cfg.DeployConfig.GasPriceOracleOperatorFeeScalar == 0 && cfg.DeployConfig.GasPriceOracleOperatorFeeConstant == 0 {
			// if both operator fee params are 0, they aren't added to receipts.
			require.Nil(t, receipt.OperatorFeeScalar, "operator fee constant in receipt is nil")
			require.Nil(t, receipt.OperatorFeeConstant, "operator fee constant in receipt is nil")
		} else {
			require.Equal(t, cfg.DeployConfig.GasPriceOracleOperatorFeeScalar, uint32(*receipt.OperatorFeeScalar), "operator fee constant in receipt is correct")
			require.Equal(t, cfg.DeployConfig.GasPriceOracleOperatorFeeConstant, *receipt.OperatorFeeConstant, "operator fee constant in receipt is correct")
		}
	}

	// Calculate total fee
	baseFeeRecipientDiff.Add(baseFeeRecipientDiff, coinbaseDiff)
	totalFee := new(big.Int).Add(baseFeeRecipientDiff, l1FeeRecipientDiff)
	totalFee = new(big.Int).Add(totalFee, operatorFeeVaultDiff)
	balanceDiff := new(big.Int).Sub(startBalance, endBalance)
	balanceDiff.Sub(balanceDiff, transferAmount)
	require.Equal(t, balanceDiff, totalFee, "balances should add up")
}

// l1Cost returns the L1 data fee of a user transaction in the L2 block at blockTime, whose L1
// attributes are l1Info.
func l1Cost(rollupCfg *rollup.Config, blockTime uint64, rcd opfees.RollupCostData, l1Info *derive.L1BlockInfo) *big.Int {
	// The Ecotone activation block still carries Bedrock L1 attributes and is charged as Bedrock.
	if !rollupCfg.IsEcotone(blockTime) || rollupCfg.IsEcotoneActivationBlock(blockTime) {
		calldataGas := rcd.Zeroes*params.TxDataZeroGas + rcd.Ones*params.TxDataNonZeroGasEIP2028
		if !rollupCfg.IsRegolith(blockTime) {
			// Before Regolith, 68 non-zero bytes are charged on top for the signature and RLP overhead.
			calldataGas += 68 * params.TxDataNonZeroGasEIP2028
		}
		overhead := new(big.Int).SetBytes(l1Info.L1FeeOverhead[:])
		scalar := new(big.Int).SetBytes(l1Info.L1FeeScalar[:])
		return opfees.L1CostBedrock(calldataGas, l1Info.BaseFee, overhead, scalar)
	}
	feeParams := opfees.L1FeeParams{
		L1BaseFee:     l1Info.BaseFee,
		L1BlobBaseFee: l1Info.BlobBaseFee,
		BaseFeeScalar: new(big.Int).SetUint64(uint64(l1Info.BaseFeeScalar)),
		BlobFeeScalar: new(big.Int).SetUint64(uint64(l1Info.BlobBaseFeeScalar)),
	}
	if rollupCfg.IsFjord(blockTime) {
		return opfees.L1CostFjord(rcd, feeParams)
	}
	return opfees.L1CostEcotone(rcd, feeParams)
}

// operatorCost returns the operator fee of a transaction that used gasUsed in the L2 block at
// blockTime, whose L1 attributes are l1Info.
func operatorCost(rollupCfg *rollup.Config, blockTime, gasUsed uint64, l1Info *derive.L1BlockInfo) *big.Int {
	scalar, constant := uint64(l1Info.OperatorFeeScalar), l1Info.OperatorFeeConstant
	switch {
	case rollupCfg.IsJovian(blockTime):
		return opfees.OperatorCostJovian(gasUsed, scalar, constant)
	case rollupCfg.IsIsthmus(blockTime):
		return opfees.OperatorCostIsthmus(gasUsed, scalar, constant)
	default:
		return new(big.Int)
	}
}
