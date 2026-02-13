package batcher

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxID_String(t *testing.T) {
	for _, test := range []struct {
		desc   string
		id     txID
		expStr string
	}{
		{
			desc:   "empty",
			id:     []frameID{},
			expStr: "",
		},
		{
			desc:   "nil",
			id:     nil,
			expStr: "",
		},
		{
			desc: "single",
			id: []frameID{{
				chID:        [16]byte{0: 0xca, 15: 0xaf},
				frameNumber: 42,
			}},
			expStr: "ca0000000000000000000000000000af:42",
		},
		{
			desc: "multi",
			id: []frameID{
				{
					chID:        [16]byte{0: 0xca, 15: 0xaf},
					frameNumber: 42,
				},
				{
					chID:        [16]byte{0: 0xca, 15: 0xaf},
					frameNumber: 33,
				},
				{
					chID:        [16]byte{0: 0xbe, 15: 0xef},
					frameNumber: 0,
				},
				{
					chID:        [16]byte{0: 0xbe, 15: 0xef},
					frameNumber: 128,
				},
			},
			expStr: "ca0000000000000000000000000000af:42+33|be0000000000000000000000000000ef:0+128",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			require.Equal(t, test.expStr, test.id.String())
		})
	}
}
