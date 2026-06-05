package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type config struct {
	rpcURL           string
	sequencerRPCURL  string
	opRbuilderRPCURL string
	privateKeyHex    string
	mnemonic         string
	mnemonicIndex    uint64
	contract         string
	blockNum         uint64
	fundL2           bool
	l1RPCURL         string
	l1PrivateKey     string
	portal           string
	fundAmount       string
	fundGasLimit     uint64
	flashblocksURL   string
	txsPerFlashblock int
	txSpacing        time.Duration
	batchSize        int
	slotCount        uint64
	minUserTxs       int
	attempts         int
	gasLimit         uint64
	deployGasLimit   uint64
	timeout          time.Duration
	receiptTimeout   time.Duration
	pollInterval     time.Duration
	skipReceipts     bool
	skipReplay       bool
	skipSDMOptIn     bool
	jsonOut          bool
}

type workloadResult struct {
	RPCURL          string                  `json:"rpc_url"`
	From            common.Address          `json:"from"`
	Contract        common.Address          `json:"contract"`
	Attempt         int                     `json:"attempt"`
	SubmittedTxs    int                     `json:"submitted_txs"`
	IncludedByBlock map[uint64]int          `json:"included_by_block"`
	Validation      *sdm.ValidationResult   `json:"validation"`
	Flashblocks     []sdm.FlashblockSummary `json:"flashblocks,omitempty"`
}

func main() {
	cfg := parseFlags()
	if err := run(cfg); err != nil {
		log.Fatalf("sdm-devnet failed: %v", err)
	}
}

func parseFlags() config {
	defaultRPC := os.Getenv("L2_RPC_URL")
	if defaultRPC == "" {
		defaultRPC = "http://localhost:55126"
	}

	var cfg config
	flag.StringVar(&cfg.rpcURL, "rpc", defaultRPC, "L2 execution RPC URL (or L2_RPC_URL)")
	flag.StringVar(&cfg.sequencerRPCURL, "sequencer-rpc", firstEnv("SEQUENCER_RPC_URL", "OP_NODE_RPC_URL"), "sequencer/op-node admin RPC URL for admin_setSdmPostExecOptIn (or SEQUENCER_RPC_URL/OP_NODE_RPC_URL)")
	flag.StringVar(&cfg.opRbuilderRPCURL, "op-rbuilder-rpc", os.Getenv("OP_RBUILDER_RPC_URL"), "op-rbuilder admin RPC URL for admin_setSdmPostExecOptIn (or OP_RBUILDER_RPC_URL)")
	flag.StringVar(&cfg.privateKeyHex, "private-key", os.Getenv("PRIVATE_KEY"), "sender private key hex (defaults to devkeys test mnemonic)")
	flag.StringVar(&cfg.mnemonic, "mnemonic", devkeys.TestMnemonic, "mnemonic used when -private-key is empty")
	flag.Uint64Var(&cfg.mnemonicIndex, "mnemonic-index", 10_000, "devkeys user index used when -private-key is empty")
	flag.StringVar(&cfg.contract, "contract", "", "existing StateBloat contract address; deploys one when empty")
	flag.Uint64Var(&cfg.blockNum, "block", 0, "validate an existing block and skip workload submission")
	flag.BoolVar(&cfg.fundL2, "fund-l2", false, "deposit ETH through OptimismPortal if the L2 sender balance is zero")
	flag.StringVar(&cfg.l1RPCURL, "l1-rpc", os.Getenv("L1_RPC_URL"), "L1 RPC URL for -fund-l2")
	flag.StringVar(&cfg.l1PrivateKey, "l1-private-key", "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", "L1 private key for -fund-l2")
	flag.StringVar(&cfg.portal, "portal", "", "OptimismPortalProxy address for -fund-l2")
	flag.StringVar(&cfg.fundAmount, "fund-amount", "1000000000000000000", "wei to deposit with -fund-l2")
	flag.Uint64Var(&cfg.fundGasLimit, "fund-gas-limit", 200_000, "L1 gas limit for OptimismPortal deposit")
	flag.StringVar(&cfg.flashblocksURL, "flashblocks-url", os.Getenv("FLASHBLOCKS_WS_URL"), "op-rbuilder flashblocks websocket URL; enables per-flashblock tx counts")
	flag.IntVar(&cfg.txsPerFlashblock, "txs-per-flashblock", 0, "when -flashblocks-url is set, submit this many txs after each observed flashblock to spread workload across flashblocks")
	flag.DurationVar(&cfg.txSpacing, "tx-spacing", 0, "delay between tx submissions when not using -txs-per-flashblock")
	flag.IntVar(&cfg.batchSize, "batch-size", 12, "number of run(uint256) txs to submit per attempt; keep below per-account txpool limits")
	flag.Uint64Var(&cfg.slotCount, "slot-count", 20, "storage slots touched by each run(uint256) tx")
	flag.IntVar(&cfg.minUserTxs, "min-user-txs", 2, "minimum submitted tx receipts in a block before validation")
	flag.IntVar(&cfg.attempts, "attempts", 3, "workload attempts")
	flag.Uint64Var(&cfg.gasLimit, "gas-limit", 1_000_000, "gas limit for workload txs")
	flag.Uint64Var(&cfg.deployGasLimit, "deploy-gas-limit", 2_000_000, "gas limit for deploying StateBloat")
	flag.DurationVar(&cfg.timeout, "timeout", 2*time.Minute, "overall timeout")
	flag.DurationVar(&cfg.receiptTimeout, "receipt-timeout", 45*time.Second, "timeout for each submitted tx receipt")
	flag.DurationVar(&cfg.pollInterval, "poll-interval", 500*time.Millisecond, "receipt polling interval")
	flag.BoolVar(&cfg.skipReceipts, "skip-receipts", false, "do not compare payload entries against receipt opGasRefund")
	flag.BoolVar(&cfg.skipReplay, "skip-replay", false, "do not call debug_replaySDMBlock")
	flag.BoolVar(&cfg.skipSDMOptIn, "skip-sdm-opt-in", false, "do not call admin_setSdmPostExecOptIn(true) before submitting workload")
	flag.BoolVar(&cfg.jsonOut, "json", false, "print JSON result")
	flag.Parse()
	return cfg
}

