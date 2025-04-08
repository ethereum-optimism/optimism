package sysext

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

const (
	ELServiceName = "el"
	CLServiceName = "cl"

	HTTPProtocol    = "http"
	RPCProtocol     = "rpc"
	MetricsProtocol = "metrics"

	FeatureInterop = "interop"
)

func (orch *Orchestrator) rpcClient(t devtest.T, endpoint string) client.RPC {
	opts := []client.RPCOption{}
	if !orch.useEagerRPCClients {
		opts = append(opts, client.WithLazyDial())
	}

	// TODO function arg
	var serviceName string

	// op-service client util:
	//	ReconnectingClient
	//    contains a callback method to dynamically get the endpoint,
	//    whenever there is a connection issue
	//
	opts = append(opts, client.WithReconnector(func() string {
		newEndpoint, ok := orch.diff[serviceName]
		if !ok {
			// no changes, can use original inventory (aka the endpoint arg)
			return endpoint
		}
		return newEndpoint
	}))
	cl, err := client.NewRPC(t.Ctx(), t.Logger(), endpoint, opts...)

	// wrap the RPC
	// on CallContext/Subscribe, check error
	// if error is connection issue ->
	//  1. get the new endpoint
	//	2. swap underlying RPC client for new one, that dials the new endpoint

	// TODO: when op-node restarts in kurtosis,
	//  does the supervisor in kurtosis automatically get aware of the new op-node?
	// I.e. who is responsible for the supervisor_addL2RPC call to the supervisor?
	// Likely related to https://github.com/ethereum-optimism/optimism/issues/15243

	t.Require().NoError(err)
	t.Cleanup(cl.Close)
	return cl
}

func (orch *Orchestrator) findProtocolService(service *descriptors.Service, protocol string) (string, error) {
	for proto, endpoint := range service.Endpoints {
		if proto == protocol {
			port := endpoint.Port
			if orch.usePrivatePorts {
				port = endpoint.PrivatePort
			}
			return fmt.Sprintf("http://%s:%d", endpoint.Host, port), nil
		}
	}
	return "", fmt.Errorf("protocol %s not found", protocol)
}

func decodePrivateKey(key string) (*ecdsa.PrivateKey, error) {
	b := common.FromHex(key)
	return crypto.ToECDSA(b)
}
