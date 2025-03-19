package system2

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSystemTypes is a quick test for type-checking, ensuring the system shims can all be composed, without dialing any actual services or hydrating configs.
func TestSystemTypes(t *testing.T) {
	logger := testlog.Logger(t, log.LevelInfo)

	setup := &Setup{
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

	l1Chain := NewL1Chain(L1ChainConfig{
		ChainConfig: ChainConfig{
			CommonConfig: setup.CommonConfig(),
			ChainCfg:     &params.ChainConfig{ChainID: big.NewInt(900)},
		},
		ID: L1ChainID{Key: "devnet", ChainID: eth.ChainIDFromUInt64(900)},
	})
	setup.System.AddL1Chain(l1Chain)

	l1EL := NewL1ELNode(L1ELNodeConfig{
		ELNodeConfig: ELNodeConfig{
			CommonConfig: setup.CommonConfig(),
			Client:       nil,
			ChainID:      l1Chain.ChainID(),
		},
		ID: L1ELNodeID{Key: "miner", ChainID: l1Chain.ID().ChainID},
	})
	l1CL := NewL1CLNode(L1CLNodeConfig{
		ID:           L1CLNodeID{Key: "miner", ChainID: l1Chain.ID().ChainID},
		CommonConfig: setup.CommonConfig(),
		Client:       nil,
	})
	l1Chain.AddL1ELNode(l1EL)
	l1Chain.AddL1CLNode(l1CL)

	priv, err := crypto.GenerateKey()
	require.NoError(t, err)
	user := NewUser(UserConfig{
		CommonConfig: setup.CommonConfig(),
		ID:           UserID{Key: "user", ChainID: l1Chain.ID().ChainID},
		Priv:         priv,
		EL:           l1EL,
	})
	setup.System.AddUser(user)

	superchain := NewSuperchain(SuperchainConfig{
		CommonConfig: setup.CommonConfig(),
		ID:           SuperchainID("devnet"),
	})
	setup.System.AddSuperchain(superchain)

	cluster := NewCluster(ClusterConfig{
		CommonConfig: setup.CommonConfig(),
		ID:           L2ClusterID("devnet"),
		DepSet:       nil,
	})
	setup.System.AddCluster(cluster)

	addL2 := func(chainID eth.ChainID) {
		l1ChainID := l1Chain.ChainID()
		l2Chain := NewL2Chain(L2ChainConfig{
			ChainConfig: ChainConfig{
				CommonConfig: setup.CommonConfig(),
				ChainCfg:     &params.ChainConfig{ChainID: chainID.ToBig()},
			},
			ID: L2ChainID{Key: "devnet", ChainID: chainID},
			RollupConfig: &rollup.Config{
				L1ChainID: l1ChainID.ToBig(),
				L2ChainID: chainID.ToBig(),
			},
			Deployment: nil,
			Keys:       nil,
			Superchain: nil,
			L1:         l1Chain,
			Cluster:    nil,
		})
		setup.System.AddL2Chain(l2Chain)

		l2EL := NewL2ELNode(L2ELNodeConfig{
			ELNodeConfig: ELNodeConfig{
				CommonConfig: setup.CommonConfig(),
				Client:       nil,
				ChainID:      l2Chain.ChainID(),
			},
			ID: L2ELNodeID{Key: "sequencer", ChainID: l2Chain.ID().ChainID},
		})
		l2Chain.AddL2ELNode(l2EL)
		l2CL := NewL2CLNode(L2CLNodeConfig{
			ID:           L2CLNodeID{Key: "sequencer", ChainID: l2Chain.ID().ChainID},
			CommonConfig: setup.CommonConfig(),
			Client:       nil,
		})
		l2Chain.AddL2CLNode(l2CL)

		l2Batcher := NewL2Batcher(L2BatcherConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           L2BatcherID{Key: "main", ChainID: l2Chain.ID().ChainID},
		})
		l2Chain.AddL2Batcher(l2Batcher)

		l2Proposer := NewL2Proposer(L2ProposerConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           L2ProposerID{Key: "main", ChainID: l2Chain.ID().ChainID},
		})
		l2Chain.AddL2Proposer(l2Proposer)

		l2Challenger := NewL2Challenger(L2ChallengerConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           L2ChallengerID{Key: "main", ChainID: l2Chain.ID().ChainID},
		})
		l2Chain.AddL2Challenger(l2Challenger)
	}

	addL2(eth.ChainIDFromUInt64(1000))
	addL2(eth.ChainIDFromUInt64(1001))

	l2Chains := setup.System.L2Chains()
	require.Equal(t, 2, len(l2Chains))
	require.Equal(t, eth.ChainIDFromUInt64(1000), l2Chains[0].ChainID)
	require.Equal(t, eth.ChainIDFromUInt64(1001), l2Chains[1].ChainID)

	l1Chains := setup.System.L1Chains()
	require.Equal(t, 1, len(l1Chains))
	require.Equal(t, l1Chain.ChainID(), l1Chains[0].ChainID)

	require.Len(t, l1Chain.L1ELNodes(), 1)
	require.Len(t, l1Chain.L1CLNodes(), 1)
	l1EL.Logger().Info("L1 EL Node")
	l1CL.Logger().Info("L1 CL Node")

	l2ChainA := setup.System.L2Chain(l2Chains[0])
	require.Len(t, l2ChainA.L2ELNodes(), 1)
	require.Len(t, l2ChainA.L2CLNodes(), 1)
	require.Len(t, l2ChainA.L2Batchers(), 1)
	require.Len(t, l2ChainA.L2Proposers(), 1)
	require.Len(t, l2ChainA.L2Challengers(), 1)

	l2ChainB := setup.System.L2Chain(l2Chains[1])
	require.Len(t, l2ChainB.L2ELNodes(), 1)
	require.Len(t, l2ChainB.L2CLNodes(), 1)
	require.Len(t, l2ChainB.L2Batchers(), 1)
	require.Len(t, l2ChainB.L2Proposers(), 1)
	require.Len(t, l2ChainB.L2Challengers(), 1)

	require.Equal(t, eth.ChainIDFromUInt64(1000), eth.ChainIDFromBig(l2ChainA.ChainConfig().ChainID))
	require.Equal(t, eth.ChainIDFromUInt64(1001), eth.ChainIDFromBig(l2ChainB.ChainConfig().ChainID))

	batcher := l2ChainA.L2Batcher(l2ChainA.L2Batchers()[0])
	batcher.Logger().Info("batcher")
	proposer := l2ChainA.L2Proposer(l2ChainA.L2Proposers()[0])
	proposer.Logger().Info("proposer")
	challenger := l2ChainA.L2Challenger(l2ChainA.L2Challengers()[0])
	challenger.Logger().Info("challenger")

	clNode := l2ChainA.L2CLNode(l2ChainA.L2CLNodes()[0])
	clNode.Logger().Info("L2 CL Node")

	elNode := l2ChainA.L2ELNode(l2ChainA.L2ELNodes()[0])
	elNode.Logger().Info("L2 EL Node")
}
