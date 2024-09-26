package derive

import (
	"bytes"
	"io"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive/mocks"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPruneFrameQueue(t *testing.T) {
	for _, tt := range []struct {
		desc string
		fs   []testFrame
		exp  []testFrame
	}{
		{
			desc: "empty",
			fs:   []testFrame{},
			exp:  []testFrame{},
		},
		{
			desc: "one",
			fs:   []testFrame{"a:2:"},
			exp:  []testFrame{"a:2:"},
		},
		{
			desc: "one-last",
			fs:   []testFrame{"a:2:!"},
			exp:  []testFrame{"a:2:!"},
		},
		{
			desc: "last-new",
			fs:   []testFrame{"a:2:!", "b:0:"},
			exp:  []testFrame{"a:2:!", "b:0:"},
		},
		{
			desc: "last-ooo",
			fs:   []testFrame{"a:2:!", "b:1:"},
			exp:  []testFrame{"a:2:!"},
		},
		{
			desc: "middle-lastooo",
			fs:   []testFrame{"b:1:", "a:2:!"},
			exp:  []testFrame{"b:1:"},
		},
		{
			desc: "middle-first",
			fs:   []testFrame{"b:1:", "a:0:"},
			exp:  []testFrame{"a:0:"},
		},
		{
			desc: "last-first",
			fs:   []testFrame{"b:1:!", "a:0:"},
			exp:  []testFrame{"b:1:!", "a:0:"},
		},
		{
			desc: "last-ooo",
			fs:   []testFrame{"b:1:!", "b:2:"},
			exp:  []testFrame{"b:1:!"},
		},
		{
			desc: "ooo",
			fs:   []testFrame{"b:1:", "b:3:"},
			exp:  []testFrame{"b:1:"},
		},
		{
			desc: "other-ooo",
			fs:   []testFrame{"b:1:", "c:3:"},
			exp:  []testFrame{"b:1:"},
		},
		{
			desc: "other-ooo-last",
			fs:   []testFrame{"b:1:", "c:3:", "b:2:!"},
			exp:  []testFrame{"b:1:", "b:2:!"},
		},
		{
			desc: "first-discards-multiple",
			fs:   []testFrame{"c:0:", "c:1:", "c:2:", "d:0:", "c:3:!"},
			exp:  []testFrame{"d:0:"},
		},
		{
			desc: "complex",
			fs:   []testFrame{"b:1:", "b:2:!", "a:0:", "c:1:!", "a:1:", "a:2:!", "c:0:", "c:1:", "d:0:", "c:2:!", "e:0:"},
			exp:  []testFrame{"b:1:", "b:2:!", "a:0:", "a:1:", "a:2:!", "e:0:"},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			pfs := pruneFrameQueue(testFramesToFrames(tt.fs...))
			require.Equal(t, testFramesToFrames(tt.exp...), pfs)
		})
	}
}

func TestFrameQueue_NextFrame(t *testing.T) {
	t.Run("pre-holocene", func(t *testing.T) { testFrameQueue_NextFrame(t, false) })
	t.Run("holocene", func(t *testing.T) { testFrameQueue_NextFrame(t, true) })
}

func testFrameQueue_NextFrame(t *testing.T, holocene bool) {
	lgr := testlog.Logger(t, slog.LevelWarn)
	cfg := &rollup.Config{}
	dp := mocks.NewNextDataProvider(t)
	fq := NewFrameQueue(lgr, cfg, dp)

	inFrames := testFramesToFrames("b:1:", "b:2:!", "a:0:", "c:1:!", "a:1:", "a:2:!", "c:0:", "c:1:", "d:0:", "c:2:!", "e:0:")
	var expFrames []Frame
	if holocene {
		cfg.HoloceneTime = ptr(uint64(0))
		// expect pruned frames with Holocene
		expFrames = testFramesToFrames("b:1:", "b:2:!", "a:0:", "a:1:", "a:2:!", "e:0:")
	} else {
		expFrames = inFrames
	}

	var inBuf bytes.Buffer
	inBuf.WriteByte(DerivationVersion0)
	for _, f := range inFrames {
		require.NoError(t, f.MarshalBinary(&inBuf))
	}

	dp.On("Origin").Return(eth.L1BlockRef{})
	dp.On("NextData", mock.Anything).Return(inBuf.Bytes(), nil).Once()
	dp.On("NextData", mock.Anything).Return(nil, io.EOF)

	gotFrames := make([]Frame, 0, len(expFrames))
	for i := 0; i <= len(inFrames); i++ { // make sure we hit EOF case
		frame, err := fq.NextFrame(nil)
		if err != nil {
			require.ErrorIs(t, err, io.EOF)
			break
		}
		require.NoError(t, err)
		gotFrames = append(gotFrames, frame)
	}
	require.Equal(t, expFrames, gotFrames)
}

func ptr[T any](t T) *T { return &t }

func testFramesToFrames(tfs ...testFrame) []Frame {
	fs := make([]Frame, 0, len(tfs))
	for _, f := range tfs {
		fs = append(fs, f.ToFrame())
	}
	return fs
}
