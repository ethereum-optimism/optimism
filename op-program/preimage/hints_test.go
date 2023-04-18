package preimage

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type rawHint string

func (rh rawHint) Hint() string {
	return string(rh)
}

func TestHints(t *testing.T) {
	// Note: pretty much every string is valid communication:
	// length, payload, 0. Worst case you run out of data, or allocate too much.
	testHint := func(hints ...string) {
		var buf bytes.Buffer
		hw := NewHintWriter(&buf)
		for _, h := range hints {
			hw.Hint(rawHint(h))
		}
		hr := NewHintReader(&buf)
		var got []string
		for i := 0; i < 100; i++ { // sanity limit
			err := hr.NextHint(func(hint string) error {
				got = append(got, hint)
				return nil
			})
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}
		require.Equal(t, len(hints), len(got), "got all hints")
		for i, h := range hints {
			require.Equal(t, h, got[i], "hints match")
		}
	}

	t.Run("empty hint", func(t *testing.T) {
		testHint("")
	})
	t.Run("hello world", func(t *testing.T) {
		testHint("hello world")
	})
	t.Run("zero byte", func(t *testing.T) {
		testHint(string([]byte{0}))
	})
	t.Run("many zeroes", func(t *testing.T) {
		testHint(string(make([]byte, 1000)))
	})
	t.Run("random data", func(t *testing.T) {
		dat := make([]byte, 1000)
		_, _ = rand.Read(dat[:])
		testHint(string(dat))
	})
	t.Run("multiple hints", func(t *testing.T) {
		testHint("give me header a", "also header b", "foo bar")
	})
	t.Run("unexpected EOF", func(t *testing.T) {
		var buf bytes.Buffer
		hw := NewHintWriter(&buf)
		hw.Hint(rawHint("hello"))
		_, _ = buf.Read(make([]byte, 1)) // read one byte so it falls short, see if it's detected
		hr := NewHintReader(&buf)
		err := hr.NextHint(func(hint string) error { return nil })
		require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
	t.Run("cb error", func(t *testing.T) {
		var buf bytes.Buffer
		hw := NewHintWriter(&buf)
		hw.Hint(rawHint("one"))
		hw.Hint(rawHint("two"))
		hr := NewHintReader(&buf)
		cbErr := errors.New("fail")
		err := hr.NextHint(func(hint string) error { return cbErr })
		require.ErrorIs(t, err, cbErr)
		var readHint string
		err = hr.NextHint(func(hint string) error {
			readHint = hint
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, readHint, "two")
	})
}
