package actions

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type SyncStatusAPI interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

type BlocksAPI interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

type L1TxAPI interface {
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
}

type BatcherCfg struct {
	// Limit the size of txs
	MinL1TxSize uint64
	MaxL1TxSize uint64

	BatcherKey *ecdsa.PrivateKey

	GarbageCfg *GarbageChannelCfg
}

// L2Batcher buffers and submits L2 batches to L1.
//
// TODO: note the batcher shares little logic/state with actual op-batcher,
// tests should only use this actor to build batch contents for rollup node actors to consume,
// until the op-batcher is refactored and can be covered better.
type L2Batcher struct {
	log log.Logger

	rollupCfg *rollup.Config

	syncStatusAPI SyncStatusAPI
	l2            BlocksAPI
	l1            L1TxAPI

	l1Signer types.Signer

	l2ChannelOut     ChannelOutIface
	l2Submitting     bool // when the channel out is being submitted, and not safe to write to without resetting
	l2BufferedBlock  eth.BlockID
	l2SubmittedBlock eth.BlockID
	l2BatcherCfg     *BatcherCfg
	batcherAddr      common.Address
}

func NewL2Batcher(log log.Logger, rollupCfg *rollup.Config, batcherCfg *BatcherCfg, api SyncStatusAPI, l1 L1TxAPI, l2 BlocksAPI) *L2Batcher {
	return &L2Batcher{
		log:           log,
		rollupCfg:     rollupCfg,
		syncStatusAPI: api,
		l1:            l1,
		l2:            l2,
		l2BatcherCfg:  batcherCfg,
		l1Signer:      types.LatestSignerForChainID(rollupCfg.L1ChainID),
		batcherAddr:   crypto.PubkeyToAddress(batcherCfg.BatcherKey.PublicKey),
	}
}

// SubmittingData indicates if the actor is submitting buffer data.
// All data must be submitted before it can safely continue buffering more L2 blocks.
func (s *L2Batcher) SubmittingData() bool {
	return s.l2Submitting
}

// ActL2BatchBuffer adds the next L2 block to the batch buffer.
// If the buffer is being submitted, the buffer is wiped.
func (s *L2Batcher) ActL2BatchBuffer(t Testing) {
	require.NoError(t, s.Buffer(t), "failed to add block to channel")
}

