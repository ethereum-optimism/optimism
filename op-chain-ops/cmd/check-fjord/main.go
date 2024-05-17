package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum/core/types"
)

func main() {
	app := cli.NewApp()
	app.Name = "check-fjord"
	app.Usage = "Check Fjord upgrade results."
	app.Description = "Check Fjord upgrade results."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		makeCommand("all", checkAll),
		makeCommand("rip-7212", checkRIP7212),
		{
			Name: "fast-lz",
			Subcommands: []*cli.Command{
				makeCommand("gas-price-oracle", checkGasPriceOracle),
				makeCommand("tx-send-eth", checkTxSendEth),
				makeCommand("tx-all-zero", checkTxAllZero),
				makeCommand("tx-all-42", checkTxAll42),
				makeCommand("tx-random", checkTxRandom),
				makeCommand("all", checkAllFastLz),
			},
			Flags:  makeFlags(),
			Action: makeCommandAction(checkAllFastLz),
		},
		makeCommand("fastLz", checkGasPriceOracle),
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}

type actionEnv struct {
	log       log.Logger
	l1        *ethclient.Client
	l2        *ethclient.Client
	rollupCl  *sources.RollupClient
	key       *ecdsa.PrivateKey
	addr      common.Address
	gasUsed   uint64
	l1GasUsed uint64
}

type CheckAction func(ctx context.Context, env *actionEnv) error

var (
	prefix     = "CHECK_FJORD"
	EndpointL1 = &cli.StringFlag{
		Name:    "l1",
		Usage:   "L1 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L1"),
		Value:   "http://localhost:8545",
	}
	EndpointL2 = &cli.StringFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
		Value:   "http://localhost:9545",
	}
	EndpointRollup = &cli.StringFlag{
		Name:    "rollup",
		Usage:   "L2 rollup-node RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "ROLLUP"),
		Value:   "http://localhost:7545",
	}
	AccountKey = &cli.StringFlag{
		Name:    "account",
		Usage:   "Private key (hex-formatted string) of test account to perform test txs with",
		EnvVars: op_service.PrefixEnvVar(prefix, "ACCOUNT"),
	}
)

func makeFlags() []cli.Flag {
	flags := []cli.Flag{
		EndpointL1,
		EndpointL2,
		EndpointRollup,
		AccountKey,
	}
	return append(flags, oplog.CLIFlags(prefix)...)
}

func makeCommand(name string, fn CheckAction) *cli.Command {
	return &cli.Command{
		Name:   name,
		Action: makeCommandAction(fn),
		Flags:  cliapp.ProtectFlags(makeFlags()),
	}
}

func makeCommandAction(fn CheckAction) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		logCfg := oplog.ReadCLIConfig(c)
		logger := oplog.NewLogger(c.App.Writer, logCfg)

		c.Context = opio.CancelOnInterrupt(c.Context)
		l1Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL1.Name))
		if err != nil {
			return fmt.Errorf("failed to dial L1 RPC: %w", err)
		}
		l2Cl, err := ethclient.DialContext(c.Context, c.String(EndpointL2.Name))
		if err != nil {
			return fmt.Errorf("failed to dial L2 RPC: %w", err)
		}
		rollupCl, err := dial.DialRollupClientWithTimeout(c.Context, time.Second*20, logger, c.String(EndpointRollup.Name))
		if err != nil {
			return fmt.Errorf("failed to dial rollup node RPC: %w", err)
		}
		key, err := crypto.HexToECDSA(c.String(AccountKey.Name))
		if err != nil {
			return fmt.Errorf("failed to parse test private key: %w", err)
		}
		if err := fn(c.Context, &actionEnv{
			log:      logger,
			l1:       l1Cl,
			l2:       l2Cl,
			rollupCl: rollupCl,
			key:      key,
			addr:     crypto.PubkeyToAddress(key.PublicKey),
		}); err != nil {
			return fmt.Errorf("command error: %w", err)
		}
		return nil
	}
}

var (
	rip7212Precompile = common.HexToAddress("0x0000000000000000000000000000000000000100")
	invalid7212Data   = []byte{0x00}
	// This is a valid hash, r, s, x, y params for RIP-7212 taken from:
	// https://gist.github.com/ulerdogan/8f1714895e23a54147fc529ea30517eb
	valid7212Data = common.FromHex("4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e")
)

