package dsl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
)

var emptyHash = common.Hash{}

// L2ELNode wraps a stack.L2ELNode interface for DSL operations
type L2ELNode struct {
	*elNode
	inner   stack.L2ELNode
	control stack.ControlPlane
}

// NewL2ELNode creates a new L2ELNode DSL wrapper
func NewL2ELNode(inner stack.L2ELNode, control stack.ControlPlane) *L2ELNode {
	return &L2ELNode{
		elNode:  newELNode(commonFromT(inner.T()), inner),
		inner:   inner,
		control: control,
	}
}

func (el *L2ELNode) String() string {
	return el.inner.ID().String()
}

// Escape returns the underlying stack.L2ELNode
func (el *L2ELNode) Escape() stack.L2ELNode {
	return el.inner
}

func (el *L2ELNode) ID() stack.L2ELNodeID {
	return el.inner.ID()
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

func (el *L2ELNode) AdvancedFn(label eth.BlockLabel, block uint64) CheckFunc {
	return func() error {
		initial := el.BlockRefByLabel(label)
		target := initial.Number + block
		el.log.Info("expecting chain to advance", "chain", el.inner.ChainID(), "label", label, "target", target)
		attempts := int(block + 3) // intentionally allow few more attempts for avoid flaking
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
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

func (el *L2ELNode) NotAdvancedFn(label eth.BlockLabel) CheckFunc {
	return func() error {
		el.log.Info("expecting chain not to advance", "chain", el.inner.ChainID(), "label", label)
		initial := el.BlockRefByLabel(label)
		attempts := 5 // check few times to make sure head does not advance
		for range attempts {
			time.Sleep(2 * time.Second)
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
		logger := el.log.With("id", el.inner.ID(), "chain", el.ChainID(), "label", label, "target", target)
		logger.Info("Expecting L2EL to reach")
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				head := el.BlockRefByLabel(label)
				if head.Number >= target {
					logger.Info("L2EL advanced", "target", target)
					return nil
				}
				logger.Info("L2EL sync status", "current", head.Number)
				return fmt.Errorf("expected head to advance: %s", label)
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

// ReorgTriggeredFn returns a lambda that checks that a L2 reorg occurred on the expected block
// Composable with other lambdas to wait in parallel
func (el *L2ELNode) ReorgTriggeredFn(target eth.L2BlockRef, attempts int) CheckFunc {
	return func() error {
		el.log.Info("expecting chain to reorg on block ref", "id", el.inner.ID(), "chain", el.inner.ID().ChainID(), "target", target)
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				reorged, err := el.inner.EthClient().BlockRefByNumber(el.ctx, target.Number)
				if err != nil {
					if strings.Contains(err.Error(), "not found") { // reorg is happening wait a bit longer
						el.log.Info("chain still hasn't been reorged", "chain", el.inner.ID().ChainID(), "error", err)
						return err
					}
					return err
				}

				if target.Hash == reorged.Hash { // want not equal
					el.log.Info("chain still hasn't been reorged", "chain", el.inner.ID().ChainID(), "ref", reorged)
					return fmt.Errorf("expected head to reorg %s, but got %s", target, reorged)
				}

				if target.ParentHash != reorged.ParentHash && target.ParentHash != emptyHash {
					return fmt.Errorf("expected parent of target to be the same as the parent of the reorged head, but they are different")
				}

				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "pre_blockref", target)
				el.log.Info("reorg on divergence block", "chain", el.inner.ID().ChainID(), "post_blockref", reorged)

				return nil
			})
	}
}

func (el *L2ELNode) Advanced(label eth.BlockLabel, block uint64) {
	el.require.NoError(el.AdvancedFn(label, block)())
}

func (el *L2ELNode) Reached(label eth.BlockLabel, block uint64, attempts int) {
	el.require.NoError(el.ReachedFn(label, block, attempts)())
}

func (el *L2ELNode) NotAdvanced(label eth.BlockLabel) {
	el.require.NoError(el.NotAdvancedFn(label)())
}

func (el *L2ELNode) ReorgTriggered(target eth.L2BlockRef, attempts int) {
	el.require.NoError(el.ReorgTriggeredFn(target, attempts)())
}

func (el *L2ELNode) TransactionTimeout() time.Duration {
	return el.inner.TransactionTimeout()
}

// L1OriginReachedFn returns a lambda that waits for the L1 origin to reach the target block number.
func (el *L2ELNode) L1OriginReachedFn(label eth.BlockLabel, l1OriginTarget uint64, attempts int) CheckFunc {
	return func() error {
		logger := el.log.With("id", el.inner.ID(), "chain", el.ChainID(), "label", label, "l1OriginTarget", l1OriginTarget)
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

// VerifyWithdrawalHashChangedIn verifies that the withdrawal hash changed between the parent and current block
// This is used to verify that the withdrawal hash changed in the block where the withdrawal was initiated
func (el *L2ELNode) VerifyWithdrawalHashChangedIn(blockHash common.Hash) {
	l2Client := el.inner.L2EthClient()

	postBlockWithdrawalInfo, err := l2Client.InfoByHash(el.ctx, blockHash)
	el.require.NoError(err, "failed to get post-withdrawal block info")

	parentBlockInfo, err := l2Client.InfoByHash(el.ctx, postBlockWithdrawalInfo.ParentHash())
	el.require.NoError(err, "failed to get parent block info")

	postProof, err := l2Client.GetProof(el.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, blockHash.String())
	el.require.NoError(err, "failed to get post-withdrawal storage proof")

	parentProof, err := l2Client.GetProof(el.ctx, predeploys.L2ToL1MessagePasserAddr, []common.Hash{}, postBlockWithdrawalInfo.ParentHash().String())
	el.require.NoError(err, "failed to get parent storage proof")

	el.require.NotEqual(parentProof.StorageHash, postProof.StorageHash, "withdrawal hash should have changed between parent and current block")

	el.require.Equal(postProof.StorageHash, *postBlockWithdrawalInfo.WithdrawalsRoot(), "post-withdrawal storage root should match block header withdrawal root")
	el.require.Equal(parentProof.StorageHash, *parentBlockInfo.WithdrawalsRoot(), "parent storage root should match block header withdrawal root")

	el.log.Info("Withdrawal hash verification successful",
		"parentBlock", postBlockWithdrawalInfo.ParentHash(),
		"currentBlock", blockHash,
		"parentStorageRoot", parentProof.StorageHash,
		"currentStorageRoot", postProof.StorageHash)
}

func (el *L2ELNode) Stop() {
	el.log.Info("Stopping", "id", el.inner.ID())
	el.control.L2ELNodeState(el.inner.ID(), stack.Stop)
}

func (el *L2ELNode) Start() {
	el.control.L2ELNodeState(el.inner.ID(), stack.Start)
}

func (el *L2ELNode) PeerWith(peer *L2ELNode) {
	sysgo.ConnectP2P(el.ctx, el.require, el.inner.L2EthClient().RPC(), peer.inner.L2EthClient().RPC())
}

func (el *L2ELNode) DisconnectPeerWith(peer *L2ELNode) {
	sysgo.DisconnectP2P(el.ctx, el.require, el.inner.L2EthClient().RPC(), peer.inner.L2EthClient().RPC())
}

func (el *L2ELNode) PayloadByNumber(number uint64) *eth.ExecutionPayloadEnvelope {
	payload, err := el.inner.L2EthExtendedClient().PayloadByNumber(el.ctx, number)
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

func (el *L2ELNode) ChainSyncStatus(chainID eth.ChainID, lvl types.SafetyLevel) eth.BlockID {
	el.require.Equal(chainID, el.inner.ID().ChainID(), "chain ID mismatch")
	var blockRef eth.L2BlockRef
	switch lvl {
	case types.Finalized:
		blockRef = el.BlockRefByLabel(eth.Finalized)
	case types.CrossSafe, types.LocalSafe:
		blockRef = el.BlockRefByLabel(eth.Safe)
	case types.CrossUnsafe, types.LocalUnsafe:
		blockRef = el.BlockRefByLabel(eth.Unsafe)
	default:
		el.require.NoError(errors.New("invalid safety level"))
	}
	return blockRef.ID()
}

func (el *L2ELNode) MatchedFn(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int) CheckFunc {
	return MatchedFn(el, refNode, el.log, el.ctx, lvl, el.ChainID(), attempts)
}

func (el *L2ELNode) Matched(refNode SyncStatusProvider, lvl types.SafetyLevel, attempts int) {
	el.require.NoError(el.MatchedFn(refNode, lvl, attempts)())
}

func (el *L2ELNode) UnsafeHead() *BlockRefResult {
	return &BlockRefResult{T: el.t, BlockRef: el.BlockRefByLabel(eth.Unsafe)}
}

func (el *L2ELNode) SafeHead() *BlockRefResult {
	return &BlockRefResult{T: el.t, BlockRef: el.BlockRefByLabel(eth.Safe)}
}

type BlockRefResult struct {
	T        devtest.T
	BlockRef eth.L2BlockRef
}

func (r *BlockRefResult) NumEqualTo(num uint64) *BlockRefResult {
	r.T.Require().Equal(num, r.BlockRef.Number)
	return r
}
