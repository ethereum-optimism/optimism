package sysext

import (
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"

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

func (orch *Orchestrator) rpcClient(t devtest.T, service *descriptors.Service, protocol string) client.RPC {
	t.Helper()

	endpoint, header, err := orch.findProtocolService(service, protocol)
	t.Require().NoError(err)

	opts := []client.RPCOption{}
	if !orch.useEagerRPCClients {
		opts = append(opts, client.WithLazyDial())
	}

	if orch.env.ReverseProxyURL != "" && !orch.useDirectCnx {
		opts = append(
			opts,
			client.WithGethRPCOptions(
				rpc.WithHeaders(header),
				// we need both Header["Host"] and req.Host to be set
				rpc.WithHTTPClient(&http.Client{
					Transport: hostAwareRoundTripper(header),
				}),
			),
		)
	}

	cl, err := client.NewRPC(t.Ctx(), t.Logger(), endpoint, opts...)
	t.Require().NoError(err)
	t.Cleanup(cl.Close)
	return cl
}

func (orch *Orchestrator) httpClient(t devtest.T, service *descriptors.Service, protocol string) *client.BasicHTTPClient {
	t.Helper()

	endpoint, header, err := orch.findProtocolService(service, protocol)
	t.Require().NoError(err)

	opts := []client.BasicHTTPClientOption{}

	if orch.env.ReverseProxyURL != "" && !orch.useDirectCnx {
		opts = append(
			opts,
			client.WithHeader(header),
			client.WithTransport(hostAwareRoundTripper(header)),
		)
	}

	return client.NewBasicHTTPClient(endpoint, t.Logger(), opts...)
}

func (orch *Orchestrator) findProtocolService(service *descriptors.Service, protocol string) (string, http.Header, error) {
	for proto, endpoint := range service.Endpoints {
		if proto == protocol {
			if orch.env.ReverseProxyURL != "" && !orch.useDirectCnx {
				return orch.env.ReverseProxyURL, endpoint.ReverseProxyHeader, nil
			}

			port := endpoint.Port
			if orch.usePrivatePorts {
				port = endpoint.PrivatePort
			}
			return fmt.Sprintf("http://%s:%d", endpoint.Host, port), nil, nil
		}
	}
	return "", nil, fmt.Errorf("protocol %s not found", protocol)
}

type hostSettingRoundTripper struct {
	host string
	rt   http.RoundTripper
}

func (h *hostSettingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Host = h.host
	return h.rt.RoundTrip(req)
}

func hostAwareRoundTripper(header http.Header) http.RoundTripper {
	return &hostSettingRoundTripper{
		host: header.Get("Host"),
		rt:   http.DefaultTransport,
	}
}
