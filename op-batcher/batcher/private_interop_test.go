package batcher

import (
	"bytes"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	opparams "github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	derivepar "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/params"
)

const (
	piL1Genesis = uint64(1_000_000)
	piL2Genesis = uint64(1_000_006)
	piBlockTime = uint64(2)
	piCadence   = uint64(8)
)

var (
	piKey, _     = crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	piRegistry   = common.HexToAddress("0x00000000000000000000000000000000000e9e9e")
	piReplayer   = common.HexToAddress("0x00000000000000000000000000000000000e0e0e")
	piOtherAddr  = common.HexToAddress("0x00000000000000000000000000000000000f00d1")
	piTerminal   = crypto.Keccak256Hash([]byte("rendering-terminal"))
	piChainIDBig = big.NewInt(901)
)

func piRollupCfg() *rollup.Config {
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L2Time:       piL2Genesis,
			SystemConfig: eth.SystemConfig{GasLimit: 60_000_000},
		},
		BlockTime:         piBlockTime,
		MaxSequencerDrift: 1800,
		SeqWindowSize:     3600,
		L2ChainID:         piChainIDBig,
		ChainOpConfig:     &opparams.OptimismConfig{EIP1559Elasticity: 6},
	}
	cfg.ActivateAtGenesis(forks.Delta)
	return cfg
}

func piL1Chain(n int) []eth.L1BlockRef {
	out := make([]eth.L1BlockRef, 0, n)
	var parent common.Hash
	for i := range n {
		h := crypto.Keccak256Hash([]byte{byte(i), 'l', '1'})
		out = append(out, eth.L1BlockRef{Hash: h, Number: uint64(i), ParentHash: parent, Time: piL1Genesis + uint64(i)*12})
		parent = h
	}
	return out
}

// staticRanges is the mocked RangeSource. The production one dials a node following the rendering;
// nothing here needs more than a fixed answer.
type staticRanges struct {
	start RangeStart
	calls int
}

func (s *staticRanges) RangeStart(context.Context, uint64) (RangeStart, error) {
	s.calls++
	return s.start, nil
}

var _ RangeSource = (*staticRanges)(nil)

// failingClaimTxs is the batcher tx builder with a switchable failure in ClaimTx: the one way a
// test can make renderChannelOut.Close fail after the private object has been encoded, which is how
// the "no claim, no frame" ordering is asserted from the failure side.
type failingClaimTxs struct {
	render.ReplayTxBuilder
	err error
}

func (f *failingClaimTxs) ClaimTx(claim *codec.RangeClaim) (*types.Transaction, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ReplayTxBuilder.ClaimTx(claim)
}

// fixedReceipts is the private EL's receipt source: one export, one import, one private log per
// block, so every block exercises the filter and the interleaving.
type fixedReceipts struct{ calls int }

func (f *fixedReceipts) FetchReceipts(_ context.Context, hash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	f.calls++
	num := new(big.Int).SetBytes(hash[24:]).Uint64()
	logs := []*types.Log{
		{Address: piOtherAddr, Topics: []common.Hash{{0x1}}, Data: []byte{1}},
		piExportLog(num),
		piImportLog(num),
	}
	rec := &types.Receipt{Status: types.ReceiptStatusSuccessful, BlockHash: hash, BlockNumber: new(big.Int).SetUint64(num)}
	for i, l := range logs {
		l.BlockNumber, l.BlockHash, l.Index = num, hash, uint(i)
		rec.Logs = append(rec.Logs, l)
	}
	return nil, optypes.Receipts{optypes.FromGethReceipt(rec)}, nil
}

func piExportLog(nonce uint64) *types.Log {
	msg := []byte{0xde, 0xad, byte(nonce)}
	data := make([]byte, 0, 96+32)
	data = append(data, common.LeftPadBytes(piOtherAddr.Bytes(), 32)...)
	data = append(data, common.BigToHash(big.NewInt(64)).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(len(msg)))).Bytes()...)
	data = append(data, msg...)
	data = append(data, make([]byte, 32-len(msg))...)
	return &types.Log{
		Address: predeploys.L2toL2CrossDomainMessengerAddr,
		Topics: []common.Hash{
			render.SentMessageEventTopic,
			common.BigToHash(big.NewInt(902)),
			common.BytesToHash(piOtherAddr.Bytes()),
			common.BigToHash(new(big.Int).SetUint64(nonce)),
		},
		Data: data,
	}
}

