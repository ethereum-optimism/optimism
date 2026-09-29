// Package sdmcheck drives an SDM workload and validates the resulting PostExec block.
package sdmcheck

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	batchSize           = 12
	slotCount           = 20
	minUserTxs          = 2
	attempts            = 3
	gasLimit            = 1_000_000
	deployGasLimit      = 2_000_000
	receiptTimeout      = 45 * time.Second
	verifyTimeout       = 2 * time.Minute
	pollInterval        = 500 * time.Millisecond
	optInRestoreTimeout = 10 * time.Second
)

// RollupClient is the subset required for activation and safe-head checks.
type RollupClient interface {
	RollupConfig(ctx context.Context) (*rollup.Config, error)
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

// Config selects the producer, workload account, and optional verifiers.
type Config struct {
	Log      log.Logger
	RPCURL   string
	Key      *ecdsa.PrivateKey
	Contract *common.Address
	OptIn    bool

	Verifier sdm.Caller
	Rollup   RollupClient
}

// Result is the machine-readable output of an SDM workload or block check.
type Result struct {
	RPCURL          string                 `json:"rpc_url"`
	From            common.Address         `json:"from"`
	Contract        common.Address         `json:"contract"`
	OptInToggled    bool                   `json:"opt_in_toggled"`
	Attempt         int                    `json:"attempt"`
	SubmittedTxs    int                    `json:"submitted_txs"`
	IncludedByBlock map[uint64]int         `json:"included_by_block,omitempty"`
	Conformance     *sdm.ConformanceResult `json:"conformance"`
	VerifierChecked bool                   `json:"verifier_checked"`
	SafeHeadChecked bool                   `json:"safe_head_checked"`
}

// DeriveSDMAccountKey derives a stable, domain-separated key from funder.
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

// EnsureFundedAccount tops up recipient from funder to at least minimumBalance.
func EnsureFundedAccount(ctx context.Context, rpcURL string, funder *ecdsa.PrivateKey, recipient common.Address, minimumBalance *big.Int) error {
	if funder == nil {
		return errors.New("funding account private key is required")
	}
	if minimumBalance == nil || minimumBalance.Sign() <= 0 {
		return errors.New("minimum funded balance must be positive")
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
		return fmt.Errorf("fund SDM account on %s: %w", rpcURL, err)
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

// CheckAll deploys the workload if needed, drives transactions until a block
// with a PostExec payload is produced, and runs every configured check.
func CheckAll(ctx context.Context, cfg Config) (*Result, error) {
	if cfg.Log == nil {
		cfg.Log = log.New()
	}
	if cfg.RPCURL == "" {
		return nil, errors.New("SDM producer RPC URL is required")
	}
	if cfg.Key == nil {
		return nil, errors.New("SDM account private key is required")
	}
	sender, err := sdm.DialTxSender(ctx, cfg.RPCURL, cfg.Key, publicAddress(cfg.Key), gasLimit)
	if err != nil {
		return nil, err
	}
	defer sender.Close()

	balance, err := sender.Eth.BalanceAt(ctx, sender.From, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch sender balance: %w", err)
	}
	if balance.Sign() == 0 {
		return nil, fmt.Errorf("SDM sender %s has zero balance", sender.From)
	}
	cfg.Log.Info("connected to SDM producer", "rpc", cfg.RPCURL, "chain", sender.ChainID, "account", sender.From, "balance", balance)

	optInToggled, err := ensureOptIn(ctx, cfg, sender.RPC)
	if err != nil {
		return nil, err
	}
	if optInToggled {
		defer restoreOptIn(cfg, sender.RPC)
	}

	contract, err := resolveContract(ctx, cfg, sender)
	if err != nil {
		return nil, err
	}
	var attemptErrs []error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := submitAttempt(ctx, cfg, sender, contract, attempt)
		if err == nil {
			result.OptInToggled = optInToggled
			return result, nil
		}
		attemptErrs = append(attemptErrs, fmt.Errorf("attempt %d: %w", attempt, err))
		if ctx.Err() != nil {
			break
		}
		cfg.Log.Warn("SDM workload attempt did not validate", "attempt", attempt, "error", err)
	}
	return nil, fmt.Errorf("no attempt produced a conforming SDM PostExec block: %w", errors.Join(attemptErrs...))
}

type sdmStatus struct {
	OperatorSDMOptIn bool `json:"operatorSdmOptIn"`
}

// ensureOptIn reports whether it switched the producer's opt-in on, in which
// case the caller must restore it. It only does so when cfg.OptIn is set.
func ensureOptIn(ctx context.Context, cfg Config, producer sdm.Caller) (bool, error) {
	var status sdmStatus
	if err := producer.CallContext(ctx, &status, "admin_sdmStatus"); err != nil {
		if !cfg.OptIn {
			cfg.Log.Warn("cannot read admin_sdmStatus; assuming SDM opt-in", "error", err)
			return false, nil
		}
		return false, fmt.Errorf("admin_sdmStatus: %w", err)
	}
	if status.OperatorSDMOptIn {
		return false, nil
	}
	if !cfg.OptIn {
		return false, errors.New("producer has not opted in to SDM; pass --sdm-opt-in to enable it for the check")
	}
	cfg.Log.Warn("temporarily enabling SDM opt-in on the producer", "rpc", cfg.RPCURL)
	if err := producer.CallContext(ctx, nil, "admin_setOperatorSdmOptIn", true); err != nil {
		// The enable may have landed even though the call failed.
		restoreOptIn(cfg, producer)
		return false, fmt.Errorf("admin_setOperatorSdmOptIn(true): %w", err)
	}
	return true, nil
}

func restoreOptIn(cfg Config, producer sdm.Caller) {
	ctx, cancel := context.WithTimeout(context.Background(), optInRestoreTimeout)
	defer cancel()
	if err := producer.CallContext(ctx, nil, "admin_setOperatorSdmOptIn", false); err != nil {
		cfg.Log.Error("failed to restore SDM opt-in; run admin_setOperatorSdmOptIn(false)", "rpc", cfg.RPCURL, "error", err)
		return
	}
	cfg.Log.Info("restored SDM opt-in to disabled", "rpc", cfg.RPCURL)
}

// CheckBlock validates an existing SDM block without submitting transactions.
func CheckBlock(ctx context.Context, cfg Config, blockNum uint64) (*Result, error) {
	if cfg.RPCURL == "" {
		return nil, errors.New("SDM producer RPC URL is required")
	}
	sender, err := sdm.DialTxSender(ctx, cfg.RPCURL, cfg.Key, publicAddress(cfg.Key), gasLimit)
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

func resolveContract(ctx context.Context, cfg Config, sender *sdm.TxSender) (common.Address, error) {
	if cfg.Contract != nil {
		return *cfg.Contract, nil
	}
	receiptCtx, cancel := context.WithTimeout(ctx, receiptTimeout)
	defer cancel()
	receipt, err := sdm.DeployStateBloat(receiptCtx, sender, deployGasLimit, pollInterval)
	if err != nil {
		return common.Address{}, err
	}
	cfg.Log.Info("deployed StateBloat", "address", *receipt.ContractAddress, "block", receipt.BlockNum())
	return *receipt.ContractAddress, nil
}

func submitAttempt(ctx context.Context, cfg Config, sender *sdm.TxSender, contract common.Address, attempt int) (*Result, error) {
	txs, err := sdm.SubmitWorkload(ctx, sender, contract, batchSize, slotCount, gasLimit, 0)
	if err != nil {
		return nil, err
	}
	byBlock, err := sdm.CollectWorkloadReceipts(ctx, sender, txs, receiptTimeout, pollInterval)
	if err != nil {
		return nil, err
	}
	counts := sdm.CountByBlock(byBlock)

	var validationErrs []error
	for _, blockNum := range sdm.SortBlockNumsByTxCount(byBlock) {
		if len(byBlock[blockNum]) < minUserTxs {
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
	return nil, fmt.Errorf("no block had at least %d workload transactions; included_by_block=%v", minUserTxs, counts)
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
		err := waitFor(ctx, verifyTimeout, func(verifyCtx context.Context) error {
			return sdm.ValidateVerifierAgreement(verifyCtx, conformance, cfg.Verifier)
		})
		if err != nil {
			return nil, false, false, fmt.Errorf("verifier did not agree on SDM block %d: %w", blockNum, err)
		}
		verifierChecked = true
	}
	safeHeadChecked := false
	if cfg.Rollup != nil {
		if err := checkSafeHead(ctx, cfg, producer, conformance.Block, blockNum); err != nil {
			return nil, verifierChecked, false, err
		}
		safeHeadChecked = true
	}
	return conformance, verifierChecked, safeHeadChecked, nil
}

// checkSafeHead waits for the safe head to pass the SDM block, then checks the
// rollup node and producer still agree on the validated block.
func checkSafeHead(ctx context.Context, cfg Config, producer sdm.Caller, validated *sdm.RPCBlock, blockNum uint64) error {
	err := waitFor(ctx, verifyTimeout, func(verifyCtx context.Context) error {
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
		return fmt.Errorf("safe head did not reach SDM block %d: %w", blockNum, err)
	}
	output, err := cfg.Rollup.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("fetch rollup node block %d: %w", blockNum, err)
	}
	if output.BlockRef.Hash != validated.Hash {
		return fmt.Errorf("rollup node's safe block %d is %s, but the validated SDM block is %s", blockNum, output.BlockRef.Hash, validated.Hash)
	}
	canonical, err := sdm.GetBlockWithTxs(ctx, producer, blockNum)
	if err != nil {
		return fmt.Errorf("refetch safe SDM block %d: %w", blockNum, err)
	}
	if canonical.Hash != validated.Hash {
		return fmt.Errorf("SDM block %d was reorged before becoming safe: validated %s, canonical %s", blockNum, validated.Hash, canonical.Hash)
	}
	return nil
}

// waitFor retries op each second until it succeeds or timeout expires, keeping
// op's last error when the deadline cuts the wait short.
func waitFor(ctx context.Context, timeout time.Duration, op func(context.Context) error) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var lastErr error
	err := retry.Do0(waitCtx, int(timeout/time.Second)+1, retry.Fixed(time.Second), func() error {
		lastErr = op(waitCtx)
		return lastErr
	})
	if err != nil && lastErr != nil && !errors.Is(err, lastErr) {
		return errors.Join(err, lastErr)
	}
	return err
}

func publicAddress(key *ecdsa.PrivateKey) common.Address {
	if key == nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}
