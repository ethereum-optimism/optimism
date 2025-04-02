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
	cl, err := client.NewRPC(t.Ctx(), t.Logger(), endpoint, opts...)
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
