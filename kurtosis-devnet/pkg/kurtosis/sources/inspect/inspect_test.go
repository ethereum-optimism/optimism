package inspect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
)

func TestNewInspector(t *testing.T) {
	inspector := NewInspector("test-enclave")
	assert.NotNil(t, inspector)
	assert.Equal(t, "test-enclave", inspector.enclaveID)
}

func TestShortenedUUIDString(t *testing.T) {
	assert.Equal(t, "f47ac10b-58c", ShortenedUUIDString("f47ac10b-58cc-4372-a567-0e02b2c3d479"))
	assert.Equal(t, "abc", ShortenedUUIDString("abc"))
	assert.Equal(t, "", ShortenedUUIDString(""))
	assert.Equal(t, "123456789012", ShortenedUUIDString("123456789012"))
	assert.Equal(t, "test2-devnet", ShortenedUUIDString("test2-devnet-2151908"))
}

func TestInspectData(t *testing.T) {
	data := &InspectData{
		FileArtifacts: []string{"genesis.json", "jwt.txt"},
		UserServices: ServiceMap{
			"op-node": &Service{
				Labels: map[string]string{"app": "op-node", "role": "sequencer"},
				Ports: PortMap{
					"rpc": &descriptors.PortInfo{Host: "127.0.0.1", Port: 8545},
					"p2p": &descriptors.PortInfo{Host: "127.0.0.1", Port: 9222},
				},
			},
		},
	}

	assert.Len(t, data.FileArtifacts, 2)
	assert.Len(t, data.UserServices, 1)
	assert.Contains(t, data.FileArtifacts, "genesis.json")

	service := data.UserServices["op-node"]
	assert.Equal(t, "op-node", service.Labels["app"])
	assert.Equal(t, "sequencer", service.Labels["role"])

	rpcPort, exists := service.Ports["rpc"]
	require.True(t, exists)
	assert.Equal(t, 8545, rpcPort.Port)

	_, exists = service.Ports["nonexistent"]
	assert.False(t, exists)
}

func TestServiceMap(t *testing.T) {
	services := ServiceMap{
		"seq0":      &Service{Labels: map[string]string{"role": "sequencer"}, Ports: PortMap{"rpc": &descriptors.PortInfo{Port: 8545}}},
		"seq1":      &Service{Labels: map[string]string{"role": "sequencer"}, Ports: PortMap{"rpc": &descriptors.PortInfo{Port: 8645}}},
		"conductor": &Service{Labels: map[string]string{"app": "conductor"}, Ports: PortMap{"rpc": &descriptors.PortInfo{Port: 8547}}},
	}

	assert.Len(t, services, 3)

	seq0, exists := services["seq0"]
	require.True(t, exists)
	assert.Equal(t, "sequencer", seq0.Labels["role"])

	sequencerCount := 0
	for _, svc := range services {
		if svc.Labels["role"] == "sequencer" {
			sequencerCount++
		}
	}
	assert.Equal(t, 2, sequencerCount)
}
