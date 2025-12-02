package dsl

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// L2Network wraps a stack.L2Network interface for DSL operations
type L2Network struct {
	commonImpl
	inner   stack.L2Network
	control stack.ControlPlane
}

// NewL2Network creates a new L2Network DSL wrapper
func NewL2Network(inner stack.L2Network, control stack.ControlPlane) *L2Network {
	return &L2Network{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
		control:    control,
	}
}

func (n *L2Network) String() string {
	return n.inner.ID().String()
}

func (n *L2Network) ChainID() eth.ChainID {
	return n.inner.ChainID()
}

// Escape returns the underlying stack.L2Network
func (n *L2Network) Escape() stack.L2Network {
	return n.inner
}

func (n *L2Network) L2ELNodes() []*L2ELNode {
	innerNodes := n.inner.L2ELNodes()
	nodes := make([]*L2ELNode, len(innerNodes))
	for i, inner := range innerNodes {
		nodes[i] = NewL2ELNode(inner, n.control)
	}
	return nodes
}

func (n *L2Network) CatchUpTo(o *L2Network) {
	this := n.inner.L2ELNode(match.FirstL2EL)
	other := o.inner.L2ELNode(match.FirstL2EL)

	err := wait.For(n.ctx, 5*time.Second, func() (bool, error) {
		a, err := this.EthClient().InfoByLabel(n.ctx, "latest")
		if err != nil {
			return false, err
		}

		b, err := other.EthClient().InfoByLabel(n.ctx, "latest")
		if err != nil {
			return false, err
		}

		eps := 6.0 // 6 seconds
		if math.Abs(float64(a.Time()-b.Time())) > eps {
			n.log.Warn("L2 networks too far off each other", n.String(), a.Time(), o.String(), b.Time())
			return false, nil
		}

		return true, nil
	})
	n.require.NoError(err, "Expected to get latest block from L2 execution clients")
}

func (n *L2Network) WaitForBlock() eth.BlockRef {
	return NewL2ELNode(n.inner.L2ELNode(match.FirstL2EL), n.control).WaitForBlock()
}

func (n *L2Network) PublicRPC() *L2ELNode {
	if proxyds := match.Proxyd.Match(n.Escape().L2ELNodes()); len(proxyds) > 0 {
		n.log.Info("PublicRPC - Using proxyd", "network", n.String())
		return NewL2ELNode(proxyds[0], n.control)
	}

	n.log.Info("PublicRPC - Using fallback instead of proxyd", "network", n.String())
	// Fallback since sysgo doesn't have proxyd support at the moment, and may never get it.
	return NewL2ELNode(n.inner.L2ELNode(match.FirstL2EL), n.control)
}

// PrintChain is used for testing/debugging, it prints the blockchain hashes and parent hashes to logs, which is useful when developing reorg tests
func (n *L2Network) PrintChain() {
	l2_el := n.inner.L2ELNode(match.FirstL2EL)
	l2_cl := n.inner.L2CLNode(match.FirstL2CL)

	l1_el := n.inner.L1().L1ELNode(match.FirstL1EL)

	biAddr := n.inner.RollupConfig().BatchInboxAddress
	dgfAddr := n.inner.Deployment().DisputeGameFactoryProxyAddr()

	var entries []string
	var totalL2Txs int
	err := retry.Do0(n.ctx, 3, &retry.FixedStrategy{Dur: 200 * time.Millisecond}, func() error {
		entries = []string{}
		totalL2Txs = 0

		ref := n.unsafeHeadRef()

		for i := ref.Number; i > 0; i-- {
			ref, err := l2_el.L2EthClient().L2BlockRefByNumber(n.ctx, i)
			if err != nil {
				return err
			}

			_, l2Txs, err := l2_el.EthClient().InfoAndTxsByHash(n.ctx, ref.Hash)
			if err != nil {
				return err
			}

			_, txs, err := l1_el.EthClient().InfoAndTxsByHash(n.ctx, ref.L1Origin.Hash)
			if err != nil {
				return err
			}

			var batchTxs, dgfTxs int
			for _, tx := range txs {
				to := tx.To()
				if to != nil && *to == biAddr {
					batchTxs++
				}
				if to != nil && *to == dgfAddr {
					dgfTxs++
				}
			}

			entries = append(entries, fmt.Sprintf("Time: %d Block: %s Parent: %s L1 Origin: %s Txs (L2: %d; Batch: %d; DGF: %d)", ref.Time, ref, ref.ParentID(), ref.L1Origin, len(l2Txs), batchTxs, dgfTxs))
			totalL2Txs += len(l2Txs)
		}

		return nil
	})
	n.require.NoError(err, "could not PrintChain after many attempts")

	syncStatus, err := l2_cl.RollupAPI().SyncStatus(n.ctx)
	n.require.NoError(err, "Expected to get sync status")

	entries = append(entries, "")
	entries = append(entries, fmt.Sprintf("Total L2 Txs: %d", totalL2Txs))
	entries = append(entries, "")
	entries = append(entries, "Supervisor Sync view")
	entries = append(entries, "")
	entries = append(entries, fmt.Sprintf("Current L1:      %s", syncStatus.CurrentL1))
	entries = append(entries, fmt.Sprintf("Head L1:         %s", syncStatus.HeadL1))
	entries = append(entries, fmt.Sprintf("Safe L1:         %s", syncStatus.SafeL1))
	entries = append(entries, fmt.Sprintf("Unsafe L2:       %s", syncStatus.UnsafeL2))
	entries = append(entries, fmt.Sprintf("Local-Safe L2:   %s", syncStatus.LocalSafeL2))
	entries = append(entries, fmt.Sprintf("Cross-Unsafe L2: %s", syncStatus.CrossUnsafeL2))
	entries = append(entries, fmt.Sprintf("Cross-Safe L2:   %s", syncStatus.SafeL2))

	n.log.Info("Printing block hashes and parent hashes", "network", n.String(), "chain", n.ChainID())
	spew.Dump(entries)
}

