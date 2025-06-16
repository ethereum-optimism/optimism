package sysext

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

const (
	ELServiceName        = "el"
	CLServiceName        = "cl"
	RBuilderServiceName  = "rbuilder"
	ConductorServiceName = "conductor"

	HTTPProtocol                 = "http"
	RPCProtocol                  = "rpc"
	MetricsProtocol              = "metrics"
	WebsocketFlashblocksProtocol = "ws-flashblocks"

	FeatureInterop     = "interop"
	FeatureFlashblocks = "flashblocks"
)

func (orch *Orchestrator) rpcClient(t devtest.T, service *descriptors.Service, protocol string, path string) client.RPC {
	t.Helper()

	endpoint, header, err := orch.findProtocolService(service, protocol)
	t.Require().NoError(err)

	endpoint, err = url.JoinPath(endpoint, path)
	t.Require().NoError(err)

	opts := []client.RPCOption{}
	if !orch.useEagerRPCClients {
		opts = append(opts, client.WithLazyDial())
	}

	if orch.env.Env.ReverseProxyURL != "" && len(header) > 0 && !orch.useDirectCnx {
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

func (orch *Orchestrator) httpClient(t devtest.T, service *descriptors.Service, protocol string, path string) *client.BasicHTTPClient {
	t.Helper()

	endpoint, header, err := orch.findProtocolService(service, protocol)
	t.Require().NoError(err)

	endpoint, err = url.JoinPath(endpoint, path)
	t.Require().NoError(err)

	opts := []client.BasicHTTPClientOption{}

	if orch.env.Env.ReverseProxyURL != "" && !orch.useDirectCnx {
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
			if orch.env.Env.ReverseProxyURL != "" && len(endpoint.ReverseProxyHeader) > 0 && !orch.useDirectCnx {
				return orch.env.Env.ReverseProxyURL, endpoint.ReverseProxyHeader, nil
			}

			port := endpoint.Port
			if orch.usePrivatePorts {
				port = endpoint.PrivatePort
			}
			scheme := endpoint.Scheme
			if scheme == "" {
				scheme = HTTPProtocol
			}
			host := endpoint.Host
			path := ""
			if strings.Contains(host, "/") {
				parts := strings.SplitN(host, "/", 2)
				host = parts[0]
				path = "/" + parts[1]
			}
			return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path), nil, nil
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
