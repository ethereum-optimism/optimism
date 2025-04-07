package kt

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/devnet-sdk/controller/surface"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/run"
)

type KurtosisControllerSurface struct {
	runner *run.KurtosisRunner
}

func NewKurtosisControllerSurface(enclave string) (*KurtosisControllerSurface, error) {
	runner, err := run.NewKurtosisRunner(run.WithKurtosisRunnerEnclave(enclave))
	if err != nil {
		return nil, err
	}
	return &KurtosisControllerSurface{
		runner: runner,
	}, nil
}

func (s *KurtosisControllerSurface) StartService(ctx context.Context, serviceName string) error {
	script := fmt.Sprintf(`
def run(plan):
	plan.start_service(name="%s")
`, serviceName)
	// start_service is not idempotent, and doesn't return a typed error,
	// so we need to check the error message
	if err := s.runner.RunScript(ctx, script); err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "is already in use by container") {
			return nil
		}
		return err
	}
	return nil
}

func (s *KurtosisControllerSurface) StopService(ctx context.Context, serviceName string) error {
	script := fmt.Sprintf(`
def run(plan):
	plan.stop_service(name="%s")
`, serviceName)
	// stop_service is idempotent
	return s.runner.RunScript(ctx, script)
}

var _ surface.ServiceLifecycleSurface = (*KurtosisControllerSurface)(nil)
