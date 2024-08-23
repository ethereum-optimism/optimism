package interop

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestInterop(t *testing.T) {
	rec := interopgen.InteropDevRecipe{
		L1ChainID:        900100,
		L2ChainIDs:       []uint64{900200, 900201},
		GenesisTimestamp: uint64(1234567),
	}
	hd, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	worldCfg, err := rec.Build(hd)
	require.NoError(t, err)

	logger := testlog.Logger(t, log.LevelInfo)
	require.NoError(t, worldCfg.Check(logger))

	fa := foundry.OpenArtifactsDir("../../packages/contracts-bedrock/forge-artifacts")
	srcFS := foundry.NewSourceMapFS(os.DirFS("../../packages/contracts-bedrock"))

	worldDeployment, worldOutput, err := interopgen.Deploy(logger, fa, srcFS, worldCfg)
	require.NoError(t, err)

	_ = worldDeployment
	_ = worldOutput

	/* TODO: refactor E2E beacon setup
	// Create a fake Beacon node to hold on to blobs created by the L1 miner, and to serve them to L2
	bcn := fakebeacon.NewBeacon(testlog.Logger(t, log.LevelInfo).New("role", "l1_cl"),
		path.Join(cfg.BlobsPath, "l1_cl"), l1Genesis.Timestamp, cfg.DeployConfig.L1BlockTime)
	t.Cleanup(func() {
		_ = bcn.Close()
	})
	require.NoError(t, bcn.Start("127.0.0.1:0"))
	beaconApiAddr := bcn.BeaconAddr()
	require.NotEmpty(t, beaconApiAddr, "beacon API listener must be up")
	sys.L1BeaconAPIAddr = beaconApiAddr
	*/

	/* TODO refactor E2E L1 EL setup
	l1Node, l1Backend, err := geth.InitL1(cfg.DeployConfig.L1ChainID,
		cfg.DeployConfig.L1BlockTime, cfg.L1FinalizedDistance, l1Genesis, c,
		path.Join(cfg.BlobsPath, "l1_el"), bcn, cfg.GethOptions[RoleL1]...)
	if err != nil {
		return nil, err
	}
	sys.EthInstances[RoleL1] = &GethInstance{
		Backend: l1Backend,
		Node:    l1Node,
	}
	err = l1Node.Start()
	if err != nil {
		return nil, err
	}
	*/

	/* TODO refactor E2E L2 setup
	node, backend, err := geth.InitL2(name, big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, cfg.JWTFilePath, cfg.GethOptions[name]...)
	if err != nil {
		return nil, err
	}
	gethInst := &GethInstance{
		Backend: backend,
		Node:    node,
	}
	err = gethInst.Node.Start()
	if err != nil {
		return nil, err
	}
	*/

	/* TODO refactor op-e2e op-node setup
	configureL1(nodeCfg, sys.EthInstances[RoleL1], sys.L1BeaconEndpoint())
	configureL2(nodeCfg, sys.EthInstances[name], cfg.JWTSecret)
	if sys.RollupConfig.EcotoneTime != nil {
		nodeCfg.Beacon = &rollupNode.L1BeaconEndpointConfig{BeaconAddr: sys.L1BeaconAPIAddr}
	}

	var cycle cliapp.Lifecycle
	c.Cancel = func(errCause error) {
		l.Warn("node requested early shutdown!", "err", errCause)
		go func() {
			postCtx, postCancel := context.WithCancel(context.Background())
			postCancel() // don't allow the stopping to continue for longer than needed
			if err := cycle.Stop(postCtx); err != nil {
				t.Error(err)
			}
			l.Warn("closed op-node!")
		}()
	}
	node, err := rollupNode.New(context.Background(), &c, l, "", metrics.NewMetrics(""))
	if err != nil {
		return nil, err
	}
	cycle = node
	err = node.Start(context.Background())
	if err != nil {
		return nil, err
	}
	sys.RollupNodes[name] = node
	*/

	// TODO op-proposer

	// TODO op-batcher

	// TODO op-supervisor
}
