package dsl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var emptyHash = common.Hash{}

// L2ELNode wraps a stack.L2ELNode interface for DSL operations
type L2ELNode struct {
	*elNode
	inner stack.L2ELNode
}

// NewL2ELNode creates a new L2ELNode DSL wrapper
func NewL2ELNode(inner stack.L2ELNode) *L2ELNode {
	return &L2ELNode{
		elNode: newELNode(commonFromT(inner.T()), inner),
		inner:  inner,
	}
}

func (el *L2ELNode) String() string {
	return el.inner.Name()
}

// Escape returns the underlying stack.L2ELNode
func (el *L2ELNode) Escape() stack.L2ELNode {
	return el.inner
}

func (el *L2ELNode) EthClient() apis.EthClient {
	return el.inner.EthClient()
}

func (el *L2ELNode) BlockRefByLabel(label eth.BlockLabel) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByLabel(ctx, label)
	el.require.NoError(err, "block not found using block label")
	return block
}

func (el *L2ELNode) BlockRefByHash(hash common.Hash) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByHash(ctx, hash)
	el.require.NoError(err, "block not found using block hash")
	return block
}

// AdvancedOption configures an AdvancedFn call.
type AdvancedOption func(*advancedOpts)

type advancedOpts struct {
	attempts int // number of 2-second polling attempts
}

// WithTimeout overrides AdvancedFn's default polling budget. The argument is
// the number of attempts; each attempt polls 2 seconds apart, so attempts*2
// is the total wall-clock timeout.
func WithTimeout(attempts int) AdvancedOption {
	return func(o *advancedOpts) { o.attempts = attempts }
}

func (el *L2ELNode) AdvancedFn(label eth.BlockLabel, block uint64, opts ...AdvancedOption) CheckFunc {
	o := advancedOpts{attempts: int(block + 30)}
	for _, opt := range opts {
		opt(&o)
	}
	return func() error {
		initial := el.BlockRefByLabel(label)
		target := initial.Number + block
		el.log.Info("expecting chain to advance", "chain", el.inner.ChainID(), "label", label, "target", target, "attempts", o.attempts)
		return retry.Do0(el.ctx, o.attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := el.BlockRefByLabel(label)
				if head.Number >= target {
					el.log.Info("chain advanced", "chain", el.inner.ChainID(), "target", target)
					return nil
				}
				el.log.Info("chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", target)
				return fmt.Errorf("expected head to advance: %s", label)
			})
	}
}

func (el *L2ELNode) NotAdvancedFn(label eth.BlockLabel, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain not to advance", "chain", el.inner.ChainID(), "label", label)
		initial := el.BlockRefByLabel(label)
		for range attempts {
			if err := clock.SystemClock.SleepCtx(el.ctx, 2*time.Second); err != nil { // nosemgrep: flake-sleep-in-test -- asserting absence of progress; no chain event to wait on
				return err
			}
			head := el.BlockRefByLabel(label)
			el.log.Info("chain sync status", "chain", el.inner.ChainID(), "initial", initial.Number, "current", head.Number, "target", initial.Number)
			if head.Hash == initial.Hash {
				continue
			}
			return fmt.Errorf("expected head not to advance: %s", label)
		}
		return nil
	}
}

func (el *L2ELNode) ReachedFn(label eth.BlockLabel, target uint64, attempts int) CheckFunc {
	return func() error {
		logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", label, "target", target)
		logger.Info("Expecting L2EL to reach")
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := el.BlockRefByLabel(label)
				if head.Number >= target {
					logger.Info("L2EL advanced", "target", target)
					return nil
				}
				logger.Info("L2EL sync status", "current", head.Number)
				return fmt.Errorf("expected head for label=%s to advance to target=%d, but got current=%d", label, target, head.Number)
			})
	}
}

func (el *L2ELNode) BlockRefByNumber(num uint64) eth.L2BlockRef {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	block, err := el.inner.L2EthClient().L2BlockRefByNumber(ctx, num)
	el.require.NoError(err, "block not found using block number %d", num)
	return block
}

// ReorgTriggeredFn returns a lambda that checks that a L2 reorg occurred on or before the expected block
// Composable with other lambdas to wait in parallel
func (el *L2ELNode) ReorgTriggeredFn(target eth.L2BlockRef, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain to reorg on block ref", "name", el.inner.Name(), "chain", el.inner.ChainID(), "target", target)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				reorged, err := el.reorgTriggered(target)
				if err == nil {
					el.log.Info("reorg on divergence block", "chain", el.inner.ChainID(), "pre_blockref", target, "post_blockref", reorged)
				}
				return err
			})
	}
}

