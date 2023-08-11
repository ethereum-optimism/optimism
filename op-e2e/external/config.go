package external

import (
	"encoding/json"
	"os"
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
