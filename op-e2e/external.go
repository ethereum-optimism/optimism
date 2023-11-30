package op_e2e

import (
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/external"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/onsi/gomega/gexec"
	"github.com/stretchr/testify/require"
)

type ExternalRunner struct {
	Name    string
	BinPath string
	Genesis *core.Genesis
	JWTPath string
	// 4844: a datadir specifically for tx-pool blobs
	BlobPoolPath string
}

type ExternalEthClient struct {
	Session   *gexec.Session
	Endpoints external.Endpoints
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

func (eec *ExternalEthClient) Close() error {
	eec.Session.Terminate()
	select {
	case <-time.After(5 * time.Second):
		eec.Session.Kill()
		select {
		case <-time.After(30 * time.Second):
			return errors.New("external client failed to terminate")
		case <-eec.Session.Exited:
		}
	case <-eec.Session.Exited:
	}
	return nil
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
			Config:     params.OptimismTestConfig,
			Difficulty: big.NewInt(0),
		}
	}

	workDir := t.TempDir()

	config := external.Config{
		DataDir:            filepath.Join(workDir, "datadir"),
		JWTPath:            er.JWTPath,
		ChainID:            er.Genesis.Config.ChainID.Uint64(),
		GenesisPath:        filepath.Join(workDir, "genesis.json"),
		EndpointsReadyPath: filepath.Join(workDir, "endpoints.json"),
		Verbosity:          uint64(config.EthNodeVerbosity),
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

	cmd := exec.Command(er.BinPath, "--config", configPath)
	cmd.Dir = filepath.Dir(er.BinPath)
	sess, err := gexec.Start(
		cmd,
		gexec.NewPrefixedWriter("[extout:"+er.Name+"]", os.Stdout),
		gexec.NewPrefixedWriter("[exterr:"+er.Name+"]", os.Stderr),
	)
	require.NoError(t, err)

	// 2 minutes may seem like a long timeout, and, it definitely is.  That
	// being said, when running these tests with high parallelism turned on, the
	// node startup time can be substantial (remember, this usually is a
	// multi-step process initializing the database and then starting the
	// client).
	require.Eventually(
		t,
		func() bool {
			_, err := os.Stat(config.EndpointsReadyPath)
			return err == nil
		},
		2*time.Minute,
		10*time.Millisecond,
		"external runner did not create ready file at %s within timeout",
		config.EndpointsReadyPath,
	)

	readyFile, err := os.Open(config.EndpointsReadyPath)
	require.NoError(t, err)
	var endpoints external.Endpoints
	err = json.NewDecoder(readyFile).Decode(&endpoints)
	require.NoError(t, err)

	return &ExternalEthClient{
		Session:   sess,
		Endpoints: endpoints,
	}
}
