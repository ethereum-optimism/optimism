// Package karsttest implements the post-Karst conformance checks shared between
// the op-acceptance-tests acceptance suite and the check-karst CLI. Each
// Check{EIPName} function sends the relevant transactions, asserts the expected
// receipt status, and returns the L2 block range exercised so callers can run
// kona-host cross-checks (acceptance tests) or simply verify the checks ran
// (CLI).
package karsttest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// CLZBytecode is EVM init code that computes CLZ(1) and returns the 32-byte
// result. CLZ(1) = 255 because 1 has 255 leading zero bits in a uint256. Used
// by both CheckEIP7939 (post-Karst, where deployment must succeed) and the
// pre-Karst acceptance sub-test (where deployment must fail because the CLZ
// opcode is invalid).
var CLZBytecode = []byte{
	byte(vm.PUSH1), 1, // stack: [1]
	byte(vm.CLZ),      // stack: [255] (1 has 255 leading zeros)
	byte(vm.PUSH1), 0, // stack: [0, 255]
	byte(vm.MSTORE),    // mem[0:32] = 255
	byte(vm.PUSH1), 32, // stack: [32]
	byte(vm.PUSH1), 0, // stack: [0, 32]
	byte(vm.RETURN), // return mem[0:32]
}

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

// CheckEIP7883 verifies the post-Karst MODEXP gas-cost increase: an empty-input
// MODEXP call landing exactly on the gas floor (21,000 intrinsic + 300 execution
// gas) reverts under EIP-7883's 500-gas floor where it would have succeeded
// against EIP-2565's 200-gas floor, and a within-floor call (21,000 + 600)
// succeeds. Empty calldata avoids EIP-7623 calldata cost inflation, so intrinsic
// gas is exactly 21,000 and tx gas limit minus 21,000 is the execution budget.
// It returns the block numbers where its two transactions landed (smaller
// number first).
func CheckEIP7883(ctx context.Context, logger log.Logger, basePlan txplan.Option) (uint64, uint64, error) {
	logger.Info("EIP-7883: under-gas MODEXP call must OOG-revert against the 500-gas floor")
	underGasReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&ModExpPrecompile),
		txplan.WithGasLimit(21_300),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("under-gas MODEXP submission: %w", err)
	}
	if underGasReceipt.Status != types.ReceiptStatusFailed {
		return 0, 0, fmt.Errorf("under-gas MODEXP: expected revert, got success (block=%v, tx=%s)",
			underGasReceipt.BlockNumber, underGasReceipt.TxHash)
	}
	logger.Info("EIP-7883: under-gas MODEXP reverted as expected", "block", underGasReceipt.BlockNumber, "tx", underGasReceipt.TxHash)

	logger.Info("EIP-7883: within-floor MODEXP call must succeed")
	sufficientReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&ModExpPrecompile),
		txplan.WithGasLimit(21_600),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("within-floor MODEXP submission: %w", err)
	}
	if sufficientReceipt.Status != types.ReceiptStatusSuccessful {
		return 0, 0, fmt.Errorf("within-floor MODEXP: expected success, got revert (block=%v, tx=%s)",
			sufficientReceipt.BlockNumber, sufficientReceipt.TxHash)
	}
	logger.Info("EIP-7883: within-floor MODEXP succeeded", "block", sufficientReceipt.BlockNumber, "tx", sufficientReceipt.TxHash)

	return bigs.Uint64Strict(underGasReceipt.BlockNumber), bigs.Uint64Strict(sufficientReceipt.BlockNumber), nil
}

// CheckEIP7951 verifies the post-Karst P256VERIFY gas-cost increase: an
// empty-input call landing exactly on the pre-Karst gas budget (21,000
// intrinsic + 3,500 execution gas) reverts under EIP-7951's 6,900-gas cost
// where it would have succeeded against RIP-7212's 3,450-gas cost, and a
// within-cost call (21,000 + 7,000) succeeds. The precompile returns empty
// for non-160-byte input but charges its full cost regardless. Empty calldata
// avoids EIP-7623 calldata cost inflation, so intrinsic gas is exactly 21,000
// and tx gas limit minus 21,000 is the execution budget. It returns the block
// numbers where its two transactions landed (smaller number first).
func CheckEIP7951(ctx context.Context, logger log.Logger, basePlan txplan.Option) (uint64, uint64, error) {
	logger.Info("EIP-7951: under-gas P256VERIFY call must OOG-revert against the 6,900-gas cost")
	underGasReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&P256VerifyPrecompile),
		txplan.WithGasLimit(24_500),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("under-gas P256VERIFY submission: %w", err)
	}
	if underGasReceipt.Status != types.ReceiptStatusFailed {
		return 0, 0, fmt.Errorf("under-gas P256VERIFY: expected revert, got success (block=%v, tx=%s)",
			underGasReceipt.BlockNumber, underGasReceipt.TxHash)
	}
	logger.Info("EIP-7951: under-gas P256VERIFY reverted as expected", "block", underGasReceipt.BlockNumber, "tx", underGasReceipt.TxHash)

	logger.Info("EIP-7951: within-cost P256VERIFY call must succeed")
	sufficientReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&P256VerifyPrecompile),
		txplan.WithGasLimit(28_000),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("within-cost P256VERIFY submission: %w", err)
	}
	if sufficientReceipt.Status != types.ReceiptStatusSuccessful {
		return 0, 0, fmt.Errorf("within-cost P256VERIFY: expected success, got revert (block=%v, tx=%s)",
			sufficientReceipt.BlockNumber, sufficientReceipt.TxHash)
	}
	logger.Info("EIP-7951: within-cost P256VERIFY succeeded", "block", sufficientReceipt.BlockNumber, "tx", sufficientReceipt.TxHash)

	return bigs.Uint64Strict(underGasReceipt.BlockNumber), bigs.Uint64Strict(sufficientReceipt.BlockNumber), nil
}

