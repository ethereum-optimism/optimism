package matrix

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/stretchr/testify/require"
)

type sameCallEOFReader struct {
	idx  int
	data []byte
}

// newSameCallEOFReader returns an io.Reader that returns io.EOF in the same call that returns the final byte of data.
// This is valid as per io.Reader:
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil. The next Read should
// return 0, EOF.
func newSameCallEOFReader(data []byte) *sameCallEOFReader {
	return &sameCallEOFReader{data: data}
}

func (i *sameCallEOFReader) Read(out []byte) (int, error) {
	end := min(len(i.data), i.idx+len(out))
	n := copy(out, i.data[i.idx:end])
	i.idx += n
	if i.idx >= len(i.data) {
		return n, io.EOF
	}
	return n, nil
}

func TestImmediateEofReader(t *testing.T) {
	rng := rand.New(rand.NewSource(223))
	data := testutils.RandomData(rng, 100)

	batchSizes := []int{1, 2, 3, 5, 10, 33, 99, 100, 101}
	for _, size := range batchSizes {
		size := size
		t.Run(fmt.Sprintf("Size-%v", size), func(t *testing.T) {

			reader := &sameCallEOFReader{data: data}
			out := make([]byte, size)
			actual := make([]byte, 0, len(data))
			for {
				n, err := reader.Read(out)
				actual = append(actual, out[:n]...)
				if errors.Is(err, io.EOF) {
					break
				} else {
					require.NoError(t, err)
				}
			}
			require.Equal(t, data, actual)
			n, err := reader.Read(out)
			require.Zero(t, n)
			require.ErrorIs(t, err, io.EOF)
		})
	}
}
