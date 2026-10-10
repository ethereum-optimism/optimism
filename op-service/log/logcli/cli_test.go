package logcli_test

import (
	"bytes"
	stdlog "log"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/log/logfilter"
)

// TestSetGlobalLogHandlerTagsRecords checks that records logged through the
// global logger, and through its go-ethereum view, carry the global=true tag
// that SetGlobalLogHandler puts in the logger's default context.
func TestSetGlobalLogHandlerTagsRecords(t *testing.T) {
	root, slogDefault := log.Root(), slog.Default()
	writer, flags := stdlog.Writer(), stdlog.Flags()
	t.Cleanup(func() {
		log.SetDefault(root)
		slog.SetDefault(slogDefault)
		stdlog.SetOutput(writer)
		stdlog.SetFlags(flags)
	})

	var buf bytes.Buffer
	logcli.SetGlobalLogHandler(logfilter.WrapContextHandler(log.LogfmtHandler(&buf)))
	log.Info("package-level")
	log.ToGeth(log.Root()).Info("geth view")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	for _, line := range lines {
		require.Contains(t, line, "global=true")
	}
}
