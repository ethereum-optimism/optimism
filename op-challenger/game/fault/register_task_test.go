package fault

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/registry"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCannonRegisterTask_BottomPrestateProvider(t *testing.T) {
	// A base URL that never resolves a prestate, matching a challenger configured with --prestates-url
	// where the (placeholder) prestate for a permissioned game is not published.
	baseURL, err := url.Parse("file:///nonexistent-prestates/")
	require.NoError(t, err)
	newCfg := func(t *testing.T) *config.Config {
		return &config.Config{
			Datadir:                       t.TempDir(),
			Cannon:                        vm.Config{VmType: gameTypes.CannonGameType},
			CannonAbsolutePreStateBaseURL: baseURL,
		}
	}
	requiredPrestate := common.Hash{0xaa}

	t.Run("permissioned game uses placeholder prestate without loading it", func(t *testing.T) {
		cfg := newCfg(t)
		task := NewCannonRegisterTask(gameTypes.PermissionedGameType, cfg, metrics.NoopMetrics, nil, nil, nil, nil)
		provider, err := task.getBottomPrestateProvider(context.Background(), requiredPrestate)
		require.NoError(t, err)
		vmProvider, ok := provider.(*vm.PrestateProvider)
		require.True(t, ok)
		require.Empty(t, vmProvider.PrestatePath())
		// No load is ever attempted, so the prestates dir the downloader would create must not exist
		require.NoDirExists(t, filepath.Join(cfg.Datadir, "cannon-prestates"))
	})

	t.Run("cannon game requires prestate", func(t *testing.T) {
		cfg := newCfg(t)
		task := NewCannonRegisterTask(gameTypes.CannonGameType, cfg, metrics.NoopMetrics, nil, nil, nil, nil)
		_, err := task.getBottomPrestateProvider(context.Background(), requiredPrestate)
		require.ErrorIs(t, err, prestates.ErrPrestateUnavailable)
	})
}

func TestRegisterOracle_MissingGameImpl(t *testing.T) {
	// Test versions with and without game args support
	for _, factoryVersion := range []string{"1.2.0", "1.3.0"} {
		t.Run(factoryVersion, func(t *testing.T) {
			gameFactoryAddr := common.Address{0xaa}
			rpc := test.NewAbiBasedRpc(t, gameFactoryAddr, snapshots.LoadDisputeGameFactoryABI())
			rpc.SetResponse(gameFactoryAddr, "version", rpcblock.Latest, nil, []interface{}{factoryVersion})
			m := metrics.NoopMetrics
			caller := batching.NewMultiCaller(rpc, batching.DefaultBatchSize)
			gameFactory, err := contracts.NewDisputeGameFactoryContract(context.Background(), m, gameFactoryAddr, caller)
			require.NoError(t, err)

			logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
			oracles := registry.NewOracleRegistry()
			gameType := gameTypes.CannonGameType

			rpc.SetResponse(gameFactoryAddr, "gameImpls", rpcblock.Latest, []interface{}{gameType}, []interface{}{common.Address{}})

			err = registerOracle(context.Background(), logger, oracles, gameFactory, gameType)
			require.NoError(t, err)
			require.NotNil(t, logs.FindLog(
				testlog.NewMessageFilter("No game implementation set for game type"),
				testlog.NewAttributesFilter("gameType", gameType.String())))
		})
	}
}

