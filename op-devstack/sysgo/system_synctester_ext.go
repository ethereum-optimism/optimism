package sysgo

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

type DefaultMinimalExternalELSystemIDs struct {
	L1   stack.L1NetworkID
	L1EL stack.L1ELNodeID
	L1CL stack.L1CLNodeID

	L2   stack.L2NetworkID
	L2CL stack.L2CLNodeID
	L2EL stack.L2ELNodeID

	SyncTester stack.SyncTesterID
}

func NewDefaultMinimalExternalELSystemIDs(l1ID, l2ID eth.ChainID) DefaultMinimalExternalELSystemIDs {
	ids := DefaultMinimalExternalELSystemIDs{
		L1:         stack.L1NetworkID(l1ID),
		L1EL:       stack.NewL1ELNodeID("l1", l1ID),
		L1CL:       stack.NewL1CLNodeID("l1", l1ID),
		L2:         stack.L2NetworkID(l2ID),
		L2CL:       stack.NewL2CLNodeID("verifier", l2ID),
		L2EL:       stack.NewL2ELNodeID("sync-tester-el", l2ID),
		SyncTester: stack.NewSyncTesterID("sync-tester", l2ID),
	}
	return ids
}

// DefaultMinimalExternalELSystemWithEndpointAndSuperchainRegistry creates a minimal external EL system
// using a network from the superchain registry instead of the deployer
func DefaultMinimalExternalELSystemWithEndpointAndSuperchainRegistry(dest *DefaultMinimalExternalELSystemIDs, l1CLBeaconRPC, l1ELRPC, l2ELRPC string, l1ChainID eth.ChainID, networkName string) stack.Option[*Orchestrator] {
	chainCfg := chaincfg.ChainByName(networkName)
	if chainCfg == nil {
		panic(fmt.Sprintf("network %s not found in superchain registry", networkName))
	}
	l2ChainID := eth.ChainIDFromUInt64(chainCfg.ChainID)

	ids := NewDefaultMinimalExternalELSystemIDs(l1ChainID, l2ChainID)

	opt := stack.Combine[*Orchestrator]()
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		o.P().Logger().Info("Setting up with superchain registry network", "network", networkName)
	}))

	opt.Add(WithMnemonicKeys(devkeys.TestMnemonic))

	// Skip deployer since we're using external L1 and superchain registry for L2 config
	// Create L1 network record for external L1
	opt.Add(stack.BeforeDeploy(func(o *Orchestrator) {
		chainID, _ := ids.L1.ChainID().Uint64()
		l1Net := &L1Network{
			id: ids.L1,
			genesis: &core.Genesis{
				Config: &params.ChainConfig{
					ChainID: big.NewInt(int64(chainID)),
				},
			},
			blockTime: 12,
		}
		o.l1Nets.Set(ids.L1.ChainID(), l1Net)
	}))

	opt.Add(WithExtL1Nodes(ids.L1EL, ids.L1CL, l1ELRPC, l1CLBeaconRPC))

	// Use superchain registry instead of deployer
	opt.Add(WithL2NetworkFromSuperchainRegistryWithDependencySet(
		stack.L2NetworkID(l2ChainID),
		networkName,
	))

	// Add SyncTester service with external endpoint
	opt.Add(WithSyncTesterWithExternalEndpoint(ids.SyncTester, l2ELRPC, l2ChainID))

	// Add SyncTesterL2ELNode as the L2EL replacement for real-world EL endpoint
	opt.Add(WithSyncTesterL2ELNode(ids.L2EL, ids.L2EL))
	opt.Add(WithL2CLNode(ids.L2CL, ids.L1CL, ids.L1EL, ids.L2EL))

	opt.Add(stack.Finally(func(orch *Orchestrator) {
		*dest = ids
	}))

	return opt
}
