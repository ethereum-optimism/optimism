package faultproofs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestPrecompiles(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon)
	ctx := context.Background()

	tests := []struct {
		name    string
		address common.Address
		input   []byte
	}{
		{
			name:    "pointEvaluation",
			address: common.BytesToAddress([]byte{0x0a}),
			// precompile test vector in go-ethereum
			input: common.Hex2Bytes("01e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a"),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			cfg := op_e2e.DefaultSystemConfig(t)
			// We don't need a verifier - just the sequencer is enough
			delete(cfg.Nodes, "verifier")
			// Use a small sequencer window size to avoid test timeout while waiting for empty blocks
			// But not too small to ensure that our claim and subsequent state change is published
			cfg.DeployConfig.SequencerWindowSize = 16
			minTs := hexutil.Uint64(0)
			cfg.DeployConfig.L2GenesisDeltaTimeOffset = &minTs
			cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &minTs
			cfg.DeployConfig.L2GenesisFjordTimeOffset = &minTs

			sys, err := cfg.Start(t)
			require.Nil(t, err, "Error starting up system")
			defer sys.Close()

			log := testlog.Logger(t, log.LevelInfo)
			log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

			l1Client := sys.Clients["l1"]
			l2Seq := sys.Clients["sequencer"]
			rollupRPCClient, err := rpc.DialContext(context.Background(), sys.RollupNodes["sequencer"].HTTPEndpoint())
			require.Nil(t, err)
			rollupClient := sources.NewRollupClient(client.NewBaseRPCClient(rollupRPCClient))

			aliceKey := cfg.Secrets.Alice

			t.Log("Capture current L2 head as agreed starting point")
			latestBlock, err := l2Seq.BlockByNumber(ctx, nil)
			require.NoError(t, err)
			agreedL2Output, err := rollupClient.OutputAtBlock(ctx, latestBlock.NumberU64())
			require.NoError(t, err, "could not retrieve l2 agreed block")
			l2Head := agreedL2Output.BlockRef.Hash
			l2OutputRoot := agreedL2Output.OutputRoot

			op_e2e.SendL2Tx(t, cfg, l2Seq, aliceKey, func(opts *op_e2e.TxOpts) {
				opts.Gas = 1_000_000
				opts.ToAddr = &test.address
				opts.Value = big.NewInt(1_000)
				opts.Nonce = 0
				opts.Data = test.input
			})

			t.Log("Determine L2 claim")
			l2ClaimBlockNumber, err := l2Seq.BlockNumber(ctx)
			require.NoError(t, err, "get L2 claim block number")
			l2Output, err := rollupClient.OutputAtBlock(ctx, l2ClaimBlockNumber)
			require.NoError(t, err, "could not get expected output")
			l2Claim := l2Output.OutputRoot

			t.Log("Determine L1 head that includes all batches required for L2 claim block")
			require.NoError(t, waitForSafeHead(ctx, l2ClaimBlockNumber, rollupClient))
			l1HeadBlock, err := l1Client.BlockByNumber(ctx, nil)
			require.NoError(t, err, "get l1 head block")
			l1Head := l1HeadBlock.Hash()

			inputs := cannon.LocalGameInputs{
				L1Head:        l1Head,
				L2Head:        l2Head,
				L2Claim:       common.Hash(l2Claim),
				L2OutputRoot:  common.Hash(l2OutputRoot),
				L2BlockNumber: new(big.Int).SetUint64(l2ClaimBlockNumber),
			}
			runCannon(t, ctx, sys, inputs, "sequencer")
		})
	}
}

func runCannon(t *testing.T, ctx context.Context, sys *op_e2e.System, inputs cannon.LocalGameInputs, l2Node string) {
	l1Endpoint := sys.NodeEndpoint("l1")
	l1Beacon := sys.L1BeaconEndpoint()
	cannonOpts := challenger.WithCannon(t, sys.RollupCfg(), sys.L2Genesis(), sys.RollupEndpoint(l2Node), sys.NodeEndpoint(l2Node))
	dir := t.TempDir()
	proofsDir := filepath.Join(dir, "cannon-proofs")
	cfg := config.NewConfig(common.Address{}, l1Endpoint, l1Beacon, dir)
	cannonOpts(&cfg)

	logger := testlog.Logger(t, log.LevelInfo).New("role", "cannon")
	executor := cannon.NewExecutor(logger, metrics.NoopMetrics, &cfg, inputs)

	t.Log("Running cannon")
	err := executor.GenerateProof(ctx, proofsDir, math.MaxUint)
	require.NoError(t, err, "failed to generate proof")

	state, err := parseState(filepath.Join(proofsDir, "final.json.gz"))
	require.NoError(t, err, "failed to parse state")
	t.Logf("Completed in %d steps", state.Step)
}

func waitForSafeHead(ctx context.Context, safeBlockNum uint64, rollupClient *sources.RollupClient) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for {
		seqStatus, err := rollupClient.SyncStatus(ctx)
		if err != nil {
			return err
		}
		if seqStatus.SafeL2.Number >= safeBlockNum {
			return nil
		}
	}
}

func parseState(path string) (*mipsevm.State, error) {
	file, err := ioutil.OpenDecompressed(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open state file (%v): %w", path, err)
	}
	defer file.Close()
	var state mipsevm.State
	err = json.NewDecoder(file).Decode(&state)
	if err != nil {
		return nil, fmt.Errorf("invalid mipsevm state (%v): %w", path, err)
	}
	return &state, nil
}