func run(cfg config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	priv, from, err := loadKey(cfg)
	if err != nil {
		return err
	}
	sender, err := sdm.DialTxSender(ctx, cfg.rpcURL, priv, from, cfg.gasLimit)
	if err != nil {
		return err
	}
	defer sender.Close()

	validationOpts := sdm.DefaultValidationOptions()
	validationOpts.CheckReceipts = !cfg.skipReceipts
	validationOpts.CheckReplay = !cfg.skipReplay

	var flashblocks *sdm.FlashblockCollector
	if cfg.flashblocksURL != "" && cfg.blockNum == 0 {
		flashblocks, err = sdm.StartFlashblockCollector(ctx, cfg.flashblocksURL)
		if err != nil {
			return err
		}
		log.Printf("connected flashblocks websocket=%s", cfg.flashblocksURL)
	}

	if cfg.blockNum != 0 {
		validation, err := sdm.ValidatePostExecBlock(ctx, sender.RPC, cfg.blockNum, validationOpts)
		if err != nil {
			return err
		}
		result := workloadResult{RPCURL: cfg.rpcURL, From: from, Validation: validation}
		return printResult(cfg, result)
	}

	balance, err := sender.Eth.BalanceAt(ctx, from, nil)
	if err != nil {
		return fmt.Errorf("balance %s: %w", from, err)
	}
	log.Printf("connected rpc=%s chain_id=%s from=%s balance=%s wei", cfg.rpcURL, sender.ChainID, from, balance)

	// The sequencer and op-rbuilder must be opted in to produce PostExec txs, even
	// on an SDM-active chain. The flags are in-memory and lost on restart; this CLI
	// is the standard way to flip them for the devnet sequencer.
	if !cfg.skipSDMOptIn {
		if err := enableSDM(ctx, cfg, sender); err != nil {
			return err
		}
	}
	if balance.Sign() == 0 && cfg.fundL2 {
		if err := fundL2(ctx, cfg, sender); err != nil {
			return err
		}
		balance, err = sender.Eth.BalanceAt(ctx, from, nil)
		if err != nil {
			return fmt.Errorf("balance %s after funding: %w", from, err)
		}
		log.Printf("post-deposit L2 balance from=%s balance=%s wei", from, balance)
	}
	if balance.Sign() == 0 {
		return fmt.Errorf("sender %s has zero balance; pass -private-key for a funded devnet account or use -fund-l2", from)
	}

	contract, err := resolveContract(ctx, cfg, sender)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.attempts; attempt++ {
		result, err := submitAttempt(ctx, cfg, sender, contract, attempt, validationOpts, flashblocks)
		if err == nil {
			return printResult(cfg, *result)
		}
		lastErr = err
		log.Printf("attempt %d did not validate: %v", attempt, err)
	}
	return fmt.Errorf("no attempt produced a validated SDM post-exec block: %w", lastErr)
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}