// ReorgExactFn returns a lambda that checks that a L2 reorg occurred on the exact target L2 block.
// If an L2 block prior to target was reorged, this function will block forever.
// Composable with other lambdas to wait in parallel.
func (el *L2ELNode) ReorgExactFn(target eth.L2BlockRef, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain to reorg on block ref", "name", el.inner.Name(), "chain", el.inner.ChainID(), "target", target)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				reorged, err := el.reorgTriggered(target)
				if err != nil {
					return err
				}

				if target.ParentHash != reorged.ParentHash && target.ParentHash != emptyHash {
					return fmt.Errorf("expected parent of target to be the same as the parent of the reorged head, but they are different")
				}

				el.log.Info("reorg on divergence block", "chain", el.inner.ChainID(), "pre_blockref", target, "post_blockref", reorged)

				return nil
			})
	}
}

func (el *L2ELNode) reorgTriggered(target eth.L2BlockRef) (eth.BlockRef, error) {
	reorged, err := el.inner.EthClient().BlockRefByNumber(el.ctx, target.Number)
	if err != nil {
		if strings.Contains(err.Error(), "not found") { // reorg is happening wait a bit longer
			el.log.Info("chain still hasn't been reorged", "chain", el.inner.ChainID(), "error", err)
			return eth.BlockRef{}, err
		}
		return eth.BlockRef{}, err
	}

	if target.Hash == reorged.Hash { // want not equal
		el.log.Info("chain still hasn't been reorged", "chain", el.inner.ChainID(), "ref", reorged)
		return eth.BlockRef{}, fmt.Errorf("expected head to reorg %s, but got %s", target, reorged)
	}

	return reorged, nil
}

func (el *L2ELNode) Advanced(label eth.BlockLabel, block uint64) {
	el.require.NoError(el.AdvancedFn(label, block)())
}

func (el *L2ELNode) Reached(label eth.BlockLabel, block uint64, attempts int) {
	el.require.NoError(el.ReachedFn(label, block, attempts)())
}

func (el *L2ELNode) NotAdvanced(label eth.BlockLabel, attempts int) {
	el.require.NoError(el.NotAdvancedFn(label, attempts)())
}

func (el *L2ELNode) NotAdvancedUnsafe(attempts int) {
	el.NotAdvanced(eth.Unsafe, attempts)
}

func (el *L2ELNode) ReorgTriggered(target eth.L2BlockRef, attempts int) {
	el.require.NoError(el.ReorgTriggeredFn(target, attempts)())
}

func (el *L2ELNode) ReorgExact(target eth.L2BlockRef, attempts int) {
	el.require.NoError(el.ReorgExactFn(target, attempts)())
}

func (el *L2ELNode) TransactionTimeout() time.Duration {
	return el.inner.TransactionTimeout()
}

// L1OriginReachedFn returns a lambda that waits for the L1 origin to reach the target block number.
func (el *L2ELNode) L1OriginReachedFn(label eth.BlockLabel, l1OriginTarget uint64, attempts int) CheckFunc {
	return func() error {
		logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", label, "l1OriginTarget", l1OriginTarget)
		logger.Info("Expecting L2EL to reach L1 origin")
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 1 * time.Second},
			func() error {
				head := el.BlockRefByLabel(label)
				if head.L1Origin.Number >= l1OriginTarget {
					logger.Info("L2EL advanced L1 origin", "l1OriginTarget", l1OriginTarget)
					return nil
				}
				logger.Debug("L2EL sync status", "head", head.ID())
				return fmt.Errorf("L1 origin of %s not advanced yet", label)
			})
	}
}

// WaitL1OriginReached waits for the L1 origin to reach the target block number.
func (el *L2ELNode) WaitL1OriginReached(label eth.BlockLabel, l1OriginTarget uint64, attempts int) {
	el.require.NoError(el.L1OriginReachedFn(label, l1OriginTarget, attempts)())
}

