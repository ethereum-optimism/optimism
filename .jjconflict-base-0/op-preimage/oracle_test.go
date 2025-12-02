package preimage

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type readWritePair struct {
	io.Reader
	io.Writer
}

func bidirectionalPipe() (a, b io.ReadWriter) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	return readWritePair{Reader: ar, Writer: aw}, readWritePair{Reader: br, Writer: bw}
}

func TestOracle(t *testing.T) {
	testPreimage := func(preimages ...[]byte) {
		a, b := bidirectionalPipe()
		cl := NewOracleClient(a)
		srv := NewOracleServer(b)

		preimageByHash := make(map[[32]byte][]byte)
		for _, p := range preimages {
			k := Keccak256Key(Keccak256(p))
			preimageByHash[k.PreimageKey()] = p
		}
		for _, p := range preimages {
			k := Keccak256Key(Keccak256(p))

			var wg sync.WaitGroup
			wg.Add(2)

			go func(k Key, p []byte) {
				result := cl.Get(k)
				wg.Done()
				expected := preimageByHash[k.PreimageKey()]
				require.True(t, bytes.Equal(expected, result), "need correct preimage %x, got %x", expected, result)
			}(k, p)

			go func() {
				err := srv.NextPreimageRequest(func(key [32]byte) ([]byte, error) {
					dat, ok := preimageByHash[key]
					if !ok {
						return nil, fmt.Errorf("cannot find %s", key)
					}
					return dat, nil
				})
				wg.Done()
				require.NoError(t, err)
			}()
			wg.Wait()
		}
	}
	t.Run("empty preimage", func(t *testing.T) {
		testPreimage([]byte{})
	})
	t.Run("nil preimage", func(t *testing.T) {
		testPreimage(nil)
	})
	t.Run("zero", func(t *testing.T) {
		testPreimage([]byte{0})
	})
	t.Run("multiple", func(t *testing.T) {
		testPreimage([]byte("tx from alice"), []byte{0x13, 0x37}, []byte("tx from bob"))
	})
	t.Run("zeroes", func(t *testing.T) {
		testPreimage(make([]byte, 1000))
	})
	t.Run("random", func(t *testing.T) {
		dat := make([]byte, 1000)
		_, _ = rand.Read(dat[:])
		testPreimage(dat)
	})
}
