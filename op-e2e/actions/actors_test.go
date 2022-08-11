package actions

import (
	"context"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var etherScalar = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func ether(v uint64) *big.Int {
	return new(big.Int).Mul(new(big.Int).SetUint64(v), etherScalar)
}

func TestActors(t *testing.T) {
	jwtPath := writeDefaultJWT(t)
	log := testlog.Logger(t, log.LvlInfo)

	userSpawner := &UserSpawner{
		log: log,
		rng: rand.New(rand.NewSource(1234)),
	}
	userA := userSpawner.SpawnUser()
	userB := userSpawner.SpawnUser()
	t.Log("user A", userA.address)
	t.Log("user B", userB.address)

	tp := &TestParams{
		MaxSequencerDrift:   40,
		SequencerWindowSize: 120,
		ChannelTimeout:      120,
	}
	dp := MakeDeployParams(t, tp)
	// TODO better addresses/allocation DSL
	alloc := &AllocParams{
		L1Alloc: map[common.Address]core.GenesisAccount{
			dp.Addresses.Batcher: core.GenesisAccount{
				Balance: ether(1e6),
			},
			userA.address: core.GenesisAccount{
				Balance: ether(1e6),
			},
			userB.address: core.GenesisAccount{
				Balance: ether(1e6),
			},
		},
		L2Alloc: map[common.Address]core.GenesisAccount{
			userA.address: core.GenesisAccount{
				Balance: ether(1e6),
			},
			userB.address: core.GenesisAccount{
				Balance: ether(1e6),
			},
		},
	}
	sd := Setup(t, dp, alloc)

	l1BlockTime := uint64(12)
	canonL1 := L1CanonSrc(nil)
	l1Miner := NewL1Miner(log, sd.L1Cfg, l1BlockTime, canonL1)
	l1Cl := ethclient.NewClient(l1Miner.RPCClient())

	l2Eng := NewL2Engine(log, sd.L2Cfg, sd.RollupCfg.Genesis.L2, jwtPath)
	l2Cl := ethclient.NewClient(l2Eng.RPCClient())

	l1Client, err := sources.NewL1Client(l1Miner.RPCClient(), log, nil, sources.L1ClientDefaultConfig(sd.RollupCfg, false))
	require.NoError(t, err)

	engClient, err := sources.NewEngineClient(l2Eng.RPCClient(), log, nil, sources.EngineClientDefaultConfig(sd.RollupCfg))
	require.NoError(t, err)

	l2Seq := NewL2Sequencer(log, l1Client, engClient, sd.RollupCfg)

	batcherCfg := &BatcherCfg{
		MinL1TxSize: 0,
		MaxL1TxSize: 128_000,
		BatcherKey:  dp.Secrets.Batcher,
	}
	batcher := NewL2Batcher(log, sd.RollupCfg, batcherCfg, l2Seq, l1Cl, l2Cl)

	portal, err := bindings.NewOptimismPortal(sd.RollupCfg.DepositContractAddress, l1Cl)
	require.NoError(t, err)

	userEnv := &UserEnvironment{
		l1:             l1Cl,
		l2:             l2Cl,
		l1ChainID:      sd.L1Cfg.Config.ChainID,
		l2ChainID:      sd.L2Cfg.Config.ChainID,
		l1Signer:       types.LatestSigner(sd.L1Cfg.Config),
		l2Signer:       types.LatestSigner(sd.L2Cfg.Config),
		bindingPortal:  portal,
		addressCorpora: CollectAddresses(sd, dp),
	}
	userA.SetEnv(userEnv)
	userB.SetEnv(userEnv)

	actions := []Action{
		// initial state
		l2Seq.actL2PipelineFull,
		// make some l2 txs
		userA.actL2AddTx,
		userB.actL2AddTx,
		// build a l2 block
		l2Seq.actL2StartBlock,
		l2Eng.actL2IncludeTx(userB.address),
		l2Eng.actL2IncludeTx(userA.address),
		l2Seq.actL2EndBlock,
		// batch submit it
		batcher.actL2BatchBuffer,
		batcher.actL2BatchSubmit,
		// include it on l1
		l1Miner.actL1StartBlock,
		l1Miner.actL1IncludeTx(sd.RollupCfg.BatchSenderAddress),
		l1Miner.actL1EndBlock,
		// derive, will make unsafe now safe
		l2Seq.actL2PipelineFull,
	}

	st := &StandardTesting{TestingBase: t}
	ctx := context.Background()
	for i, act := range actions {
		actCtx, cancel := context.WithCancel(ctx)
		st.Reset(actCtx)
		act(st)
		cancel()
		switch st.State() {
		case ActionOK:
			continue
		case ActionInvalid:
			log.Warn("skipping invalid action", "index", i)
			continue
		default:
			t.Fatalf("unrecognized action state %d", st.State())
		}
	}
}
