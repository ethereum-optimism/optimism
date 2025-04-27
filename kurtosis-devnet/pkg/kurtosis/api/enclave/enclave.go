package enclave

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/util"
)

// DockerManager defines the interface for Docker operations
type DockerManager interface {
	DestroyDockerResources(ctx context.Context, enclave ...string) error
}

// DefaultDockerManager implements DockerManager using the util package
type DefaultDockerManager struct{}

func (d *DefaultDockerManager) DestroyDockerResources(ctx context.Context, enclave ...string) error {
	return util.DestroyDockerResources(ctx, enclave...)
}

type KurtosisEnclaveManager struct {
	kurtosisCtx interfaces.KurtosisContextInterface
	dockerMgr   DockerManager
}

type KurtosisEnclaveManagerOptions func(*KurtosisEnclaveManager)

func WithKurtosisContext(kurtosisCtx interfaces.KurtosisContextInterface) KurtosisEnclaveManagerOptions {
	return func(manager *KurtosisEnclaveManager) {
		manager.kurtosisCtx = kurtosisCtx
	}
}

func WithDockerManager(dockerMgr DockerManager) KurtosisEnclaveManagerOptions {
	return func(manager *KurtosisEnclaveManager) {
		manager.dockerMgr = dockerMgr
	}
}

func NewKurtosisEnclaveManager(opts ...KurtosisEnclaveManagerOptions) (*KurtosisEnclaveManager, error) {
	manager := &KurtosisEnclaveManager{}

	for _, opt := range opts {
		opt(manager)
	}

	if manager.kurtosisCtx == nil {
		var err error
		manager.kurtosisCtx, err = wrappers.GetDefaultKurtosisContext()
		if err != nil {
			return nil, fmt.Errorf("failed to create Kurtosis context: %w", err)
		}
	}
	return manager, nil
}

func (mgr *KurtosisEnclaveManager) GetEnclave(ctx context.Context, enclave string) (interfaces.EnclaveContext, error) {
	// Try to get existing enclave first
	enclaveCtx, err := mgr.kurtosisCtx.GetEnclave(ctx, enclave)
	if err != nil {
		// If enclave doesn't exist, create a new one
		fmt.Printf("Creating a new enclave for Starlark to run inside...\n")
		enclaveCtx, err = mgr.kurtosisCtx.CreateEnclave(ctx, enclave)
		if err != nil {
			return nil, fmt.Errorf("failed to create enclave: %w", err)
		}
		fmt.Printf("Enclave '%s' created successfully\n\n", enclave)
	} else {
		fmt.Printf("Using existing enclave '%s'\n\n", enclave)
	}

	return enclaveCtx, nil
}

// cleanupEnclave handles the common cleanup logic for both stopped and empty enclaves
func (mgr *KurtosisEnclaveManager) cleanupEnclave(ctx context.Context, enclave string) error {
	// Remove the enclave
	err := mgr.kurtosisCtx.DestroyEnclave(ctx, enclave)
	if err != nil {
		fmt.Printf("failed to destroy enclave: %v", err)
	} else {
		fmt.Printf("Destroyed enclave: %s\n", enclave)
	}
	var errDocker error
	if mgr.dockerMgr != nil {
		errDocker = mgr.dockerMgr.DestroyDockerResources(ctx, enclave)
		if errDocker != nil {
			fmt.Printf("failed to destroy docker resources: %v", errDocker)
		} else {
			fmt.Printf("Destroyed docker resources for enclave: %s\n", enclave)
		}
	}
	if err != nil {
		return err
	}
	if errDocker != nil {
		return errDocker
	}
	return nil
}

func (mgr *KurtosisEnclaveManager) Autofix(ctx context.Context, enclave string) error {
	fmt.Printf("Autofixing enclave '%s'\n", enclave)
	status, err := mgr.kurtosisCtx.GetEnclaveStatus(ctx, enclave)
	if err != nil {
		// Means the enclave doesn't exist, so we're good
		fmt.Printf("Enclave '%s' does not exist, skipping autofix\n", enclave)
		return nil
	}
	switch status {
	case interfaces.EnclaveStatusRunning:
		fmt.Printf("Enclave '%s' is running, skipping autofix\n", enclave)
		return nil
	case interfaces.EnclaveStatusStopped:
		fmt.Printf("Enclave '%s' is stopped, removing\n", enclave)
		return mgr.cleanupEnclave(ctx, enclave)
	case interfaces.EnclaveStatusEmpty:
		fmt.Printf("Enclave '%s' is empty, removing\n", enclave)
		return mgr.cleanupEnclave(ctx, enclave)
	}
	return fmt.Errorf("unknown enclave status: %s", status)
}

func (mgr *KurtosisEnclaveManager) Nuke(ctx context.Context) error {
	enclaves, err := mgr.kurtosisCtx.Clean(ctx, true)
	if err != nil {
		fmt.Printf("failed to clean enclaves: %v", err)
	} else {
		fmt.Printf("Cleaned enclaves\n")
	}
	for _, enclave := range enclaves {
		fmt.Printf("Nuked enclave: %s\n", enclave.GetName())
	}
	var errDocker error
	if mgr.dockerMgr != nil {
		errDocker = mgr.dockerMgr.DestroyDockerResources(ctx)
		if errDocker != nil {
			fmt.Printf("failed to destroy docker resources: %v", errDocker)
		} else {
			fmt.Printf("Destroyed docker resources\n")
		}
	}
	if err != nil {
		return err
	}
	if errDocker != nil {
		return errDocker
	}
	return nil
}