// WaitL1OriginHash polls until the L2 chain at the given label references the target L1 block.
// If the head's L1 origin has advanced past the target number (e.g. due to a large batch),
// it walks back through L2 blocks to find one with the target L1 origin number and checks its hash.
func (el *L2ELNode) WaitL1OriginHash(label eth.BlockLabel, target eth.BlockID, attempts int) {
	logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "label", label, "target", target)
	logger.Info("Expecting L2EL L1 origin to match")
	el.require.NoError(retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
		func() error {
			head := el.BlockRefByLabel(label)
			if head.L1Origin.Number < target.Number {
				logger.Debug("L2EL L1 origin not yet reached", "head", head.ID(), "l1Origin", head.L1Origin)
				return fmt.Errorf("L1 origin of %s head has not reached target yet", label)
			}
			// Head's L1 origin is at or past the target number. Walk back to find
			// the L2 block whose L1 origin number matches the target.
			block := head
			for block.L1Origin.Number > target.Number && block.Number > 0 {
				block = el.BlockRefByNumber(block.Number - 1)
			}
			if block.L1Origin.Hash == target.Hash {
				logger.Info("L2EL L1 origin matched", "l2Block", block.ID(), "l1Origin", block.L1Origin)
				return nil
			}
			logger.Debug("L2EL L1 origin hash mismatch", "l2Block", block.ID(), "l1Origin", block.L1Origin)
			return fmt.Errorf("L1 origin hash of %s head does not match target", label)
		}))
}

// VerifyWithdrawalHashChangedIn verifies that the withdrawal hash changed between the parent and current block.
// This is used to verify that the withdrawal hash changed in the block where the withdrawal was initiated.
//
// Some EL backends, such as op-reth, can briefly lag in serving historical proofs for a block that has
// already been inserted. Retry until the proof backend catches up instead of failing immediately.
func (el *L2ELNode) VerifyWithdrawalHashChangedIn(blockHash common.Hash) {
	l2Client := el.inner.L2EthClient()

	el.require.Eventually(func() bool {
		postBlockWithdrawalInfo, err := l2Client.InfoByHash(el.ctx, blockHash)
		if err != nil {
			el.log.Debug("Waiting for post-withdrawal block info", "blockHash", blockHash, "err", err)
			return false
		}
		if postBlockWithdrawalInfo.WithdrawalsRoot() == nil {
			err = fmt.Errorf("post-withdrawal block %s has no withdrawals root", blockHash)
			el.log.Debug("Waiting for post-withdrawal withdrawals root", "blockHash", blockHash, "err", err)
			return false
		}

		parentBlockInfo, err := l2Client.InfoByHash(el.ctx, postBlockWithdrawalInfo.ParentHash())
		if err != nil {
			el.log.Debug("Waiting for parent block info", "blockHash", postBlockWithdrawalInfo.ParentHash(), "err", err)
			return false
		}
		if parentBlockInfo.WithdrawalsRoot() == nil {
			err = fmt.Errorf("parent block %s has no withdrawals root", postBlockWithdrawalInfo.ParentHash())
			el.log.Debug("Waiting for parent withdrawals root", "blockHash", postBlockWithdrawalInfo.ParentHash(), "err", err)
			return false
		}

		postProof, err := l2Client.GetProof(el.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, blockHash.String())
		if err != nil {
			el.log.Debug("Waiting for post-withdrawal storage proof", "blockHash", blockHash, "err", err)
			return false
		}

		parentProof, err := l2Client.GetProof(el.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, postBlockWithdrawalInfo.ParentHash().String())
		if err != nil {
			el.log.Debug("Waiting for parent storage proof", "blockHash", postBlockWithdrawalInfo.ParentHash(), "err", err)
			return false
		}

		if parentProof.StorageHash == postProof.StorageHash {
			err = fmt.Errorf("withdrawal hash did not change between parent %s and current %s", postBlockWithdrawalInfo.ParentHash(), blockHash)
			el.log.Debug("Waiting for withdrawal hash change",
				"parentBlock", postBlockWithdrawalInfo.ParentHash(),
				"currentBlock", blockHash,
				"storageRoot", postProof.StorageHash,
				"err", err)
			return false
		}

		if postProof.StorageHash != *postBlockWithdrawalInfo.WithdrawalsRoot() {
			err = fmt.Errorf("post-withdrawal storage root mismatch: proof=%s header=%s", postProof.StorageHash, *postBlockWithdrawalInfo.WithdrawalsRoot())
			el.log.Debug("Waiting for post-withdrawal storage root to match header",
				"blockHash", blockHash,
				"proofStorageRoot", postProof.StorageHash,
				"headerWithdrawalsRoot", *postBlockWithdrawalInfo.WithdrawalsRoot(),
				"err", err)
			return false
		}

		if parentProof.StorageHash != *parentBlockInfo.WithdrawalsRoot() {
			err = fmt.Errorf("parent storage root mismatch: proof=%s header=%s", parentProof.StorageHash, *parentBlockInfo.WithdrawalsRoot())
			el.log.Debug("Waiting for parent storage root to match header",
				"blockHash", postBlockWithdrawalInfo.ParentHash(),
				"proofStorageRoot", parentProof.StorageHash,
				"headerWithdrawalsRoot", *parentBlockInfo.WithdrawalsRoot(),
				"err", err)
			return false
		}

		el.log.Info("Withdrawal hash verification successful",
			"parentBlock", postBlockWithdrawalInfo.ParentHash(),
			"currentBlock", blockHash,
			"parentStorageRoot", parentProof.StorageHash,
			"currentStorageRoot", postProof.StorageHash)

		return true
	}, 30*time.Second, 200*time.Millisecond, "withdrawal proof data did not become available in time")
}

