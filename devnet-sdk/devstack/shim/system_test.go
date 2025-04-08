package shim

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSystemTypes is a quick test for type-checking, ensuring the system shims can all be composed, without dialing any actual services or hydrating configs.
func TestSystemTypes(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	setup := &stack.Setup{
		Ctx:     context.Background(),
		Log:     logger,
		T:       t,
		Require: require.New(t),
		System: NewSystem(SystemConfig{
			CommonConfig: CommonConfig{
				Log: logger,
				T:   t,
			},
		}),
		Orchestrator: nil,
	}

	l1Net := NewL1Network(L1NetworkConfig{
		NetworkConfig: NetworkConfig{
			CommonConfig: CommonConfigFromSetup(setup),
			ChainConfig:  &params.ChainConfig{ChainID: big.NewInt(900)},
		},
		ID: stack.L1NetworkID{Key: "devnet", ChainID: eth.ChainIDFromUInt64(900)},
	})
	setup.System.AddL1Network(l1Net)

	l1EL := NewL1ELNode(L1ELNodeConfig{
		ELNodeConfig: ELNodeConfig{
			CommonConfig: CommonConfigFromSetup(setup),
			Client:       nil,
			ChainID:      l1Net.ChainID(),
		},
		ID: stack.L1ELNodeID{Key: "miner", ChainID: l1Net.ID().ChainID},
	})
	l1CL := NewL1CLNode(L1CLNodeConfig{
		ID:           stack.L1CLNodeID{Key: "miner", ChainID: l1Net.ID().ChainID},
		CommonConfig: CommonConfigFromSetup(setup),
		Client:       nil,
	})
	l1Net.AddL1ELNode(l1EL)
	l1Net.AddL1CLNode(l1CL)

	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	userA := NewUser(UserConfig{
		CommonConfig: CommonConfigFromSetup(setup),
		ID:           stack.UserID{Key: "userA", ChainID: l1Net.ID().ChainID},
		Priv:         priv,
		EL:           l1EL,
	})
	l1Net.AddUser(userA)

	superchain := NewSuperchain(SuperchainConfig{
		CommonConfig: CommonConfigFromSetup(setup),
		ID:           stack.SuperchainID("devnet"),
	})
	setup.System.AddSuperchain(superchain)

	cluster := NewCluster(ClusterConfig{
		CommonConfig:  CommonConfigFromSetup(setup),
		ID:            stack.ClusterID("devnet"),
		DependencySet: nil,
	})
	setup.System.AddCluster(cluster)

	supervisor := NewSupervisor(SupervisorConfig{
		CommonConfig: CommonConfigFromSetup(setup),
		ID:           stack.SupervisorID("supervisor0"),
		Client:       nil,
	})
	setup.System.AddSupervisor(supervisor)

	addL2 := func(chainID eth.ChainID) {
		l1ChainID := l1Net.ChainID()
		l2Net := NewL2Network(L2NetworkConfig{
			NetworkConfig: NetworkConfig{
				CommonConfig: CommonConfigFromSetup(setup),
				ChainConfig:  &params.ChainConfig{ChainID: chainID.ToBig()},
			},
			ID: stack.L2NetworkID{Key: "devnet", ChainID: chainID},
			RollupConfig: &rollup.Config{
				L1ChainID: l1ChainID.ToBig(),
				L2ChainID: chainID.ToBig(),
			},
			Deployment: nil,
			Keys:       nil,
			Superchain: nil,
			L1:         l1Net,
			Cluster:    nil,
		})
		setup.System.AddL2Network(l2Net)

		l2EL := NewL2ELNode(L2ELNodeConfig{
			ELNodeConfig: ELNodeConfig{
				CommonConfig: CommonConfigFromSetup(setup),
				Client:       nil,
				ChainID:      l2Net.ChainID(),
			},
			ID: stack.L2ELNodeID{Key: "sequencer", ChainID: l2Net.ID().ChainID},
		})
		l2Net.AddL2ELNode(l2EL)
		l2CL := NewL2CLNode(L2CLNodeConfig{
			ID:           stack.L2CLNodeID{Key: "sequencer", ChainID: l2Net.ID().ChainID},
			CommonConfig: CommonConfigFromSetup(setup),
			Client:       nil,
		})
		l2Net.AddL2CLNode(l2CL)

		l2Batcher := NewL2Batcher(L2BatcherConfig{
			CommonConfig: CommonConfigFromSetup(setup),
			ID:           stack.L2BatcherID{Key: "main", ChainID: l2Net.ID().ChainID},
		})
		l2Net.AddL2Batcher(l2Batcher)

		l2Proposer := NewL2Proposer(L2ProposerConfig{
			CommonConfig: CommonConfigFromSetup(setup),
			ID:           stack.L2ProposerID{Key: "main", ChainID: l2Net.ID().ChainID},
		})
		l2Net.AddL2Proposer(l2Proposer)

		l2Challenger := NewL2Challenger(L2ChallengerConfig{
			CommonConfig: CommonConfigFromSetup(setup),
			ID:           stack.L2ChallengerID{Key: "main", ChainID: l2Net.ID().ChainID},
		})
		l2Net.AddL2Challenger(l2Challenger)
	}

	addL2(eth.ChainIDFromUInt64(1000))
	addL2(eth.ChainIDFromUInt64(1001))

	l2Networks := setup.System.L2Networks()
	require.Equal(t, 2, len(l2Networks))
	require.Equal(t, eth.ChainIDFromUInt64(1000), l2Networks[0].ChainID)
	require.Equal(t, eth.ChainIDFromUInt64(1001), l2Networks[1].ChainID)

	l1Networks := setup.System.L1Networks()
	require.Equal(t, 1, len(l1Networks))
	require.Equal(t, l1Net.ChainID(), l1Networks[0].ChainID)

	users := l1Net.Users()
	require.Equal(t, 1, len(users))
	require.Equal(t, userA.ID(), users[0])

	require.Len(t, l1Net.L1ELNodes(), 1)
	require.Len(t, l1Net.L1CLNodes(), 1)
	l1EL.Logger().Info("L1 EL Node")
	l1CL.Logger().Info("L1 CL Node")

	require.Equal(t, supervisor, setup.System.Supervisor(stack.SupervisorID("supervisor0")))
	supervisor.Logger().Info("supervisor is registered")

	l2NetA := setup.System.L2Network(l2Networks[0])
	require.Len(t, l2NetA.L2ELNodes(), 1)
	require.Len(t, l2NetA.L2CLNodes(), 1)
	require.Len(t, l2NetA.L2Batchers(), 1)
	require.Len(t, l2NetA.L2Proposers(), 1)
	require.Len(t, l2NetA.L2Challengers(), 1)

	l2NetB := setup.System.L2Network(l2Networks[1])
	require.Len(t, l2NetB.L2ELNodes(), 1)
	require.Len(t, l2NetB.L2CLNodes(), 1)
	require.Len(t, l2NetB.L2Batchers(), 1)
	require.Len(t, l2NetB.L2Proposers(), 1)
	require.Len(t, l2NetB.L2Challengers(), 1)

	require.Equal(t, eth.ChainIDFromUInt64(1000), eth.ChainIDFromBig(l2NetA.ChainConfig().ChainID))
	require.Equal(t, eth.ChainIDFromUInt64(1001), eth.ChainIDFromBig(l2NetB.ChainConfig().ChainID))

	batcher := l2NetA.L2Batcher(l2NetA.L2Batchers()[0])
	batcher.Logger().Info("batcher")
	proposer := l2NetA.L2Proposer(l2NetA.L2Proposers()[0])
	proposer.Logger().Info("proposer")
	challenger := l2NetA.L2Challenger(l2NetA.L2Challengers()[0])
	challenger.Logger().Info("challenger")

	clNode := l2NetA.L2CLNode(l2NetA.L2CLNodes()[0])
	clNode.Logger().Info("L2 CL Node")

	elNode := l2NetA.L2ELNode(l2NetA.L2ELNodes()[0])
	elNode.Logger().Info("L2 EL Node")
}
