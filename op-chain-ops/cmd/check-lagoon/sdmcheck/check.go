// Package sdmcheck drives a non-trivial SDM workload and validates the
// resulting PostExec block against the Lagoon RPC and accounting rules.
package sdmcheck

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

const (
	defaultBatchSize      = 12
	defaultSlotCount      = 20
	defaultMinUserTxs     = 2
	defaultAttempts       = 3
	defaultGasLimit       = 1_000_000
	defaultDeployGasLimit = 2_000_000
	defaultReceiptTimeout = 45 * time.Second
	defaultPollInterval   = 500 * time.Millisecond
	defaultVerifyTimeout  = 2 * time.Minute
)

// RollupClient is the subset required for activation and safe-head checks.
type RollupClient interface {
	RollupConfig(ctx context.Context) (*rollup.Config, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

// Config controls workload generation and optional independent verification.
type Config struct {
	Log         log.Logger
	RPCURL      string
	Key         *ecdsa.PrivateKey
	Contract    *common.Address
	OptIn       bool
	BatchSize   int
	SlotCount   uint64
	MinUserTxs  int
	Attempts    int
	GasLimit    uint64
	DeployLimit uint64
	TxSpacing   time.Duration

	ReceiptTimeout time.Duration
	PollInterval   time.Duration
	VerifyTimeout  time.Duration

	FundL2       bool
	L1RPCURL     string
	L1Key        *ecdsa.PrivateKey
	Portal       common.Address
	FundAmount   *big.Int
	FundGasLimit uint64

	Verifier sdm.Caller
	Rollup   RollupClient
}

// Result is the machine-readable output of an SDM workload or block check.
type Result struct {
	RPCURL          string                 `json:"rpc_url"`
	From            common.Address         `json:"from"`
	Contract        common.Address         `json:"contract"`
	Attempt         int                    `json:"attempt"`
	SubmittedTxs    int                    `json:"submitted_txs"`
	IncludedByBlock map[uint64]int         `json:"included_by_block,omitempty"`
	Conformance     *sdm.ConformanceResult `json:"conformance"`
	VerifierChecked bool                   `json:"verifier_checked"`
	SafeHeadChecked bool                   `json:"safe_head_checked"`
}

// ApplyDefaults fills zero-valued workload tuning fields.
func (cfg *Config) ApplyDefaults() {
	if cfg.Log == nil {
		cfg.Log = log.New()
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.SlotCount == 0 {
		cfg.SlotCount = defaultSlotCount
	}
	if cfg.MinUserTxs == 0 {
		cfg.MinUserTxs = defaultMinUserTxs
	}
	if cfg.Attempts == 0 {
		cfg.Attempts = defaultAttempts
	}
	if cfg.GasLimit == 0 {
		cfg.GasLimit = defaultGasLimit
	}
	if cfg.DeployLimit == 0 {
		cfg.DeployLimit = defaultDeployGasLimit
	}
	if cfg.ReceiptTimeout == 0 {
		cfg.ReceiptTimeout = defaultReceiptTimeout
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.VerifyTimeout == 0 {
		cfg.VerifyTimeout = defaultVerifyTimeout
	}
	if cfg.FundAmount == nil {
		cfg.FundAmount = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	}
	if cfg.FundGasLimit == 0 {
		cfg.FundGasLimit = 200_000
	}
}

// DeriveSDMAccountKey derives a stable, domain-separated test key from a funded
// account. Repeated smoke runs therefore reuse the same SDM address without
// persisting or printing another private key.
func DeriveSDMAccountKey(funder *ecdsa.PrivateKey) (*ecdsa.PrivateKey, error) {
	if funder == nil {
		return nil, errors.New("funding account private key is required")
	}
	secret := crypto.FromECDSA(funder)
	const domain = "check-lagoon/sdm-account/v1"
	for counter := 0; counter < 256; counter++ {
		material := make([]byte, 0, len(domain)+len(secret)+1)
		material = append(material, domain...)
		material = append(material, secret...)
		material = append(material, byte(counter))
		key, err := crypto.ToECDSA(crypto.Keccak256(material))
		if err == nil {
			return key, nil
		}
	}
	return nil, errors.New("failed to derive a valid SDM account key")
}

// EnsureFundedAccount transfers enough ETH from funder to make recipient's
// balance reach minimumBalance. It is a no-op when the recipient is already funded.
func EnsureFundedAccount(
	ctx context.Context,
	rpcURL string,
	funder *ecdsa.PrivateKey,
	recipient common.Address,
	minimumBalance *big.Int,
	receiptTimeout time.Duration,
	pollInterval time.Duration,
) error {
	if funder == nil {
		return errors.New("funding account private key is required")
	}
	if minimumBalance == nil || minimumBalance.Sign() <= 0 {
		return errors.New("minimum funded balance must be positive")
	}
	if receiptTimeout <= 0 {
		receiptTimeout = defaultReceiptTimeout
	}
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	from := crypto.PubkeyToAddress(funder.PublicKey)
	if from == recipient {
		return errors.New("funding and recipient accounts must differ")
	}
	sender, err := sdm.DialTxSender(ctx, rpcURL, funder, from, 21_000)
	if err != nil {
		return err
	}
	defer sender.Close()
	balance, err := sender.Eth.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("fetch recipient balance on %s: %w", rpcURL, err)
	}
	deficit := fundingDeficit(balance, minimumBalance)
	if deficit.Sign() == 0 {
		return nil
	}
	funderBalance, err := sender.Eth.BalanceAt(ctx, from, nil)
	if err != nil {
		return fmt.Errorf("fetch funder balance on %s: %w", rpcURL, err)
	}
	if funderBalance.Cmp(deficit) <= 0 {
		return fmt.Errorf("funder %s balance %s cannot cover transfer %s on %s", from, funderBalance, deficit, rpcURL)
	}
	nonce, err := sender.Eth.PendingNonceAt(ctx, from)
	if err != nil {
		return fmt.Errorf("fetch funding nonce on %s: %w", rpcURL, err)
	}
	tx, err := sender.SendCallValue(ctx, nonce, recipient, deficit, nil, 21_000)
	if err != nil {
		return fmt.Errorf("fund generated SDM account on %s: %w", rpcURL, err)
	}
	receiptCtx, cancel := context.WithTimeout(ctx, receiptTimeout)
	defer cancel()
	receipt, err := sdm.WaitReceipt(receiptCtx, sender.Eth, tx.Hash(), pollInterval)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("funding transaction %s failed with status %d", tx.Hash(), receipt.Status)
	}
	finalBalance, err := sender.Eth.BalanceAt(ctx, recipient, nil)
	if err != nil {
		return fmt.Errorf("fetch funded balance on %s: %w", rpcURL, err)
	}
	if finalBalance.Cmp(minimumBalance) < 0 {
		return fmt.Errorf("funded account balance is %s on %s, want at least %s", finalBalance, rpcURL, minimumBalance)
	}
	return nil
}

func fundingDeficit(balance, minimumBalance *big.Int) *big.Int {
	if balance.Cmp(minimumBalance) >= 0 {
		return new(big.Int)
	}
	return new(big.Int).Sub(minimumBalance, balance)
}

// CheckAll opts the producer into SDM, deploys the StateBloat workload when
// needed, drives transactions until a non-empty PostExec payload is produced,
// and runs every configured conformance check.
func CheckAll(ctx context.Context, cfg Config) (*Result, error) {
	cfg.ApplyDefaults()
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	sender, err := sdm.DialTxSender(ctx, cfg.RPCURL, cfg.Key, publicAddress(cfg.Key), cfg.GasLimit)
	if err != nil {
		return nil, err
	}
	defer sender.Close()

	balance, err := sender.Eth.BalanceAt(ctx, sender.From, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch sender balance: %w", err)
	}
	if balance.Sign() == 0 && cfg.FundL2 {
		if err := fundL2(ctx, cfg, sender); err != nil {
			return nil, err
		}
		balance, err = sender.Eth.BalanceAt(ctx, sender.From, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch sender balance after funding: %w", err)
		}
	}
	if balance.Sign() == 0 {
		return nil, fmt.Errorf("SDM sender %s has zero balance; fund it or configure the optional L1 deposit inputs", sender.From)
	}
	cfg.Log.Info("connected to SDM producer", "rpc", cfg.RPCURL, "chain", sender.ChainID, "account", sender.From, "balance", balance)
	if cfg.OptIn {
		cfg.Log.Warn("enabling the in-memory SDM operator opt-in on the producer")
		if err := sender.RPC.CallContext(ctx, nil, "admin_setOperatorSdmOptIn", true); err != nil {
			return nil, fmt.Errorf("admin_setOperatorSdmOptIn(true): %w", err)
		}
	}

	contract, err := resolveContract(ctx, cfg, sender)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 1; attempt <= cfg.Attempts; attempt++ {
		result, err := submitAttempt(ctx, cfg, sender, contract, attempt)
		if err == nil {
			return result, nil
		}
		lastErr = err
		cfg.Log.Warn("SDM workload attempt did not validate", "attempt", attempt, "error", err)
	}
	return nil, fmt.Errorf("no attempt produced a conforming SDM PostExec block: %w", lastErr)
}

// CheckBlock validates an existing SDM block without submitting transactions.
func CheckBlock(ctx context.Context, cfg Config, blockNum uint64) (*Result, error) {
	cfg.ApplyDefaults()
	if cfg.RPCURL == "" {
		return nil, errors.New("SDM producer RPC URL is required")
	}
	sender, err := sdm.DialTxSender(ctx, cfg.RPCURL, cfg.Key, publicAddress(cfg.Key), cfg.GasLimit)
	if err != nil {
		return nil, err
	}
	defer sender.Close()
	conformance, verifierChecked, safeHeadChecked, err := checkConformance(ctx, cfg, sender.RPC, blockNum)
	if err != nil {
		return nil, err
	}
	return &Result{
		RPCURL: cfg.RPCURL, From: sender.From, Conformance: conformance,
		VerifierChecked: verifierChecked, SafeHeadChecked: safeHeadChecked,
	}, nil
}

func validateConfig(cfg Config) error {
	if cfg.RPCURL == "" {
		return errors.New("SDM producer RPC URL is required")
	}
	if cfg.Key == nil {
		return errors.New("SDM funded account private key is required")
	}
	if cfg.BatchSize < 1 || cfg.MinUserTxs < 1 || cfg.Attempts < 1 {
		return errors.New("batch size, minimum user transactions, and attempts must be positive")
	}
	if cfg.MinUserTxs > cfg.BatchSize {
		return fmt.Errorf("minimum user transactions %d exceeds batch size %d", cfg.MinUserTxs, cfg.BatchSize)
	}
	if cfg.FundL2 && (cfg.L1RPCURL == "" || cfg.L1Key == nil || cfg.Portal == (common.Address{})) {
		return errors.New("SDM L1 funding requires the L1 RPC, L1 account, and OptimismPortal address")
	}
	if cfg.FundAmount == nil || cfg.FundAmount.Sign() <= 0 {
		return errors.New("SDM funding amount must be positive")
	}
	return nil
}

func fundL2(ctx context.Context, cfg Config, l2Sender *sdm.TxSender) error {
	l1From := crypto.PubkeyToAddress(cfg.L1Key.PublicKey)
	l1Sender, err := sdm.DialTxSender(ctx, cfg.L1RPCURL, cfg.L1Key, l1From, cfg.FundGasLimit)
	if err != nil {
		return fmt.Errorf("dial L1 funding RPC: %w", err)
	}
	defer l1Sender.Close()
	portalABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"depositTransaction","inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"},{"name":"_gasLimit","type":"uint64"},{"name":"_isCreation","type":"bool"},{"name":"_data","type":"bytes"}],"outputs":[]}]`))
	if err != nil {
		return err
	}
	calldata, err := portalABI.Pack("depositTransaction", l2Sender.From, cfg.FundAmount, uint64(21_000), false, []byte{})
	if err != nil {
		return fmt.Errorf("pack OptimismPortal deposit: %w", err)
	}
	nonce, err := l1Sender.Eth.PendingNonceAt(ctx, l1From)
	if err != nil {
		return fmt.Errorf("fetch L1 funding nonce: %w", err)
	}
	tx, err := l1Sender.SendCallValue(ctx, nonce, cfg.Portal, cfg.FundAmount, calldata, cfg.FundGasLimit)
	if err != nil {
		return fmt.Errorf("send OptimismPortal deposit: %w", err)
	}
	receiptCtx, cancel := context.WithTimeout(ctx, cfg.ReceiptTimeout)
	defer cancel()
	receipt, err := sdm.WaitReceipt(receiptCtx, l1Sender.Eth, tx.Hash(), cfg.PollInterval)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("L1 deposit transaction %s failed with status %d", tx.Hash(), receipt.Status)
	}
	waitCtx, waitCancel := context.WithTimeout(ctx, cfg.VerifyTimeout)
	defer waitCancel()
	return retry.Do0(waitCtx, int(cfg.VerifyTimeout/time.Second)+1, retry.Fixed(time.Second), func() error {
		balance, err := l2Sender.Eth.BalanceAt(waitCtx, l2Sender.From, nil)
		if err != nil {
			return err
		}
		if balance.Cmp(cfg.FundAmount) < 0 {
			return fmt.Errorf("L2 balance is %s, waiting for at least %s", balance, cfg.FundAmount)
		}
		return nil
	})
}

func resolveContract(ctx context.Context, cfg Config, sender *sdm.TxSender) (common.Address, error) {
	if cfg.Contract != nil {
		cfg.Log.Info("using existing StateBloat contract", "address", *cfg.Contract)
		return *cfg.Contract, nil
	}
	nonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return common.Address{}, fmt.Errorf("fetch deployment nonce: %w", err)
	}
	bytecode, err := sdm.DecodeHexBytes(sdm.StateBloatBin)
	if err != nil {
		return common.Address{}, err
	}
	tx, err := sender.SendContractCreation(ctx, nonce, bytecode, cfg.DeployLimit)
	if err != nil {
		return common.Address{}, err
	}
	receiptCtx, cancel := context.WithTimeout(ctx, cfg.ReceiptTimeout)
	defer cancel()
	receipt, err := sdm.WaitRPCReceipt(receiptCtx, sender.RPC, tx.Hash(), cfg.PollInterval)
	if err != nil {
		return common.Address{}, err
	}
	if uint64(receipt.Status) != types.ReceiptStatusSuccessful || receipt.ContractAddress == nil {
		return common.Address{}, fmt.Errorf("StateBloat deployment %s failed or omitted contractAddress", tx.Hash())
	}
	cfg.Log.Info("deployed StateBloat workload", "address", *receipt.ContractAddress, "block", receipt.BlockNum())
	return *receipt.ContractAddress, nil
}

func submitAttempt(ctx context.Context, cfg Config, sender *sdm.TxSender, contract common.Address, attempt int) (*Result, error) {
	startNonce, err := sender.Eth.PendingNonceAt(ctx, sender.From)
	if err != nil {
		return nil, fmt.Errorf("fetch workload nonce: %w", err)
	}
	data := sdm.EncodeRun(cfg.SlotCount)
	txs := make([]*types.Transaction, 0, cfg.BatchSize)
	for i := 0; i < cfg.BatchSize; i++ {
		if i > 0 && cfg.TxSpacing > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(cfg.TxSpacing):
			}
		}
		tx, err := sender.SendCall(ctx, startNonce+uint64(i), contract, data, cfg.GasLimit)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	byBlock := make(map[uint64][]*sdm.RPCReceipt)
	counts := make(map[uint64]int)
	for i, tx := range txs {
		receiptCtx, cancel := context.WithTimeout(ctx, cfg.ReceiptTimeout)
		receipt, err := sdm.WaitRPCReceipt(receiptCtx, sender.RPC, tx.Hash(), cfg.PollInterval)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("workload transaction %d receipt: %w", i, err)
		}
		if uint64(receipt.Status) != types.ReceiptStatusSuccessful {
			return nil, fmt.Errorf("workload transaction %d failed with status %d", i, receipt.Status)
		}
		blockNum := receipt.BlockNum()
		byBlock[blockNum] = append(byBlock[blockNum], receipt)
		counts[blockNum]++
	}

	var validationErrs []error
	for _, blockNum := range sdm.SortBlockNumsByTxCount(byBlock) {
		if len(byBlock[blockNum]) < cfg.MinUserTxs {
			continue
		}
		conformance, verifierChecked, safeHeadChecked, err := checkConformance(ctx, cfg, sender.RPC, blockNum)
		if err != nil {
			validationErrs = append(validationErrs, fmt.Errorf("block %d (%d workload transactions): %w", blockNum, len(byBlock[blockNum]), err))
			continue
		}
		return &Result{
			RPCURL: cfg.RPCURL, From: sender.From, Contract: contract,
			Attempt: attempt, SubmittedTxs: len(txs), IncludedByBlock: counts,
			Conformance: conformance, VerifierChecked: verifierChecked, SafeHeadChecked: safeHeadChecked,
		}, nil
	}
	if len(validationErrs) > 0 {
		return nil, errors.Join(validationErrs...)
	}
	return nil, fmt.Errorf("no block had at least %d workload transactions; included_by_block=%v", cfg.MinUserTxs, counts)
}

func checkConformance(ctx context.Context, cfg Config, producer sdm.Caller, blockNum uint64) (*sdm.ConformanceResult, bool, bool, error) {
	conformance, err := sdm.ValidatePostExecConformance(ctx, producer, blockNum)
	if err != nil {
		return nil, false, false, err
	}
	if cfg.Rollup != nil {
		rollupCfg, err := cfg.Rollup.RollupConfig(ctx)
		if err != nil {
			return nil, false, false, fmt.Errorf("fetch rollup config: %w", err)
		}
		if !rollupCfg.IsLagoon(uint64(conformance.Block.Timestamp)) {
			return nil, false, false, fmt.Errorf("Lagoon is not active at SDM block %d timestamp %d", blockNum, conformance.Block.Timestamp)
		}
	}
	verifierChecked := false
	if cfg.Verifier != nil {
		verifyCtx, cancel := context.WithTimeout(ctx, cfg.VerifyTimeout)
		defer cancel()
		err := retry.Do0(verifyCtx, int(cfg.VerifyTimeout/time.Second)+1, retry.Fixed(time.Second), func() error {
			return sdm.ValidateVerifierAgreement(verifyCtx, conformance, cfg.Verifier)
		})
		if err != nil {
			return nil, false, false, fmt.Errorf("verifier did not agree on SDM block %d: %w", blockNum, err)
		}
		verifierChecked = true
	}
	safeHeadChecked := false
	if cfg.Rollup != nil {
		verifyCtx, cancel := context.WithTimeout(ctx, cfg.VerifyTimeout)
		defer cancel()
		err := retry.Do0(verifyCtx, int(cfg.VerifyTimeout/time.Second)+1, retry.Fixed(time.Second), func() error {
			status, err := cfg.Rollup.SyncStatus(verifyCtx)
			if err != nil {
				return err
			}
			if status.SafeL2.Number < blockNum {
				return fmt.Errorf("safe head is %d, waiting for %d", status.SafeL2.Number, blockNum)
			}
			return nil
		})
		if err != nil {
			return nil, verifierChecked, false, fmt.Errorf("safe head did not reach SDM block %d: %w", blockNum, err)
		}
		canonical, err := sdm.GetBlockWithTxs(verifyCtx, producer, blockNum)
		if err != nil {
			return nil, verifierChecked, false, fmt.Errorf("refetch safe SDM block %d: %w", blockNum, err)
		}
		if canonical.Hash != conformance.Block.Hash {
			return nil, verifierChecked, false, fmt.Errorf("SDM block %d was reorged before becoming safe: validated %s, canonical %s", blockNum, conformance.Block.Hash, canonical.Hash)
		}
		safeHeadChecked = true
	}
	return conformance, verifierChecked, safeHeadChecked, nil
}

func publicAddress(key *ecdsa.PrivateKey) common.Address {
	if key == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}
