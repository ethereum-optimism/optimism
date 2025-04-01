package system

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/shell/env"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

type system struct {
	identifier string
	l1         Chain
	l2s        []L2Chain
}

// system implements System
var _ System = (*system)(nil)

func NewSystemFromURL(url string) (System, error) {
	devnetEnv, err := env.LoadDevnetFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to load devnet from URL: %w", err)
	}

	sys, err := systemFromDevnet(devnetEnv.Config, devnetEnv.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create system from devnet: %w", err)
	}
	return sys, nil
}

func (s *system) L1() Chain {
	return s.l1
}

func (s *system) L2s() []L2Chain {
	return s.l2s
}

func (s *system) Identifier() string {
	return s.identifier
}

func systemFromDevnet(dn descriptors.DevnetEnvironment, identifier string) (System, error) {
	l1, err := newChainFromDescriptor(dn.L1)
	if err != nil {
		return nil, fmt.Errorf("failed to add L1 chain: %w", err)
	}

	l2s := make([]L2Chain, len(dn.L2))
	for i, l2 := range dn.L2 {
		l2s[i], err = newL2ChainFromDescriptor(l2)
		if err != nil {
			return nil, fmt.Errorf("failed to add L2 chain: %w", err)
		}
	}

	sys := &system{
		identifier: identifier,
		l1:         l1,
		l2s:        l2s,
	}

	if slices.Contains(dn.Features, "interop") {
		// TODO(14849): this will break as soon as we have a dependency set that
		// doesn't include all L2s.
		supervisorRPC := dn.L2[0].Services["supervisor"].Endpoints["rpc"]
		return &interopSystem{
			system:        sys,
			supervisorRPC: fmt.Sprintf("http://%s:%d", supervisorRPC.Host, supervisorRPC.Port),
		}, nil
	}

	return sys, nil
}

type interopSystem struct {
	*system

	supervisorRPC string
	supervisor    Supervisor
	mu            sync.Mutex
}

// interopSystem implements InteropSystem
var _ InteropSystem = (*interopSystem)(nil)

func (i *interopSystem) InteropSet() InteropSet {
	return i.system // TODO: the interop set might not contain all L2s
}

func (i *interopSystem) Supervisor(ctx context.Context) (Supervisor, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.supervisor != nil {
		return i.supervisor, nil
	}

	cl, err := client.NewRPC(ctx, nil, i.supervisorRPC)
	if err != nil {
		return nil, fmt.Errorf("failed to dial supervisor RPC: %w", err)
	}
	supervisor := sources.NewSupervisorClient(cl, nil)
	i.supervisor = supervisor
	return supervisor, nil
}
