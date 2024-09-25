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
		stateConverter := cannon.NewStateConverter()
		prestate, err := getPrestate(prestateHash, cfg.CannonAbsolutePreStateBaseURL, cfg.CannonAbsolutePreState, dir, stateConverter)
		if err != nil {
			return nil, err
		}
		prestateProvider := vm.NewPrestateProvider(prestate, stateConverter)
		return cannon.NewTraceProvider(logger, m, cfg.Cannon, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	case types.TraceTypeAsterisc:
		vmConfig := vm.NewOpProgramServerExecutor()
		stateConverter := asterisc.NewStateConverter()
		prestate, err := getPrestate(prestateHash, cfg.AsteriscAbsolutePreStateBaseURL, cfg.AsteriscAbsolutePreState, dir, stateConverter)
		if err != nil {
			return nil, err
		}
		prestateProvider := vm.NewPrestateProvider(prestate, stateConverter)
		return asterisc.NewTraceProvider(logger, m, cfg.Asterisc, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	case types.TraceTypeAsteriscKona:
		vmConfig := vm.NewKonaExecutor()
		stateConverter := asterisc.NewStateConverter()
		prestate, err := getPrestate(prestateHash, cfg.AsteriscKonaAbsolutePreStateBaseURL, cfg.AsteriscKonaAbsolutePreState, dir, stateConverter)
		if err != nil {
			return nil, err
		}
		prestateProvider := vm.NewPrestateProvider(prestate, stateConverter)
		return asterisc.NewTraceProvider(logger, m, cfg.AsteriscKona, vmConfig, prestateProvider, prestate, localInputs, dir, 42), nil
	}
	return nil, errors.New("invalid trace type")
}

func createMTTraceProvider(
	logger log.Logger,
	m vm.Metricer,
	vmConfig vm.Config,
	prestateHash common.Hash,
	absolutePrestateBaseURL *url.URL,
	traceType types.TraceType,
	localInputs utils.LocalGameInputs,
	dir string,
) (types.TraceProvider, error) {
	executor := vm.NewOpProgramServerExecutor()
	stateConverter := cannon.NewStateConverter()

	prestateSource := prestates.NewMultiPrestateProvider(absolutePrestateBaseURL, filepath.Join(dir, "prestates"), cannon.NewStateConverter())
	prestatePath, err := prestateSource.PrestatePath(prestateHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get prestate %v: %w", prestateHash, err)
	}
	prestateProvider := vm.NewPrestateProvider(prestatePath, stateConverter)
	return cannon.NewTraceProvider(logger, m, vmConfig, executor, prestateProvider, prestatePath, localInputs, dir, 42), nil
}

func getPrestate(prestateHash common.Hash, prestateBaseUrl *url.URL, prestatePath string, dataDir string, stateConverter vm.StateConverter) (string, error) {
	prestateSource := prestates.NewPrestateSource(
		prestateBaseUrl,
		prestatePath,
		filepath.Join(dataDir, "prestates"),
		stateConverter)

	prestate, err := prestateSource.PrestatePath(prestateHash)
	if err != nil {
		return "", fmt.Errorf("failed to get prestate %v: %w", prestateHash, err)
	}
	return prestate, nil
}
