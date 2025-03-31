package syskt

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	ELServiceName = "el"
	CLServiceName = "cl"

	HTTPProtocol    = "http"
	RPCProtocol     = "rpc"
	MetricsProtocol = "metrics"

	FeatureInterop = "interop"
)

func getOrchestrator(setup *stack.Setup) *Orchestrator {
	o, ok := setup.Orchestrator.(*Orchestrator)
	setup.Require.True(ok, "orchestrator is not a valid Orchestrator")
	return o
}

func rpcClient(setup *stack.Setup, endpoint string) client.RPC {
	orchestrator := getOrchestrator(setup)

	opts := []client.RPCOption{}
	if !orchestrator.useEagerRPCClients {
		opts = append(opts, client.WithLazyDial())
	}
	cl, err := client.NewRPC(setup.Ctx, setup.Log, endpoint, opts...)
	setup.Require.NoError(err)
	return cl
}

func findProtocolService(setup *stack.Setup, svc string, protocol string, services descriptors.ServiceMap) (string, error) {
	orchestrator := getOrchestrator(setup)

	for name, service := range services {
		if name == svc {
			for proto, endpoint := range service.Endpoints {
				if proto == protocol {
					port := endpoint.Port
					if orchestrator.usePrivatePorts {
						port = endpoint.PrivatePort
					}
					return fmt.Sprintf("http://%s:%d", endpoint.Host, port), nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s not found", svc)
}

func decodePrivateKey(key string) (*ecdsa.PrivateKey, error) {
	b := common.FromHex(key)
	return crypto.ToECDSA(b)
}
