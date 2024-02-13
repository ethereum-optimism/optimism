package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

func main() {
	app := cli.NewApp()
	app.Name = "check-ecotone"
	app.Usage = "Check Ecotone upgrade results."
	app.Description = "Check Ecotone upgrade results."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr
	app.Commands = []*cli.Command{
		{
			Name: "cancun",
			Subcommands: []*cli.Command{
				makeCommand("eip-1153-tstore", checkEIP1153),
				makeCommand("eip-4844-blobhash", checkBlobDataHash),
				makeCommand("eip-4844-precompile", check4844Precompile),
				makeCommand("eip-5656-mcopy", checkMcopy),
				makeCommand("eip-6780-selfdestruct", checkSelfdestruct),
				makeCommand("eip-4844-blobtx", checkBlobTxDenial),
				makeCommand("eip-4788", checkBeaconBlockRoot),
				makeCommand("all", checkAllCancun),
			},
			Flags:  makeFlags(),
			Action: makeCommandAction(checkAllCancun),
		},
		makeCommand("upgrade", checkUpgradeTxs),
		{
			Name: "contracts",
			Subcommands: []*cli.Command{
				makeCommand("l1block", checkL1Block),
				makeCommand("gpo", checkGPO),
			},
		},
		makeCommand("fees", checkL1Fees),
		makeCommand("all", checkALL),
		{
			Name: "gen-key",
			Action: func(c *cli.Context) error {
				key, err := crypto.GenerateKey()
				if err != nil {
					return err
				}
				fmt.Println("address: " + crypto.PubkeyToAddress(key.PublicKey).String())
				return crypto.SaveECDSA("hotkey.txt", key)
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v", err)
		os.Exit(1)
	}
}

type actionEnv struct {
	log      log.Logger
	l1       *ethclient.Client
	l2       *ethclient.Client
	rollupCl *sources.RollupClient
	key      *ecdsa.PrivateKey
	addr     common.Address
}

type CheckAction func(ctx context.Context, env *actionEnv) error

var (
	prefix     = "CHECK_ECOTONE"
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
		Value:   "http://localhost:5545",
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

// assuming a 0 (fail) or non-zero (success) on the stack, this performs a revert or self-destruct
func conditionalCode(data []byte) []byte {
	suffix := []byte{
		// add jump dest
		byte(vm.PUSH4),
		0xff, 0xff, 0xff, 0xff,
		byte(vm.JUMPI),
		// error case
		byte(vm.PUSH0),
		byte(vm.PUSH0),
		byte(vm.REVERT),
		// success case
		byte(vm.JUMPDEST),
		byte(vm.CALLER),
		byte(vm.SELFDESTRUCT),
		byte(vm.STOP),
	}
	binary.BigEndian.PutUint32(suffix[1:5], uint32(len(data))+9)
	out := make([]byte, 0, len(data)+len(suffix))
	out = append(out, data...)
	out = append(out, suffix...)
	return out
}

func checkEIP1153(ctx context.Context, env *actionEnv) error {
	input := conditionalCode([]byte{
		// store 0xc0ffee at 0x42
		byte(vm.PUSH3),
		0xc0, 0xff, 0xee,
		byte(vm.PUSH1),
		0x42,
		byte(vm.TSTORE),
		// retrieve it
		byte(vm.PUSH1),
		0x42,
		byte(vm.TLOAD),
		// check value
		byte(vm.PUSH3),
		0xc0, 0xff, 0xee,
		byte(vm.EQ),
	})
	return execTx(ctx, nil, input, false, env)
}

func checkBlobDataHash(ctx context.Context, env *actionEnv) error {
	// revert on non-blob tx
	input := []byte{
		byte(vm.BLOBHASH),
	}
	return execTx(ctx, nil, input, true, env)
}

func check4844Precompile(ctx context.Context, env *actionEnv) error {
	head, err := env.l2.HeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}
	// reverts on all-0 time input
	if err := execTx(ctx, &predeploys.EIP4788ContractAddr, make([]byte, 32), true, env); err != nil {
		return fmt.Errorf("expected revert on empty input: %w", err)
	}

	conf, err := env.rollupCl.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("config retrieval failed: %w", err)
	}
	t := head.Time
	alignment := head.Time % conf.BlockTime
	for i := 0; i < 20; i++ {
		ti := t - uint64(i)
		if !conf.IsEcotone(ti) {
			continue
		}
		env.log.Info("Beacon block root query timestamp", "query_timestamp", ti)
		// revert when timestamp doesn't exist (when not aligned it won't exist)
		input := new(uint256.Int).SetUint64(ti).Bytes32()
		if err := execTx(ctx, &predeploys.EIP4788ContractAddr, input[:], ti%conf.BlockTime != alignment, env); err != nil {
			return fmt.Errorf("failed at t = %d", ti)
		}
	}
	return nil
}

func checkMcopy(ctx context.Context, env *actionEnv) error {
	input := conditionalCode([]byte{
		// push info & mstore it
		byte(vm.PUSH3),
		0xc0, 0xff, 0xee,
		byte(vm.PUSH0), // store at 0
		byte(vm.MSTORE),
		// copy the memory
		byte(vm.PUSH1), // length
		0x2,            // only copy the C0FF part
		byte(vm.PUSH1), // src
		32 - 3,         // right-aligned bytes3
		byte(vm.PUSH1), // dst
		0x42,
		byte(vm.MCOPY),
		byte(vm.PUSH1),  // copy from destination
		0x42 - (32 - 3), // a little to the left, so it's left-padded
		byte(vm.MLOAD),  // load the memory from copied location
		// check if it matches, with zero 3rd byte
		byte(vm.PUSH3),
		0xc0, 0xff, 0x00,
		byte(vm.EQ),
	})
	return execTx(ctx, nil, input, false, env)
}

func checkSelfdestruct(ctx context.Context, env *actionEnv) error {
	input := conditionalCode([]byte{
		// prepare code in memory
		byte(vm.PUSH2), // value
		byte(vm.CALLER),
		byte(vm.SELFDESTRUCT),
		byte(vm.PUSH1), // offset
		byte(vm.MSTORE),
		// create contract
		byte(vm.PUSH1), // size, just a 2 byte contract
		2,
		byte(vm.PUSH0),       // ETH value
		byte(vm.PUSH0),       // offset
		byte(vm.CREATE),      // pushes address on stack. Contract will immediately self-destruct
		byte(vm.EXTCODESIZE), // size should be 0
		byte(vm.ISZERO),      // check that it is
	})
	return execTx(ctx, nil, input, false, env)
}

func execTx(ctx context.Context, to *common.Address, data []byte, expectRevert bool, env *actionEnv) error {
	nonce, err := env.l2.PendingNonceAt(ctx, env.addr)
	if err != nil {
		return fmt.Errorf("pending nonce retrieval failed: %w", err)
	}
	head, err := env.l2.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve head header: %w", err)
	}

	tip := big.NewInt(params.GWei)
	maxFee := new(big.Int).Mul(head.BaseFee, big.NewInt(2))
	maxFee = maxFee.Add(maxFee, tip)

	chainID, err := env.l2.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get chainID: %w", err)
	}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce,
		GasTipCap: tip, GasFeeCap: maxFee, Gas: 500000, To: to, Data: data,
	})
	signer := types.NewCancunSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, env.key)
	if err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	env.log.Info("sending tx", "txhash", signedTx.Hash(), "to", to, "data", hexutil.Bytes(data))
	if err := env.l2.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
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
				return fmt.Errorf("error while checking tx receipt: %w", err)
			}
		}
		if expectRevert {
			if receipt.Status == types.ReceiptStatusFailed {
				env.log.Info("tx reverted as expected", "txhash", signedTx.Hash())
				return nil
			} else {
				return fmt.Errorf("tx %s unexpectedly completed without revert", signedTx.Hash())
			}
		} else {
			if receipt.Status == types.ReceiptStatusSuccessful {
				env.log.Info("tx confirmed", "txhash", signedTx.Hash())
				return nil
			} else {
				return fmt.Errorf("tx %s failed", signedTx.Hash())
			}
		}
	}
	return fmt.Errorf("failed to confirm tx: %s", signedTx.Hash())
}

