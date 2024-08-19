package runner

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/asterisc"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func createTraceProvider(
	logger log.Logger,
	m vm.Metricer,
	cfg *config.Config,
	prestateHash common.Hash,
	traceType types.TraceType,
	localInputs utils.LocalGameInputs,
	dir string,
) (types.TraceProvider, error) {
	switch traceType {
	case types.TraceTypeCannon:
		vmConfig := vm.NewOpProgramServerExecutor()
		prestate, err := getPrestate(prestateHash, cfg.CannonAbsolutePreStateBaseURL, cfg.CannonAbsolutePreState, dir)
		if err != nil {
			return nil, err
		}
		prestateProvider := cannon.NewPrestateProvider(prestate)
		return cannon.NewTraceProvider(logger, m, cfg.Cannon, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	case types.TraceTypeAsterisc:
		vmConfig := vm.NewOpProgramServerExecutor()
		prestate, err := getPrestate(prestateHash, cfg.AsteriscAbsolutePreStateBaseURL, cfg.AsteriscAbsolutePreState, dir)
		if err != nil {
			return nil, err
		}
		prestateProvider := asterisc.NewPrestateProvider(prestate)
		return asterisc.NewTraceProvider(logger, m, cfg.Asterisc, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	case types.TraceTypeAsteriscKona:
		vmConfig := vm.NewKonaServerExecutor()
		prestate, err := getPrestate(prestateHash, cfg.AsteriscAbsolutePreStateBaseURL, cfg.AsteriscAbsolutePreState, dir)
		if err != nil {
			return nil, err
		}
		prestateProvider := asterisc.NewPrestateProvider(prestate)
		return asterisc.NewTraceProvider(logger, m, cfg.Asterisc, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	}
	return nil, errors.New("invalid trace type")
}

func getPrestate(prestateHash common.Hash, prestateBaseUrl *url.URL, prestatePath string, dataDir string) (string, error) {
	prestateSource := prestates.NewPrestateSource(
		prestateBaseUrl,
		prestatePath,
		filepath.Join(dataDir, "prestates"))

	prestate, err := prestateSource.PrestatePath(prestateHash)
	if err != nil {
		return "", fmt.Errorf("failed to get prestate %v: %w", prestateHash, err)
	}
	return prestate, nil
}