func (n *L2Network) unsafeHeadRef() eth.L2BlockRef {
	l2_el := n.inner.L2ELNode(match.FirstL2EL)

	unsafeHead, err := l2_el.EthClient().InfoByLabel(n.ctx, eth.Unsafe)
	n.require.NoError(err, "Expected to get latest block from L2 execution client")

	unsafeHeadRef, err := l2_el.L2EthClient().L2BlockRefByHash(n.ctx, unsafeHead.Hash())
	n.require.NoError(err, "Expected to get block ref by hash")

	return unsafeHeadRef
}

// IsActivated checks if a given fork has been activated
func (n *L2Network) IsActivated(timestamp uint64) bool {
	blockNum, err := n.Escape().RollupConfig().TargetBlockNumber(timestamp)
	n.require.NoError(err)

	el := n.Escape().L2ELNode(match.FirstL2EL)
	head, err := el.EthClient().BlockRefByLabel(n.ctx, eth.Unsafe)
	n.require.NoError(err)

	return head.Number >= blockNum
}

func (n *L2Network) IsForkActive(fork forks.Name) bool {
	el := NewL2ELNode(n.inner.L2ELNode(match.FirstL2EL), n.control)
	timestamp := el.BlockRefByLabel(eth.Unsafe).Time
	return n.IsForkActiveAt(fork, timestamp)
}

func (n *L2Network) IsForkActiveAt(forkName forks.Name, timestamp uint64) bool {
	return n.Escape().RollupConfig().IsForkActive(forkName, timestamp)
}

// LatestBlockBeforeTimestamp finds the latest block before fork activation
func (n *L2Network) LatestBlockBeforeTimestamp(t devtest.T, timestamp uint64) eth.BlockRef {
	require := t.Require()

	t.Gate().Greater(timestamp, uint64(0), "Must not start fork at genesis")

	blockNum, err := n.Escape().RollupConfig().TargetBlockNumber(timestamp)
	require.NoError(err)

	el := n.Escape().L2ELNode(match.FirstL2EL)
	head, err := el.EthClient().BlockRefByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(err)

	t.Logger().Info("Preparing",
		"head", head, "head_time", head.Time,
		"target_num", blockNum, "target_time", timestamp)

	if head.Number < blockNum {
		t.Logger().Info("No block with given timestamp yet, checking head block instead")
		return head
	} else {
		t.Logger().Info("Reached block already, proceeding with last block before timestamp")
		v, err := el.EthClient().BlockRefByNumber(t.Ctx(), blockNum-1)
		require.NoError(err)
		return v
	}
}

// AwaitActivation awaits the fork activation time, and returns the activation block
func (n *L2Network) AwaitActivation(t devtest.T, forkName rollup.ForkName) eth.BlockID {
	require := t.Require()

	el := n.Escape().L2ELNode(match.FirstL2EL)

	rollupCfg := n.Escape().RollupConfig()
	maybeActivationTime := rollupCfg.ActivationTime(forkName)
	require.NotNil(maybeActivationTime, "Required fork is not scheduled for activation")
	activationTime := *maybeActivationTime
	if activationTime == 0 {
		block, err := el.EthClient().BlockRefByNumber(t.Ctx(), 0)
		require.NoError(err, "Fork activated at genesis, but failed to get genesis block")
		return block.ID()
	}
	blockNum, err := rollupCfg.TargetBlockNumber(activationTime)
	require.NoError(err)
	activationBlock := eth.ToBlockID(NewL2ELNode(el, n.control).WaitForBlockNumber(blockNum))
	t.Logger().Info("Activation block", "block", activationBlock)
	return activationBlock

}