func (el *L2ELNode) Stop() {
	el.log.Info("Stopping", "name", el.inner.Name())
	lifecycle, ok := el.inner.(stack.Lifecycle)
	el.require.Truef(ok, "L2EL node %s is not lifecycle-controllable", el.inner.Name())
	lifecycle.Stop()
}

func (el *L2ELNode) Start() {
	lifecycle, ok := el.inner.(stack.Lifecycle)
	el.require.Truef(ok, "L2EL node %s is not lifecycle-controllable", el.inner.Name())
	lifecycle.Start()
}

func (el *L2ELNode) PeerWith(peer *L2ELNode) {
	sysgo.ConnectP2P(el.ctx, el.require, el.inner.L2EthClient().RPC(), peer.inner.L2EthClient().RPC(), false)
}

func (el *L2ELNode) DisconnectPeerWith(peer *L2ELNode) {
	sysgo.DisconnectP2P(el.ctx, el.require, el.inner.L2EthClient().RPC(), peer.inner.L2EthClient().RPC())
}

func (el *L2ELNode) PayloadByNumber(number uint64) *eth.ExecutionPayloadEnvelope {
	payload, err := el.inner.L2EthClient().PayloadByNumber(el.ctx, number)
	el.require.NoError(err, "failed to get payload")
	return payload
}

// NewPayload fetches payload for target number from the reference EL Node, and inserts the payload
func (el *L2ELNode) NewPayload(refNode *L2ELNode, number uint64) *NewPayloadResult {
	el.log.Info("NewPayload", "number", number, "node", el, "refNode", refNode)
	payload := refNode.PayloadByNumber(number)
	return el.NewPayloadRaw(payload)
}

func (el *L2ELNode) NewPayloadRaw(payload *eth.ExecutionPayloadEnvelope) *NewPayloadResult {
	el.log.Info("NewPayloadRaw", "number", payload.ExecutionPayload.BlockNumber)
	status, err := el.inner.L2EngineClient().NewPayload(el.ctx, payload.ExecutionPayload, payload.ParentBeaconBlockRoot)
	return &NewPayloadResult{T: el.t, Status: status, Err: err}
}

// ForkchoiceUpdate fetches FCU target hashes from the reference EL node, and FCU update with attributes
func (el *L2ELNode) ForkchoiceUpdate(refNode *L2ELNode, unsafe, safe, finalized uint64, attr *eth.PayloadAttributes) *ForkchoiceUpdateResult {
	unsafeHash := refNode.BlockRefByNumber(unsafe).Hash
	safeHash := refNode.BlockRefByNumber(safe).Hash
	finalizedHash := refNode.BlockRefByNumber(finalized).Hash
	el.log.Info("ForkchoiceUpdate with reference node", "unsafe", unsafe, "safe", safe, "finalized", finalized, "node", el, "refNode", refNode)
	return el.ForkchoiceUpdateRaw(unsafeHash, safeHash, finalizedHash, attr)
}

