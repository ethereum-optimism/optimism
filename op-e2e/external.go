package op_e2e

import (
	"encoding/json"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/stretchr/testify/require"
)

type ExternalRunner struct {
	Name    string
	BinPath string
	Genesis *core.Genesis
	JWTPath string
}

type ExternalConfig struct {
	DataDir     string `json:"data_dir"`
	JWTPath     string `json:"jwt_path"`
	ChainID     uint64 `json:"chain_id"`
	GasCeil     uint64 `json:"gas_ceil"`
	GenesisPath string `json:"genesis_path"`

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

type ExternalEndpoints struct {
	HTTPEndpoint     string `json:"http_endpoint"`
	WSEndpoint       string `json:"ws_endpoint"`
	HTTPAuthEndpoint string `json:"http_auth_endpoint"`
	WSAuthEndpoint   string `json:"ws_auth_endpoint"`
}

type ExternalEthClient struct {
	Session   *gexec.Session
	Endpoints ExternalEndpoints
}

func (eec *ExternalEthClient) HTTPEndpoint() string {
	return eec.Endpoints.HTTPEndpoint
}

func (eec *ExternalEthClient) WSEndpoint() string {
	return eec.Endpoints.WSEndpoint
}

func (eec *ExternalEthClient) HTTPAuthEndpoint() string {
	return eec.Endpoints.HTTPAuthEndpoint
}

func (eec *ExternalEthClient) WSAuthEndpoint() string {
	return eec.Endpoints.WSAuthEndpoint
}

func (eec *ExternalEthClient) Close() {
	eec.Session.Terminate()
	select {
	case <-time.After(5 * time.Second):
		eec.Session.Kill()
	case <-eec.Session.Exited:
	}
}

func (er *ExternalRunner) Run(t *testing.T) *ExternalEthClient {
	if er.BinPath == "" {
		t.Error("no external bin path set")
	}

	if er.JWTPath == "" {
		er.JWTPath = writeDefaultJWT(t)
	}

	if er.Genesis == nil {
		er.Genesis = &core.Genesis{
			Alloc: core.GenesisAlloc{
				common.Address{1}: core.GenesisAccount{Balance: big.NewInt(1)},
			},
			Config:     &params.ChainConfig{ChainID: big.NewInt(901)},
			Difficulty: big.NewInt(0),
		}
	}

	workDir := t.TempDir()

	config := ExternalConfig{
		DataDir:            filepath.Join(workDir, "datadir"),
		JWTPath:            er.JWTPath,
		ChainID:            er.Genesis.Config.ChainID.Uint64(),
		GenesisPath:        filepath.Join(workDir, "genesis.json"),
		EndpointsReadyPath: filepath.Join(workDir, "endpoints.json"),
	}

	err := os.Mkdir(config.DataDir, 0o700)
	require.NoError(t, err)

	genesisFile, err := os.Create(config.GenesisPath)
	require.NoError(t, err)
	err = json.NewEncoder(genesisFile).Encode(er.Genesis)
	require.NoError(t, err)

	configPath := filepath.Join(workDir, "config.json")
	configFile, err := os.Create(configPath)
	require.NoError(t, err)
	err = json.NewEncoder(configFile).Encode(config)
	require.NoError(t, err)

	gt := gomega.NewWithT(t)
	cmd := exec.Command(er.BinPath, "--config", configPath)
	cmd.Dir = filepath.Dir(er.BinPath)
	sess, err := gexec.Start(
		cmd,
		gexec.NewPrefixedWriter("[extout:"+er.Name+"]", os.Stdout),
		gexec.NewPrefixedWriter("[exterr:"+er.Name+"]", os.Stderr),
	)
	gt.Expect(err).NotTo(gomega.HaveOccurred())

	// 2 minutes may seem like a long timeout, and, it definitely is.  That
	// being said, when running these tests with high parallelism turned on, the
	// node startup time can be substantial (remember, this usually is a
	// multi-step process initializing the database and then starting the
	// client).
	gt.Eventually(config.EndpointsReadyPath, 2*time.Minute).Should(gomega.BeARegularFile(), "external runner did not create ready file at %s within timeout", config.EndpointsReadyPath)

	readyFile, err := os.Open(config.EndpointsReadyPath)
	require.NoError(t, err)
	var endpoints ExternalEndpoints
	err = json.NewDecoder(readyFile).Decode(&endpoints)
	require.NoError(t, err)

	return &ExternalEthClient{
		Session:   sess,
		Endpoints: endpoints,
	}
}