func piImportLog(nonce uint64) *types.Log {
	id := messages.Identifier{
		Origin: predeploys.L2toL2CrossDomainMessengerAddr, BlockNumber: nonce, LogIndex: 1,
		Timestamp: piL2Genesis, ChainID: eth.ChainIDFromUInt64(902),
	}
	data := make([]byte, 0, 32*5)
	data = append(data, common.LeftPadBytes(id.Origin.Bytes(), 32)...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.BlockNumber)).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(id.LogIndex))).Bytes()...)
	data = append(data, common.BigToHash(new(big.Int).SetUint64(id.Timestamp)).Bytes()...)
	chainID := id.ChainID.Bytes32()
	data = append(data, chainID[:]...)
	return &types.Log{
		Address: predeploys.CrossL2InboxAddr,
		Topics:  []common.Hash{messages.ExecutingMessageEventTopic, {0x77}},
		Data:    data,
	}
}

// piPayload is a private execution payload: the L1-attributes deposit plus one ordinary private
// transaction whose logs the fixture supplies. Its transactions never appear on the rendering.
func piPayload(t *testing.T, number uint64) *eth.ExecutionPayload {
	t.Helper()
	// Six 2 s L2 blocks per 12 s L1 block, so the fixture crosses epoch boundaries and carries real
	// sequence numbers — which is what makes the origin-copy assertions mean anything.
	l1 := piL1Chain(40)
	origin := l1[(number-900)/6]
	seq := (number - 900) % 6
	l1Info := testutils.MockBlockInfo{
		InfoHash: origin.Hash, InfoNum: origin.Number, InfoTime: origin.Time,
		InfoBaseFee: big.NewInt(1), InfoRoot: common.Hash{0x9},
	}
	raw, err := derive.L1InfoDepositBytes(piRollupCfg(), params.MergedTestChainConfig,
		eth.SystemConfig{BatcherAddr: common.Address{0x42}}, seq, &l1Info, piL2Genesis+number*piBlockTime)
	require.NoError(t, err)
	// One ordinary private transaction after the deposit: the private chain's own business, which
	// must never appear on the rendering. Its logs are what the fixture's receipts describe.
	priv, err := types.SignNewTx(piKey, types.LatestSignerForChainID(piChainIDBig), &types.DynamicFeeTx{
		ChainID: piChainIDBig, Nonce: number, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2),
		Gas: 100_000, To: &piOtherAddr,
	})
	require.NoError(t, err)
	privRaw, err := priv.MarshalBinary()
	require.NoError(t, err)
	return &eth.ExecutionPayload{
		BlockNumber:  hexutil.Uint64(number),
		Timestamp:    hexutil.Uint64(piL2Genesis + number*piBlockTime),
		BlockHash:    common.BigToHash(new(big.Int).SetUint64(number)),
		ParentHash:   common.BigToHash(new(big.Int).SetUint64(number - 1)),
		Transactions: []eth.Data{raw, privRaw},
	}
}

// piTxs is the batcher transaction builder the seam signs with.
func piTxs() render.ReplayTxBuilder {
	txs := render.NewBatcherTxBuilder(piChainIDBig, render.DefaultGasPolicy(), render.PrivateKeySigner(piKey, piChainIDBig))
	txs.SetEventReplayer(piReplayer)
	txs.SetRegistry(piRegistry)
	return txs
}

func piEncoder(t *testing.T) (*PrivateInteropEncoder, *staticRanges, *fixedReceipts) {
	t.Helper()
	return piEncoderWithTxs(t, piTxs())
}