func checkRIP7212(ctx context.Context, env *actionEnv) error {
	env.log.Info("checking rip-7212")
	// invalid request returns empty response, this is how the spec denotes an error.
	response, err := env.l2.CallContract(ctx, ethereum.CallMsg{
		To:   &rip7212Precompile,
		Data: invalid7212Data,
	}, nil)
	if err != nil {
		return err
	}

	if !bytes.Equal(response, []byte{}) {
		return fmt.Errorf("precompile should return empty response for invalid signature, but got %s", response)
	}
	env.log.Info("confirmed precompile returns empty reponse for invalid signature")

	// valid request returns one
	response, err = env.l2.CallContract(ctx, ethereum.CallMsg{
		To:   &rip7212Precompile,
		Data: valid7212Data,
	}, nil)
	if err != nil {
		return err
	}
	if !bytes.Equal(response, common.LeftPadBytes([]byte{1}, 32)) {
		return fmt.Errorf("precompile should return 1 for valid signature, but got %s", response)
	}
	env.log.Info("confirmed precompile returns 1 for valid signature")

	return nil
}

func checkAllFastLz(ctx context.Context, env *actionEnv) error {
	env.log.Info("beginning all FastLz feature tests")
	if err := checkGasPriceOracle(ctx, env); err != nil {
		return fmt.Errorf("gas-price-oracle: %w", err)
	}
	if err := checkTxSendEth(ctx, env); err != nil {
		return fmt.Errorf("tx-send-eth: %w", err)
	}
	if err := checkTxAllZero(ctx, env); err != nil {
		return fmt.Errorf("tx-all-zero: %w", err)
	}
	if err := checkTxAll42(ctx, env); err != nil {
		return fmt.Errorf("tx-all-42: %w", err)
	}
	if err := checkTxRandom(ctx, env); err != nil {
		return fmt.Errorf("tx-random: %w", err)
	}
	env.log.Info("completed all FastLz feature tests successfully")
	return nil
}

func checkGasPriceOracle(ctx context.Context, env *actionEnv) error {
	env.log.Info("beginning GasPriceOracle checks")
	expectedGasPriceOracleAddress := crypto.CreateAddress(derive.GasPriceOracleFjordDeployerAddress, 0)

	// Gas Price Oracle Proxy is updated
	updatedGasPriceOracleAddress, err := env.l2.StorageAt(context.Background(), predeploys.GasPriceOracleAddr, genesis.ImplementationSlot, nil)
	if err != nil {
		return err
	}
	if expectedGasPriceOracleAddress != common.BytesToAddress(updatedGasPriceOracleAddress) {
		return fmt.Errorf("expected GasPriceOracle address does not match actual address")
	}
	env.log.Info("confirmed GasPriceOracle address meets expectation")

	code, err := env.l2.CodeAt(context.Background(), expectedGasPriceOracleAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to read codeAt expectedGasPriceOracleAddress")
	}
	if len(code) == 0 {
		return fmt.Errorf("codeAt expectedGasPriceOracleAddress is empty")
	}
	codeHash := crypto.Keccak256Hash(code)
	fjordGasPriceOracleCodeHash := common.HexToHash("0xa88fa50a2745b15e6794247614b5298483070661adacb8d32d716434ed24c6b2")

	if codeHash != fjordGasPriceOracleCodeHash {
		return fmt.Errorf("GasPriceOracle codeHash does not match expectation")
	}
	env.log.Info("confirmed GasPriceOracle codeHash meets expectation")

	// Get gas price from oracle
	gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings for new GaspriceOracleCaller")
	}

	// Check that Fjord was activated
	isFjord, err := gasPriceOracle.IsFjord(nil)
	if err != nil {
		return fmt.Errorf("failed when calling GasPriceOracle.IsFjord function: %w", err)
	}
	if !isFjord {
		return fmt.Errorf("GasPriceOracle.IsFjord function returned false")
	}
	env.log.Info("confirmed GasPriceOracle reports Fjord is activated")
	return nil
}

