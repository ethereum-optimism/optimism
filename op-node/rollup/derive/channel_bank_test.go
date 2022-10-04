package derive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-node/testutils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type fakeChannelBankInput struct {
	origin eth.L1BlockRef
	data   []struct {
		data []byte
		err  error
	}
}

func (f *fakeChannelBankInput) Origin() eth.L1BlockRef {
	return f.origin
}

func (f *fakeChannelBankInput) NextData(_ context.Context) ([]byte, error) {
	out := f.data[0]
	f.data = f.data[1:]
	return out.data, out.err
}

func (f *fakeChannelBankInput) AddOutput(data []byte, err error) {
	f.data = append(f.data, struct {
		data []byte
		err  error
	}{data: data, err: err})
}

// ExpectNextFrameData takes a set of test frame & turns into the raw data
// for reading into the channel bank via `NextData`
func (f *fakeChannelBankInput) AddFrames(frames ...testFrame) {
	data := new(bytes.Buffer)
	data.WriteByte(DerivationVersion0)
	for _, frame := range frames {
		ff := frame.ToFrame()
		if err := ff.MarshalBinary(data); err != nil {
			panic(fmt.Errorf("error in making frame during test: %w", err))
		}
	}
	f.AddOutput(data.Bytes(), nil)
}

var _ NextFrameProvider = (*fakeChannelBankInput)(nil)

// format: <channelID-data>:<frame-number>:<content><optional-last-frame-marker "!">
// example: "abc:0:helloworld!"
type testFrame string

func (tf testFrame) ChannelID() ChannelID {
	parts := strings.Split(string(tf), ":")
	var chID ChannelID
	copy(chID[:], parts[0])
	return chID
}

func (tf testFrame) FrameNumber() uint64 {
	parts := strings.Split(string(tf), ":")
	frameNum, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		panic(err)
	}
	return frameNum
}

func (tf testFrame) IsLast() bool {
	parts := strings.Split(string(tf), ":")
	return strings.HasSuffix(parts[2], "!")
}

func (tf testFrame) Content() []byte {
	parts := strings.Split(string(tf), ":")
	return []byte(strings.TrimSuffix(parts[2], "!"))
}

func (tf testFrame) ToFrame() Frame {
	return Frame{
		ID:          tf.ChannelID(),
		FrameNumber: uint16(tf.FrameNumber()),
		Data:        tf.Content(),
		IsLast:      tf.IsLast(),
	}
}

func TestChannelBankSimple(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)

	input := &fakeChannelBankInput{origin: a}
	input.AddFrames("a:0:first", "a:2:third!")
	input.AddFrames("a:1:second")
	input.AddOutput(nil, io.EOF)

	cfg := &rollup.Config{ChannelTimeout: 10}

	cb := NewChannelBank(testlog.Logger(t, log.LvlCrit), cfg, input, nil)

	// Load the first + third frame
	out, err := cb.NextData(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Equal(t, []byte(nil), out)

	// Load the second frame
	out, err = cb.NextData(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Equal(t, []byte(nil), out)

	// Pull out the channel data
	out, err = cb.NextData(context.Background())
	require.Nil(t, err)
	require.Equal(t, "firstsecondthird", string(out))

	// No more data
	out, err = cb.NextData(context.Background())
	require.Nil(t, out)
	require.Equal(t, io.EOF, err)
}

func TestChannelBankDuplicates(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	a := testutils.RandomBlockRef(rng)

	input := &fakeChannelBankInput{origin: a}
	input.AddFrames("a:0:first", "a:2:third!")
	input.AddFrames("a:0:altfirst", "a:2:altthird!")
	input.AddFrames("a:1:second")
	input.AddOutput(nil, io.EOF)

	cfg := &rollup.Config{ChannelTimeout: 10}

	cb := NewChannelBank(testlog.Logger(t, log.LvlCrit), cfg, input, nil)

	// Load the first + third frame
	out, err := cb.NextData(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Equal(t, []byte(nil), out)

	// Load the duplicate frames
	out, err = cb.NextData(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Equal(t, []byte(nil), out)

	// Load the second frame
	out, err = cb.NextData(context.Background())
	require.ErrorIs(t, err, NotEnoughData)
	require.Equal(t, []byte(nil), out)

	// Pull out the channel data. Expect to see the original set & not the duplicates
	out, err = cb.NextData(context.Background())
	require.Nil(t, err)
	require.Equal(t, "firstsecondthird", string(out))

	// No more data
	out, err = cb.NextData(context.Background())
	require.Nil(t, out)
	require.Equal(t, io.EOF, err)
}