func piEncoderWithTxs(t *testing.T, txs render.ReplayTxBuilder) (*PrivateInteropEncoder, *staticRanges, *fixedReceipts) {
	t.Helper()
	ranges := &staticRanges{
		start: RangeStart{
			PrevTerminalRenderingHash: piTerminal,
			StartNonce:                5,
		},
	}
	receipts := &fixedReceipts{}
	enc, err := NewPrivateInteropEncoder(PrivateInteropConfig{
		Rollup:            piRollupCfg(),
		PrivateRollup:     piRollupCfg(),
		MaxBlocksPerRange: piCadence,
		MaxRangeBytes:     512 * 1024,
		RollupConfigHash:  common.Hash{0x1b},
		DepSetHash:        common.Hash{0x1c},
		Receipts:          receipts,
		Ranges:            ranges,
		Txs:               txs,
	})
	require.NoError(t, err)
	return enc, ranges, receipts
}

// drive runs the seam exactly as the stock batcher does: enrich each loaded block, add it to the
// channel out, close, and drain frames.
func drive(t *testing.T, enc *PrivateInteropEncoder, first, count uint64) (*renderChannelOut, [][]byte) {
	t.Helper()
	cfg := piRollupCfg()
	out, err := enc.ChannelOut(ChannelConfig{MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib}}, cfg)
	require.NoError(t, err)
	co := out.(*renderChannelOut)

	for i := range count {
		p := piPayload(t, first+i)
		require.NoError(t, enc.PrepareBlock(context.Background(), p))
		_, err := co.AddBlock(cfg, p)
		require.NoError(t, err)
	}
	require.NoError(t, co.Close())

	var frames [][]byte
	for {
		var buf bytes.Buffer
		_, err := co.OutputFrame(&buf, 100_000)
		if buf.Len() > 0 {
			frames = append(frames, buf.Bytes())
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}
	return co, frames
}

// TestPrivateInteropSeamProducesStockFrames is the seam's gate: the terminal stage of the STOCK
// batcher lifecycle turns private blocks into stock-shaped frames with a derived channel ID, and
// does it identically every time.
func TestPrivateInteropSeamProducesStockFrames(t *testing.T) {
	enc, _, receipts := piEncoder(t)
	co, frames := drive(t, enc, 901, piCadence)

	require.Equal(t, int(piCadence), receipts.calls, "one receipt fetch per loaded private block")
	require.NotEmpty(t, frames)
	require.Equal(t, builder.ChannelID(piTerminal, 901), co.ID(),
		"the channel ID is derived from the previous range's terminal rendering hash")

	// Every frame decodes as a stock frame, carries the derived channel ID, and only the last is
	// marked last. This is what "indistinguishable from stock batcher output" has to mean.
	for i, raw := range frames {
		var f derive.Frame
		require.NoError(t, f.UnmarshalBinary(bytes.NewReader(raw)), "frame %d", i)
		require.Equal(t, co.ID(), f.ID)
		require.Equal(t, uint16(i), f.FrameNumber)
		require.Equal(t, i == len(frames)-1, f.IsLast)
	}

	// And the blobs the batcher would post carry the plain 0x00 derivation version: this design has
	// no raw or skipped blob anywhere.
	built := co.BuiltRange()
	require.NotNil(t, built)
	require.Len(t, built.Blobs, len(frames))
	for i, f := range frames {
		var blob eth.Blob
		require.NoError(t, blob.FromData(append([]byte{derivepar.DerivationVersion0}, f...)))
		require.Equal(t, built.Blobs[i][:], blob[:], "blob %d matches what txData.Blobs() would build", i)
	}
}

// TestPrivateInteropSeamIsByteDeterministic re-runs the whole terminal path from fresh state.
func TestPrivateInteropSeamIsByteDeterministic(t *testing.T) {
	run := func() ([][]byte, *builder.BuiltRange) {
		enc, _, _ := piEncoder(t)
		co, frames := drive(t, enc, 901, piCadence)
		return frames, co.BuiltRange()
	}
	firstFrames, firstBuilt := run()
	secondFrames, secondBuilt := run()

	require.Equal(t, firstFrames, secondFrames, "every frame byte-identical across fresh builders")
	require.Equal(t, firstBuilt.ChannelData, secondBuilt.ChannelData)
	require.Equal(t, firstBuilt.Blocks, secondBuilt.Blocks)
	require.Len(t, firstBuilt.Blobs, len(secondBuilt.Blobs))
	for i := range firstBuilt.Blobs {
		require.Equal(t, firstBuilt.Blobs[i][:], secondBuilt.Blobs[i][:])
	}
}

