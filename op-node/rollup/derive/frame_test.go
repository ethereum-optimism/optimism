package derive

import (
	"bytes"
	"testing"
)

func FuzzFrameUnmarshalBinary(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		buf := bytes.NewBuffer(data)
		var f Frame
		_ = (&f).UnmarshalBinary(buf)
	})
}

func FuzzParseFrames(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		frames, err := ParseFrames(data)
		if err != nil && len(frames) != 0 {
			t.Fatal("non-nil error with an amount of return data")
		} else if err == nil && len(frames) == 0 {
			t.Fatal("must return data with a non-nil error")
		}
	})
}
