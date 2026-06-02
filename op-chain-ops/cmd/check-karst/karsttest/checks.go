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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
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

// Precompile addresses referenced by post-Karst checks.
var (
	ModExpPrecompile     = common.HexToAddress("0x0000000000000000000000000000000000000005")
	Bn256PairPrecompile  = common.HexToAddress("0x0000000000000000000000000000000000000008")
	P256VerifyPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000100")
)

// Per-EIP probe parameters shared between the pre-Karst acceptance sub-tests
// and the post-Karst Check{EIP} functions. Each value is the exact input that
// makes the pre-/post-Karst behaviors diverge — pre-Karst the call succeeds,
// post-Karst it reverts — so the pre-/post-Karst symmetry can never silently
// drift if a floor moves on one side.
const (
	// EIP7883BoundaryGas is the MODEXP probe gas limit: 21,000 intrinsic + 300
	// execution. Empty calldata avoids EIP-7623 calldata cost inflation, so
	// intrinsic gas is exactly 21,000 and the 300-gas execution budget covers
	// EIP-2565's 200-gas floor (pre-Karst) but not EIP-7883's 500-gas floor
	// (post-Karst).
	EIP7883BoundaryGas = 21_300

	// EIP7951BoundaryGas is the P256VERIFY probe gas limit: 21,000 intrinsic +
	// 3,500 execution. Empty calldata avoids EIP-7623 calldata cost inflation,
	// so intrinsic gas is exactly 21,000 and the 3,500-gas execution budget
	// covers RIP-7212's 3,450-gas cost (pre-Karst) but not EIP-7951's
	// 6,900-gas cost (post-Karst).
	EIP7951BoundaryGas = 24_500

	// EIP7823OversizedGasLimit is enough gas to fully process a MODEXP call
	// with the oversized modulus produced by NewEIP7823OversizedModExpInput.
	EIP7823OversizedGasLimit = 2_000_000

	// Bn256PairElementLen is the byte length of one (G1, G2) pair fed to the
	// bn256 pairing precompile: 64 bytes for the G1 point + 128 bytes for the G2
	// point.
	Bn256PairElementLen = 192

	// KarstBn256PairMaxInputSize is the post-Karst max input size for the bn256
	// pairing precompile in bytes: 300 pairs × 192 bytes/pair. Down from Jovian's
	// 81,984 bytes (427 pairs). The same curve is variously called bn128, bn254,
	// or bn256 across the codebase; this matches `BN256_MAX_PAIRING_SIZE_KARST`
	// in kona's FPVM module and `bn254_pair::KARST_MAX_INPUT_SIZE` in op-revm.
	KarstBn256PairMaxInputSize = 300 * Bn256PairElementLen

	// KarstBn256PairProbeGasLimit is the tx gas limit used by every bn256
	// pairing probe (pre-Karst 301-pair success, post-Karst 301-pair length
	// halt, and post-Karst 300-pair success). It must be high enough to fully
	// execute 301 pairs pre-Karst — 301 × 34,000 + 45,000 = 10,279,000
	// precompile gas plus calldata + intrinsic — otherwise an OOG-revert would
	// masquerade as the post-Karst length halt and the post-Karst check would
	// pass against a pre-Karst chain. 12M leaves headroom and stays under the
	// post-Karst EIP-7825 tx-gas cap of 2^24 = 16,777,216.
	KarstBn256PairProbeGasLimit = 12_000_000
)

// NewEIP7823OversizedModExpInput returns MODEXP input whose declared modulus
// is 1025 bytes — one byte over the EIP-7823 1024-byte cap. The high byte is
// non-zero so the modulus parses as a non-zero value. Pre-Karst the call
// succeeds; post-Karst it reverts.
func NewEIP7823OversizedModExpInput() []byte {
	const oversizeModSize = 1025
	mod := make([]byte, oversizeModSize)
	mod[oversizeModSize-1] = 5
	return BuildModExpInput([]byte{2}, []byte{3}, mod)
}

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
	overReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&ModExpPrecompile),
		txplan.WithData(NewEIP7823OversizedModExpInput()),
		txplan.WithGasLimit(EIP7823OversizedGasLimit),
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
		txplan.WithGasLimit(EIP7883BoundaryGas),
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
		txplan.WithGasLimit(EIP7951BoundaryGas),
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

