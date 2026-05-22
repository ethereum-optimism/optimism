package pipeline

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/current"
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
		input := opcm.DeployAlphabetVMInput{
			"absolutePrestate": game.DisputeAbsolutePrestate,
			"preimageOracle":   st.ImplementationsDeployment.PreimageOracleImpl,
		}

		var out opcm.DeployAlphabetVMOutput
		var err error

		if env.UseForge {
			lgr.Info("using Forge for DeployAlphabetVM")
			forgeEnv := &opcm.ForgeEnv{
				Client:     env.ForgeClient,
				Context:    env.Context,
				L1RPCUrl:   env.L1RPCUrl,
				PrivateKey: env.PrivateKey,
			}
			out, err = opcm.DeployAlphabetVMViaForge(forgeEnv, input)
			if err != nil {
				return err
			}
		} else {
			deployAlphabetVM, err := opcm.NewDeployAlphabetVMScript(env.L1ScriptHost)
			if err != nil {
				return fmt.Errorf("failed to load DeployAlphabetVM script: %w", err)
			}
			out, err = deployAlphabetVM.Run(input)
			if err != nil {
				return fmt.Errorf("failed to deploy Alphabet VM: %w", err)
			}
		}
		vmAddr = out.Address("alphabetVM")
	case state.VMTypeCannon, state.VMTypeCannonNext:
		if env.UseForge {
			lgr.Info("using Forge for DeployMIPS")
			forgeInput := opcm.DeployMIPSInput{
				"mipsVersion":    new(big.Int).SetUint64(game.VMType.MipsVersion()),
				"preimageOracle": st.ImplementationsDeployment.PreimageOracleImpl,
			}
			forgeEnv := &opcm.ForgeEnv{
				Client:     env.ForgeClient,
				Context:    env.Context,
				L1RPCUrl:   env.L1RPCUrl,
				PrivateKey: env.PrivateKey,
			}
			forgeOut, err := opcm.DeployMIPSViaForge(forgeEnv, forgeInput)
			if err != nil {
				return err
			}
			vmAddr = forgeOut.Address("mipsSingleton")
		} else {
			input := opcm.DeployMIPSInput{
				"mipsVersion":    new(big.Int).SetUint64(game.VMType.MipsVersion()),
				"preimageOracle": st.ImplementationsDeployment.PreimageOracleImpl,
			}
			out, err := opcm.DeployMIPS(env.L1ScriptHost, input)
			if err != nil {
				return fmt.Errorf("failed to deploy MIPS VM: %w", err)
			}
			vmAddr = out.Address("mipsSingleton")
		}
	case state.VMTypeZK:
		zkImpl := st.ImplementationsDeployment.ZkDisputeGameImpl
		if zkImpl == (common.Address{}) {
			return fmt.Errorf("ZkDisputeGameImpl is not deployed; ensure ZKDisputeGameFlag is set in devFeatureBitmap")
		}
		if game.ZKDisputeGame == nil {
			return fmt.Errorf("ZKDisputeGame params must be set when VMType is ZK")
		}
		if game.DisputeGameType != uint32(current.GameTypeZKDisputeGame) {
			return fmt.Errorf("DisputeGameType must be %d for ZK dispute game, got %d", current.GameTypeZKDisputeGame, game.DisputeGameType)
		}
		zk := game.ZKDisputeGame
		if zk.ChallengerBond == nil || zk.ChallengerBond.ToInt().Sign() <= 0 {
			return fmt.Errorf("ZKDisputeGame.ChallengerBond must be set to a positive value")
		}
		gameArgs := gameargs.ZKGameArgs{
			AbsolutePrestate:     zk.AbsolutePrestate,
			Verifier:             zk.Verifier,
			MaxChallengeDuration: zk.MaxChallengeDuration,
			MaxProveDuration:     zk.MaxProveDuration,
			ChallengerBond:       zk.ChallengerBond.ToInt(),
			AnchorStateRegistry:  thisState.OpChainContracts.AnchorStateRegistryProxy,
			Weth:                 thisState.OpChainContracts.DelayedWethPermissionlessGameProxy,
			L2ChainID:            new(big.Int).SetBytes(thisIntent.ID[:]),
		}.Pack()
		zkInput := opcm.SetDisputeGameImplInput{
			Factory:             thisState.OpChainContracts.DisputeGameFactoryProxy,
			Impl:                zkImpl,
			AnchorStateRegistry: common.Address{},
			GameType:            game.DisputeGameType,
			GameArgs:            gameArgs,
		}
		if game.MakeRespected {
			zkInput.AnchorStateRegistry = thisState.OpChainContracts.AnchorStateRegistryProxy
		}
		if err := opcm.SetDisputeGameImpl(env.L1ScriptHost, zkInput); err != nil {
			return fmt.Errorf("failed to set ZK dispute game impl: %w", err)
		}
		thisState.AdditionalDisputeGames = append(thisState.AdditionalDisputeGames, state.AdditionalDisputeGameState{
			GameType:    game.DisputeGameType,
			VMType:      game.VMType,
			GameAddress: zkImpl,
		})
		return nil
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

	input := opcm.DeployDisputeGameInput{
		"release":                  "dev",
		"vmAddress":                vmAddr,
		"gameKind":                 "FaultDisputeGame",
		"gameType":                 game.DisputeGameType,
		"absolutePrestate":         game.DisputeAbsolutePrestate,
		"maxGameDepth":             new(big.Int).SetUint64(game.DisputeMaxGameDepth),
		"splitDepth":               new(big.Int).SetUint64(game.DisputeSplitDepth),
		"clockExtension":           game.DisputeClockExtension,
		"maxClockDuration":         game.DisputeMaxClockDuration,
		"delayedWethProxy":         thisState.OpChainContracts.DelayedWethPermissionedGameProxy,
		"anchorStateRegistryProxy": thisState.OpChainContracts.AnchorStateRegistryProxy,
		"l2ChainId":                new(big.Int).SetBytes(thisIntent.ID[:]),
		"proposer":                 thisIntent.Roles.Proposer,
		"challenger":               thisIntent.Roles.Challenger,
	}

	var out opcm.DeployDisputeGameOutput
	var err error

	if env.UseForge {
		lgr.Info("using Forge for DeployDisputeGame")
		forgeEnv := &opcm.ForgeEnv{
			Client:     env.ForgeClient,
			Context:    env.Context,
			L1RPCUrl:   env.L1RPCUrl,
			PrivateKey: env.PrivateKey,
		}
		out, err = opcm.DeployDisputeGameViaForge(forgeEnv, input)
		if err != nil {
			return err
		}
	} else {
		out, err = env.Scripts.DeployDisputeGame.Run(input)
		if err != nil {
			return fmt.Errorf("failed to deploy dispute game: %w", err)
		}
	}
	disputeGameImpl := out.Address("disputeGameImpl")
	lgr.Info("dispute game deployed", "impl", disputeGameImpl)

	lgr.Info("setting dispute game impl on factory", "respected", game.MakeRespected)
	sdgiInput := opcm.SetDisputeGameImplInput{
		Factory:             thisState.OpChainContracts.DisputeGameFactoryProxy,
		Impl:                disputeGameImpl,
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
		GameAddress:   disputeGameImpl,
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