func enableSDM(ctx context.Context, cfg config, sender *sdm.TxSender) error {
	targets := []struct {
		name   string
		rpcURL string
		client *gethrpc.Client
	}{
		{name: "execution", rpcURL: cfg.rpcURL, client: sender.RPC},
		{name: "sequencer", rpcURL: cfg.sequencerRPCURL},
		{name: "op-rbuilder", rpcURL: cfg.opRbuilderRPCURL},
	}

	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		rpcURL := strings.TrimRight(target.rpcURL, "/")
		if rpcURL == "" {
			continue
		}
		if _, ok := seen[rpcURL]; ok {
			continue
		}
		seen[rpcURL] = struct{}{}

		client := target.client
		if client == nil {
			var err error
			client, err = gethrpc.DialContext(ctx, target.rpcURL)
			if err != nil {
				return fmt.Errorf("dial %s SDM admin RPC %s: %w", target.name, target.rpcURL, err)
			}
			defer client.Close()
		}
		if err := client.CallContext(ctx, nil, "admin_setSdmPostExecOptIn", true); err != nil {
			return fmt.Errorf("%s admin_setSdmPostExecOptIn(true) rpc=%s: %w", target.name, target.rpcURL, err)
		}
		log.Printf("admin_setSdmPostExecOptIn(true) ok target=%s rpc=%s", target.name, target.rpcURL)
	}
	return nil
}

func loadKey(cfg config) (*ecdsa.PrivateKey, common.Address, error) {
	if cfg.privateKeyHex != "" {
		priv, err := parsePrivateKey(cfg.privateKeyHex)
		if err != nil {
			return nil, common.Address{}, fmt.Errorf("parse -private-key: %w", err)
		}
		return priv, crypto.PubkeyToAddress(priv.PublicKey), nil
	}

	keys, err := devkeys.NewMnemonicDevKeys(cfg.mnemonic)
	if err != nil {
		return nil, common.Address{}, err
	}
	priv, err := keys.Secret(devkeys.UserKey(cfg.mnemonicIndex))
	if err != nil {
		return nil, common.Address{}, err
	}
	return priv, crypto.PubkeyToAddress(priv.PublicKey), nil
}

func parsePrivateKey(input string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(strings.TrimPrefix(input, "0x"))
}