// ForkchoiceUpdateRaw calls FCU with block hashes with attributes
func (el *L2ELNode) ForkchoiceUpdateRaw(unsafe, safe, finalized common.Hash, attr *eth.PayloadAttributes) *ForkchoiceUpdateResult {
	result := &ForkchoiceUpdateResult{T: el.t}
	refresh := func() {
		result.RefreshCnt += 1
		el.log.Info("ForkchoiceUpdateRaw", "unsafe", unsafe, "safe", safe, "finalized", finalized, "attr", attr, "node", el)
		state := &eth.ForkchoiceState{
			HeadBlockHash:      unsafe,
			SafeBlockHash:      safe,
			FinalizedBlockHash: finalized,
		}
		res, err := el.inner.L2EngineClient().ForkchoiceUpdate(el.ctx, state, attr)
		result.Result = res
		result.Err = err
		if result.Result != nil {
			switch result.Result.PayloadStatus.Status {
			case eth.ExecutionValid:
				result.ValidCnt += 1
			case eth.ExecutionSyncing:
				result.SyncingCnt += 1
			case eth.ExecutionInvalid:
				result.InvalidCnt += 1
			default:
				el.require.NoError(fmt.Errorf("invalid fcu payload status: %s", result.Result.PayloadStatus.Status))
			}
		}
	}
	result.Refresh = refresh
	result.Refresh()
	return result
}

func (el *L2ELNode) FinishedELSync(refNode *L2ELNode, unsafe, safe, finalized uint64) {
	el.log.Info("Trigger EL Sync", "unsafe", unsafe, "safe", safe, "finalized", finalized)
	trial := 1
	el.require.NoError(retry.Do0(el.ctx, 5, &retry.FixedStrategy{Dur: 2 * time.Second}, func() error {
		el.log.Info("FCU to trigger EL Sync", "trial", trial)
		res := el.ForkchoiceUpdate(refNode, unsafe, safe, finalized, nil)
		// If EL Sync triggered, Example logs from L2EL(geth)
		//  New skeleton head announced
		//  Backfilling with the network
		if res.Result.PayloadStatus.Status == eth.ExecutionValid {
			el.log.Info("Finished EL Sync")
			return nil
		}
		trial += 1
		return errors.New("EL Sync not finished")
	}))
}

func (el *L2ELNode) ChainSyncStatus(chainID eth.ChainID, lvl safety.Level) eth.BlockID {
	el.require.Equal(chainID, el.inner.ChainID(), "chain ID mismatch")
	var blockRef eth.L2BlockRef
	switch lvl {
	case safety.Finalized:
		blockRef = el.BlockRefByLabel(eth.Finalized)
	case safety.CrossSafe, safety.LocalSafe:
		blockRef = el.BlockRefByLabel(eth.Safe)
	case safety.CrossUnsafe, safety.LocalUnsafe:
		blockRef = el.BlockRefByLabel(eth.Unsafe)
	default:
		el.require.NoError(errors.New("invalid safety level"))
	}
	return blockRef.ID()
}

func (el *L2ELNode) ChainBlockID(chainID eth.ChainID, number uint64) (eth.BlockID, error) {
	el.require.Equal(chainID, el.inner.ChainID(), "chain ID mismatch")
	ref, err := el.inner.L2EthClient().L2BlockRefByNumber(el.ctx, number)
	if err != nil {
		return eth.BlockID{}, err
	}
	return ref.ID(), nil
}

// WaitForReceipt waits for a transaction receipt to be available, retrying until found or timeout.
func (el *L2ELNode) WaitForReceipt(txHash common.Hash) *types.Receipt {
	var receipt *types.Receipt
	err := retry.Do0(el.ctx, 30, &retry.FixedStrategy{Dur: 500 * time.Millisecond}, func() error {
		var err error
		receipt, err = el.inner.EthClient().TransactionReceipt(el.ctx, txHash)
		if err != nil {
			return fmt.Errorf("waiting for receipt of %s: %w", txHash.Hex(), err)
		}
		return nil
	})
	el.require.NoError(err, "failed to get receipt for tx %s", txHash.Hex())
	return receipt
}

func (el *L2ELNode) MatchedFn(refNode SyncStatusProvider, lvl safety.Level, attempts int) CheckFunc {
	return MatchedFn(el, refNode, el.log, el.ctx, lvl, el.ChainID(), attempts)
}

func (el *L2ELNode) InSyncFn(other SyncStatusProvider, lvl safety.Level, attempts int) CheckFunc {
	return InSyncFn(el, other, el.log, el.ctx, lvl, el.ChainID(), attempts)
}

func (el *L2ELNode) Matched(refNode SyncStatusProvider, lvl safety.Level, attempts int) {
	el.require.NoError(el.MatchedFn(refNode, lvl, attempts)())
}