func (s *L2Batcher) Buffer(t Testing) error {
	if s.l2Submitting { // break ongoing submitting work if necessary
		s.l2ChannelOut = nil
		s.l2Submitting = false
	}
	syncStatus, err := s.syncStatusAPI.SyncStatus(t.Ctx())
	require.NoError(t, err, "no sync status error")
	// If we just started, start at safe-head
	if s.l2SubmittedBlock == (eth.BlockID{}) {
		s.log.Info("Starting batch-submitter work at safe-head", "safe", syncStatus.SafeL2)
		s.l2SubmittedBlock = syncStatus.SafeL2.ID()
		s.l2BufferedBlock = syncStatus.SafeL2.ID()
		s.l2ChannelOut = nil
	}
	// If it's lagging behind, catch it up.
	if s.l2SubmittedBlock.Number < syncStatus.SafeL2.Number {
		s.log.Warn("last submitted block lagged behind L2 safe head: batch submission will continue from the safe head now", "last", s.l2SubmittedBlock, "safe", syncStatus.SafeL2)
		s.l2SubmittedBlock = syncStatus.SafeL2.ID()
		s.l2BufferedBlock = syncStatus.SafeL2.ID()
		s.l2ChannelOut = nil
	}
	// Add the next unsafe block to the channel
	if s.l2BufferedBlock.Number >= syncStatus.UnsafeL2.Number {
		if s.l2BufferedBlock.Number > syncStatus.UnsafeL2.Number || s.l2BufferedBlock.Hash != syncStatus.UnsafeL2.Hash {
			s.log.Error("detected a reorg in L2 chain vs previous buffered information, resetting to safe head now", "safe_head", syncStatus.SafeL2)
			s.l2SubmittedBlock = syncStatus.SafeL2.ID()
			s.l2BufferedBlock = syncStatus.SafeL2.ID()
			s.l2ChannelOut = nil
		} else {
			s.log.Info("nothing left to submit")
			return nil
		}
	}
	// Create channel if we don't have one yet
	if s.l2ChannelOut == nil {
		var ch ChannelOutIface
		if s.l2BatcherCfg.GarbageCfg != nil {
			ch, err = NewGarbageChannelOut(s.l2BatcherCfg.GarbageCfg)
		} else {
			c, e := compressor.NewRatioCompressor(compressor.Config{
				TargetFrameSize:  s.l2BatcherCfg.MaxL1TxSize,
				TargetNumFrames:  1,
				ApproxComprRatio: 1,
			})
			require.NoError(t, e, "failed to create compressor")
			ch, err = derive.NewChannelOut(c)
		}
		require.NoError(t, err, "failed to create channel")
		s.l2ChannelOut = ch
	}
	block, err := s.l2.BlockByNumber(t.Ctx(), big.NewInt(int64(s.l2BufferedBlock.Number+1)))
	require.NoError(t, err, "need l2 block %d from sync status", s.l2SubmittedBlock.Number+1)
	if block.ParentHash() != s.l2BufferedBlock.Hash {
		s.log.Error("detected a reorg in L2 chain vs previous submitted information, resetting to safe head now", "safe_head", syncStatus.SafeL2)
		s.l2SubmittedBlock = syncStatus.SafeL2.ID()
		s.l2BufferedBlock = syncStatus.SafeL2.ID()
		s.l2ChannelOut = nil
	}
	if _, err := s.l2ChannelOut.AddBlock(block); err != nil { // should always succeed
		return err
	}
	s.l2BufferedBlock = eth.ToBlockID(block)
	return nil
}

func (s *L2Batcher) ActL2ChannelClose(t Testing) {
	// Don't run this action if there's no data to submit
	if s.l2ChannelOut == nil {
		t.InvalidAction("need to buffer data first, cannot batch submit with empty buffer")
		return
	}
	require.NoError(t, s.l2ChannelOut.Close(), "must close channel before submitting it")
}

// ActL2BatchSubmit constructs a batch tx from previous buffered L2 blocks, and submits it to L1
func (s *L2Batcher) ActL2BatchSubmit(t Testing, txOpts ...func(tx *types.DynamicFeeTx)) {
	// Don't run this action if there's no data to submit
	if s.l2ChannelOut == nil {
		t.InvalidAction("need to buffer data first, cannot batch submit with empty buffer")
		return
	}
	// Collect the output frame
	data := new(bytes.Buffer)
	data.WriteByte(derive.DerivationVersion0)
	// subtract one, to account for the version byte
	if _, err := s.l2ChannelOut.OutputFrame(data, s.l2BatcherCfg.MaxL1TxSize-1); err == io.EOF {
		s.l2ChannelOut = nil
		s.l2Submitting = false
	} else if err != nil {
		s.l2Submitting = false
		t.Fatalf("failed to output channel data to frame: %v", err)
	}

	nonce, err := s.l1.PendingNonceAt(t.Ctx(), s.batcherAddr)
	require.NoError(t, err, "need batcher nonce")

	gasTipCap := big.NewInt(2 * params.GWei)
	pendingHeader, err := s.l1.HeaderByNumber(t.Ctx(), big.NewInt(-1))
	require.NoError(t, err, "need l1 pending header for gas price estimation")
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))

	rawTx := &types.DynamicFeeTx{
		ChainID:   s.rollupCfg.L1ChainID,
		Nonce:     nonce,
		To:        &s.rollupCfg.BatchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      data.Bytes(),
	}
	for _, opt := range txOpts {
		opt(rawTx)
	}
	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true, false)
	require.NoError(t, err, "need to compute intrinsic gas")
	rawTx.Gas = gas

	tx, err := types.SignNewTx(s.l2BatcherCfg.BatcherKey, s.l1Signer, rawTx)
	require.NoError(t, err, "need to sign tx")

	err = s.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "need to send tx")
}

