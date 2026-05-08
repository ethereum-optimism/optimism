// Package karsttest implements the post-Karst conformance checks shared between
// the op-acceptance-tests acceptance suite and the check-karst CLI. Each
// Check{EIPName} function sends the relevant transactions, asserts the expected
// receipt status, and returns the L2 block range exercised so callers can run
// kona-host cross-checks (acceptance tests) or simply verify the checks ran
// (CLI).
package karsttest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// CheckResult is the L2 block range that a Check function exercised. Callers
// that want to run a kona-host cross-check pass these into RunKonaNative; the
// CLI ignores the result.
type CheckResult struct {
	AgreedBlock uint64 // last block before the check's first tx
	ClaimBlock  uint64 // block of the check's last tx
}

// Precompile addresses referenced by post-Karst checks.
var (
	ModExpPrecompile     = common.HexToAddress("0x0000000000000000000000000000000000000005")
	P256VerifyPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000100")
)

// NewBasePlan returns a txplan.Option. Each per-tx Check{EIPName} composes its
// own options on top of this base plan; gas-limit overrides via txplan.WithGasLimit
// reset the estimator, so the same base plan handles both reverting and successful txs.
func NewBasePlan(cl *ethclient.Client, key *ecdsa.PrivateKey) txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(cl),
		txplan.WithPrivateKey(key),
		txplan.WithPendingNonce(cl),
		txplan.WithAgainstLatestBlockEthClient(cl),
		txplan.WithEstimator(cl, true),
		txplan.WithRetrySubmission(cl, 5, retry.Exponential()),
		txplan.WithRetryInclusion(cl, 5, retry.Exponential()),
	)
}

// BuildModExpInput constructs input data for the MODEXP precompile
// (address 0x05). Format:
//
//	<Bsize (32 bytes)> <Esize (32 bytes)> <Msize (32 bytes)> <B> <E> <M>
func BuildModExpInput(base, exp, mod []byte) []byte {
	input := make([]byte, 0, 96+len(base)+len(exp)+len(mod))
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(base))).Bytes(), 32)...)
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(exp))).Bytes(), 32)...)
	input = append(input, common.LeftPadBytes(new(big.Int).SetInt64(int64(len(mod))).Bytes(), 32)...)
	input = append(input, base...)
	input = append(input, exp...)
	input = append(input, mod...)
	return input
}

// CheckEIP7823 verifies the post-Karst MODEXP upper-bound rule: a MODEXP call
// whose declared input lengths exceed 1024 bytes is included on-chain but
// reverts, while a within-limit call still succeeds. It returns the block numbers
// where its two transactions landed (smaller number first).
func CheckEIP7823(ctx context.Context, logger log.Logger, basePlan txplan.Option) (uint64, uint64, error) {
	logger.Info("EIP-7823: oversized MODEXP call must revert")
	oversizeMod := make([]byte, 1025)
	oversizeMod[1024] = 5
	overReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&ModExpPrecompile),
		txplan.WithData(BuildModExpInput([]byte{2}, []byte{3}, oversizeMod)),
		txplan.WithGasLimit(2_000_000),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("oversized MODEXP submission: %w", err)
	}
	if overReceipt.Status != types.ReceiptStatusFailed {
		return 0, 0, fmt.Errorf("oversized MODEXP: expected revert, got success (block=%v, tx=%s)",
			overReceipt.BlockNumber, overReceipt.TxHash)
	}
	logger.Info("EIP-7823: oversized MODEXP reverted as expected", "block", overReceipt.BlockNumber, "tx", overReceipt.TxHash)

	logger.Info("EIP-7823: within-limit MODEXP call must succeed")
	okReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&ModExpPrecompile),
		txplan.WithData(BuildModExpInput([]byte{2}, []byte{3}, []byte{5})),
		txplan.WithGasLimit(200_000),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("within-limit MODEXP submission: %w", err)
	}
	if okReceipt.Status != types.ReceiptStatusSuccessful {
		return 0, 0, fmt.Errorf("within-limit MODEXP: expected success, got revert (block=%v, tx=%s)",
			okReceipt.BlockNumber, okReceipt.TxHash)
	}
	logger.Info("EIP-7823: within-limit MODEXP succeeded", "block", okReceipt.BlockNumber, "tx", okReceipt.TxHash)

	return bigs.Uint64Strict(overReceipt.BlockNumber), bigs.Uint64Strict(okReceipt.BlockNumber), nil
}

// CheckAll runs every implemented post-Karst check in sequence. It is intended
// for the CLI; the acceptance test invokes individual Check functions per
// sub-test so each can run in parallel and gate its own kona-host cross-check.
func CheckAll(ctx context.Context, logger log.Logger, basePlan txplan.Option) error {
	logger.Info("starting Karst checks")
	if _, _, err := CheckEIP7823(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7823: %w", err)
	}
	logger.Info("completed all Karst checks successfully")
	return nil
}
