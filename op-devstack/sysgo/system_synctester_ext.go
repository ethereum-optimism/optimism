package sysgo

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
)

type DefaultMinimalExternalELSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2           stack.L2NetworkID
	L2CL         stack.L2CLNodeID
	L2EL         stack.L2ELNodeID
	L2ELReadOnly stack.L2ELNodeID

	SyncTester stack.SyncTesterID
}

func NewExternalELSystemIDs(l1ID, l2ID eth.ChainID) DefaultMinimalExternalELSystemIDs {
	ids := DefaultMinimalExternalELSystemIDs{
		L1:           stack.L1NetworkID(l1ID),
		L1EL:         stack.NewL1ELNodeID("l1", l1ID),
		L1CL:         stack.NewL1CLNodeID("l1", l1ID),
		L2:           stack.L2NetworkID(l2ID),
		L2CL:         stack.NewL2CLNodeID("verifier", l2ID),
		L2EL:         stack.NewL2ELNodeID("sync-tester-el", l2ID),
		L2ELReadOnly: stack.NewL2ELNodeID("l2-el-readonly", l2ID),
		SyncTester:   stack.NewSyncTesterID("sync-tester", l2ID),
	}
	return ids
}

// ExternalELSystemWithEndpointAndSuperchainRegistry creates a minimal external EL system
// using a network from the superchain registry instead of the deployer
func ExternalELSystemWithEndpointAndSuperchainRegistry(dest *DefaultMinimalExternalELSystemIDs, networkPreset stack.ExtNetworkConfig) stack.Option[*Orchestrator] {
	chainCfg := chaincfg.ChainByName(networkPreset.L2NetworkName)
	if chainCfg == nil {
		panic(fmt.Sprintf("network %s not found in superchain registry", networkPreset.L2NetworkName))
	}
	l2ChainID := eth.ChainIDFromUInt64(chainCfg.ChainID)

	ids := NewExternalELSystemIDs(networkPreset.L1ChainID, l2ChainID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up with superchain registry network", "network", networkPreset.L2NetworkName)
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	// We must supply the full L1 Chain Config, so look that up or fail if unknown
	chainID := ids.L1.ChainID()
	l1ChainConfig := eth.L1ChainConfigByChainID(chainID)
	if l1ChainConfig == nil {
		panic(fmt.Sprintf("unsupported L1 chain ID: %s", chainID))
	}

	// Skip deployer since we're using external L1 and superchain registry for L2 config
	// Create L1 network record for external L1
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		l1Net := &L1Network{
			id: ids.L1,
			genesis: &core.Genesis{
				Config: l1ChainConfig,
			},
			blockTime: 12,
		}
		o.l1Nets.Set(ids.L1.ChainID(), l1Net)
	}))

	opt.Add(WithExtL1Nodes(ids.L1EL, ids.L1CL, networkPreset.L1ELEndpoint, networkPreset.L1CLBeaconEndpoint))

	// Use superchain registry instead of deployer
	opt.Add(WithL2NetworkFromSuperchainRegistryWithDependencySet(
		stack.L2NetworkID(l2ChainID),
		networkPreset.L2NetworkName,
	))

	// Add SyncTester service with external endpoint
	opt.Add(WithSyncTesterWithExternalEndpoint(ids.SyncTester, networkPreset.L2ELEndpoint, l2ChainID))

	// Add SyncTesterL2ELNode as the L2EL replacement for real-world EL endpoint
	opt.Add(WithSyncTesterL2ELNode(ids.L2EL, ids.L2EL))
	opt.Add(WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL))

	opt.Add(WithExtL2Node(ids.L2ELReadOnly, networkPreset.L2ELEndpoint))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