func fundL2(ctx context.Context, cfg config, l2Sender *sdm.TxSender) error {
	if cfg.l1RPCURL == "" {
		return errors.New("-fund-l2 requires -l1-rpc or L1_RPC_URL")
	}
	if !common.IsHexAddress(cfg.portal) {
		return fmt.Errorf("-fund-l2 requires valid -portal OptimismPortalProxy address, got %q", cfg.portal)
	}
	amount, ok := new(big.Int).SetString(cfg.fundAmount, 0)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid -fund-amount %q", cfg.fundAmount)
	}
	l1Priv, err := parsePrivateKey(cfg.l1PrivateKey)
	if err != nil {
		return fmt.Errorf("parse -l1-private-key: %w", err)
	}
	l1From := crypto.PubkeyToAddress(l1Priv.PublicKey)
	l1Sender, err := sdm.DialTxSender(ctx, cfg.l1RPCURL, l1Priv, l1From, cfg.fundGasLimit)
	if err != nil {
		return err
	}
	defer l1Sender.Close()

	portalABI, err := abi.JSON(strings.NewReader(`[{
		"type":"function",
		"name":"depositTransaction",
		"inputs":[
			{"name":"_to","type":"address"},
			{"name":"_value","type":"uint256"},
			{"name":"_gasLimit","type":"uint64"},
			{"name":"_isCreation","type":"bool"},
			{"name":"_data","type":"bytes"}
		],
		"outputs":[]
	}]`))
	if err != nil {
		return err
	}
	calldata, err := portalABI.Pack("depositTransaction", l2Sender.From, amount, uint64(21_000), false, []byte{})
	if err != nil {
		return fmt.Errorf("pack depositTransaction: %w", err)
	}
	l1Nonce, err := l1Sender.Eth.PendingNonceAt(ctx, l1From)
	if err != nil {
		return fmt.Errorf("L1 pending nonce: %w", err)
	}
	portal := common.HexToAddress(cfg.portal)
	tx, err := l1Sender.SendCallValue(ctx, l1Nonce, portal, amount, calldata, cfg.fundGasLimit)
	if err != nil {
		return fmt.Errorf("send OptimismPortal deposit: %w", err)
	}
	log.Printf("submitted L1 deposit tx=%s from=%s to_l2=%s amount=%s portal=%s", tx.Hash(), l1From, l2Sender.From, amount, portal)
	receipt, err := sdm.WaitReceipt(ctx, l1Sender.Eth, tx.Hash(), cfg.pollInterval)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("L1 deposit tx %s failed with status %d", tx.Hash(), receipt.Status)
	}

	want := new(big.Int).Set(amount)
	deadline := time.NewTimer(cfg.receiptTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()
	for {
		balance, err := l2Sender.Eth.BalanceAt(ctx, l2Sender.From, nil)
		if err != nil {
			return fmt.Errorf("poll L2 balance after deposit: %w", err)
		}
		if balance.Cmp(want) >= 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for L2 deposit balance >= %s", want)
		case <-ticker.C:
		}
	}
}

func resolveContract(ctx context.Context, cfg config, sender *sdm.TxSender) (common.Address, error) {
	if cfg.contract != "" {
		if !common.IsHexAddress(cfg.contract) {
			return common.Address{}, fmt.Errorf("invalid -contract address %q", cfg.contract)
		}
		contract := common.HexToAddress(cfg.contract)
		log.Printf("using existing StateBloat contract %s", contract)
		return contract, nil
	}

	nonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return common.Address{}, fmt.Errorf("pending nonce for deployer %s: %w", sender.From, err)
	}
	bytecode, err := sdm.DecodeHexBytes(sdm.StateBloatBin)
	if err != nil {
		return common.Address{}, err
	}
	deployTx, err := sender.SendContractCreation(ctx, nonce, bytecode, cfg.deployGasLimit)
	if err != nil {
		return common.Address{}, err
	}
	log.Printf("submitted StateBloat deploy tx=%s nonce=%d", deployTx.Hash(), nonce)
	receiptCtx, cancel := context.WithTimeout(ctx, cfg.receiptTimeout)
	defer cancel()
	receipt, err := sdm.WaitRPCReceipt(receiptCtx, sender.RPC, deployTx.Hash(), cfg.pollInterval)
	if err != nil {
		return common.Address{}, err
	}
	if uint64(receipt.Status) != types.ReceiptStatusSuccessful {
		return common.Address{}, fmt.Errorf("StateBloat deploy tx %s failed with status %d", deployTx.Hash(), receipt.Status)
	}
	if receipt.ContractAddress == nil {
		return common.Address{}, fmt.Errorf("StateBloat deploy tx %s receipt missing contractAddress", deployTx.Hash())
	}
	log.Printf("deployed StateBloat contract=%s block=%d gas_used=%d", *receipt.ContractAddress, receipt.BlockNum(), receipt.GasUsed)
	return *receipt.ContractAddress, nil
}