// TestPrivateInteropSeamRendersTheRightContent checks what the range actually says: the private
// transactions are gone, and each block carries one replay transaction per rendered log, with the
// range's claim leading in its first block only.
func TestPrivateInteropSeamRendersTheRightContent(t *testing.T) {
	enc, _, _ := piEncoder(t)
	co, _ := drive(t, enc, 901, piCadence)
	built := co.BuiltRange()

	require.Equal(t, uint64(901), built.FirstBlock)
	require.Equal(t, uint64(900+piCadence), built.LastBlock)
	require.Len(t, built.Blocks, int(piCadence))

	// The opening block: the claim LEADS, then two replay transactions (the private business log is
	// filtered out). The claim emits no log, so replay transaction k still emits rendering log k.
	opening := built.Blocks[0]
	require.Len(t, opening.Txs, 3)
	require.Equal(t, piRegistry, *decodeRenderTx(t, opening.Txs[0]).To())
	require.Equal(t, predeploys.L2toL2CrossDomainMessengerAddr, *decodeRenderTx(t, opening.Txs[1]).To())
	require.Equal(t, predeploys.CrossL2InboxAddr, *decodeRenderTx(t, opening.Txs[2]).To())

	// The claim the seam actually posted describes the range it opened, and commits to the private
	// chain's own terminal hash — which the seam already had, from the payload it loaded.
	require.Equal(t, uint64(901), built.Claim.FirstBlock)
	require.Equal(t, uint64(900+piCadence), built.Claim.LastBlock)
	require.Equal(t, common.BigToHash(new(big.Int).SetUint64(900+piCadence)), built.Claim.PrivateTerminalBlockHash,
		"the private block hash of the range's last block, with no new dependency")

	for _, blk := range built.Blocks[1:] {
		require.Len(t, blk.Txs, 2, "every other block is exactly its replay transactions")
	}
	// Nonces run from the range's start, unbroken, across every block.
	nonce := uint64(5)
	for _, blk := range built.Blocks {
		for _, raw := range blk.Txs {
			require.Equal(t, nonce, decodeRenderTx(t, raw).Nonce())
			nonce++
		}
	}
	require.Equal(t, nonce, built.NextNonce)
}

func TestPrivateInteropSeamStopsAtTheCadence(t *testing.T) {
	enc, _, _ := piEncoder(t)
	cfg := piRollupCfg()
	out, err := enc.ChannelOut(ChannelConfig{MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib}}, cfg)
	require.NoError(t, err)
	co := out.(*renderChannelOut)

	for i := range piCadence {
		p := piPayload(t, 901+i)
		require.NoError(t, enc.PrepareBlock(context.Background(), p))
		_, err := co.AddBlock(cfg, p)
		require.NoError(t, err)
	}
	require.ErrorIs(t, co.FullErr(), derive.ErrCompressorFull,
		"the cadence ends the range; the batcher then closes the channel as it does for a full one")
}

func TestPrivateInteropSeamStopsAtTheRangeByteBudget(t *testing.T) {
	enc, _, _ := piEncoder(t)
	enc.cfg.MaxRangeBytes = 1
	cfg := piRollupCfg()
	out, err := enc.ChannelOut(ChannelConfig{
		MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib},
	}, cfg)
	require.NoError(t, err)
	p := piPayload(t, 901)
	require.NoError(t, enc.PrepareBlock(context.Background(), p))
	_, err = out.AddBlock(cfg, p)
	require.NoError(t, err)
	require.ErrorIs(t, out.FullErr(), derive.ErrCompressorFull)
	require.Greater(t, out.InputBytes(), 1, "the budget counts estimated serialized bytes, not actions")
}

func TestPrivateInteropSeamNeedsReceipts(t *testing.T) {
	enc, _, _ := piEncoder(t)
	cfg := piRollupCfg()
	out, err := enc.ChannelOut(ChannelConfig{MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib}}, cfg)
	require.NoError(t, err)
	// A block that was never enriched must fail loudly rather than render as if it had no logs:
	// silently emitting an empty block would drop every message in it.
	_, err = out.AddBlock(cfg, piPayload(t, 901))
	require.ErrorContains(t, err, "no receipts prepared")
}

