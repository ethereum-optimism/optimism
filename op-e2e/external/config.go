package external

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type Config struct {
	DataDir     string `json:"data_dir"`
	JWTPath     string `json:"jwt_path"`
	ChainID     uint64 `json:"chain_id"`
	GasCeil     uint64 `json:"gas_ceil"`
	GenesisPath string `json:"genesis_path"`
	Verbosity   uint64 `json:"verbosity"`

	// EndpointsReadyPath is the location to write the endpoint configuration file.
	// Note, this should be written atomically by writing the JSON, then moving
	// it to this path to avoid races.  A helper AtomicEncode is provided for
	// golang clients.
	EndpointsReadyPath string `json:"endpoints_ready_path"`
}

// AtomicEncode json encodes val to path+".atomic" then moves the path+".atomic"
// file to path
func AtomicEncode(path string, val any) error {
	atomicPath := path + ".atomic"
	atomicFile, err := os.Create(atomicPath)
	if err != nil {
		return err
	}
	if err = json.NewEncoder(atomicFile).Encode(val); err != nil {
		return err
	}
	return os.Rename(atomicPath, path)
}

type Endpoints struct {
	HTTPEndpoint     string `json:"http_endpoint"`
	WSEndpoint       string `json:"ws_endpoint"`
	HTTPAuthEndpoint string `json:"http_auth_endpoint"`
	WSAuthEndpoint   string `json:"ws_auth_endpoint"`
}

type TestParms struct {
	// SkipTests is a map from test name to skip message.  The skip message may
	// be arbitrary, but the test name should match the skipped test (either
	// base, or a sub-test) exactly.  Precisely, the skip name must match rune for
	// rune starting with the first rune.  If the skip name does not match all
	// runes, the first mismatched rune must be a '/'.
	SkipTests map[string]string `json:"skip_tests"`
}

func (tp TestParms) SkipIfNecessary(t *testing.T) {
	if len(tp.SkipTests) == 0 {
		return
	}
	var base bytes.Buffer
	for _, name := range strings.Split(t.Name(), "/") {
		base.WriteString(name)
		if msg, ok := tp.SkipTests[base.String()]; ok {
			t.Skip(msg)
		}
		base.WriteRune('/')
	}
}
