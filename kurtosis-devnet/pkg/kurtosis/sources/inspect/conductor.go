package inspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
)

type ConductorSequencer struct {
	RaftAddr        string `json:"raft_addr" toml:"raft_addr"`
	ConductorRPCURL string `json:"conductor_rpc_url" toml:"conductor_rpc_url"`
	NodeRPCURL      string `json:"node_rpc_url" toml:"node_rpc_url"`
	Voting          bool   `json:"voting" toml:"voting"`
}

type ConductorNetwork struct {
	Sequencers []string `json:"sequencers" toml:"sequencers"`
}

type ConductorConfig struct {
	Networks   map[string]*ConductorNetwork   `json:"networks" toml:"networks"`
	Sequencers map[string]*ConductorSequencer `json:"sequencers" toml:"sequencers"`
}

func ExtractConductorConfig(ctx context.Context, enclaveID string) (*ConductorConfig, error) {
	kurtosisCtx, err := wrappers.GetDefaultKurtosisContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kurtosis context: %w", err)
	}

	enclaveCtx, err := kurtosisCtx.GetEnclave(ctx, enclaveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enclave: %w", err)
	}

	services, err := enclaveCtx.GetServices()
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	conductorServices := make(map[string]map[string]interface{})
	opNodeServices := make(map[string]map[string]interface{})

	for svcName := range services {
		svcNameStr := string(svcName)

		svcCtx, err := enclaveCtx.GetService(svcNameStr)
		if err != nil {
			continue
		}

		labels := svcCtx.GetLabels()
		ports := make(map[string]*descriptors.PortInfo)

		for portName, portSpec := range svcCtx.GetPublicPorts() {
			ports[portName] = &descriptors.PortInfo{
				Host: svcCtx.GetMaybePublicIPAddress(),
				Port: int(portSpec.GetNumber()),
			}
		}

		if labels["op.kind"] == "conductor" {
			conductorServices[svcNameStr] = map[string]interface{}{
				"labels": labels,
				"ports":  ports,
			}
		}

		if labels["op.kind"] == "cl" && labels["op.cl.type"] == "op-node" {
			opNodeServices[svcNameStr] = map[string]interface{}{
				"labels": labels,
				"ports":  ports,
			}
		}
	}

	if len(conductorServices) == 0 {
		return nil, nil
	}

	networks := make(map[string]*ConductorNetwork)
	sequencers := make(map[string]*ConductorSequencer)

	networkSequencers := make(map[string][]string)

	for conductorSvcName, conductorData := range conductorServices {
		labels := conductorData["labels"].(map[string]string)
		ports := conductorData["ports"].(map[string]*descriptors.PortInfo)

		networkID := labels["op.network.id"]
		if networkID == "" {
			continue
		}

		// Find the network name from service name (e.g., "op-conductor-2151908-op-kurtosis-node0")
		parts := strings.Split(conductorSvcName, "-")
		var networkName string
		if len(parts) >= 4 {
			networkName = strings.Join(parts[2:len(parts)-1], "-")
		}
		if networkName == "" {
			networkName = "unknown"
		}

		networkSequencers[networkName] = append(networkSequencers[networkName], conductorSvcName)

		participantName := labels["op.network.participant.name"]
		var nodeRPCURL string

		// Look for matching op-node service
		for _, nodeData := range opNodeServices {
			nodeLabels := nodeData["labels"].(map[string]string)
			nodePorts := nodeData["ports"].(map[string]*descriptors.PortInfo)

			if nodeLabels["op.network.participant.name"] == participantName &&
				nodeLabels["op.network.id"] == networkID {
				if rpcPort, ok := nodePorts["rpc"]; ok {
					nodeRPCURL = fmt.Sprintf("http://127.0.0.1:%d", rpcPort.Port)
				}
				break
			}
		}

		var raftAddr, conductorRPCURL string

		if consensusPort, ok := ports["consensus"]; ok {
			raftAddr = fmt.Sprintf("127.0.0.1:%d", consensusPort.Port)
		}

		if rpcPort, ok := ports["rpc"]; ok {
			conductorRPCURL = fmt.Sprintf("http://127.0.0.1:%d", rpcPort.Port)
		}

		sequencers[conductorSvcName] = &ConductorSequencer{
			RaftAddr:        raftAddr,
			ConductorRPCURL: conductorRPCURL,
			NodeRPCURL:      nodeRPCURL,
			Voting:          true,
		}
	}

	for networkName, sequencerNames := range networkSequencers {
		networks[networkName] = &ConductorNetwork{
			Sequencers: sequencerNames,
		}
	}

	return &ConductorConfig{
		Networks:   networks,
		Sequencers: sequencers,
	}, nil
}
