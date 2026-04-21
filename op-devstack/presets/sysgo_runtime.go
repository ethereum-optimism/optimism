package presets

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func newL1ELFrontend(t devtest.T, name string, chainID eth.ChainID, userRPC string) *l1ELFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	return newPresetL1ELNode(t, name, chainID, rpcCl)
}

func newL1CLFrontend(t devtest.T, name string, chainID eth.ChainID, beaconHTTPAddr string, lifecycle ...stack.Lifecycle) *l1CLFrontend {
	beaconCl := client.NewBasicHTTPClient(beaconHTTPAddr, t.Logger())
	l1CL := newPresetL1CLNode(t, name, chainID, beaconCl)
	if len(lifecycle) > 0 {
		l1CL.lifecycle = lifecycle[0]
	}
	return l1CL
}

func newL2ELFrontend(t devtest.T, name string, chainID eth.ChainID, userRPC string, engineRPC string, jwtPath string, rollupCfg *rollup.Config, lifecycle ...stack.Lifecycle) *l2ELFrontend {
	userRPCCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(userRPCCl.Close)
	jwtSecret := readJWTSecret(t, jwtPath)
	engineRPCCl, err := client.NewRPC(
		t.Ctx(),
		t.Logger(),
		engineRPC,
		client.WithLazyDial(),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))),
	)
	t.Require().NoError(err)
	t.Cleanup(engineRPCCl.Close)
	l2EL := newPresetL2ELNode(t, name, chainID, userRPCCl, engineRPCCl, rollupCfg)
	if len(lifecycle) > 0 {
		l2EL.lifecycle = lifecycle[0]
	}
	return l2EL
}

func readJWTSecret(t devtest.T, jwtPath string) [32]byte {
	t.Require().NotEmpty(jwtPath, "missing jwt path")
	content, err := os.ReadFile(jwtPath)
	t.Require().NoError(err, "failed to read jwt path %s", jwtPath)
	raw, err := hexutil.Decode(strings.TrimSpace(string(content)))
	t.Require().NoError(err, "failed to decode jwt secret from %s", jwtPath)
	t.Require().Len(raw, 32, "invalid jwt secret length from %s", jwtPath)
	var secret [32]byte
	copy(secret[:], raw)
	return secret
}

func newL2CLFrontend(t devtest.T, name string, chainID eth.ChainID, userRPC string, node sysgo.L2CLNode) *l2CLFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	interopEndpoint, interopJWT := node.InteropRPC()
	l2CL := newPresetL2CLNode(t, name, chainID, rpcCl, userRPC, interopEndpoint, interopJWT)
	if lifecycle, ok := any(node).(stack.Lifecycle); ok {
		l2CL.lifecycle = lifecycle
	}
	return l2CL
}

func newL2BatcherFrontend(t devtest.T, name string, chainID eth.ChainID, rpcEndpoint string) *l2BatcherFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), rpcEndpoint, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	return newPresetL2Batcher(t, name, chainID, rpcCl)
}

func newOPRBuilderFrontend(t devtest.T, name string, chainID eth.ChainID, userRPC string, flashblocksWSURL string, updateRuleSet func(string) error, rollupCfg *rollup.Config, lifecycle ...stack.Lifecycle) *oprBuilderFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)

	t.Require().NotEmpty(flashblocksWSURL, "missing flashblocks ws url for %s", name)
	wsCl, err := client.DialWS(t.Ctx(), client.WSConfig{
		URL: flashblocksWSURL,
		Log: t.Logger(),
	})
	t.Require().NoError(err)

	oprb := newPresetOPRBuilderNode(t, name, chainID, rpcCl, rollupCfg, wsCl, updateRuleSet)
	if len(lifecycle) > 0 {
		oprb.lifecycle = lifecycle[0]
	}
	return oprb
}

func newRollupBoostFrontend(t devtest.T, name string, chainID eth.ChainID, userRPC string, flashblocksWSURL string, rollupCfg *rollup.Config, lifecycle ...stack.Lifecycle) *rollupBoostFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)

	t.Require().NotEmpty(flashblocksWSURL, "missing flashblocks ws url for %s", name)
	wsCl, err := client.DialWS(t.Ctx(), client.WSConfig{
		URL: flashblocksWSURL,
		Log: t.Logger(),
	})
	t.Require().NoError(err)

	rollupBoost := newPresetRollupBoostNode(t, name, chainID, rpcCl, rollupCfg, wsCl)
	if len(lifecycle) > 0 {
		rollupBoost.lifecycle = lifecycle[0]
	}
	return rollupBoost
}

func newSupervisorFrontend(t devtest.T, name string, userRPC string, lifecycle ...stack.Lifecycle) *supervisorFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	supervisor := newPresetSupervisor(t, name, rpcCl)
	if len(lifecycle) > 0 {
		supervisor.lifecycle = lifecycle[0]
	}
	return supervisor
}

func newSupernodeFrontend(t devtest.T, name string, userRPC string) *supernodeFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), userRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	return newPresetSupernode(t, name, rpcCl)
}

func newConductorFrontend(t devtest.T, name string, chainID eth.ChainID, rpcEndpoint string) *conductorFrontend {
	rpcCl, err := rpc.DialContext(t.Ctx(), rpcEndpoint)
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	return newPresetConductor(t, name, chainID, rpcCl)
}

func newTestSequencerFrontend(t devtest.T, name string, adminRPC string, controlRPCs map[eth.ChainID]string, jwtSecret [32]byte) *testSequencerFrontend {
	opts := []client.RPCOption{
		client.WithLazyDial(),
		client.WithGethRPCOptions(rpc.WithHTTPAuth(gn.NewJWTAuth(jwtSecret))),
	}

	adminRPCCl, err := client.NewRPC(t.Ctx(), t.Logger(), adminRPC, opts...)
	t.Require().NoError(err)
	t.Cleanup(adminRPCCl.Close)

	controlClients := make(map[eth.ChainID]client.RPC, len(controlRPCs))
	for chainID, endpoint := range controlRPCs {
		rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), endpoint, opts...)
		t.Require().NoErrorf(err, "failed to create control RPC client for chain %s", chainID)
		t.Cleanup(rpcCl.Close)
		controlClients[chainID] = rpcCl
	}
	return newPresetTestSequencer(t, name, adminRPCCl, controlClients)
}

func newSyncTesterFrontend(t devtest.T, name string, chainID eth.ChainID, syncTesterRPC string) *syncTesterFrontend {
	rpcCl, err := client.NewRPC(t.Ctx(), t.Logger(), syncTesterRPC, client.WithLazyDial())
	t.Require().NoError(err)
	t.Cleanup(rpcCl.Close)
	return newPresetSyncTester(t, name, chainID, syncTesterRPC, rpcCl)
}