func sendTxAndCheckFees(ctx context.Context, env *actionEnv, to *common.Address, txData []byte) error {
	gasPriceOracle, err := bindings.NewGasPriceOracleCaller(predeploys.GasPriceOracleAddr, env.l2)
	if err != nil {
		return fmt.Errorf("creating bindings for new GaspriceOracleCaller: %w", err)
	}

	tx, err := execTx(ctx, to, txData, false, env)
	if err != nil {
		return fmt.Errorf("executing tx: %w", err)
	}
	blockHash := tx.receipt.BlockHash
	opts := &bind.CallOpts{BlockHash: blockHash}
	txUnsigned, err := tx.unsigned.MarshalBinary()
	if err != nil {
		return fmt.Errorf("binary-encoding unsigned tx: %w", err)
	}
	txSigned, err := tx.signed.MarshalBinary()
	if err != nil {
		return fmt.Errorf("binary-encoding signed tx: %w", err)
	}
	env.log.Info("Transaction confirmed",
		"unsigned_len", len(txUnsigned),
		"signed", len(txSigned),
		"block_hash", blockHash,
	)

	gpoL1GasUsed, err := gasPriceOracle.GetL1GasUsed(opts, txUnsigned)
	if err != nil {
		return fmt.Errorf("calling GasPriceOracle.GetL1GasUsed: %w", err)
	}

	env.log.Info("retrieved L1 gas used", "gpoL1GasUsed", gpoL1GasUsed.Uint64())

	// Check that GetL1Fee takes into account fast LZ
	gpoFee, err := gasPriceOracle.GetL1Fee(opts, txUnsigned)
	if err != nil {
		return fmt.Errorf("calling GasPriceOracle.GetL1Fee: %w", err)
	}

	gethGPOFee, err := fjordL1Cost(gasPriceOracle, blockHash, uint64(types.FlzCompressLen(txUnsigned)+68))
	if err != nil {
		return fmt.Errorf("calculating GPO fjordL1Cost: %w", err)
	}
	if gethGPOFee.Uint64() != gpoFee.Uint64() {
		return fmt.Errorf("gethGPOFee (%s) does not match gpoFee (%s)", gethGPOFee, gpoFee)
	}
	env.log.Info("gethGPOFee matches gpoFee")

	gethFee, err := fjordL1Cost(gasPriceOracle, blockHash, uint64(types.FlzCompressLen(txSigned)))
	if err != nil {
		return fmt.Errorf("calculating receipt fjordL1Cost: %w", err)
	}
	if gethFee.Uint64() != tx.receipt.L1Fee.Uint64() {
		return fmt.Errorf("gethFee (%s) does not match receipt L1Fee (%s)", gethFee, tx.receipt.L1Fee)
	}
	env.log.Info("gethFee matches receipt fee")

	// Check that L1FeeUpperBound works
	upperBound, err := gasPriceOracle.GetL1FeeUpperBound(opts, big.NewInt(int64(len(txUnsigned))))
	if err != nil {
		return fmt.Errorf("failed when calling GasPriceOracle.GetL1FeeUpperBound function: %w", err)
	}

	txLenGPO := len(txUnsigned) + 68
	flzUpperBound := uint64(txLenGPO + txLenGPO/255 + 16)
	upperBoundCost, err := fjordL1Cost(gasPriceOracle, blockHash, flzUpperBound)
	if err != nil {
		return fmt.Errorf("failed to calculate fjordL1Cost: %w", err)
	}
	if upperBoundCost.Uint64() != upperBound.Uint64() {
		return fmt.Errorf("upperBound (%s) does not meet expectation (%s)", upperBound, upperBoundCost)
	}
	env.log.Info("GPO upper bound matches")
	return nil
}

func checkTxSendEth(ctx context.Context, env *actionEnv) error {
	txData := []byte(nil)
	to := &env.addr
	env.log.Info("Attempting tx-send-eth...")
	return sendTxAndCheckFees(ctx, env, to, txData)
}

func checkTxAllZero(ctx context.Context, env *actionEnv) error {
	txData := make([]byte, 256)
	for i := range txData {
		txData[i] = 0x00
	}
	to := &env.addr
	env.log.Info("Attempting tx-all-zero...")
	return sendTxAndCheckFees(ctx, env, to, txData)
}

func checkTxAll42(ctx context.Context, env *actionEnv) error {
	txData := make([]byte, 256)
	for i := range txData {
		txData[i] = 0x42
	}
	to := &env.addr
	env.log.Info("Attempting tx-all-42...")
	return sendTxAndCheckFees(ctx, env, to, txData)
}

func checkTxRandom(ctx context.Context, env *actionEnv) error {
	txData := make([]byte, 256)
	rand.Read(txData)
	to := &env.addr
	env.log.Info("Attempting tx-random...")
	return sendTxAndCheckFees(ctx, env, to, txData)
}

