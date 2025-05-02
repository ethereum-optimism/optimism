package kt

import (
	"context"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/kurtosis-devnet/pkg/kurtosis/api/interfaces"
)

// a hack because some L2 services are duplicated across chains
type redundantService []*descriptors.Service

func (s redundantService) getEndpoints() descriptors.EndpointMap {
	if len(s) == 0 {
		return nil
	}

	return s[0].Endpoints
}

func (s redundantService) setEndpoints(endpoints descriptors.EndpointMap) {
	for _, svc := range s {
		svc.Endpoints = endpoints
	}
}

func (s redundantService) refreshEndpoints(serviceCtx interfaces.ServiceContext) {
	endpoints := s.getEndpoints()

	publicPorts := serviceCtx.GetPublicPorts()
	privatePorts := serviceCtx.GetPrivatePorts()

	for name, info := range publicPorts {
		endpoints[name].Port = int(info.GetNumber())
	}
	for name, info := range privatePorts {
		endpoints[name].PrivatePort = int(info.GetNumber())
	}

	s.setEndpoints(endpoints)
}

func findSvcInEnv(env *descriptors.DevnetEnvironment, serviceName string) redundantService {
	if svc := findSvcInChain(env.L1, serviceName); svc != nil {
		return redundantService{svc}
	}

	var services redundantService = nil
	for _, l2 := range env.L2 {
		if svc := findSvcInChain(l2.Chain, serviceName); svc != nil {
			services = append(services, svc)
		}
	}
	return services
}

func findSvcInChain(chain *descriptors.Chain, serviceName string) *descriptors.Service {
	for _, svc := range chain.Services {
		if svc.Name == serviceName {
			return svc
		}
	}

	for _, node := range chain.Nodes {
		for _, svc := range node.Services {
			if svc.Name == serviceName {
				return svc
			}
		}
	}

	return nil
}

func (s *KurtosisControllerSurface) updateDevnetEnvironmentService(ctx context.Context, serviceName string, on bool) (bool, error) {
	svc := findSvcInEnv(s.env, serviceName)
	if svc == nil {
		// service is not part of the env, so we don't need to do anything
		return false, nil
	}

	// get the enclave
	enclaveCtx, err := s.kurtosisCtx.GetEnclave(ctx, s.env.Name)
	if err != nil {
		return false, err
	}

	serviceCtx, err := enclaveCtx.GetService(serviceName)
	if err != nil {
		return false, err
	}

	if on {
		svc.refreshEndpoints(serviceCtx)
	}
	// otherwise the service is down anyway, it doesn't matter if it has outdated endpoints
	return on, nil
}
