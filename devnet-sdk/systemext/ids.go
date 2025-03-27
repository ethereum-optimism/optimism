package systemext

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func collectSystemExtIDs(env *descriptors.DevnetEnvironment) DefaultSystemExtIDs {
	l1ID := eth.ChainIDFromBig(env.L1.Config.ChainID)
	l1Nodes := make([]DefaultSystemExtL1NodeIDs, len(env.L1.Nodes))
	for i := range env.L1.Nodes {
		l1Nodes[i] = DefaultSystemExtL1NodeIDs{
			EL: system2.L1ELNodeID{Key: fmt.Sprintf("el-%d", i), ChainID: l1ID},
			CL: system2.L1CLNodeID{Key: fmt.Sprintf("cl-%d", i), ChainID: l1ID},
		}
	}

	l2s := make([]DefaultSystemExtL2IDs, len(env.L2))
	for idx, l2 := range env.L2 {
		l2ID := eth.ChainIDFromBig(l2.Config.ChainID)
		id := system2.L2NetworkID{Key: l2.Name, ChainID: l2ID}

		nodes := make([]DefaultSystemExtL2NodeIDs, len(l2.Nodes))
		for i := range l2.Nodes {
			nodes[i] = DefaultSystemExtL2NodeIDs{
				EL: system2.L2ELNodeID{Key: fmt.Sprintf("el-%s-%d", l2.Name, i), ChainID: l2ID},
				CL: system2.L2CLNodeID{Key: fmt.Sprintf("cl-%s-%d", l2.Name, i), ChainID: l2ID},
			}
		}

		l2s[idx] = DefaultSystemExtL2IDs{
			L2:    id,
			Nodes: nodes,

			L2Batcher:    system2.L2BatcherID{Key: fmt.Sprintf("batcher-%s", l2.Name), ChainID: l2ID},
			L2Proposer:   system2.L2ProposerID{Key: fmt.Sprintf("proposer-%s", l2.Name), ChainID: l2ID},
			L2Challenger: system2.L2ChallengerID{Key: fmt.Sprintf("challenger-%s", l2.Name), ChainID: l2ID},
		}
	}

	ids := DefaultSystemExtIDs{
		L1: system2.L1NetworkID{
			Key:     env.L1.Name,
			ChainID: l1ID,
		},
		Nodes:      l1Nodes,
		Superchain: system2.SuperchainID(env.Name),
		Cluster:    system2.ClusterID(env.Name),
		L2s:        l2s,
	}

	if isInterop(env) {
		ids.Supervisor = system2.SupervisorID(env.Name)
	}

	return ids
}