func checkBlobTxDenial(ctx context.Context, env *actionEnv) error {
	// verify we cannot submit a blob tx to RPC
	var blob eth.Blob
	_, err := rand.Read(blob[:])
	if err != nil {
		return fmt.Errorf("failed randomnes: %w", err)
	}
	// get the field-elements into a valid range
	for i := 0; i < 4096; i++ {
		blob[32*i] &= 0b0011_1111
	}
	sidecar, blobHashes, err := txmgr.MakeSidecar([]*eth.Blob{&blob})
	if err != nil {
		return fmt.Errorf("failed to make sidecar: %w", err)
	}
	latestHeader, err := env.l1.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get header: %w", err)
	}
	if latestHeader.ExcessBlobGas == nil {
		return fmt.Errorf("the L1 block %s (time %d) is not ecotone yet", latestHeader.Hash(), latestHeader.Time)
	}
	blobBaseFee := eip4844.CalcBlobFee(*latestHeader.ExcessBlobGas)
	blobFeeCap := new(uint256.Int).Mul(uint256.NewInt(2), uint256.MustFromBig(blobBaseFee))
	if blobFeeCap.Lt(uint256.NewInt(params.GWei)) { // ensure we meet 1 gwei geth tx-pool minimum
		blobFeeCap = uint256.NewInt(params.GWei)
	}
	gasTipCap := big.NewInt(2 * params.GWei)
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(latestHeader.BaseFee, big.NewInt(2)))

	nonce, err := env.l1.PendingNonceAt(ctx, env.addr)
	if err != nil {
		return fmt.Errorf("failed to get pending nonce: %w", err)
	}
	rollupCfg, err := env.rollupCl.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	txData := &types.BlobTx{
		To:         rollupCfg.BatchInboxAddress,
		Data:       nil,
		Gas:        params.TxGas, // intrinsic gas only
		BlobHashes: blobHashes,
		Sidecar:    sidecar,
		ChainID:    uint256.MustFromBig(rollupCfg.L2ChainID),
		GasTipCap:  uint256.MustFromBig(gasTipCap),
		GasFeeCap:  uint256.MustFromBig(gasFeeCap),
		BlobFeeCap: blobFeeCap,
		Value:      uint256.NewInt(0),
		Nonce:      nonce,
	}
	// bypass signer filter by creating it manually and using the L2 chain ID
	signer := types.NewCancunSigner(rollupCfg.L2ChainID)
	tx, err := types.SignNewTx(env.key, signer, txData)
	if err != nil {
		return fmt.Errorf("failed to sign blob tx: %w", err)
	}
	err = env.l2.SendTransaction(ctx, tx)
	if err == nil {
		return errors.New("expected tx error, but got none")
	}
	if !strings.Contains(err.Error(), "transaction type not supported") {
		return fmt.Errorf("unexpected tx submission error: %w", err)
	}
	return nil
}

