package syskt

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func collectSystemExtIDs(env *descriptors.DevnetEnvironment) DefaultSystemExtIDs {
	l1ID := eth.ChainIDFromBig(env.L1.Config.ChainID)
	l1Nodes := make([]DefaultSystemExtL1NodeIDs, len(env.L1.Nodes))
	for i := range env.L1.Nodes {
		l1Nodes[i] = DefaultSystemExtL1NodeIDs{
			EL: stack.L1ELNodeID{Key: fmt.Sprintf("el-%d", i), ChainID: l1ID},
			CL: stack.L1CLNodeID{Key: fmt.Sprintf("cl-%d", i), ChainID: l1ID},
		}
	}

	l2s := make([]DefaultSystemExtL2IDs, len(env.L2))
	for idx, l2 := range env.L2 {
		l2ID := eth.ChainIDFromBig(l2.Config.ChainID)
		id := stack.L2NetworkID{Key: l2.Name, ChainID: l2ID}

		nodes := make([]DefaultSystemExtL2NodeIDs, len(l2.Nodes))
		for i := range l2.Nodes {
			nodes[i] = DefaultSystemExtL2NodeIDs{
				EL: stack.L2ELNodeID{Key: fmt.Sprintf("el-%s-%d", l2.Name, i), ChainID: l2ID},
				CL: stack.L2CLNodeID{Key: fmt.Sprintf("cl-%s-%d", l2.Name, i), ChainID: l2ID},
			}
		}

		l2s[idx] = DefaultSystemExtL2IDs{
			L2:    id,
			Nodes: nodes,

			L2Batcher:    stack.L2BatcherID{Key: fmt.Sprintf("batcher-%s", l2.Name), ChainID: l2ID},
			L2Proposer:   stack.L2ProposerID{Key: fmt.Sprintf("proposer-%s", l2.Name), ChainID: l2ID},
			L2Challenger: stack.L2ChallengerID{Key: fmt.Sprintf("challenger-%s", l2.Name), ChainID: l2ID},
		}
	}

	ids := DefaultSystemExtIDs{
		L1: stack.L1NetworkID{
			Key:     env.L1.Name,
			ChainID: l1ID,
		},
		Nodes:      l1Nodes,
		Superchain: stack.SuperchainID(env.Name),
		Cluster:    stack.ClusterID(env.Name),
		L2s:        l2s,
	}

	if isInterop(env) {
		ids.Supervisor = stack.SupervisorID(env.Name)
	}

	return ids
}
