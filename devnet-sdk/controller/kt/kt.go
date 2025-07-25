package kt

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/devnet-sdk/controller/surface"
	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/kt/fs"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/run"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
)

type KurtosisControllerSurface struct {
	env         *descriptors.DevnetEnvironment
	kurtosisCtx interfaces.KurtosisContextInterface
	runner      *run.KurtosisRunner
	devnetfs    *fs.DevnetFS

	// control operations are disruptive, let's make sure we don't run them
	// concurrently so that test logic has a fighting chance of being correct.
	mtx sync.Mutex
}

func NewKurtosisControllerSurface(env *descriptors.DevnetEnvironment) (*KurtosisControllerSurface, error) {
	enclave := env.Name

	kurtosisCtx, err := wrappers.GetDefaultKurtosisContext()
	if err != nil {
		return nil, err
	}

	runner, err := run.NewKurtosisRunner(
		run.WithKurtosisRunnerEnclave(enclave),
		run.WithKurtosisRunnerKurtosisContext(kurtosisCtx),
	)
	if err != nil {
		return nil, err
	}

	enclaveFS, err := fs.NewEnclaveFS(context.TODO(), enclave)
	if err != nil {
		return nil, err
	}
	// Create a new DevnetFS instance using the enclaveFS
	devnetfs := fs.NewDevnetFS(enclaveFS)

	return &KurtosisControllerSurface{
		env:         env,
		kurtosisCtx: kurtosisCtx,
		runner:      runner,
		devnetfs:    devnetfs,
	}, nil
}

func (s *KurtosisControllerSurface) StartService(ctx context.Context, serviceName string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	script := fmt.Sprintf(`
def run(plan):
	plan.start_service(name="%s")
`, serviceName)
	// start_service is not idempotent, and doesn't return a typed error,
	// so we need to check the error message
	if err := s.runner.RunScript(ctx, script); err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "is already in use by container") {
			// we know we don't need to update the env, as the service was already running
			return nil
		}
		return err
	}
	return s.updateDevnetEnvironmentForService(ctx, serviceName, true)
}

func (s *KurtosisControllerSurface) StopService(ctx context.Context, serviceName string) error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	script := fmt.Sprintf(`
def run(plan):
	plan.stop_service(name="%s")
`, serviceName)
	// stop_service is idempotent, so errors here are real
	if err := s.runner.RunScript(ctx, script); err != nil {
		return err
	}
	// conversely, we don't know if the service was running or not, so we need to update the env
	return s.updateDevnetEnvironmentForService(ctx, serviceName, false)
}

func (s *KurtosisControllerSurface) updateDevnetEnvironmentForService(ctx context.Context, serviceName string, on bool) error {
	//
	refreshed, err := s.updateDevnetEnvironmentService(ctx, serviceName, on)
	if err != nil {
		return err
	}
	if !refreshed {
		return nil
	}

	return s.devnetfs.UploadDevnetDescriptor(ctx, s.env)
}

var _ surface.ServiceLifecycleSurface = (*KurtosisControllerSurface)(nil)
