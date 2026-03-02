package derive

import (
	"bytes"
	"compress/zlib"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	opderive "github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func testRollupConfig() *rollup.Config {
	zero := uint64(0)
	return &rollup.Config{
		Genesis: rollup.Genesis{
			L1: eth.BlockID{
				Hash:   common.HexToHash("0x01"),
				Number: 0,
			},
			L2: eth.BlockID{
				Hash:   common.HexToHash("0x02"),
				Number: 0,
			},
			L2Time:       0,
			SystemConfig: testSystemConfig(),
		},
		BlockTime:             2,
		MaxSequencerDrift:     600,
		SeqWindowSize:         10,
		ChannelTimeoutBedrock: 50,
		L1ChainID:             big.NewInt(1),
		L2ChainID:             big.NewInt(10),
		// Activate all forks at genesis for post-Karst only pipeline
		RegolithTime:           &zero,
		CanyonTime:             &zero,
		DeltaTime:              &zero,
		EcotoneTime:            &zero,
		FjordTime:              &zero,
		GraniteTime:            &zero,
		HoloceneTime:           &zero,
		JovianTime:             &zero,
		KarstTime:              &zero,
		BatchInboxAddress:      common.HexToAddress("0xff00000000000000000000000000000000000010"),
		DepositContractAddress: common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
	}
}

func testSystemConfig() eth.SystemConfig {
	return eth.SystemConfig{
		BatcherAddr: common.HexToAddress("0xba7c4e500000000000000000000000000000ba7c"),
		GasLimit:    30_000_000,
	}
}

func testL1ChainConfig() *gethparams.ChainConfig {
	return gethparams.AllDevChainProtocolChanges
}

func testSafeHead(cfg *rollup.Config) eth.L2BlockRef {
	return eth.L2BlockRef{
		Hash:       cfg.Genesis.L2.Hash,
		Number:     cfg.Genesis.L2.Number,
		ParentHash: common.Hash{},
		Time:       cfg.Genesis.L2Time,
		L1Origin:   cfg.Genesis.L1,
	}
}

func makeTestL1Input(num uint64) *L1Input {
	return &L1Input{
		Header: &types.Header{
			ParentHash: common.BigToHash(new(big.Int).SetUint64(num + 0x100 - 1)),
			Number:     new(big.Int).SetUint64(num),
			Time:       num * 2, // match L2 block time for simple epoch advancement in tests
			BaseFee:    big.NewInt(7),
			MixDigest:  common.BigToHash(new(big.Int).SetUint64(num + 0x200)),
			// ExcessBlobGas required for BlobBaseFee to work via HeaderBlockInfo
			ExcessBlobGas: ptrTo(uint64(0)),
		},
	}
}

func makeTestDeposit() *types.DepositTx {
	return &types.DepositTx{
		SourceHash: common.HexToHash("0xdead"),
		From:       common.HexToAddress("0x1111"),
		To:         ptrTo(common.HexToAddress("0x2222")),
		Value:      big.NewInt(0),
		Gas:        100_000,
		Data:       nil,
	}
}

func ptrTo[T any](v T) *T {
	return &v
}

func testL1Ref(num uint64) eth.L1BlockRef {
	input := makeTestL1Input(num)
	return input.BlockRef()
}

// makeTestL1Chain creates a sequence of properly chained L1 inputs where each
// block's ParentHash matches the previous block's actual hash. This is required
// for AddL1Block's reorg detection.
func makeTestL1Chain(count uint64) []*L1Input {
	chain := make([]*L1Input, count)
	for i := uint64(0); i < count; i++ {
		input := makeTestL1Input(i)
		if i > 0 {
			input.Header.ParentHash = chain[i-1].Header.Hash()
		}
		chain[i] = input
	}
	return chain
}