// ActL2BatchSubmitGarbage constructs a malformed channel frame and submits it to the
// batch inbox. This *should* cause the batch inbox to reject the blocks
// encoded within the frame, even if the blocks themselves are valid.
func (s *L2Batcher) ActL2BatchSubmitGarbage(t Testing, kind GarbageKind) {
	// Don't run this action if there's no data to submit
	if s.l2ChannelOut == nil {
		t.InvalidAction("need to buffer data first, cannot batch submit with empty buffer")
		return
	}

	// Collect the output frame
	data := new(bytes.Buffer)
	data.WriteByte(derive.DerivationVersion0)

	// subtract one, to account for the version byte
	if _, err := s.l2ChannelOut.OutputFrame(data, s.l2BatcherCfg.MaxL1TxSize-1); err == io.EOF {
		s.l2ChannelOut = nil
		s.l2Submitting = false
	} else if err != nil {
		s.l2Submitting = false
		t.Fatalf("failed to output channel data to frame: %v", err)
	}

	outputFrame := data.Bytes()

	// Malform the output frame
	switch kind {
	// Strip the derivation version byte from the output frame
	case STRIP_VERSION:
		outputFrame = outputFrame[1:]
	// Replace the output frame with random bytes of length [1, 512]
	case RANDOM:
		i, err := rand.Int(rand.Reader, big.NewInt(512))
		require.NoError(t, err, "error generating random bytes length")
		buf := make([]byte, i.Int64()+1)
		_, err = rand.Read(buf)
		require.NoError(t, err, "error generating random bytes")
		outputFrame = buf
	// Remove 4 bytes from the tail end of the output frame
	case TRUNCATE_END:
		outputFrame = outputFrame[:len(outputFrame)-4]
	// Append 4 garbage bytes to the end of the output frame
	case DIRTY_APPEND:
		outputFrame = append(outputFrame, []byte{0xBA, 0xD0, 0xC0, 0xDE}...)
	case INVALID_COMPRESSION:
		// Do nothing post frame encoding- the `GarbageChannelOut` used for this case is modified to
		// use gzip compression rather than zlib, which is invalid.
		break
	case MALFORM_RLP:
		// Do nothing post frame encoding- the `GarbageChannelOut` used for this case is modified to
		// write malformed RLP each time a block is added to the channel.
		break
	default:
		t.Fatalf("Unexpected garbage kind: %v", kind)
	}

	nonce, err := s.l1.PendingNonceAt(t.Ctx(), s.batcherAddr)
	require.NoError(t, err, "need batcher nonce")

	gasTipCap := big.NewInt(2 * params.GWei)
	pendingHeader, err := s.l1.HeaderByNumber(t.Ctx(), big.NewInt(-1))
	require.NoError(t, err, "need l1 pending header for gas price estimation")
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(pendingHeader.BaseFee, big.NewInt(2)))

	rawTx := &types.DynamicFeeTx{
		ChainID:   s.rollupCfg.L1ChainID,
		Nonce:     nonce,
		To:        &s.rollupCfg.BatchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      outputFrame,
	}
	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true, false)
	require.NoError(t, err, "need to compute intrinsic gas")
	rawTx.Gas = gas

	tx, err := types.SignNewTx(s.l2BatcherCfg.BatcherKey, s.l1Signer, rawTx)
	require.NoError(t, err, "need to sign tx")

	err = s.l1.SendTransaction(t.Ctx(), tx)
	require.NoError(t, err, "need to send tx")
}

func (s *L2Batcher) ActBufferAll(t Testing) {
	stat, err := s.syncStatusAPI.SyncStatus(t.Ctx())
	require.NoError(t, err)
	for s.l2BufferedBlock.Number < stat.UnsafeL2.Number {
		s.ActL2BatchBuffer(t)
	}
}

func (s *L2Batcher) ActSubmitAll(t Testing) {
	s.ActBufferAll(t)
	s.ActL2ChannelClose(t)
	s.ActL2BatchSubmit(t)
}