func (n *L2Network) DisputeGameFactoryProxyAddr() common.Address {
	return n.inner.Deployment().DisputeGameFactoryProxyAddr()
}

func (n *L2Network) DepositContractAddr() common.Address {
	return n.inner.RollupConfig().DepositContractAddress
}

func (n *L2Network) DeriveData(blocks int) (channels []derive.ChannelID, channelFrames map[derive.ChannelID][]derive.Frame, l2Txs map[common.Address][]*ethtypes.Transaction) {
	l := n.log
	ctx := n.ctx

	channelFrames = make(map[derive.ChannelID][]derive.Frame)
	channels = make([]derive.ChannelID, 0)
	l2Txs = make(map[common.Address][]*ethtypes.Transaction)

	rollupCfg := n.inner.RollupConfig()
	batchInboxAddr := rollupCfg.BatchInboxAddress

	l1EC := n.inner.L1().L1ELNode(match.FirstL1EL).EthClient()

	// Get current L1 block number before starting to monitor
	startBlockRef, err := l1EC.BlockRefByLabel(ctx, eth.Unsafe)
	n.require.NoError(err, "Failed to get start block number")

	seenChannels := make(map[derive.ChannelID]bool)
	lastBlockRef := startBlockRef

	// Monitor L1 blocks for batch transactions
	for range blocks {
		NewL1ELNode(n.inner.L1().L1ELNode(match.FirstL1EL)).WaitForBlock()

		// Get current block number
		currentBlockRef, err := l1EC.BlockRefByLabel(ctx, eth.Unsafe)
		n.require.NoError(err, "Failed to get current block number")
		blockNum := currentBlockRef.Number
		lastBlockRef = currentBlockRef

		_, txs, err := l1EC.InfoAndTxsByNumber(ctx, blockNum)
		n.require.NoError(err, "Failed to get block %d", blockNum)

		// Process transactions in this block
		for _, tx := range txs {
			// Check if transaction is targeted to BatchInbox
			if tx.To() != nil && *tx.To() == batchInboxAddr {
				// Get transaction sender
				chainID := n.inner.L1().ChainID()
				chainIDBig := chainID.ToBig()
				signer := ethtypes.LatestSignerForChainID(chainIDBig)
				sender, err := signer.Sender(tx)
				n.require.NoError(err, "Failed to get transaction sender")

				l.Debug("Found batch transaction",
					"txHash", tx.Hash(),
					"block", blockNum,
					"sender", sender)

				var datas [][]byte
				if tx.Type() != ethtypes.BlobTxType {
					// Regular transaction - data is in tx.Data()
					datas = append(datas, tx.Data())
				} else {
					// Blob transaction - need to fetch blobs from beacon
					// For now, log that we found a blob tx but skip detailed parsing
					// as it requires beacon API access
					l.Error("Found blob transaction (skipping blob fetch for now)",
						"txHash", tx.Hash(),
						"blobHashes", tx.BlobHashes())
					continue
				}

				// Parse frames from transaction data
				for _, data := range datas {
					frames, err := derive.ParseFrames(data)
					if err != nil {
						l.Warn("Failed to parse frames from transaction",
							"txHash", tx.Hash(),
							"error", err)
						n.require.NoError(err)
					}

					l.Debug("Parsed frames from transaction",
						"txHash", tx.Hash(),
						"frameCount", len(frames))

					// Process each frame
					for _, frame := range frames {
						channelID := frame.ID
						if !seenChannels[channelID] {
							seenChannels[channelID] = true
							l.Debug("Found new channel",
								"channelID", channelID.String(),
								"txHash", tx.Hash(),
								"block", blockNum)
							channels = append(channels, channelID)
						}
						channelFrames[channelID] = append(channelFrames[channelID], frame)
						l.Debug("Frame added to channel",
							"channelID", channelID.String(),
							"frameNumber", frame.FrameNumber,
							"dataLength", len(frame.Data),
							"isLast", frame.IsLast,
							"txHash", tx.Hash())
					}
				}
			}
		}
	}

	// Reassemble channels and extract batches
	for channelID, frames := range channelFrames {
		l.Debug("Processing channel",
			"channelID", channelID.String(),
			"frameCount", len(frames))

		// Sort frames by frame number
		sortedFrames := make([]derive.Frame, len(frames))
		copy(sortedFrames, frames)
		for i := 0; i < len(sortedFrames); i++ {
			for j := i + 1; j < len(sortedFrames); j++ {
				if sortedFrames[i].FrameNumber > sortedFrames[j].FrameNumber {
					sortedFrames[i], sortedFrames[j] = sortedFrames[j], sortedFrames[i]
				}
			}
		}

		// Create a channel and add frames to it
		// We need an L1 block ref for the channel - use the last processed block as the origin
		originBlock := lastBlockRef
		ch := derive.NewChannel(channelID, originBlock, false)

		for _, frame := range sortedFrames {
			err := ch.AddFrame(frame, originBlock)
			if err != nil {
				l.Warn("Failed to add frame to channel",
					"channelID", channelID.String(),
					"frameNumber", frame.FrameNumber,
					"error", err)
				continue
			}
		}

		l.Debug("Channel is ready, extracting batches",
			"channelID", channelID.String(),
			"size", ch.Size())

		channelReader := ch.Reader()
		channelData, err := io.ReadAll(channelReader)
		if err != nil {
			l.Warn("Failed to read channel data",
				"channelID", channelID.String(),
				"error", err)
			continue
		}

		l.Debug("Read channel data",
			"channelID", channelID.String(),
			"dataLength", len(channelData))

		spec := rollup.NewChainSpec(rollupCfg)
		maxRLPBytes := spec.MaxRLPBytesPerChannel(originBlock.Time)
		isFjord := rollupCfg.IsFjord(originBlock.Time)
		batchReader, err := derive.BatchReader(bytes.NewReader(channelData), maxRLPBytes, isFjord)
		if err != nil {
			l.Warn("Failed to create batch reader",
				"channelID", channelID.String(),
				"error", err)
			continue
		}

		// Read all batches from the channel
		batchCount := 0
		for {
			batchData, err := batchReader()
			if err == io.EOF {
				break
			}
			if err != nil {
				l.Warn("Failed to read batch from channel",
					"channelID", channelID.String(),
					"batchCount", batchCount,
					"error", err)
				break
			}

			batchCount++
			batchType := batchData.GetBatchType()

			l.Debug("Found batch in channel",
				"channelID", channelID.String(),
				"batchNumber", batchCount,
				"batchType", batchType,
				"compressionAlgo", batchData.ComprAlgo)

			// Decode the batch based on type
			if batchType == derive.SingularBatchType {
				singularBatch, err := derive.GetSingularBatch(batchData)
				if err != nil {
					l.Warn("Failed to decode singular batch",
						"channelID", channelID.String(),
						"batchNumber", batchCount,
						"error", err)
					n.require.NoError(err)
				}

				for _, txData := range singularBatch.Transactions {
					var tx ethtypes.Transaction
					n.require.NoError(tx.UnmarshalBinary(txData))

					signer := ethtypes.LatestSignerForChainID(rollupCfg.L2ChainID)
					fromAddr, err := signer.Sender(&tx)
					n.require.NoError(err)

					l2Txs[fromAddr] = append(l2Txs[fromAddr], &tx)
				}

			} else if batchType == derive.SpanBatchType {
				spanBatch, err := derive.DeriveSpanBatch(
					batchData,
					rollupCfg.BlockTime,
					rollupCfg.Genesis.L2Time,
					rollupCfg.L2ChainID,
				)
				if err != nil {
					l.Warn("Failed to decode span batch",
						"channelID", channelID.String(),
						"batchNumber", batchCount,
						"error", err)
					continue
				}

				for blockIdx, batchElement := range spanBatch.Batches {
					l.Debug("L2 block in span batch",
						"channelID", channelID.String(),
						"batchNumber", batchCount,
						"blockIndex", blockIdx,
						"epochNum", batchElement.EpochNum,
						"timestamp", batchElement.Timestamp,
						"txCount", len(batchElement.Transactions))

					for _, txData := range batchElement.Transactions {
						var tx ethtypes.Transaction
						n.require.NoError(tx.UnmarshalBinary(txData))

						signer := ethtypes.LatestSignerForChainID(rollupCfg.L2ChainID)
						fromAddr, err := signer.Sender(&tx)
						n.require.NoError(err)

						l2Txs[fromAddr] = append(l2Txs[fromAddr], &tx)
					}
				}
			} else {
				l.Warn("Unknown batch type",
					"channelID", channelID.String(),
					"batchNumber", batchCount,
					"batchType", batchType)
			}
		}

		l.Debug("Finished processing channel",
			"channelID", channelID.String(),
			"totalBatches", batchCount)
	}

	return channels, channelFrames, l2Txs
}