// TestPrivateInteropSeamClaimsTheRangesPrivateInput is the commitment gate.
//
// The claim's privateDataHash must be the keccak of the range's private derivation input as this
// very range's blocks encode it. Nothing publishes those bytes — they travel the operator's
// firewalled p2p network — so the hash is the entire binding between a public claim and the private
// range it stands for, and a hash computed from anything but these blocks binds nothing.
func TestPrivateInteropSeamClaimsTheRangesPrivateInput(t *testing.T) {
	enc, _, _ := piEncoder(t)
	co, frames := drive(t, enc, 901, piCadence)
	built := co.BuiltRange()
	require.NotEmpty(t, frames)

	require.Equal(t, builder.PrivateDataHash(piPrivateObject(t, 901, piCadence)), built.Claim.PrivateDataHash,
		"the claim commits to the object these blocks encode to")

	// The claim's remaining operator inputs: configuration from the seam's config.
	require.Equal(t, common.Hash{0x1b}, built.Claim.RollupConfigHash)
	require.Equal(t, common.Hash{0x1c}, built.Claim.DepSetHash)
	require.Empty(t, built.Claim.Proof, "v1 is attested, never proven")

	// ORIGIN-COPY through the whole seam: every rendering block carries the origin and sequence
	// number its private payload's own L1-info deposit declared, and no L1 client was consulted —
	// the RangeSource has no L1View to call any more. l1Head follows from it.
	l1 := piL1Chain(40)
	for i, blk := range built.Blocks {
		num := 901 + uint64(i)
		require.Equal(t, l1[(num-900)/6].ID(), blk.Origin, "block %d copies its private origin", num)
		require.Equal(t, (num-900)%6, blk.SeqNum, "block %d copies its private sequence number", num)
	}
	require.Equal(t, built.Blocks[len(built.Blocks)-1].Origin.Hash, built.Claim.L1Head,
		"l1Head is the terminal block's own origin, readable off the rendering")
	require.Equal(t, common.BigToHash(new(big.Int).SetUint64(900+piCadence-1)), built.Claim.PrivateTerminalParentHash,
		"and the claim carries the terminal parent hash, the one ref field public data cannot supply")
}

// TestPrivateInteropSeamWithoutAClaimHasNothingToSend is the ordering rule from the failure side:
// while the claim cannot be assembled there is no range, and — the part that matters — no frame for
// the batcher to put in a transaction. The ordering is structural, not remembered.
func TestPrivateInteropSeamWithoutAClaimHasNothingToSend(t *testing.T) {
	txs := &failingClaimTxs{ReplayTxBuilder: piTxs(), err: errors.New("the claim registry moved")}
	enc, _, _ := piEncoderWithTxs(t, txs)

	cfg := piRollupCfg()
	out, err := enc.ChannelOut(ChannelConfig{MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib}}, cfg)
	require.NoError(t, err)
	co := out.(*renderChannelOut)
	for i := range piCadence {
		p := piPayload(t, 901+i)
		require.NoError(t, enc.PrepareBlock(context.Background(), p))
		_, err := co.AddBlock(cfg, p)
		require.NoError(t, err)
	}

	require.ErrorContains(t, co.Close(), "the claim registry moved")
	require.Nil(t, co.BuiltRange(), "no range was built")
	require.Zero(t, co.ReadyBytes(), "the batcher is offered nothing to send")
	var buf bytes.Buffer
	_, err = co.OutputFrame(&buf, 100_000)
	require.ErrorIs(t, err, io.EOF)
	require.Zero(t, buf.Len(), "not one byte of a frame exists")

	// The stock retry path calls Close again once the cause is gone, and the range completes with
	// the SAME commitment: the private object was encoded and hashed on the first attempt and is
	// not recomputed, so a failed Close must neither leave the channel closed nor change the hash.
	txs.err = nil
	require.NoError(t, co.Close())
	require.NotNil(t, co.BuiltRange())
	require.Equal(t, builder.PrivateDataHash(piPrivateObject(t, 901, piCadence)), co.BuiltRange().Claim.PrivateDataHash)
}