func checkBeaconBlockRoot(ctx context.Context, env *actionEnv) error {
	latest, err := env.l2.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}
	if latest.ParentBeaconRoot == nil {
		return fmt.Errorf("block %d misses beacon block root", latest.Number)
	}
	beaconBlockRootsContract, err := env.l2.CodeAt(ctx, predeploys.EIP4788ContractAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to retrieve beacon block root contract code: %w", err)
	}
	codeHash := crypto.Keccak256Hash(beaconBlockRootsContract)
	if codeHash != predeploys.EIP4788ContractCodeHash {
		return fmt.Errorf("unexpected 4788 contract code: %w", err)
	}

	rollupCfg, err := env.rollupCl.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	l2RPC := client.NewBaseRPCClient(env.l2.Client())
	l2EthCl, err := sources.NewL2Client(l2RPC, env.log, nil,
		sources.L2ClientDefaultConfig(rollupCfg, false))
	if err != nil {
		return fmt.Errorf("failed to create eth client")
	}
	result, err := l2EthCl.GetProof(ctx, predeploys.EIP4788ContractAddr, nil, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get account proof to inspect storage-root")
	}
	if result.StorageHash == types.EmptyRootHash {
		return fmt.Errorf("expected contract storage to be set, but got none (%s)",
			result.StorageHash)
	}

	payload, err := l2EthCl.PayloadByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get head ref: %w", err)
	}
	if payload.ParentBeaconBlockRoot == nil {
		return fmt.Errorf("payload %s misses parent beacon block root", payload.ExecutionPayload.ID())
	}
	headRef, err := derive.PayloadToBlockRef(rollupCfg, payload.ExecutionPayload)
	if err != nil {
		return fmt.Errorf("failed to convert to block-ref: %w", err)
	}
	l1Header, err := env.l1.HeaderByHash(ctx, headRef.L1Origin.Hash)
	if err != nil {
		return fmt.Errorf("failed to retrieve matching L1 block %s: %w", headRef, err)
	}
	var l1ParentBeaconBlockRoot common.Hash // zero before Dencun activates on L1
	if l1Header.ParentBeaconRoot != nil {
		l1ParentBeaconBlockRoot = *l1Header.ParentBeaconRoot
	}
	if l1ParentBeaconBlockRoot != *payload.ParentBeaconBlockRoot {
		return fmt.Errorf("parent beacon block root mismatch, L1: %s, L2: %s",
			l1ParentBeaconBlockRoot, *payload.ParentBeaconBlockRoot)
	}
	return nil
}

