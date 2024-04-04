package derive

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

var (
	rollupCfg rollup.Config
)

// basic implementation of the Compressor interface that does no compression
type nonCompressor struct {
	bytes.Buffer
}

func (s *nonCompressor) Flush() error {
	return nil
}

func (s *nonCompressor) Close() error {
	return nil
}

func (s *nonCompressor) FullErr() error {
	return nil
}

// channelTypes allows tests to run against different channel types
var channelTypes = []struct {
	ChannelOut func(t *testing.T) ChannelOut
	Name       string
}{
	{
		Name: "Singular",
		ChannelOut: func(t *testing.T) ChannelOut {
			cout, err := NewSingularChannelOut(&nonCompressor{})
			require.NoError(t, err)
			return cout
		},
	},
	{
		Name: "Span",
		ChannelOut: func(t *testing.T) ChannelOut {
			cout, err := NewSpanChannelOut(0, big.NewInt(0), 128_000)
			require.NoError(t, err)
			return cout
		},
	},
}

func TestChannelOutAddBlock(t *testing.T) {
	for _, tcase := range channelTypes {
		t.Run(fmt.Sprintf("%s - returns err if first tx is not an l1info tx", tcase.Name), func(t *testing.T) {
			cout := tcase.ChannelOut(t)
			header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
			block := types.NewBlockWithHeader(header).WithBody(
				[]*types.Transaction{
					types.NewTx(&types.DynamicFeeTx{}),
				},
				nil,
			)
			_, err := cout.AddBlock(&rollupCfg, block)
			require.Error(t, err)
			require.Equal(t, ErrNotDepositTx, err)
		})
	}
}

// TestOutputFrameSmallMaxSize tests that calling [OutputFrame] with a small
// max size that is below the fixed frame size overhead of 23, will return
// an error.
func TestOutputFrameSmallMaxSize(t *testing.T) {
	for _, tcase := range channelTypes {
		t.Run(fmt.Sprintf("%s - returns ErrMaxFrameSizeTooSmall", tcase.Name), func(t *testing.T) {
			cout := tcase.ChannelOut(t)
			// Call OutputFrame with the range of small max size values that err
			var w bytes.Buffer
			for i := 0; i < 23; i++ {
				fid, err := cout.OutputFrame(&w, uint64(i))
				require.ErrorIs(t, err, ErrMaxFrameSizeTooSmall)
				require.Zero(t, fid)
			}
		})
	}
}

func TestOutputFrameNoEmptyLastFrame(t *testing.T) {
	for _, tcase := range channelTypes {
		t.Run(fmt.Sprintf("%s - no empty last frame", tcase.Name), func(t *testing.T) {
			cout := tcase.ChannelOut(t)

			rng := rand.New(rand.NewSource(0x543331))
			chainID := big.NewInt(0)
			txCount := 1
			singularBatch := RandomSingularBatch(rng, txCount, chainID)

			written, err := cout.AddSingularBatch(singularBatch, 0)
			require.NoError(t, err)

			require.NoError(t, cout.Close())
			// span batches return the length of the RLP structure, not the compressed length
			// so we need to collect the compressed length from the span batch
			// no production code relies on the written value from either type of batch
			// (span batches don't always compress their data, and can't always determine the compressed length for this reason)
			if span, ok := cout.(*SpanChannelOut); ok {
				written = uint64(span.compressed.Len())
			}

			var buf bytes.Buffer
			// Output a frame which needs exactly `written` bytes. This frame is expected to be the last frame.
			_, err = cout.OutputFrame(&buf, written+FrameV0OverHeadSize)
			require.ErrorIs(t, err, io.EOF)
		})
	}
}

// TestRLPByteLimit ensures that stream encoder is properly limiting the length.
// It will decode the input if `len(input) <= inputLimit`.
func TestRLPByteLimit(t *testing.T) {
	// Should succeed if `len(input) == inputLimit`
	enc := []byte("\x8bhello world") // RLP encoding of the string "hello world"
	in := bytes.NewBuffer(enc)
	var out string
	stream := rlp.NewStream(in, 12)
	err := stream.Decode(&out)
	require.Nil(t, err)
	require.Equal(t, out, "hello world")

	// Should fail if the `inputLimit = len(input) - 1`
	enc = []byte("\x8bhello world") // RLP encoding of the string "hello world"
	in = bytes.NewBuffer(enc)
	var out2 string
	stream = rlp.NewStream(in, 11)
	err = stream.Decode(&out2)
	require.Equal(t, err, rlp.ErrValueTooLarge)
	require.Equal(t, out2, "")
}

