package op_e2e

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestErigonBuildPath(t *testing.T) {
	require.FileExists(t, erigonBinPath)
}

func TestErigonRunner(t *testing.T) {
	er := &ErigonRunner{
		BinPath: erigonBinPath,
	}
	es := er.Run(t)
	t.Cleanup(es.Shutdown)
	_, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", es.HTTPPort), time.Second)
	require.NoError(t, err, "could not connect to HTTP/WS port")
	_, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", es.EnginePort), time.Second)
	require.NoError(t, err, "could not connect to Engine port")
}
