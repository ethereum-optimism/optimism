package enclave

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/wrappers"
)

type KurtosisEnclaveManager struct {
	kurtosisCtx interfaces.KurtosisContextInterface
}

type KurtosisEnclaveManagerOptions func(*KurtosisEnclaveManager)

func WithKurtosisContext(kurtosisCtx interfaces.KurtosisContextInterface) KurtosisEnclaveManagerOptions {
	return func(manager *KurtosisEnclaveManager) {
		manager.kurtosisCtx = kurtosisCtx
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