func (el *L2ELNode) InSync(other SyncStatusProvider, lvl safety.Level, attempts int) {
	el.require.NoError(el.InSyncFn(other, lvl, attempts)())
}

func (el *L2ELNode) MatchedUnsafe(refNode SyncStatusProvider, attempts int) {
	el.Matched(refNode, safety.LocalUnsafe, attempts)
}

// WaitForPendingNonceMatchFn returns a lambda that waits for the pending nonce of an account to match the provided reference nonce
func (el *L2ELNode) WaitForPendingNonceMatchFn(account common.Address, nonce uint64, attempts int, duration time.Duration) CheckFunc {
	return func() error {
		logger := el.log.With("name", el.inner.Name(), "account", account)
		logger.Debug("Expecting pending nonce to match with reference nonce", "nonce", nonce)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: duration},
			func() error {
				baseNonce, err := el.inner.EthClient().PendingNonceAt(el.ctx, account)
				if err != nil {
					return fmt.Errorf("failed to get pending nonce from node: %w", err)
				}

				if baseNonce == nonce {
					logger.Debug("Pending nonce matched", "nonce", baseNonce)
					return nil
				}

				logger.Debug("Pending nonce mismatch", "node nonce", baseNonce, "nonce", nonce)
				return fmt.Errorf("expected pending nonce to match: node nonce=%d, reference nonce=%d", baseNonce, nonce)
			})
	}
}

// WaitForPendingNonceMatch waits for the pending nonce of an account to match the reference nonce
func (el *L2ELNode) WaitForPendingNonceMatch(account common.Address, nonce uint64, attempts int, duration time.Duration) {
	el.require.NoError(el.WaitForPendingNonceMatchFn(account, nonce, attempts, duration)())
}

func (el *L2ELNode) UnsafeHead() *BlockRefResult {
	return &BlockRefResult{T: el.t, BlockRef: el.BlockRefByLabel(eth.Unsafe)}
}

func (el *L2ELNode) SafeHead() *BlockRefResult {
	return &BlockRefResult{T: el.t, BlockRef: el.BlockRefByLabel(eth.Safe)}
}

func (el *L2ELNode) FinalizedHead() *BlockRefResult {
	return &BlockRefResult{T: el.t, BlockRef: el.BlockRefByLabel(eth.Finalized)}
}

func (el *L2ELNode) AssertExecMessageNotInBlock(execMessage *ExecMessage) {
	el.AssertTxNotInBlock(bigs.Uint64Strict(execMessage.BlockNumber()), execMessage.TxHash())
}

// AssertTxNotInBlock asserts that a transaction with the given hash does not exist in the block at the given number.
func (el *L2ELNode) AssertTxNotInBlock(blockNumber uint64, txHash common.Hash) {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()

	_, txs, err := el.inner.EthClient().InfoAndTxsByNumber(ctx, blockNumber)
	el.require.NoError(err, "failed to fetch block %d", blockNumber)

	for _, tx := range txs {
		el.require.NotEqualf(tx.Hash(), txHash, "transaction should not exist in block", "Found tx %v in block %v", tx.Hash(), blockNumber)
	}
	el.log.Info("confirmed transaction not in block", "blockNumber", blockNumber, "txHash", txHash)
}

// AssertTxInBlock asserts that a transaction with the given hash does not exist in the block at the given number.
func (el *L2ELNode) AssertTxInBlock(blockNumber uint64, txHash common.Hash) {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()

	_, txs, err := el.inner.EthClient().InfoAndTxsByNumber(ctx, blockNumber)
	el.require.NoError(err, "failed to fetch block %d", blockNumber)

	for _, tx := range txs {
		if tx.Hash() == txHash {
			el.log.Info("confirmed transaction in block", "blockNumber", blockNumber, "txHash", txHash)
			return
		}
	}
	el.require.Fail("transaction should exist in block", "blockNumber", blockNumber, "txHash", txHash)
}

type BlockRefResult struct {
	T        devtest.T
	BlockRef eth.L2BlockRef
}

func (r *BlockRefResult) NumEqualTo(num uint64) *BlockRefResult {
	r.T.Require().Equal(num, r.BlockRef.Number)
	return r
}

func (r *BlockRefResult) IsGenesis() *BlockRefResult {
	return r.NumEqualTo(0)
}