// CheckKarstBn256PairInputLimit verifies the post-Karst bn256 pairing
// precompile input-size cap of 57,600 bytes (300 pairs × 192). A 301-pair
// (57,792-byte) call halts the precompile with Bn254PairLength — consuming
// all tx gas and surfacing as a failed receipt — while a 300-pair within-
// limit call succeeds. Both inputs are all zeros; per EIP-197, (0,0) decodes
// as the G1/G2 point at infinity, and pairing identity pairs yields 1
// (the identity element of F_p12), so the precompile returns the 32-byte
// little-endian 1 for the within-limit call. Returns the block numbers
// where its two transactions landed (smaller number first).
func CheckKarstBn256PairInputLimit(ctx context.Context, logger log.Logger, basePlan txplan.Option) (uint64, uint64, error) {
	logger.Info("Karst bn256 pair: over-limit (301-pair) call must revert")
	overInput := make([]byte, KarstBn256PairMaxInputSize+Bn256PairElementLen) // 301 pairs × 192
	overReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&Bn256PairPrecompile),
		txplan.WithData(overInput),
		txplan.WithGasLimit(KarstBn256PairProbeGasLimit),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("over-limit bn256 pair submission: %w", err)
	}
	if overReceipt.Status != types.ReceiptStatusFailed {
		return 0, 0, fmt.Errorf("over-limit bn256 pair: expected revert, got success (block=%v, tx=%s)",
			overReceipt.BlockNumber, overReceipt.TxHash)
	}
	logger.Info("Karst bn256 pair: over-limit call reverted as expected",
		"block", overReceipt.BlockNumber, "tx", overReceipt.TxHash)

	logger.Info("Karst bn256 pair: within-limit (300-pair) call must succeed")
	okInput := make([]byte, KarstBn256PairMaxInputSize) // 300 pairs × 192
	okReceipt, err := txplan.NewPlannedTx(basePlan,
		txplan.WithTo(&Bn256PairPrecompile),
		txplan.WithData(okInput),
		txplan.WithGasLimit(KarstBn256PairProbeGasLimit),
	).Included.Eval(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("within-limit bn256 pair submission: %w", err)
	}
	if okReceipt.Status != types.ReceiptStatusSuccessful {
		return 0, 0, fmt.Errorf("within-limit bn256 pair: expected success, got revert (block=%v, tx=%s)",
			okReceipt.BlockNumber, okReceipt.TxHash)
	}
	logger.Info("Karst bn256 pair: within-limit call succeeded",
		"block", okReceipt.BlockNumber, "tx", okReceipt.TxHash)

	return bigs.Uint64Strict(overReceipt.BlockNumber), bigs.Uint64Strict(okReceipt.BlockNumber), nil
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

// CheckEIP7825DepositBypass verifies that deposit transactions bypass the
// post-Karst EIP-7825 2^24 gas cap. It submits an L1 deposit with a gas limit
// above the cap, finds the resulting `TransactionDeposited` event, waits for
// the L2 deposit receipt, and asserts the L2 inclusion succeeded with the
// requested gas. Deposits are forced onto L2 by the derivation pipeline rather
// than passing through the txpool; if the cap (a tx-validity rule) applied to
// them, an attacker could trivially brick the rollup. Returns the L2 block
// where the deposit landed.
func CheckEIP7825DepositBypass(
	ctx context.Context,
	logger log.Logger,
	l2 apis.ReceiptFetcher,
	portalAddr, l1Sender common.Address,
	l1Plan txplan.Option,
	depositAmount eth.ETH,
) (uint64, error) {
	depositGasLimit := params.MaxTxGas + 1

	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithTo(portalAddr),
	)
	callPlan, err := contractio.Plan(
		portal.DepositTransaction(l1Sender, depositAmount, depositGasLimit, false, []byte{}),
	)
	if err != nil {
		return 0, fmt.Errorf("plan deposit call: %w", err)
	}

	logger.Info("EIP-7825-deposit: submitting high-gas deposit on L1",
		"gas", depositGasLimit, "amount", depositAmount, "portal", portalAddr)
	// Skip eth_estimateGas: the estimator caps its binary search at MaxTxGas,
	// but ResourceMetering's Burn.gas inside depositTransaction needs to burn
	// ~depositGasLimit gas on L1. WithGasLimit overrides the estimator.
	l1Receipt, err := txplan.NewPlannedTx(l1Plan, callPlan,
		txplan.WithValue(depositAmount),
		txplan.WithGasLimit(depositGasLimit+1_000_000),
	).Included.Eval(ctx)
	if err != nil {
		return 0, fmt.Errorf("L1 deposit submission: %w", err)
	}
	if l1Receipt.Status != types.ReceiptStatusSuccessful {
		return 0, fmt.Errorf("L1 deposit failed: status=%d, tx=%s", l1Receipt.Status, l1Receipt.TxHash)
	}
	logger.Info("EIP-7825-deposit: L1 deposit included", "block", l1Receipt.BlockNumber, "tx", l1Receipt.TxHash)

	var l2DepositTx *types.DepositTx
	for _, log := range l1Receipt.Logs {
		var unmarshalErr error
		if l2DepositTx, unmarshalErr = derive.UnmarshalDepositLogEvent(log); unmarshalErr == nil {
			break
		}
	}
	if l2DepositTx == nil {
		return 0, fmt.Errorf("no TransactionDeposited event in L1 receipt: tx=%s", l1Receipt.TxHash)
	}
	if l2DepositTx.Gas != depositGasLimit {
		return 0, fmt.Errorf("L2 deposit tx gas: got %d, want %d", l2DepositTx.Gas, depositGasLimit)
	}

	l2DepositHash := types.NewTx(l2DepositTx).Hash()
	logger.Info("EIP-7825-deposit: waiting for L2 deposit receipt", "tx", l2DepositHash)
	var l2Receipt *types.Receipt
	for {
		var err error
		l2Receipt, err = l2.TransactionReceipt(ctx, l2DepositHash)
		if err == nil && l2Receipt != nil {
			break
		}
		logger.Info("Could not find transaction receipt", "err", err)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if l2Receipt.Status != types.ReceiptStatusSuccessful {
		return 0, fmt.Errorf("L2 deposit reverted: tx=%s, block=%v", l2DepositHash, l2Receipt.BlockNumber)
	}
	logger.Info("EIP-7825-deposit: L2 deposit included successfully",
		"block", l2Receipt.BlockNumber, "tx", l2DepositHash)

	return bigs.Uint64Strict(l2Receipt.BlockNumber), nil
}

