package logpipe

import (
	"bytes"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestPipeLogs(t *testing.T) {
	logger, capt := testlog.CaptureLogger(t, log.LevelTrace)

	wg := new(sync.WaitGroup)
	wg.Add(2)

	r, w := io.Pipe()
	// Write the log output to the pipe
	go func() {
		defer wg.Done()
		_, err := io.Copy(w, bytes.NewReader([]byte(`{"level": "DEBUG", "fields": {"message": "hello", "foo": 1}}`+"\n")))
		require.NoError(t, err)
		_, err = io.Copy(w, bytes.NewReader([]byte(`test invalid JSON`+"\n")))
		require.NoError(t, err)
		_, err = io.Copy(w, bytes.NewReader([]byte(`{"fields": {"message": "world", "bar": "sunny"}, "level": "INFO"}`+"\n")))
		require.NoError(t, err)
		require.NoError(t, w.Close())
	}()
	// Read the log output from the pipe
	go func() {
		defer wg.Done()
		toLogger := ToLogger(logger)
		logProc := func(line []byte) {
			toLogger(ParseRustStructuredLogs(line))
		}
		err := PipeLogs(r, logProc)
		require.NoError(t, err)
	}()
	wg.Wait()

	entry1 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelDebug),
		testlog.NewAttributesContainsFilter("foo", "1"))
	require.NotNil(t, entry1)
	require.Equal(t, "hello", entry1.Message)

	entry2 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewAttributesContainsFilter("line", "test invalid JSON"))
	require.NotNil(t, entry2)
	require.Equal(t, "Invalid JSON", entry2.Message)

	entry3 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelInfo),
		testlog.NewAttributesContainsFilter("bar", "sunny"))
	require.NotNil(t, entry3)
	require.Equal(t, "world", entry3.Message)
}