// CheckEIP7939 verifies the post-Karst CLZ opcode (0x1e). It deploys a contract
// whose init code computes CLZ(1) = 255 and returns the 32-byte result. Pre-Karst
// the opcode is invalid and the init code aborts; post-Karst it executes and
// the deployed code is the 32-byte left-padded CLZ(1) value. Returns the block
// number where the deployment landed.
func CheckEIP7939(ctx context.Context, logger log.Logger, l2 apis.EthCode, basePlan txplan.Option) (uint64, error) {
	logger.Info("EIP-7939: CLZ contract deployment must succeed")
	receipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithData(CLZBytecode),
		txplan.WithGasLimit(100_000),
	).Included.Eval(ctx)
	if err != nil {
		return 0, fmt.Errorf("CLZ deploy submission: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return 0, fmt.Errorf("CLZ deploy: expected success, got revert (block=%v, tx=%s)",
			receipt.BlockNumber, receipt.TxHash)
	}
	logger.Info("EIP-7939: CLZ deployment succeeded", "block", receipt.BlockNumber, "tx", receipt.TxHash, "addr", receipt.ContractAddress)

	// The deployed code IS the 32-byte CLZ(1) result.
	deployedCode, err := l2.CodeAtHash(ctx, receipt.ContractAddress, receipt.BlockHash)
	if err != nil {
		return 0, fmt.Errorf("CLZ deployed code lookup: %w", err)
	}
	expected := common.LeftPadBytes([]byte{0xff}, 32)
	if !bytes.Equal(deployedCode, expected) {
		return 0, fmt.Errorf("CLZ(1) deployed code mismatch: expected %x, got %x", expected, deployedCode)
	}
	logger.Info("EIP-7939: CLZ(1) = 255 verified via deployed code")
	return bigs.Uint64Strict(receipt.BlockNumber), nil
}

// CheckEIP7825 verifies the post-Karst transaction-gas-limit cap of 2^24:
// op-reth's RPC must reject a tx whose gas limit exceeds the cap at submission
// time, so the tx never lands on chain. Returns no block range because no tx
// is included — the cap is a tx-validity rule, not an EVM rule, so there is
// nothing for kona-host to cross-check.
func CheckEIP7825(ctx context.Context, logger log.Logger, basePlan txplan.Option) error {
	logger.Info("EIP-7825: tx with gas > 2^24 must be rejected at submission")
	_, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&common.Address{}),
		txplan.WithGasLimit(params.MaxTxGas+1),
	).Included.Eval(ctx)
	if err == nil {
		return fmt.Errorf("expected rejection for gas > 2^24, got success")
	}
	logger.Info("EIP-7825: high-gas tx rejected as expected", "err", err)
	return nil
}

// CheckAll runs every implemented post-Karst check in sequence. It is intended
// for the CLI; the acceptance test invokes individual Check functions per
// sub-test so each can run in parallel and gate its own kona-host cross-check.
func CheckAll(ctx context.Context, logger log.Logger, l2 apis.EthCode, basePlan txplan.Option) error {
	logger.Info("starting Karst checks")
	if _, _, err := CheckEIP7823(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7823: %w", err)
	}
	if _, _, err := CheckEIP7883(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7883: %w", err)
	}
	if _, _, err := CheckEIP7951(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7951: %w", err)
	}
	if _, err := CheckEIP7939(ctx, logger, l2, basePlan); err != nil {
		return fmt.Errorf("EIP-7939: %w", err)
	}
	if err := CheckEIP7825(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7825: %w", err)
	}
	logger.Info("completed all Karst checks successfully")
	return nil
}