func checkAllCancun(ctx context.Context, env *actionEnv) error {
	if err := checkEIP1153(ctx, env); err != nil {
		return fmt.Errorf("eip-1153 error: %w", err)
	}
	if err := checkBlobDataHash(ctx, env); err != nil {
		return fmt.Errorf("eip-4844 blobhash error: %w", err)
	}
	if err := check4844Precompile(ctx, env); err != nil {
		return fmt.Errorf("eip-4844 precompile error: %w", err)
	}
	if err := checkMcopy(ctx, env); err != nil {
		return fmt.Errorf("eip-5656 mcopy error: %w", err)
	}
	if err := checkSelfdestruct(ctx, env); err != nil {
		return fmt.Errorf("eip-6780 selfdestruct error: %w", err)
	}
	if err := checkBlobTxDenial(ctx, env); err != nil {
		return fmt.Errorf("eip-4844 blob-tx denial error: %w", err)
	}
	if err := checkBeaconBlockRoot(ctx, env); err != nil {
		return fmt.Errorf("eip-4788 beacon-block-roots error: %w", err)
	}

	return nil
}

func checkUpgradeTxs(ctx context.Context, env *actionEnv) error {
	rollupCfg, err := env.rollupCl.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}

	activationBlockNum := rollupCfg.Genesis.L2.Number +
		((*rollupCfg.EcotoneTime - rollupCfg.Genesis.L2Time) / rollupCfg.BlockTime)
	env.log.Info("upgrade block num", "num", activationBlockNum)
	l2RPC := client.NewBaseRPCClient(env.l2.Client())
	l2EthCl, err := sources.NewL2Client(l2RPC, env.log, nil,
		sources.L2ClientDefaultConfig(rollupCfg, false))
	if err != nil {
		return fmt.Errorf("failed to create eth client")
	}
	activBlock, txs, err := l2EthCl.InfoAndTxsByNumber(ctx, activationBlockNum)
	if err != nil {
		return fmt.Errorf("failed to get activation block: %w", err)
	}
	if len(txs) < 7 {
		return fmt.Errorf("expected at least 7 txs in Ecotone activation block, but got %d", len(txs))
	}
	for i, tx := range txs {
		if !tx.IsDepositTx() {
			return fmt.Errorf("unexpected non-deposit tx in activation block, index %d, hash %s", i, tx.Hash())
		}
	}
	_, receipts, err := l2EthCl.FetchReceipts(ctx, activBlock.Hash())
	if err != nil {
		return fmt.Errorf("failed to fetch receipts of activation block: %w", err)
	}
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			return fmt.Errorf("failed tx receipt: %d", i)
		}
		switch i {
		case 1, 2, 6: // 2 implementations + 4788 contract deployment
			if rec.ContractAddress == (common.Address{}) {
				return fmt.Errorf("expected contract deployment, but got none")
			}
		case 3, 4, 5: // proxy upgrades and setEcotone call
			if rec.ContractAddress != (common.Address{}) {
				return fmt.Errorf("unexpected contract deployment")
			}
		}
	}
	return nil
}