func fjordL1Cost(gasPriceOracle *bindings.GasPriceOracleCaller, block common.Hash, fastLzSize uint64) (*big.Int, error) {
	opts := &bind.CallOpts{BlockHash: block}
	baseFeeScalar, err := gasPriceOracle.BaseFeeScalar(opts)
	if err != nil {
		return nil, err
	}
	l1BaseFee, err := gasPriceOracle.L1BaseFee(opts)
	if err != nil {
		return nil, err
	}
	blobBaseFeeScalar, err := gasPriceOracle.BlobBaseFeeScalar(opts)
	if err != nil {
		return nil, err
	}
	blobBaseFee, err := gasPriceOracle.BlobBaseFee(opts)
	if err != nil {
		return nil, err
	}

	costFunc := types.NewL1CostFuncFjord(
		l1BaseFee,
		blobBaseFee,
		new(big.Int).SetUint64(uint64(baseFeeScalar)),
		new(big.Int).SetUint64(uint64(blobBaseFeeScalar)))

	fee, _ := costFunc(types.RollupCostData{FastLzSize: fastLzSize})
	return fee, nil
}

type txExecution struct {
	unsigned *types.Transaction
	signed   *types.Transaction
	receipt  *types.Receipt
}

func execTx(ctx context.Context, to *common.Address, data []byte, expectRevert bool, env *actionEnv) (*txExecution, error) {
	nonce, err := env.l2.PendingNonceAt(ctx, env.addr)
	if err != nil {
		return nil, fmt.Errorf("pending nonce retrieval failed: %w", err)
	}
	head, err := env.l2.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve head header: %w", err)
	}

	tip := big.NewInt(params.GWei)
	maxFee := new(big.Int).Mul(head.BaseFee, big.NewInt(2))
	maxFee = maxFee.Add(maxFee, tip)

	chainID, err := env.l2.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chainID: %w", err)
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: maxFee,
		Gas:       500000,
		To:        to,
		Data:      data,
		Value:     big.NewInt(0),
	})

	signer := types.NewCancunSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, env.key)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	env.log.Info("sending tx", "txhash", signedTx.Hash(), "to", to, "data", hexutil.Bytes(data))
	if err := env.l2.SendTransaction(ctx, signedTx); err != nil {
		return nil, fmt.Errorf("failed to send tx: %w", err)
	}
	for i := 0; i < 30; i++ {
		env.log.Info("checking confirmation...", "txhash", signedTx.Hash())
		receipt, err := env.l2.TransactionReceipt(context.Background(), signedTx.Hash())
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				env.log.Info("not found yet, waiting...")
				time.Sleep(time.Second)
				continue
			} else {
				return nil, fmt.Errorf("error while checking tx receipt: %w", err)
			}
		}
		env.RecordGasUsed(receipt)
		if expectRevert {
			if receipt.Status == types.ReceiptStatusFailed {
				env.log.Info("tx reverted as expected", "txhash", signedTx.Hash())
				return &txExecution{unsigned: tx, signed: signedTx, receipt: receipt}, nil
			} else {
				return nil, fmt.Errorf("tx %s unexpectedly completed without revert", signedTx.Hash())
			}
		} else {
			if receipt.Status == types.ReceiptStatusSuccessful {
				env.log.Info("tx confirmed", "txhash", signedTx.Hash())
				return &txExecution{unsigned: tx, signed: signedTx, receipt: receipt}, nil
			} else {
				return nil, fmt.Errorf("tx %s failed", signedTx.Hash())
			}
		}
	}
	return nil, fmt.Errorf("confirming tx: %s", signedTx.Hash())
}

func (ae *actionEnv) RecordGasUsed(rec *types.Receipt) {
	ae.gasUsed += rec.GasUsed
	ae.l1GasUsed += rec.L1GasUsed.Uint64()
	ae.log.Debug("Recorded tx receipt gas", "gas_used", rec.GasUsed, "l1_gas_used", rec.L1GasUsed)
}

func checkAll(ctx context.Context, env *actionEnv) error {
	bal, err := env.l2.BalanceAt(ctx, env.addr, nil)
	if err != nil {
		return fmt.Errorf("failed to check balance of account: %w", err)
	}
	env.log.Info("starting checks, tx account", "addr", env.addr, "balance_wei", bal)

	if err = checkRIP7212(ctx, env); err != nil {
		return fmt.Errorf("rip-7212: %w", err)
	}

	if err = checkAllFastLz(ctx, env); err != nil {
		return fmt.Errorf("fastLz: %w", err)
	}

	finbal, err := env.l2.BalanceAt(ctx, env.addr, nil)
	if err != nil {
		return fmt.Errorf("failed to check final balance of account: %w", err)
	}
	env.log.Info("completed all tests successfully!",
		"addr", env.addr, "balance_wei", finbal,
		"spent_wei", new(big.Int).Sub(bal, finbal),
		"gas_used_total", env.gasUsed,
		"l1_gas_used_total", env.l1GasUsed,
	)
	return nil
}