// encodeBatchToChannelData RLP-encodes a singular batch and zlib-compresses
// it into channel data (the format read by the channel reader stage).
func encodeBatchToChannelData(t *testing.T, batch *opderive.SingularBatch) []byte {
	t.Helper()

	bd := opderive.NewBatchData(batch)
	batchBytes, err := bd.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}

	// Wrap in RLP string encoding as the channel reader expects RLP-encoded batch data
	var rlpBuf bytes.Buffer
	if err := rlp.Encode(&rlpBuf, batchBytes); err != nil {
		t.Fatalf("rlp encode batch: %v", err)
	}

	// zlib compress
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	if _, err := w.Write(rlpBuf.Bytes()); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}

	return compressed.Bytes()
}

// wrapInFrames wraps channel data in a single-frame batcher transaction.
// The result is a raw batcher tx data payload (DerivationVersion0 prefix + frame).
func wrapInFrames(channelData []byte, channelID opderive.ChannelID) []byte {
	frame := opderive.Frame{
		ID:          channelID,
		FrameNumber: 0,
		Data:        channelData,
		IsLast:      true,
	}

	var buf bytes.Buffer
	buf.WriteByte(params.DerivationVersion0)
	_ = frame.MarshalBinary(&buf)
	return buf.Bytes()
}

func TestHelpers(t *testing.T) {
	cfg := testRollupConfig()
	require.NotNil(t, cfg)
	require.Equal(t, uint64(2), cfg.BlockTime)
	require.Equal(t, uint64(10), cfg.SeqWindowSize)
	require.Equal(t, uint64(50), cfg.ChannelTimeoutBedrock)
	require.NotNil(t, cfg.HoloceneTime)
	require.NotNil(t, cfg.KarstTime)

	sysCfg := testSystemConfig()
	require.Equal(t, uint64(30_000_000), sysCfg.GasLimit)

	safeHead := testSafeHead(cfg)
	require.Equal(t, cfg.Genesis.L2.Hash, safeHead.Hash)
	require.Equal(t, cfg.Genesis.L2.Number, safeHead.Number)

	l1 := makeTestL1Input(5)
	require.Equal(t, uint64(5), bigs.Uint64Strict(l1.Header.Number))
	require.Equal(t, uint64(5*2), l1.Header.Time)

	dep := makeTestDeposit()
	require.NotNil(t, dep)
	require.NotNil(t, dep.To)

	ref := testL1Ref(10)
	require.Equal(t, uint64(10), ref.Number)

	l1WithBatch := makeL1WithBatch(t, cfg, 1, safeHead, sysCfg)
	require.Len(t, l1WithBatch.BatcherData, 1)
	require.NotEmpty(t, l1WithBatch.BatcherData[0])

	// Verify the batcher tx can be parsed as frames
	frames, err := opderive.ParseFrames(l1WithBatch.BatcherData[0])
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.True(t, frames[0].IsLast)
}

// makeL1WithBatch creates an L1Input containing a batcher tx with one singular batch
// targeting the given safe head as parent.
func makeL1WithBatch(t *testing.T, cfg *rollup.Config, l1Num uint64, safeHead eth.L2BlockRef, sysCfg eth.SystemConfig) *L1Input {
	t.Helper()
	_ = sysCfg // reserved for future use in batch construction

	l1 := makeTestL1Input(l1Num)
	l1Ref := l1.BlockRef()

	batch := &opderive.SingularBatch{
		ParentHash: safeHead.Hash,
		EpochNum:   rollup.Epoch(l1Ref.Number),
		EpochHash:  l1Ref.Hash,
		Timestamp:  safeHead.Time + cfg.BlockTime,
	}

	channelData := encodeBatchToChannelData(t, batch)

	var chID opderive.ChannelID
	copy(chID[:], common.Hex2Bytes("deadbeefdeadbeefdeadbeefdeadbeef"))
	batcherTx := wrapInFrames(channelData, chID)

	l1.BatcherData = [][]byte{batcherTx}
	return l1
}