func checkL1Block(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewL1Block(predeploys.L1BlockAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around L1Block contract: %w", err)
	}
	blobBaseFee, err := cl.BlobBaseFee(nil)
	if err != nil {
		return fmt.Errorf("failed to get blob basfee from L1Block contract: %w", err)
	}
	if big.NewInt(0).Cmp(blobBaseFee) == 0 {
		return errors.New("blob basefee must never be 0, EIP specifies minimum of 1")
	}
	return nil
}

func checkGPO(ctx context.Context, env *actionEnv) error {
	cl, err := bindings.NewGasPriceOracle(predeploys.GasPriceOracleAddr, env.l2)
	if err != nil {
		return fmt.Errorf("failed to create bindings around L1Block contract: %w", err)
	}
	_, err = cl.Overhead(nil)
	if err == nil || !strings.Contains(err.Error(), "revert") {
		return fmt.Errorf("expected revert on legacy overhead attribute acccess, but got %w", err)
	}
	_, err = cl.Scalar(nil)
	if err == nil || !strings.Contains(err.Error(), "revert") {
		return fmt.Errorf("expected revert on legacy scalar attribute acccess, but got %w", err)
	}
	isEcotone, err := cl.IsEcotone(nil)
	if err != nil {
		return fmt.Errorf("failed to get ecotone status: %w", err)
	}
	if !isEcotone {
		return fmt.Errorf("GPO is not set to ecotone: %w", err)
	}
	blobBaseFeeScalar, err := cl.BlobBaseFeeScalar(nil)
	if err != nil {
		return fmt.Errorf("unable to get blob basefee scalar: %w", err)
	}
	if blobBaseFeeScalar == 0 {
		env.log.Warn("blob basefee scalar is set to 0. SystemConfig needs to emit scalar change to update.")
	}
	return nil
}