func submitAttempt(ctx context.Context, cfg config, sender *sdm.TxSender, contract common.Address, attempt int, validationOpts sdm.ValidationOptions, flashblocks *sdm.FlashblockCollector) (*workloadResult, error) {
	startNonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return nil, fmt.Errorf("pending nonce: %w", err)
	}
	log.Printf("attempt=%d submitting %d txs contract=%s start_nonce=%d slot_count=%d", attempt, cfg.batchSize, contract, startNonce, cfg.slotCount)

	calldata := sdm.EncodeRun(cfg.slotCount)
	txs, err := submitWorkloadTxs(ctx, cfg, sender, contract, startNonce, calldata, flashblocks)
	if err != nil {
		return nil, err
	}

	blockTxs := make(map[uint64][]*sdm.RPCReceipt)
	includedByBlock := make(map[uint64]int)
	for i, tx := range txs {
		receiptCtx, cancel := context.WithTimeout(ctx, cfg.receiptTimeout)
		receipt, err := sdm.WaitRPCReceipt(receiptCtx, sender.RPC, tx.Hash(), cfg.pollInterval)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("tx %d %s receipt: %w", i, tx.Hash(), err)
		}
		if uint64(receipt.Status) != types.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("tx %d %s failed with status %d", i, tx.Hash(), receipt.Status)
		}
		blockNum := receipt.BlockNum()
		blockTxs[blockNum] = append(blockTxs[blockNum], receipt)
		includedByBlock[blockNum]++
	}
	log.Printf("attempt=%d included_by_block=%v", attempt, includedByBlock)

	var validationErrs []error
	for _, blockNum := range sdm.SortBlockNumsByTxCount(blockTxs) {
		if len(blockTxs[blockNum]) < cfg.minUserTxs {
			continue
		}
		validation, err := sdm.ValidatePostExecBlock(ctx, sender.RPC, blockNum, validationOpts)
		if err != nil {
			validationErrs = append(validationErrs, fmt.Errorf("block %d (%d submitted txs): %w", blockNum, len(blockTxs[blockNum]), err))
			continue
		}
		return &workloadResult{
			RPCURL:          cfg.rpcURL,
			From:            sender.From,
			Contract:        contract,
			Attempt:         attempt,
			SubmittedTxs:    len(txs),
			IncludedByBlock: includedByBlock,
			Validation:      validation,
			Flashblocks:     flashblocks.WaitSummaries(ctx, blockNum, 2*time.Second),
		}, nil
	}
	if len(validationErrs) > 0 {
		return nil, errors.Join(validationErrs...)
	}
	return nil, fmt.Errorf("no block had at least %d submitted txs; included_by_block=%v", cfg.minUserTxs, includedByBlock)
}

