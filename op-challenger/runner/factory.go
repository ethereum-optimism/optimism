package runner

import (
	"context"
	"errors"
	"net/url"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type prestateFetcher interface {
	getPrestate(ctx context.Context, logger log.Logger, prestateBaseUrl *url.URL, prestatePath string, dataDir string, stateConverter vm.StateConverter) (string, error)
}

func createTraceProvider(
	ctx context.Context,
	logger log.Logger,
	m vm.Metricer,
	cfg *config.Config,
	prestateSource prestateFetcher,
	gameType gameTypes.GameType,
	localInputs utils.LocalGameInputs,
	dir string,
) (types.TraceProvider, error) {
	switch gameType {
	case gameTypes.CannonGameType, gameTypes.SuperCannonGameType:
		serverExecutor := vm.NewOpProgramServerExecutor(logger)
		stateConverter := cannon.NewStateConverter(cfg.Cannon)
		prestate, err := prestateSource.getPrestate(ctx, logger, cfg.CannonAbsolutePreStateBaseURL, cfg.CannonAbsolutePreState, dir, stateConverter)
		if err != nil {
			return nil, err
		}
		prestateProvider := vm.NewPrestateProvider(prestate, stateConverter)
		return cannon.NewTraceProvider(logger, m, cfg.Cannon, serverExecutor, prestateProvider, prestate, localInputs, dir, 42), nil
	case gameTypes.CannonKonaGameType, gameTypes.SuperCannonKonaGameType:
		serverExecutor := vm.NewKonaExecutor()
		stateConverter := cannon.NewStateConverter(cfg.CannonKona)
		prestate, err := prestateSource.getPrestate(ctx, logger, cfg.CannonKonaAbsolutePreStateBaseURL, cfg.CannonKonaAbsolutePreState, dir, stateConverter)
		if err != nil {
			return nil, err
		}
		prestateProvider := vm.NewPrestateProvider(prestate, stateConverter)
		return cannon.NewTraceProvider(logger, m, cfg.CannonKona, serverExecutor, prestateProvider, prestate, localInputs, dir, 42), nil
	}
	return nil, errors.New("invalid game type")
}
