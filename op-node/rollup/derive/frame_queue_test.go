package derive

import (
	"testing"

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
			pfs := pruneFrameQueue(testFramesToFrames(tt.fs))
			require.Equal(t, testFramesToFrames(tt.exp), pfs)
		})
	}
}

func testFramesToFrames(tfs []testFrame) []Frame {
	fs := make([]Frame, 0, len(tfs))
	for _, f := range tfs {
		fs = append(fs, f.ToFrame())
	}
	return fs
}
