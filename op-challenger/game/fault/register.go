package fault

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/client"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/claims"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type CloseFunc func()

type Registry interface {
	RegisterGameType(gameType gameTypes.GameType, creator scheduler.PlayerCreator)
	RegisterBondContract(gameType gameTypes.GameType, creator claims.BondContractCreator)
}

type OracleRegistry interface {
	RegisterOracle(oracle keccakTypes.LargePreimageOracle)
}

type PrestateSource interface {
	// PrestatePath returns the path to the prestate file to use for the game.
	// The provided prestateHash may be used to differentiate between different states but no guarantee is made that
	// the returned prestate matches the supplied hash.
	PrestatePath(ctx context.Context, prestateHash common.Hash) (string, error)
}

func RegisterGameTypes(
	ctx context.Context,
	systemClock clock.Clock,
	l1Clock faultTypes.ClockReader,
	logger log.Logger,
	m metrics.Metricer,
	cfg *config.Config,
	registry Registry,
	oracles OracleRegistry,
	txSender TxSender,
	gameFactory *contracts.DisputeGameFactoryContract,
	clients *client.Provider,
	selective bool,
	claimants []common.Address,
) error {
	var registerTasks []*RegisterTask
	if cfg.GameTypeEnabled(gameTypes.CannonGameType) {
		l2HeaderSource, rollupClient, syncValidator, err := clients.SingleChainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewCannonRegisterTask(gameTypes.CannonGameType, cfg, m, vm.NewOpProgramServerExecutor(logger), l2HeaderSource, rollupClient, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.CannonKonaGameType) {
		l2HeaderSource, rollupClient, syncValidator, err := clients.SingleChainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewCannonKonaRegisterTask(gameTypes.CannonKonaGameType, cfg, m, vm.NewKonaExecutor(), l2HeaderSource, rollupClient, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.SuperCannonGameType) {
		rootProvider, superNodeProvider, syncValidator, err := clients.SuperchainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewSuperCannonRegisterTask(gameTypes.SuperCannonGameType, cfg, m, vm.NewOpProgramServerExecutor(logger), rootProvider, superNodeProvider, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.SuperCannonKonaGameType) {
		rootProvider, superNodeProvider, syncValidator, err := clients.SuperchainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewSuperCannonKonaRegisterTask(gameTypes.SuperCannonKonaGameType, cfg, m, vm.NewKonaSuperExecutor(), rootProvider, superNodeProvider, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.PermissionedGameType) {
		l2HeaderSource, rollupClient, syncValidator, err := clients.SingleChainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewCannonRegisterTask(gameTypes.PermissionedGameType, cfg, m, vm.NewOpProgramServerExecutor(logger), l2HeaderSource, rollupClient, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.SuperPermissionedGameType) {
		rootProvider, superNodeProvider, syncValidator, err := clients.SuperchainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewSuperCannonRegisterTask(gameTypes.SuperPermissionedGameType, cfg, m, vm.NewOpProgramServerExecutor(logger), rootProvider, superNodeProvider, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.FastGameType) {
		l2HeaderSource, rollupClient, syncValidator, err := clients.SingleChainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewAlphabetRegisterTask(gameTypes.FastGameType, l2HeaderSource, rollupClient, syncValidator))
	}
	if cfg.GameTypeEnabled(gameTypes.AlphabetGameType) {
		l2HeaderSource, rollupClient, syncValidator, err := clients.SingleChainClients()
		if err != nil {
			return err
		}
		registerTasks = append(registerTasks, NewAlphabetRegisterTask(gameTypes.AlphabetGameType, l2HeaderSource, rollupClient, syncValidator))
	}
	for _, task := range registerTasks {
		if err := task.Register(ctx, registry, oracles, systemClock, l1Clock, logger, m, txSender, gameFactory, clients.MultiCaller(), clients.L1Client(), selective, claimants, cfg.ResponseDelay, cfg.ResponseDelayAfter); err != nil {
			return fmt.Errorf("failed to register %v game type: %w", task.gameType, err)
		}
	}
	return nil
}
