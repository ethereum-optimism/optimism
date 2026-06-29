package preimage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWritePairCloseJoinsReaderAndWriterErrors(t *testing.T) {
	reader, err := os.CreateTemp(t.TempDir(), "reader")
	require.NoError(t, err)
	writer, err := os.CreateTemp(t.TempDir(), "writer")
	require.NoError(t, err)

	require.NoError(t, reader.Close())
	require.NoError(t, writer.Close())

	err = NewReadWritePair(reader, writer).Close()
	require.Error(t, err)

	errMsg := err.Error()
	require.Contains(t, errMsg, "failed to close reader")
	require.Contains(t, errMsg, "failed to close writer")
}
