package preimage

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

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
		a, b := bidirectionalPipe()
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			hw := NewHintWriter(a)
			for _, h := range hints {
				hw.Hint(rawHint(h))
			}
			wg.Done()
		}()

		got := make(chan string, len(hints))
		go func() {
			defer wg.Done()
			hr := NewHintReader(b)
			for i := 0; i < len(hints); i++ {
				err := hr.NextHint(func(hint string) error {
					got <- hint
					return nil
				})
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
			}
		}()
		if waitTimeout(&wg) {
			t.Error("hint read/write stuck")
		}

		require.Equal(t, len(hints), len(got), "got all hints")
		for _, h := range hints {
			require.Equal(t, h, <-got, "hints match")
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
		a, b := bidirectionalPipe()
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			hw := NewHintWriter(a)
			hw.Hint(rawHint("one"))
			hw.Hint(rawHint("two"))
			wg.Done()
		}()
		go func() {
			defer wg.Done()
			hr := NewHintReader(b)
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
		}()
		if waitTimeout(&wg) {
			t.Error("read/write hint stuck")
		}
	})
}

// waitTimeout returns true iff wg.Wait timed out
func waitTimeout(wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-time.After(time.Second * 30):
		return true
	case <-done:
		return false
	}
}
