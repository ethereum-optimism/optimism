package fault

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/registry"
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
			gameType := faultTypes.CannonGameType

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
			for _, gameType := range []faultTypes.GameType{faultTypes.CannonGameType, faultTypes.SuperCannonGameType, faultTypes.SuperAsteriscKonaGameType} {
				t.Run(fmt.Sprintf("%v", gameType), func(t *testing.T) {
					gameFactoryAddr := common.Address{0xaa}
					gameImplAddr := common.Address{0xbb}
					vmAddr := common.Address{0xcc}
					oracleAddr := common.Address{0xdd}
					rpc := test.NewAbiBasedRpc(t, gameFactoryAddr, snapshots.LoadDisputeGameFactoryABI())
					rpc.SetResponse(gameFactoryAddr, "version", rpcblock.Latest, nil, []interface{}{testCase.version})
					if gameType == faultTypes.CannonGameType {
						rpc.AddContract(gameImplAddr, snapshots.LoadFaultDisputeGameABI())
					} else if gameType == faultTypes.SuperCannonGameType || gameType == faultTypes.SuperAsteriscKonaGameType {
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
