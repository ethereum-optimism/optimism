package inspect

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConductorConfig(t *testing.T) {
	config := &ConductorConfig{
		Networks: map[string]*ConductorNetwork{
			"chain0": {Sequencers: []string{"seq0"}},
			"chain1": {Sequencers: []string{"seq1"}},
		},
		Sequencers: map[string]*ConductorSequencer{
			"seq0": {
				RaftAddr:        "127.0.0.1:8001",
				ConductorRPCURL: "http://127.0.0.1:8002",
				NodeRPCURL:      "http://127.0.0.1:8003",
				Voting:          true,
			},
			"seq1": {
				RaftAddr:        "127.0.0.1:8011",
				ConductorRPCURL: "http://127.0.0.1:8012",
				NodeRPCURL:      "http://127.0.0.1:8013",
				Voting:          false,
			},
		},
	}

	var buf strings.Builder
	err := toml.NewEncoder(&buf).Encode(config)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "[networks]")
	assert.Contains(t, output, "[sequencers]")
	assert.Contains(t, output, "voting = true")
	assert.Contains(t, output, "voting = false")

	var decoded ConductorConfig
	err = toml.Unmarshal([]byte(output), &decoded)
	require.NoError(t, err)
	assert.Equal(t, config.Networks, decoded.Networks)
	assert.Equal(t, config.Sequencers, decoded.Sequencers)
}

func TestConductorSequencer(t *testing.T) {
	seq := &ConductorSequencer{
		RaftAddr:        "localhost:8080",
		ConductorRPCURL: "http://localhost:9090",
		NodeRPCURL:      "http://localhost:7070",
		Voting:          true,
	}

	assert.Equal(t, "localhost:8080", seq.RaftAddr)
	assert.Equal(t, "http://localhost:9090", seq.ConductorRPCURL)
	assert.True(t, seq.Voting)
}

func TestMultiChainConfig(t *testing.T) {
	config := &ConductorConfig{
		Networks: map[string]*ConductorNetwork{
			"chain0": {Sequencers: []string{"seq0", "backup0"}},
			"chain1": {Sequencers: []string{"seq1", "observer1"}},
		},
		Sequencers: map[string]*ConductorSequencer{
			"seq0":      {RaftAddr: "127.0.0.1:8001", ConductorRPCURL: "http://127.0.0.1:8002", NodeRPCURL: "http://127.0.0.1:8003", Voting: true},
			"backup0":   {RaftAddr: "127.0.0.1:8011", ConductorRPCURL: "http://127.0.0.1:8012", NodeRPCURL: "http://127.0.0.1:8013", Voting: true},
			"seq1":      {RaftAddr: "127.0.0.1:8021", ConductorRPCURL: "http://127.0.0.1:8022", NodeRPCURL: "http://127.0.0.1:8023", Voting: true},
			"observer1": {RaftAddr: "127.0.0.1:8031", ConductorRPCURL: "http://127.0.0.1:8032", NodeRPCURL: "http://127.0.0.1:8033", Voting: false},
		},
	}

	assert.Len(t, config.Networks, 2)
	assert.Len(t, config.Sequencers, 4)
	assert.False(t, config.Sequencers["observer1"].Voting)

	var buf strings.Builder
	err := toml.NewEncoder(&buf).Encode(config)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "voting = false")
}