// LatestBlockFetcher fetches the latest block's info and transactions. It is
// the minimum-surface interface satisfied by `apis.EthClient` (the acceptance
// test passes one directly) and by a thin adapter the CLI builds around
// `*ethclient.Client`.
type LatestBlockFetcher interface {
	InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error)
}

// CheckEIP7934BlockSizeDisabled verifies that the OP Stack disables the
// EIP-7934 max-block-size limit (8 MiB) by polling the unsafe head until it
// finds a block whose total transaction-data size exceeds `params.MaxBlockSize`.
// Tx data size is a strict lower bound on RLP-encoded block size, so observing
// txData > MaxBlockSize proves the block exceeds the limit. The check is
// contingent on chain traffic — on a quiet chain it will block forever, so
// callers must bound the wait via `ctx`. Returns the L2 block number where
// the oversized block was observed.
func CheckEIP7934BlockSizeDisabled(
	ctx context.Context,
	logger log.Logger,
	l2 LatestBlockFetcher,
	blockTime time.Duration,
) error {
	for {
		info, txs, err := l2.InfoAndTxsByLabel(ctx, eth.Unsafe)
		if err != nil {
			logger.Warn("EIP-7934: failed to fetch unsafe head, retrying", "err", err)
		} else {
			var totalTxSize int
			for _, tx := range txs {
				bin, marshalErr := tx.MarshalBinary()
				if marshalErr != nil {
					return fmt.Errorf("marshal tx %s: %w", tx.Hash(), marshalErr)
				}
				totalTxSize += len(bin)
			}
			logger.Info("EIP-7934: checking L2 block",
				"number", info.NumberU64(), "txDataSize", totalTxSize, "gasUsed", info.GasUsed())
			if totalTxSize > params.MaxBlockSize {
				logger.Info("EIP-7934: observed oversized block — block-size limit is disabled",
					"number", info.NumberU64(), "txDataSize", totalTxSize, "limit", params.MaxBlockSize)
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(blockTime):
		}
	}
}

type L2 interface {
	apis.EthCode
	apis.ReceiptFetcher
	InfoAndTxsByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, types.Transactions, error)
}

// CheckAll runs every implemented post-Karst check in sequence. It is intended
// for the CLI; the acceptance test invokes individual Check functions per
// sub-test so each can run in parallel and gate its own kona-host cross-check.
func CheckAll(
	ctx context.Context,
	logger log.Logger,
	l2 L2,
	basePlan txplan.Option,
	portalAddr, l1Sender common.Address,
	l1Plan txplan.Option,
	depositAmount eth.ETH,
) error {
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
	if _, _, err := CheckKarstBn256PairInputLimit(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("Karst bn256 pair input limit: %w", err)
	}
	if _, err := CheckEIP7939(ctx, logger, l2, basePlan); err != nil {
		return fmt.Errorf("EIP-7939: %w", err)
	}
	if err := CheckEIP7825(ctx, logger, basePlan); err != nil {
		return fmt.Errorf("EIP-7825: %w", err)
	}
	if _, err := CheckEIP7825DepositBypass(ctx, logger, l2, portalAddr, l1Sender, l1Plan, depositAmount); err != nil {
		return fmt.Errorf("EIP-7825-deposit: %w", err)
	}
	if err := CheckEIP7934BlockSizeDisabled(ctx, logger, l2, 2*time.Second); err != nil {
		return fmt.Errorf("EIP-7934: %w", err)
	}
	logger.Info("completed all Karst checks successfully")
	return nil
}