// piPrivateObject rebuilds a range's private derivation-input object from the payloads the seam
// would have loaded, INDEPENDENTLY of the seam: derive.PayloadToSingularBatch on each payload, then
// builder.EncodePrivateData with the same channel config drive() gives the seam.
//
// The seam hashes this object and drops the bytes, so this is how a test gets at them. Being an
// independent rebuild is the point: a claim whose privateDataHash matches this is a claim committing
// to an object anyone holding the private blocks can reproduce.
func piPrivateObject(t *testing.T, first, count uint64) []byte {
	t.Helper()
	r := &builder.PrivateRange{FirstBlock: first, ParentHash: piPayload(t, first).ParentHash}
	for i := range count {
		p := piPayload(t, first+i)
		sb, l1Info, err := derive.PayloadToSingularBatch(piRollupCfg(), p)
		require.NoError(t, err)
		r.Batches = append(r.Batches, sb)
		r.SeqNums = append(r.SeqNums, l1Info.SequenceNumber)
	}
	data, err := builder.EncodePrivateData(builder.PrivateDataConfig{
		Rollup: piRollupCfg(), MaxFrameSize: 100_000, Compression: derive.Zlib,
	}, r)
	require.NoError(t, err)
	return data
}

// TestPrivateInteropPrivateObjectIsTheStockPrivateChannel pins WHAT the claim commits to: not a
// bespoke archive, but exactly what a stock batcher for the PRIVATE chain would have posted for
// these blocks — derivation version byte, stock frames, one span batch, the private transactions the
// rendering deliberately drops. The object is never published, so this format is consensus-relevant
// only through the hash — which is exactly why it must be pinned: a claim commits to an encoding,
// and an encoding nobody can reproduce is a commitment to nothing.
func TestPrivateInteropPrivateObjectIsTheStockPrivateChannel(t *testing.T) {
	enc, _, _ := piEncoder(t)
	co, _ := drive(t, enc, 901, piCadence)
	data := piPrivateObject(t, 901, piCadence)
	require.Equal(t, builder.PrivateDataHash(data), co.BuiltRange().Claim.PrivateDataHash,
		"the range the seam built commits to this object")

	require.Equal(t, byte(derivepar.DerivationVersion0), data[0], "one derivation version byte, then frames")
	parsed, err := derive.ParseFrames(data)
	require.NoError(t, err)
	require.NotEmpty(t, parsed)
	// The channel ID is derived, not random: the object's NAME is the hash of its bytes, so a
	// random channel ID would give the same range a new content address on every build. It is
	// seeded with the PRIVATE chain's parent hash, read off the range's first payload.
	wantID := builder.ChannelID(piPayload(t, 901).ParentHash, 901)
	var channel []byte
	for i, f := range parsed {
		require.Equal(t, wantID, f.ID)
		require.Equal(t, uint16(i), f.FrameNumber)
		require.Equal(t, i == len(parsed)-1, f.IsLast)
		channel = append(channel, f.Data...)
	}
	require.NotEqual(t, co.ID(), wantID, "the private object and the rendering are different channels")

	zr, err := zlib.NewReader(bytes.NewReader(channel))
	require.NoError(t, err)
	raw, err := io.ReadAll(zr)
	require.NoError(t, err)
	var batch derive.BatchData
	require.NoError(t, rlp.DecodeBytes(raw, &batch))
	require.Equal(t, uint8(derive.SpanBatchType), batch.GetBatchType(), "one span batch per range")

	// And it is byte-deterministic: the same private range built twice has the same content
	// address, which is the property the claim's commitment rests on.
	enc2, _, _ := piEncoder(t)
	co2, _ := drive(t, enc2, 901, piCadence)
	require.Equal(t, co.BuiltRange().Claim.PrivateDataHash, co2.BuiltRange().Claim.PrivateDataHash)
	require.Equal(t, data, piPrivateObject(t, 901, piCadence))
}

func decodeRenderTx(t *testing.T, raw hexutil.Bytes) *types.Transaction {
	t.Helper()
	var tx types.Transaction
	require.NoError(t, tx.UnmarshalBinary(raw))
	return &tx
}