func submitWorkloadTxs(
	ctx context.Context,
	cfg config,
	sender *sdm.TxSender,
	contract common.Address,
	startNonce uint64,
	calldata []byte,
	flashblocks *sdm.FlashblockCollector,
) ([]*types.Transaction, error) {
	txs := make([]*types.Transaction, 0, cfg.batchSize)
	if flashblocks != nil && cfg.txsPerFlashblock > 0 {
		latest, err := sender.Eth.BlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("latest block before flashblock-paced submit: %w", err)
		}
		drained := flashblocks.DrainPending()
		log.Printf("flashblock-paced submit enabled txs_per_flashblock=%d waiting_after_block=%d drained_pending_flashblocks=%d", cfg.txsPerFlashblock, latest, drained)

		var targetBlock uint64
		for len(txs) < cfg.batchSize {
			fb, ok := flashblocks.WaitNext(ctx)
			if !ok {
				return txs, ctx.Err()
			}
			if fb.BlockNumber <= latest {
				continue
			}
			if targetBlock == 0 {
				if fb.Index != 0 {
					log.Printf("skipping already-started flashblock block=%d index=%d; waiting for next block index=0", fb.BlockNumber, fb.Index)
					latest = fb.BlockNumber
					continue
				}
				targetBlock = fb.BlockNumber
				log.Printf("flashblock-paced submit target_block=%d", targetBlock)
			}
			if fb.BlockNumber != targetBlock {
				if len(txs) >= cfg.minUserTxs {
					log.Printf("target block ended before full batch was submitted sent=%d requested=%d next_block=%d", len(txs), cfg.batchSize, fb.BlockNumber)
					return txs, nil
				}
				targetBlock = fb.BlockNumber
			}

			log.Printf("submitting workload chunk after flashblock block=%d index=%d sent_before=%d", fb.BlockNumber, fb.Index, len(txs))
			if err := sendWorkloadChunk(ctx, cfg, sender, contract, startNonce, calldata, &txs); err != nil {
				return txs, err
			}
		}
		return txs, nil
	}

	for i := 0; i < cfg.batchSize; i++ {
		if i > 0 && cfg.txSpacing > 0 {
			select {
			case <-ctx.Done():
				return txs, ctx.Err()
			case <-time.After(cfg.txSpacing):
			}
		}
		tx, err := sender.SendCall(ctx, startNonce+uint64(i), contract, calldata, cfg.gasLimit)
		if err != nil {
			return txs, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func sendWorkloadChunk(
	ctx context.Context,
	cfg config,
	sender *sdm.TxSender,
	contract common.Address,
	startNonce uint64,
	calldata []byte,
	txs *[]*types.Transaction,
) error {
	remaining := cfg.batchSize - len(*txs)
	if remaining <= 0 {
		return nil
	}
	chunk := cfg.txsPerFlashblock
	if chunk <= 0 || chunk > remaining {
		chunk = remaining
	}
	for i := 0; i < chunk; i++ {
		nonce := startNonce + uint64(len(*txs))
		tx, err := sender.SendCall(ctx, nonce, contract, calldata, cfg.gasLimit)
		if err != nil {
			return err
		}
		*txs = append(*txs, tx)
	}
	return nil
}

func printResult(cfg config, result workloadResult) error {
	if cfg.jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if result.Validation == nil {
		return nil
	}
	v := result.Validation
	log.Printf("validated SDM post-exec block=%d hash=%s post_exec_index=%d txs=%d payload_entries=%d total_refund=%d",
		uint64(v.Block.Number), v.Block.Hash, v.PostExecIndex, len(v.Block.Transactions), len(v.Payload.GasRefundEntries), v.TotalPayloadRefund)
	if result.Contract != (common.Address{}) {
		log.Printf("workload contract=%s attempt=%d submitted_txs=%d", result.Contract, result.Attempt, result.SubmittedTxs)
	}
	if v.Replay != nil {
		log.Printf("replay raw_gas=%d canonical_gas=%d replay_refund_total=%d", v.Replay.Summary.BlockRawGasUsed, v.Replay.Summary.BlockGasUsed, v.Replay.Summary.ReplayRefundTotal)
	}
	log.Printf("flashblocks observed=%d", len(result.Flashblocks))
	for _, fb := range result.Flashblocks {
		log.Printf("flashblock block=%d index=%d txs=%d has_post_exec_tx=%t post_exec_tx_bytes=%d",
			fb.BlockNumber, fb.Index, fb.TxCount, fb.HasPostExecTx, fb.PostExecTxBytes)
	}
	return nil
}
