package main

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	contractMetrics "github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/metrics"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/tools"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	GameTypeFlag = &cli.StringFlag{
		Name:    "game-type",
		Usage:   "Game type to create, as a decimal number or a name (e.g. cannon, super-cannon-kona, zk).",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "TRACE_TYPE"),
		Value:   gameTypes.CannonGameType.String(),
	}
	OutputRootFlag = &cli.StringFlag{
		Name:    "output-root",
		Usage:   "The output root for the dispute game. For super and ZK games this is encoded into a single-chain super root proof.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "OUTPUT_ROOT"),
	}
	L2BlockNumFlag = &cli.StringFlag{
		Name:    "l2-block-num",
		Usage:   "The L2 block number for the game. For super and ZK games this is the super root timestamp.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "L2_BLOCK_NUM"),
	}
	L2ChainIDFlag = &cli.StringFlag{
		Name:    "l2-chain-id",
		Usage:   "The L2 chain ID to include in the super root proof.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "L2_CHAIN_ID"),
	}
	CreateGameParentIndexFlag = &cli.Uint64Flag{
		Name:    "parent-index",
		Usage:   "ZK games only: the factory index of the parent game. The default, uint32 max, builds on the anchor state.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ZK_PARENT_INDEX"),
		Value:   math.MaxUint32,
		Action: func(_ *cli.Context, parentIndex uint64) error {
			if parentIndex > math.MaxUint32 {
				return fmt.Errorf("parent index %d exceeds uint32 max", parentIndex)
			}
			return nil
		},
	}
)

// namedGameTypes lists every game type with a name, so create-game can target any factory
// implementation by name, not only the types op-challenger plays.
var namedGameTypes = []gameTypes.GameType{
	gameTypes.CannonGameType,
	gameTypes.PermissionedGameType,
	gameTypes.AsteriscGameType,
	gameTypes.AsteriscKonaGameType,
	gameTypes.SuperPermissionedGameType,
	gameTypes.OPSuccinctGameType,
	gameTypes.SuperAsteriscKonaGameType,
	gameTypes.CannonKonaGameType,
	gameTypes.SuperCannonKonaGameType,
	gameTypes.ZKDisputeGameType,
	gameTypes.FastGameType,
	gameTypes.AlphabetGameType,
	gameTypes.KailuaGameType,
}

func parseGameType(value string) (gameTypes.GameType, error) {
	if number, err := strconv.ParseUint(value, 10, 32); err == nil {
		return gameTypes.GameType(number), nil
	}
	for _, gameType := range namedGameTypes {
		if gameType.String() == value {
			return gameType, nil
		}
	}
	return gameTypes.UnknownGameType, fmt.Errorf("%w: %q", gameTypes.ErrUnknownGameType, value)
}

func CreateGame(ctx *cli.Context) error {
	gameType, err := parseGameType(ctx.String(GameTypeFlag.Name))
	if err != nil {
		return err
	}
	outputRoot := common.HexToHash(ctx.String(OutputRootFlag.Name))
	l2BlockNum := ctx.Uint64(L2BlockNumFlag.Name)
	l2ChainID := ctx.Uint64(L2ChainIDFlag.Name)
	switch gameType {
	case gameTypes.SuperCannonKonaGameType, gameTypes.SuperPermissionedGameType, gameTypes.ZKDisputeGameType:
		// The contracts do not check the chain ID in the super root proof, so a missing or
		// non-numeric value (read as 0) would create a game that loses its bond.
		if l2ChainID == 0 {
			return fmt.Errorf("--%v must be a non-zero chain ID for game type %v", L2ChainIDFlag.Name, gameType)
		}
	}
	parentIndex := uint32(ctx.Uint64(CreateGameParentIndexFlag.Name))

	contract, txMgr, err := NewContractWithTxMgr[*contracts.DisputeGameFactoryContract](ctx, flags.FactoryAddress,
		func(ctx context.Context, metricer contractMetrics.ContractMetricer, address common.Address, caller *batching.MultiCaller) (*contracts.DisputeGameFactoryContract, error) {
			return contracts.NewDisputeGameFactoryContract(ctx, metricer, address, caller)
		})
	if err != nil {
		return fmt.Errorf("failed to create dispute game factory bindings: %w", err)
	}

	creator := tools.NewGameCreator(contract, txMgr)
	gameAddr, err := creator.CreateGame(ctx.Context, outputRoot, uint64(gameType), l2BlockNum, l2ChainID, parentIndex)
	if err != nil {
		return fmt.Errorf("failed to create game: %w", err)
	}
	fmt.Printf("Fetched Game Address: %s\n", gameAddr.String())
	return nil
}

func createGameFlags() []cli.Flag {
	cliFlags := []cli.Flag{
		flags.L1EthRpcFlag,
		flags.NetworkFlag,
		flags.FactoryAddressFlag,
		GameTypeFlag,
		OutputRootFlag,
		L2BlockNumFlag,
		L2ChainIDFlag,
		CreateGameParentIndexFlag,
	}
	cliFlags = append(cliFlags, txmgr.CLIFlagsWithDefaults(flags.EnvVarPrefix, txmgr.DefaultChallengerFlagValues)...)
	cliFlags = append(cliFlags, logcli.CLIFlags(flags.EnvVarPrefix)...)
	return cliFlags
}

var CreateGameCommand = &cli.Command{
	Name:        "create-game",
	Usage:       "Creates a dispute game via the factory",
	Description: "Creates a dispute game via the factory",
	Action:      Interruptible(CreateGame),
	Flags:       createGameFlags(),
}
