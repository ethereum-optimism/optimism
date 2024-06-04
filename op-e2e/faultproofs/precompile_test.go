package faultproofs

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
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
	// precompile test vectors copied from go-ethereum
	tests := []struct {
		name    string
		address common.Address
		input   []byte
	}{
		{
			name:    "ecrecover",
			address: common.BytesToAddress([]byte{0x01}),
			input:   common.FromHex("18c547e4f7b0f325ad1e56f57e26c745b09a3e503d86e00e5255ff7f715d3d1c000000000000000000000000000000000000000000000000000000000000001c73b1693892219d736caba55bdb67216e485557ea6b6af75f37096c9aa6a5a75feeb940b1d03b21e36b0e47e79769f095fe2ab855bd91e3a38756b7d75a9c4549"),
		},
		{
			name:    "sha256",
			address: common.BytesToAddress([]byte{0x02}),
			input:   common.FromHex("68656c6c6f20776f726c64"),
		},
		{
			name:    "ripemd160",
			address: common.BytesToAddress([]byte{0x03}),
			input:   common.FromHex("68656c6c6f20776f726c64"),
		},
		{
			name:    "bn256Pairing",
			address: common.BytesToAddress([]byte{0x08}),
			input:   common.FromHex("1c76476f4def4bb94541d57ebba1193381ffa7aa76ada664dd31c16024c43f593034dd2920f673e204fee2811c678745fc819b55d3e9d294e45c9b03a76aef41209dd15ebff5d46c4bd888e51a93cf99a7329636c63514396b4a452003a35bf704bf11ca01483bfa8b34b43561848d28905960114c8ac04049af4b6315a416782bb8324af6cfc93537a2ad1a445cfd0ca2a71acd7ac41fadbf933c2a51be344d120a2a4cf30c1bf9845f20c6fe39e07ea2cce61f0c9bb048165fe5e4de877550111e129f1cf1097710d41c4ac70fcdfa5ba2023c6ff1cbeac322de49d1b6df7c2032c61a830e3c17286de9462bf242fca2883585b93870a73853face6a6bf411198e9393920d483a7260bfb731fb5d25f1aa493335a9e71297e485b7aef312c21800deef121f1e76426a00665e5c4479674322d4f75edadd46debd5cd992f6ed090689d0585ff075ec9e99ad690c3395bc4b313370b38ef355acdadcd122975b12c85ea5db8c6deb4aab71808dcb408fe3d1e7690c43d37b4ce6cc0166fa7daa"),
		},
		{
			name:    "blake2F",
			address: common.BytesToAddress([]byte{0x09}),
			input:   common.FromHex("0000000048c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5d182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b61626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000300000000000000000000000000000001"),
		},
		{
			name:    "kzgPointEvaluation",
			address: common.BytesToAddress([]byte{0x0a}),
			input:   common.FromHex("01e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a"),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UsesCannon)
			ctx := context.Background()
			cfg := op_e2e.DefaultSystemConfig(t)
			// We don't need a verifier - just the sequencer is enough
			delete(cfg.Nodes, "verifier")
			// Use a small sequencer window size to avoid test timeout while waiting for empty blocks
			// But not too small to ensure that our claim and subsequent state change is published
			cfg.DeployConfig.SequencerWindowSize = 16
			minTs := hexutil.Uint64(0)
			cfg.DeployConfig.L2GenesisDeltaTimeOffset = &minTs
			cfg.DeployConfig.L2GenesisEcotoneTimeOffset = &minTs

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

			receipt := op_e2e.SendL2Tx(t, cfg, l2Seq, aliceKey, func(opts *op_e2e.TxOpts) {
				opts.Gas = 1_000_000
				opts.ToAddr = &test.address
				opts.Nonce = 0
				opts.Data = test.input
			})

			t.Log("Determine L2 claim")
			l2ClaimBlockNumber := receipt.BlockNumber
			l2Output, err := rollupClient.OutputAtBlock(ctx, l2ClaimBlockNumber.Uint64())
			require.NoError(t, err, "could not get expected output")
			l2Claim := l2Output.OutputRoot

			t.Log("Determine L1 head that includes all batches required for L2 claim block")
			require.NoError(t, wait.ForSafeBlock(ctx, rollupClient, l2ClaimBlockNumber.Uint64()))
			l1HeadBlock, err := l1Client.BlockByNumber(ctx, nil)
			require.NoError(t, err, "get l1 head block")
			l1Head := l1HeadBlock.Hash()

			inputs := utils.LocalGameInputs{
				L1Head:        l1Head,
				L2Head:        l2Head,
				L2Claim:       common.Hash(l2Claim),
				L2OutputRoot:  common.Hash(l2OutputRoot),
				L2BlockNumber: l2ClaimBlockNumber,
			}
			runCannon(t, ctx, sys, inputs, "sequencer")
		})
	}
}

func runCannon(t *testing.T, ctx context.Context, sys *op_e2e.System, inputs utils.LocalGameInputs, l2Node string) {
	l1Endpoint := sys.NodeEndpoint("l1")
	l1Beacon := sys.L1BeaconEndpoint()
	rollupEndpoint := sys.RollupEndpoint("sequencer")
	l2Endpoint := sys.NodeEndpoint("sequencer")
	cannonOpts := challenger.WithCannon(t, sys.RollupCfg(), sys.L2Genesis())
	dir := t.TempDir()
	proofsDir := filepath.Join(dir, "cannon-proofs")
	cfg := config.NewConfig(common.Address{}, l1Endpoint, l1Beacon, rollupEndpoint, l2Endpoint, dir)
	cannonOpts(&cfg)

	logger := testlog.Logger(t, log.LevelInfo).New("role", "cannon")
	executor := cannon.NewExecutor(logger, metrics.NoopMetrics, &cfg, cfg.CannonAbsolutePreState, inputs)

	t.Log("Running cannon")
	err := executor.GenerateProof(ctx, proofsDir, math.MaxUint)
	require.NoError(t, err, "failed to generate proof")

	state, err := parseState(filepath.Join(proofsDir, "final.json.gz"))
	require.NoError(t, err, "failed to parse state")
	require.True(t, state.Exited, "cannon did not exit")
	require.Zero(t, state.ExitCode, "cannon failed with exit code %d", state.ExitCode)
	t.Logf("Completed in %d steps", state.Step)
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
