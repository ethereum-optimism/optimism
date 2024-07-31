package runner

import (
	"errors"
	"fmt"

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
	prestateSource prestates.PrestateSource,
	prestateHash common.Hash,
	traceType types.TraceType,
	localInputs utils.LocalGameInputs,
	dir string,
) (types.TraceProvider, error) {
	prestate, err := prestateSource.PrestatePath(prestateHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get prestate %v: %w", prestateHash, err)
	}

	switch traceType {
	case types.TraceTypeCannon:
		prestateProvider := cannon.NewPrestateProvider(prestate)
		return cannon.NewTraceProvider(logger, m, cfg.Cannon, prestateProvider, prestate, localInputs, dir, 42), nil
	case types.TraceTypeAsterisc:
		prestateProvider := asterisc.NewPrestateProvider(prestate)
		return asterisc.NewTraceProvider(logger, m, cfg.Asterisc, prestateProvider, prestate, localInputs, dir, 42), nil
	}
	return nil, errors.New("invalid trace type")
}
