package inspect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
)

func TestInspectService(t *testing.T) {
	cfg := &Config{EnclaveID: "test-enclave"}
	service := NewInspectService(cfg, log.New())

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.cfg)
}

func TestFileWriting(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &Config{
		EnclaveID:           "test-enclave",
		ConductorConfigPath: filepath.Join(tempDir, "conductor.toml"),
		EnvironmentPath:     filepath.Join(tempDir, "environment.json"),
	}
	service := NewInspectService(cfg, log.New())

	conductorConfig := &ConductorConfig{
		Networks:   map[string]*ConductorNetwork{"chain": {Sequencers: []string{"seq"}}},
		Sequencers: map[string]*ConductorSequencer{"seq": {RaftAddr: "127.0.0.1:8001", ConductorRPCURL: "http://127.0.0.1:8002", NodeRPCURL: "http://127.0.0.1:8003", Voting: true}},
	}

	inspectData := &InspectData{
		FileArtifacts: []string{"genesis.json", "jwt.txt"},
		UserServices: ServiceMap{
			"op-node": &Service{
				Labels: map[string]string{"app": "op-node"},
				Ports:  PortMap{"rpc": &descriptors.PortInfo{Host: "127.0.0.1", Port: 8545}},
			},
		},
	}

	err := service.writeFiles(inspectData, conductorConfig)
	require.NoError(t, err)

	assert.FileExists(t, cfg.ConductorConfigPath)
	assert.FileExists(t, cfg.EnvironmentPath)

	content, err := os.ReadFile(cfg.ConductorConfigPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "[networks]")
	assert.Contains(t, string(content), "[sequencers]")

	envContent, err := os.ReadFile(cfg.EnvironmentPath)
	require.NoError(t, err)
	assert.Contains(t, string(envContent), "genesis.json")
	assert.Contains(t, string(envContent), "op-node")
}