func TestRegisterOracle_AddsOracle(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		supportGameArgs bool
		useGameArgs     bool
	}{
		{
			name:            "pre-game args support",
			version:         "1.2.0",
			supportGameArgs: false,
			useGameArgs:     false,
		},
		{
			name:            "game args supported but not used",
			version:         "1.3.0",
			supportGameArgs: true,
			useGameArgs:     false,
		},
		{
			name:            "game args supported and used",
			version:         "1.3.0",
			supportGameArgs: true,
			useGameArgs:     true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			for _, gameType := range []gameTypes.GameType{gameTypes.CannonGameType, gameTypes.SuperCannonKonaGameType} {
				t.Run(fmt.Sprintf("%v", gameType), func(t *testing.T) {
					gameFactoryAddr := common.Address{0xaa}
					gameImplAddr := common.Address{0xbb}
					vmAddr := common.Address{0xcc}
					oracleAddr := common.Address{0xdd}
					rpc := test.NewAbiBasedRpc(t, gameFactoryAddr, snapshots.LoadDisputeGameFactoryABI())
					rpc.SetResponse(gameFactoryAddr, "version", rpcblock.Latest, nil, []interface{}{testCase.version})
					if gameType == gameTypes.CannonGameType {
						rpc.AddContract(gameImplAddr, snapshots.LoadFaultDisputeGameABI())
					} else if gameType == gameTypes.SuperCannonKonaGameType {
						rpc.AddContract(gameImplAddr, snapshots.LoadSuperFaultDisputeGameABI())
					} else {
						t.Fatalf("game type %v not supported", gameType)
					}
					rpc.AddContract(vmAddr, snapshots.LoadMIPSABI())
					rpc.AddContract(oracleAddr, snapshots.LoadPreimageOracleABI())
					m := metrics.NoopMetrics
					caller := batching.NewMultiCaller(rpc, batching.DefaultBatchSize)
					gameFactory, err := contracts.NewDisputeGameFactoryContract(context.Background(), m, gameFactoryAddr, caller)
					require.NoError(t, err)

					if testCase.useGameArgs {
						gameArgs := gameargs.GameArgs{
							AbsolutePrestate:    common.Hash{1},
							Vm:                  vmAddr,
							AnchorStateRegistry: common.Address{3},
							Weth:                common.Address{4},
							L2ChainID:           eth.ChainID{5},
							Proposer:            common.Address{6},
							Challenger:          common.Address{7},
						}.PackPermissionless()
						rpc.SetResponse(gameFactoryAddr, "gameArgs", rpcblock.Latest, []interface{}{gameType}, []interface{}{gameArgs})
					} else if testCase.supportGameArgs {
						rpc.SetResponse(gameFactoryAddr, "gameArgs", rpcblock.Latest, []interface{}{gameType}, []interface{}{[]byte{}})
					}

					logger := testlog.Logger(t, log.LvlInfo)
					oracles := registry.NewOracleRegistry()

					// Use the latest v1 of these contracts. Doesn't have to be an exact match for the version.
					rpc.SetResponse(gameImplAddr, "version", rpcblock.Latest, []interface{}{}, []interface{}{"1.100.0"})
					rpc.SetResponse(oracleAddr, "version", rpcblock.Latest, []interface{}{}, []interface{}{"1.100.0"})

					rpc.SetResponse(gameFactoryAddr, "gameImpls", rpcblock.Latest, []interface{}{gameType}, []interface{}{gameImplAddr})
					if !testCase.useGameArgs {
						// Can only get the vm address from the implementation contract if game args aren't being used
						rpc.SetResponse(gameImplAddr, "vm", rpcblock.Latest, []interface{}{}, []interface{}{vmAddr})
					}
					rpc.SetResponse(vmAddr, "oracle", rpcblock.Latest, []interface{}{}, []interface{}{oracleAddr})

					rpc.SetResponse(gameImplAddr, "gameType", rpcblock.Latest, []interface{}{}, []interface{}{uint32(gameType)})

					err = registerOracle(context.Background(), logger, oracles, gameFactory, gameType)
					require.NoError(t, err)
					registered := oracles.Oracles()
					require.Len(t, registered, 1)
					require.Equal(t, oracleAddr, registered[0].Addr())
				})
			}
		})
	}
}

func TestNewCannonKonaRegisterTask_UsesCannonKonaVMForPrestateConversion(t *testing.T) {
	cannonHash := common.Hash{0xca}
	cannonKonaHash := common.Hash{0xcb}
	prestatePath := filepath.Join(t.TempDir(), "prestate.bin.gz")
	cfg := &config.Config{
		Datadir:                    t.TempDir(),
		Cannon:                     vm.Config{VmBin: buildFakeCannonVM(t, cannonHash)},
		CannonKona:                 vm.Config{VmType: gameTypes.CannonKonaGameType, VmBin: buildFakeCannonVM(t, cannonKonaHash)},
		CannonKonaAbsolutePreState: prestatePath,
	}

	task := NewCannonKonaRegisterTask(gameTypes.CannonKonaGameType, cfg, metrics.NoopMetrics, nil, nil, nil, nil)
	provider, err := task.getBottomPrestateProvider(context.Background(), cannonKonaHash)
	require.NoError(t, err)

	actual, err := provider.AbsolutePreStateCommitment(context.Background())
	require.NoError(t, err)
	require.Equal(t, cannonKonaHash, actual)
}

func buildFakeCannonVM(t *testing.T, witnessHash common.Hash) string {
	t.Helper()

	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "main.go")
	binaryPath := filepath.Join(dir, "fake-cannon")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	payload := fmt.Sprintf(`{"witnessHash":%q,"witness":"0x0102","step":1,"exited":false}`, witnessHash.Hex())
	source := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 4 || os.Args[1] != "witness" || os.Args[2] != "--input" || os.Args[3] == "" {
		fmt.Fprintln(os.Stderr, "expected witness --input <path>")
		os.Exit(2)
	}
	fmt.Print(%q)
}
`, payload)
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o600))
	output, err := exec.Command("go", "build", "-o", binaryPath, sourcePath).CombinedOutput()
	require.NoError(t, err, string(output))
	return binaryPath
}