func TestForceCloseTxData(t *testing.T) {
	id := [16]byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef, 0xde, 0xad, 0xbe, 0xef}
	tests := []struct {
		frames []Frame
		errors bool
		output string
	}{
		{
			frames: []Frame{},
			errors: true,
			output: "",
		},
		{
			frames: []Frame{{FrameNumber: 0, IsLast: false}, {ID: id, FrameNumber: 1, IsLast: true}},
			errors: true,
			output: "",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 0, IsLast: false}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000001",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 0, IsLast: true}},
			errors: false,
			output: "00",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 1, IsLast: false}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000001",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 1, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 2, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00010000000000",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 1, IsLast: false}, {ID: id, FrameNumber: 3, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00020000000000",
		},
		{
			frames: []Frame{{ID: id, FrameNumber: 1, IsLast: false}, {ID: id, FrameNumber: 3, IsLast: true}, {ID: id, FrameNumber: 5, IsLast: true}},
			errors: false,
			output: "00deadbeefdeadbeefdeadbeefdeadbeef00000000000000deadbeefdeadbeefdeadbeefdeadbeef00020000000000",
		},
	}

	for i, test := range tests {
		out, err := ForceCloseTxData(test.frames)
		if test.errors {
			require.NotNil(t, err, "Should error on tc %v", i)
			require.Nil(t, out, "Should return no value in tc %v", i)
		} else {
			require.NoError(t, err, "Should not error on tc %v", i)
			require.Equal(t, common.FromHex(test.output), out, "Should match output tc %v", i)
		}
	}
}

func TestBlockToBatchValidity(t *testing.T) {
	block := new(types.Block)
	_, _, err := BlockToSingularBatch(&rollupCfg, block)
	require.ErrorContains(t, err, "has no transactions")
}

// TestSpanChannelOutCompressionOnlyOneBatch tests that the SpanChannelOut compression works as expected when there is only one batch
// and it is larger than the target size. The single batch should be compressed, and the channel should now be full
func TestSpanChannelOutCompressionOnlyOneBatch(t *testing.T) {
	// target is larger than one batch, but smaller than two batches
	target := uint64(300)
	rng := rand.New(rand.NewSource(0x543331))
	chainID := big.NewInt(rng.Int63n(1000))
	txCount := 1
	cout, err := NewSpanChannelOut(0, chainID, target)
	require.NoError(t, err)

	compressedLens := []int{}

	// adding the first batch should not cause an error
	singularBatch := RandomSingularBatch(rng, txCount, chainID)
	_, err = cout.AddSingularBatch(singularBatch, 0)
	compressedLens = append(compressedLens, cout.compressed.Len())
	require.NoError(t, err)

	// confirm compression was not skipped
	require.Greater(t, compressedLens[0], 0)
	// confirm the channel is full
	require.ErrorIs(t, cout.FullErr(), ErrCompressorFull)
}

// TestSpanChannelOutCompressionUndo tests that the SpanChannelOut compression works as expected to not accept a second batch
// if that batch would cause the channel to exceed the target size. The first batch should be compressed only
// the channel should be compressed, full, and should not have the second batch
func TestSpanChannelOutCompressionUndo(t *testing.T) {
	// target is larger than one batch, but smaller than two batches
	target := uint64(750)
	rng := rand.New(rand.NewSource(0x543331))
	chainID := big.NewInt(rng.Int63n(1000))
	txCount := 1
	cout, err := NewSpanChannelOut(0, chainID, target)
	require.NoError(t, err)

	compressedLens := []int{}
	rlpLens := []int{}

	// adding the first batch should not cause an error
	singularBatch := RandomSingularBatch(rng, txCount, chainID)
	_, err = cout.AddSingularBatch(singularBatch, 0)
	compressedLens = append(compressedLens, cout.compressed.Len())
	rlpLens = append(rlpLens, cout.activeRLP().Len())
	require.NoError(t, err)

	// adding the second batch should cause an error
	singularBatch = RandomSingularBatch(rng, txCount, chainID)
	_, err = cout.AddSingularBatch(singularBatch, 0)
	compressedLens = append(compressedLens, cout.compressed.Len())
	rlpLens = append(rlpLens, cout.activeRLP().Len())
	require.ErrorIs(t, err, ErrCompressorFull)

	// confirm that the first compression was skipped
	require.Equal(t, 0, compressedLens[0])
	// confirm that the second compression was not skipped
	require.Greater(t, compressedLens[1], 0)
	// confirm that the second rlp is tht same size as the first (because the second batch was not added)
	require.Equal(t, rlpLens[0], rlpLens[1])
}

// TestSpanChannelOutClose tests that the SpanChannelOut compression works as expected when the channel is closed.
// it should compress the batch even if it is smaller than the target size because the channel is closing
func TestSpanChannelOutClose(t *testing.T) {
	// target is larger than one batch
	target := uint64(600)
	rng := rand.New(rand.NewSource(0x543331))
	chainID := big.NewInt(rng.Int63n(1000))
	txCount := 1
	cout, err := NewSpanChannelOut(0, chainID, target)
	require.NoError(t, err)

	singularBatch := RandomSingularBatch(rng, txCount, chainID)
	_, err = cout.AddSingularBatch(singularBatch, 0)
	require.NoError(t, err)

	rlpLen := cout.activeRLP().Len()

	// confirm no compression has happened yet
	require.Equal(t, 0, cout.compressed.Len())
	require.Less(t, uint64(rlpLen), target)

	// close the channel
	require.NoError(t, cout.Close())

	// confirm that the only batch was compressed, and that the RLP did not change
	require.Greater(t, cout.compressed.Len(), 0)
	require.Equal(t, rlpLen, cout.activeRLP().Len())
}
