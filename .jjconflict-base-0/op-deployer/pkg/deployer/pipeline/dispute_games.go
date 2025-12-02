package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func DeployAdditionalDisputeGames(
	env *Env,
	intent *state.Intent,
	st *state.State,
	chainID common.Hash,
) error {
	lgr := env.Logger.New("stage", "deploy-additional-dispute-games")

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	thisState, err := st.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain state: %w", err)
	}

	if !shouldDeployAdditionalDisputeGames(thisIntent, thisState) {
		lgr.Info("additional dispute games deployment not needed")
		return nil
	}

	if thisIntent.Roles.L1ProxyAdminOwner != env.Deployer {
		return fmt.Errorf("cannot deploy additional dispute games when deployer is not L1PAO")
	}

	for _, game := range thisIntent.AdditionalDisputeGames {
		if err := deployDisputeGame(env, st, thisIntent, thisState, game); err != nil {
			return fmt.Errorf("failed to deploy additional dispute game: %w", err)
		}
	}

	return nil
}

func deployDisputeGame(
	env *Env,
	st *state.State,
	thisIntent *state.ChainIntent,
	thisState *state.ChainState,
	game state.AdditionalDisputeGame,
) error {
	lgr := env.Logger.New("gameType", game.DisputeGameType)

	lgr.Info("deploying VM", "vmType", game.VMType)
	var vmAddr common.Address
	switch game.VMType {
	case state.VMTypeAlphabet:
		deployAlphabetVM, err := opcm.NewDeployAlphabetVMScript(env.L1ScriptHost)
		if err != nil {
			return fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
		}

		out, err := deployAlphabetVM.Run(opcm.DeployAlphabetVMInput{
			AbsolutePrestate: game.DisputeAbsolutePrestate,
			PreimageOracle:   st.ImplementationsDeployment.PreimageOracleImpl,
		})
		if err != nil {
			return fmt.Errorf("failed to deploy Alphabet VM: %w", err)
		}
		vmAddr = out.AlphabetVM
	case state.VMTypeCannon, state.VMTypeCannonNext:
		out, err := opcm.DeployMIPS(env.L1ScriptHost, opcm.DeployMIPSInput{
			MipsVersion:    game.VMType.MipsVersion(),
			PreimageOracle: st.ImplementationsDeployment.PreimageOracleImpl,
		})
		if err != nil {
			return fmt.Errorf("failed to deploy MIPS VM: %w", err)
		}
		vmAddr = out.MipsSingleton
	default:
		return fmt.Errorf("unsupported VM type: %v", game.VMType)
	}
	lgr.Info("vm deployed", "vmAddr", vmAddr)

	var gameArgs []byte
	args := gameargs.GameArgs{
		AbsolutePrestate:    game.DisputeAbsolutePrestate,
		Vm:                  vmAddr,
		AnchorStateRegistry: thisState.OpChainContracts.AnchorStateRegistryProxy,
		Weth:                thisState.OpChainContracts.DelayedWethPermissionedGameProxy,
		L2ChainID:           eth.ChainIDFromBytes32(thisIntent.ID),
		Proposer:            thisIntent.Roles.Proposer,
		Challenger:          thisIntent.Roles.Challenger,
	}
	if game.DisputeGameType == uint32(gameTypes.PermissionedGameType) {
		gameArgs = args.PackPermissioned()
	} else {
		gameArgs = args.PackPermissionless()
	}

	lgr.Info("deploying dispute game")

	out, err := env.Scripts.DeployDisputeGame.Run(
		opcm.DeployDisputeGameInput{
			Release:                  "dev",
			VmAddress:                vmAddr,
			GameKind:                 "FaultDisputeGame",
			GameType:                 game.DisputeGameType,
			AbsolutePrestate:         game.DisputeAbsolutePrestate,
			MaxGameDepth:             new(big.Int).SetUint64(game.DisputeMaxGameDepth),
			SplitDepth:               new(big.Int).SetUint64(game.DisputeSplitDepth),
			ClockExtension:           game.DisputeClockExtension,
			MaxClockDuration:         game.DisputeMaxClockDuration,
			DelayedWethProxy:         thisState.OpChainContracts.DelayedWethPermissionedGameProxy,
			AnchorStateRegistryProxy: thisState.OpChainContracts.AnchorStateRegistryProxy,
			L2ChainId:                new(big.Int).SetBytes(thisIntent.ID[:]),
			Proposer:                 thisIntent.Roles.Proposer,
			Challenger:               thisIntent.Roles.Challenger,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to deploy dispute game: %w", err)
	}
	lgr.Info("dispute game deployed", "impl", out.DisputeGameImpl)

	lgr.Info("setting dispute game impl on factory", "respected", game.MakeRespected)
	sdgiInput := opcm.SetDisputeGameImplInput{
		Factory:             thisState.OpChainContracts.DisputeGameFactoryProxy,
		Impl:                out.DisputeGameImpl,
		GameType:            game.DisputeGameType,
		GameArgs:            gameArgs,
		AnchorStateRegistry: common.Address{},
	}
	if game.MakeRespected {
		sdgiInput.AnchorStateRegistry = thisState.OpChainContracts.AnchorStateRegistryProxy
	}
	if err := opcm.SetDisputeGameImpl(
		env.L1ScriptHost,
		sdgiInput,
	); err != nil {
		return fmt.Errorf("failed to set dispute game impl: %w", err)
	}

	thisState.AdditionalDisputeGames = append(thisState.AdditionalDisputeGames, state.AdditionalDisputeGameState{
		GameType:      game.DisputeGameType,
		VMType:        game.VMType,
		GameAddress:   out.DisputeGameImpl,
		OracleAddress: st.ImplementationsDeployment.PreimageOracleImpl,
		VMAddress:     vmAddr,
	})

	return nil
}

func shouldDeployAdditionalDisputeGames(thisIntent *state.ChainIntent, thisState *state.ChainState) bool {
	if len(thisIntent.AdditionalDisputeGames) == 0 {
		return false
	}

	if len(thisState.AdditionalDisputeGames) > 0 {
		return false
	}

	return true
}