func checkL1Fees(ctx context.Context, env *actionEnv) error {
	rollupCfg, err := env.rollupCl.RollupConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve rollup config: %w", err)
	}
	env.log.Info("making test tx", "addr", env.addr)
	nonce, err := env.l2.PendingNonceAt(ctx, env.addr)
	if err != nil {
		return fmt.Errorf("failed to get pending nonce: %w", err)
	}
	env.log.Info("retrieved account nonce", "nonce", nonce)
	l2RPC := client.NewBaseRPCClient(env.l2.Client())
	l2EthCl, err := sources.NewL2Client(l2RPC, env.log, nil,
		sources.L2ClientDefaultConfig(rollupCfg, false))
	if err != nil {
		return fmt.Errorf("failed to create eth client")
	}
	payload, err := l2EthCl.PayloadByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get head ref: %w", err)
	}
	headRef, err := derive.PayloadToBlockRef(rollupCfg, payload.ExecutionPayload)
	if err != nil {
		return fmt.Errorf("failed to convert to block-ref: %w", err)
	}
	l1Header, err := env.l1.HeaderByHash(ctx, headRef.L1Origin.Hash)
	if err != nil {
		return fmt.Errorf("failed to retrieve matching L1 block %s: %w", headRef, err)
	}
	gasTip := big.NewInt(2 * params.GWei)
	baseFee := (*uint256.Int)(&payload.ExecutionPayload.BaseFeePerGas).ToBig()
	gasMaxFee := new(big.Int).Add(
		new(big.Int).Mul(big.NewInt(2), baseFee), gasTip)
	to := common.Address{1, 2, 3, 5}
	txData := &types.DynamicFeeTx{
		ChainID:    rollupCfg.L2ChainID,
		Nonce:      nonce,
		GasTipCap:  gasTip,
		GasFeeCap:  gasMaxFee,
		Gas:        params.TxGas + 100, // some margin for the calldata
		To:         &to,
		Value:      big.NewInt(3 * params.GWei),
		Data:       []byte("hello"),
		AccessList: nil,
	}
	tx, err := types.SignNewTx(env.key, types.NewLondonSigner(txData.ChainID), txData)
	if err != nil {
		return fmt.Errorf("failed to sign test tx: %w", err)
	}
	env.log.Info("signed tx", "txhash", tx.Hash())
	if err := env.l2.SendTransaction(ctx, tx); err != nil {
		return fmt.Errorf("failed to send test tx: %w", err)
	}
	receipt, err := retry.Do[*types.Receipt](ctx, 20, retry.Fixed(time.Second*2), func() (*types.Receipt, error) {
		return env.l2.TransactionReceipt(ctx, tx.Hash())
	})
	if err != nil {
		return fmt.Errorf("failed to confirm tx %s timely: %w", tx.Hash(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed, gas used: %d", receipt.L1GasUsed)
	}
	env.log.Info("got receipt")
	if receipt.FeeScalar != nil {
		return fmt.Errorf("expected fee scalar attribute to be deprecated, but got %v", receipt.FeeScalar)
	}
	if receipt.L1GasPrice.Cmp(l1Header.BaseFee) != 0 {
		return fmt.Errorf("L1 gas price does not include blob fee component: %d != %d", receipt.L1GasPrice, l1Header.BaseFee)
	}
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to encode tx: %w", err)
	}
	var zero, nonZero uint64
	for _, b := range rawTx {
		if b == 0 {
			zero += 1
		} else {
			nonZero += 1
		}
	}
	expectedCalldataGas := zero*4 + nonZero*16
	env.log.Info("expecting fees", "calldatagas", expectedCalldataGas)
	env.log.Info("paid fees", "l1_fee", receipt.L1Fee, "l1_basefee", receipt.L1GasPrice)
	if new(big.Int).SetUint64(expectedCalldataGas).Cmp(receipt.L1GasUsed) != 0 {
		return fmt.Errorf("expected %d L1 gas, but only spent %d", expectedCalldataGas, receipt.L1GasUsed)
	}
	if big.NewInt(0).Cmp(receipt.L1Fee) >= 0 {
		return fmt.Errorf("calculated to low L1 fee: %d", receipt.L1Fee)
	}
	return nil
}

func checkALL(ctx context.Context, env *actionEnv) error {
	if err := checkAllCancun(ctx, env); err != nil {
		return fmt.Errorf("failed: Cancun error: %w", err)
	}
	if err := checkUpgradeTxs(ctx, env); err != nil {
		return fmt.Errorf("failed: Upgrade-tx error: %w", err)
	}
	if err := checkL1Block(ctx, env); err != nil {
		return fmt.Errorf("failed: L1Block contract error: %w", err)
	}
	if err := checkGPO(ctx, env); err != nil {
		return fmt.Errorf("failed: GPO contract error: %w", err)
	}
	if err := checkL1Fees(ctx, env); err != nil {
		return fmt.Errorf("failed: L1 fees error: %w", err)
	}
	return nil
}
