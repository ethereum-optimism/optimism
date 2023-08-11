package celestia

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeFrameRef(t *testing.T) {
	tests := []struct {
		name        string
		frameRefHex string
		frameRef    FrameRef
		isValid     bool
		err         error
	}{
		{
			"valid frame reference",
			"d20400000000000068656c6c6f20776f726c64", // 1234 + "hello world"
			FrameRef{BlockHeight: 1234, TxCommitment: []byte{0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x77, 0x6f, 0x72, 0x6c, 0x64}},
			true,
			nil,
		},
		{
			"invalid frame reference",
			"4269",
			FrameRef{},
			false,
			ErrInvalidSize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frameRef, err := hex.DecodeString(tt.frameRefHex)
			require.NoError(t, err)
			gotFrameRef := FrameRef{}
			err = gotFrameRef.UnmarshalBinary(frameRef)
			if !tt.isValid {
				require.ErrorIs(t, err, ErrInvalidSize)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.frameRef, gotFrameRef)
			frameRefHex, err := tt.frameRef.MarshalBinary()
			require.NoError(t, err)
			gotFrameRefHex := hex.EncodeToString(frameRefHex)
			require.Equal(t, tt.frameRefHex, gotFrameRefHex)
		})
	}
}
