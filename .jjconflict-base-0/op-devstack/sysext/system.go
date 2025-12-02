package sysext

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	client "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum/go-ethereum/common"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

func (o *Orchestrator) hydrateSuperchain(sys stack.ExtensibleSystem) {
	env := o.env
	sys.AddSuperchain(shim.NewSuperchain(shim.SuperchainConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           stack.SuperchainID(env.Env.Name),
		Deployment:   newL1AddressBook(sys.T(), env.Env.L1.Addresses),
	}))
}

func (o *Orchestrator) hydrateClustersMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping clusters")
		return
	}

	require := sys.T().Require()
	env := o.env

	depsets := o.env.Env.DepSets

	for _, d := range depsets {
		var depSet depset.StaticConfigDependencySet
		require.NoError(json.Unmarshal(d, &depSet))

		sys.AddCluster(shim.NewCluster(shim.ClusterConfig{
			CommonConfig:  shim.NewCommonConfig(sys.T()),
			ID:            stack.ClusterID(env.Env.Name),
			DependencySet: &depSet,
		}))
	}
}

func (o *Orchestrator) hydrateSupervisorsMaybe(sys stack.ExtensibleSystem) {
	if !o.isInterop() {
		sys.T().Logger().Info("Interop is inactive, skipping supervisors")
		return
	}

	supervisors := make(map[stack.SupervisorID]bool)
	for _, l2 := range o.env.Env.L2 {
		if supervisorService, ok := l2.Services["supervisor"]; ok {
			for _, instance := range supervisorService {
				id := stack.SupervisorID(instance.Name)
				if supervisors[id] {
					// each supervisor appears in multiple L2s (covering the dependency set),
					// so we need to deduplicate
					continue
				}
				supervisors[id] = true
				sys.AddSupervisor(shim.NewSupervisor(shim.SupervisorConfig{
					CommonConfig: shim.NewCommonConfig(sys.T()),
					ID:           id,
					Client:       o.rpcClient(sys.T(), instance, RPCProtocol, "/"),
				}))
			}
		}
	}
}

func (o *Orchestrator) hydrateTestSequencersMaybe(sys stack.ExtensibleSystem) {
	sequencers := make(map[stack.TestSequencerID]bool)

	// Collect all L2 chain IDs and the shared JWT secret
	var (
		chainIDs []eth.ChainID
		jwt      string
	)

	for _, l2 := range o.env.Env.L2 {
		chainID, _ := eth.ChainIDFromString(l2.Chain.ID)
		chainIDs = append(chainIDs, chainID)
		jwt = l2.JWT
	}

	opts := []client.RPCOption{
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(common.HexToHash(jwt)))),
	}

	for _, l2 := range o.env.Env.L2 {
		if sequencerService, ok := l2.Services["test-sequencer"]; ok {
			for _, instance := range sequencerService {
				id := stack.TestSequencerID(instance.Name)
				if sequencers[id] {
					// Each test_sequencer appears in multiple L2s
					// So we need to deduplicate
					continue
				}
				sequencers[id] = true

				cc := make(map[eth.ChainID]client.RPC, len(chainIDs))
				for _, chainID := range chainIDs {
					cc[chainID] = o.rpcClient(
						sys.T(),
						instance,
						RPCProtocol,
						fmt.Sprintf("/sequencers/sequencer-%s", chainID.String()),
						opts...,
					)
				}

				sys.AddTestSequencer(shim.NewTestSequencer(shim.TestSequencerConfig{
					CommonConfig:   shim.NewCommonConfig(sys.T()),
					ID:             id,
					Client:         o.rpcClient(sys.T(), instance, RPCProtocol, "/", opts...),
					ControlClients: cc,
				}))
			}
		}
	}
}
