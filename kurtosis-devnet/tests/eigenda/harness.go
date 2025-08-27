package eigenda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Layr-Labs/eigenda-proxy/clients/memconfig_client"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/stretchr/testify/require"
)

// The below are hardcoded constants that are specific to the eigenda-devnet enclave.
// They could be fetched dynamically from the enclave if/when needed, but for now it's easier to just hardcode them here,
// since we don't plan on changing them anytime soon.

// All tests are run in the context of the eigenda-devnet enclave.
// We assume that this enclave is already running.
const enclaveName = "eigenda-devnet"

// Test Harness, which contains all the state needed to run the tests.
// Harness also defines some higher-level "require" methods that are used in the tests.
type Harness struct {
	t                   *testing.T
	Logger              log.Logger
	Endpoints           *EnclaveServicePublicEndpoints
	Clients             *EnclaveServiceClients
	BatcherUUID         services.ServiceUUID // used to query for batcher logs
	BatchInboxAddr      common.Address
	TestStartL1BlockNum uint64
}

func NewHarness(t *testing.T) *Harness {
	logger := testlog.Logger(t, slog.LevelInfo)

	// We leave 20 seconds to build the entire testHarness.
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create a Kurtosis context
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	require.NoError(t, err)

	// Get the `enclaveName` enclave context (assuming it's already running)
	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctxWithTimeout, enclaveName)
	require.NoError(t, err, "Error getting enclave context: is enclave %v running?", enclaveName)

	endpoints, err := getPublicEndpointsFromKurtosis(enclaveCtx)
	require.NoError(t, err)
	t.Logf("Endpoints: %+v", endpoints)

	clients, err := getClientsFromEndpoints(ctxWithTimeout, logger, endpoints)
	require.NoError(t, err)

	// Get the batch inbox address from the rollup config
	rollupConfig, err := clients.OpNodeClient.RollupConfig(ctxWithTimeout)
	require.NoError(t, err)

	// Get the current L1 block number
	testStartL1BlockNum, err := clients.GethL1Client.BlockNumber(ctxWithTimeout)
	require.NoError(t, err)

	batcherCtx, err := enclaveCtx.GetServiceContext("op-batcher-2151908-op-kurtosis")
	require.NoError(t, err)

	return &Harness{
		t:                   t,
		Logger:              logger,
		Endpoints:           endpoints,
		Clients:             clients,
		BatcherUUID:         batcherCtx.GetServiceUUID(),
		BatchInboxAddr:      rollupConfig.BatchInboxAddress,
		TestStartL1BlockNum: testStartL1BlockNum,
	}
}

func (h *Harness) QueryBatcherLogs(ctx context.Context, shouldFollowLogs bool, logLineFilter *kurtosis_context.LogLineFilter) <-chan string {
	outC := make(chan string)
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	require.NoError(h.t, err)

	uuidMap := map[services.ServiceUUID]bool{
		h.BatcherUUID: true,
	}
	logsC, _, err := kurtosisCtx.GetServiceLogs(ctx, enclaveName, uuidMap, shouldFollowLogs, true, 1000, logLineFilter)
	require.NoError(h.t, err)

	go func() {
		for logContent := range logsC {
			logs := logContent.GetServiceLogsByServiceUuids()[h.BatcherUUID]
			for _, log := range logs {
				outC <- log.GetContent()
			}
		}
	}()
	return outC

}

// Localhost endpoints for the different services in the enclave
// that we need to interact with. We store the public localhost endpoints instead
// of the private enclave endpoints because we need to interact with the services
// using external shell commands like `cast rpc ...` and `cast geth ...`.
// The public endpoints are the ones that are exposed to the host machine.
type EnclaveServicePublicEndpoints struct {
	OpNodeEndpoint       string `kurtosis:"op-cl-2151908-node0-op-node,rpc"`
	OpGethEndpoint       string `kurtosis:"op-el-2151908-node0-op-geth,rpc"`
	GethL1Endpoint       string `kurtosis:"el-1-geth-teku,rpc"`
	EigendaProxyEndpoint string `kurtosis:"op-da-da-server-2151908-op-kurtosis,http"`
	// Adding new endpoints is as simple as adding a new field with a kurtosis tag
	// NewServiceEndpoint   string `kurtosis:"new-service-name,port-name"`
}

// Constructor for EnclaveServiceEndpoints struct, which assumes a running kurtosis enclave
// and queries the needed services for their public (localhost) ports, and constructs
// the struct with the endpoints.
//
// This function uses reflection to parse the `kurtosis` tags in the struct fields to get the service name and port name.
// See the comments in the EnclaveServicePublicEndpoints struct for more details on adding a new endpoint.
func getPublicEndpointsFromKurtosis(enclaveCtx *enclaves.EnclaveContext) (*EnclaveServicePublicEndpoints, error) {
	endpoints := &EnclaveServicePublicEndpoints{}

	// Get the type of the struct to iterate over fields
	t := reflect.TypeOf(endpoints).Elem()
	v := reflect.ValueOf(endpoints).Elem()

	// Iterate over all fields in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Get the kurtosis tag
		tag := field.Tag.Get("kurtosis")
		if tag == "" {
			return nil, fmt.Errorf("field %s doesn't have a kurtosis tag", field.Name)
		}

		// Parse the tag to get service name and port name
		parts := strings.Split(tag, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid kurtosis tag format for field %s: %s", field.Name, tag)
		}

		serviceName := parts[0]
		portName := parts[1]

		// Get the service context
		serviceCtx, err := enclaveCtx.GetServiceContext(serviceName)
		if err != nil {
			return nil, fmt.Errorf("GetServiceContext for %s: %w", serviceName, err)
		}

		// Get the port
		port, ok := serviceCtx.GetPublicPorts()[portName]
		if !ok {
			return nil, fmt.Errorf("service %s doesn't expose %s port", serviceName, portName)
		}

		// Set the endpoint URL in the struct field
		endpoint := fmt.Sprintf("http://localhost:%d", port.GetNumber())
		v.Field(i).SetString(endpoint)
	}

	return endpoints, nil
}

type EnclaveServiceClients struct {
	// opNode and opGeth are the L2 clients for the rollup.
	OpNodeClient *sources.RollupClient
	// opGeth is the client for the L2 execution layer client.
	OpGethClient *ethclient.Client
	// gethL1 is the client for the L1 chain execution layer client.
	GethL1Client *ethclient.Client
	// ProxyClients is the client for the eigenda-proxy's APIs: admin and memstore config.
	// It allows us to toggle the proxy's failover behavior,
	// as well as toggle the dispersal backend between V1 and V2.
	ProxyClients *ProxyClients
}

func getClientsFromEndpoints(ctx context.Context, logger log.Logger, endpoints *EnclaveServicePublicEndpoints) (*EnclaveServiceClients, error) {
	opts := []client.RPCOption{
		client.WithCallTimeout(10 * time.Second),
	}
	opNodeClient, err := dial.DialRollupClientWithTimeout(ctx, logger, endpoints.OpNodeEndpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial.DialRollupClientWithTimeout: %w", err)
	}

	opGethClient, err := dial.DialEthClientWithTimeout(ctx, 10*time.Second, logger, endpoints.OpGethEndpoint)
	if err != nil {
		return nil, fmt.Errorf("dial.DialEthClientWithTimeout: %w", err)
	}

	// TODO: prob also change to use dial.DialEthClient?
	gethL1Client, err := ethclient.Dial(endpoints.GethL1Endpoint)
	if err != nil {
		return nil, fmt.Errorf("ethclient.Dial: %w", err)
	}

	proxyClients := NewProxyClients(endpoints.EigendaProxyEndpoint)

	return &EnclaveServiceClients{
		OpNodeClient: opNodeClient,
		OpGethClient: opGethClient,
		GethL1Client: gethL1Client,
		ProxyClients: proxyClients,
	}, nil
}

// ProxyClients is a wrapper around the memconfig client that adds a Failover method
// TODO: we should upstream this to eigenda-proxy repo
type ProxyClients struct {
	*memconfig_client.Client
	*ProxyAdminAPIClient
}

func NewProxyClients(proxyEndpoint string) *ProxyClients {
	return &ProxyClients{
		Client: memconfig_client.New(&memconfig_client.Config{URL: proxyEndpoint}),
		ProxyAdminAPIClient: &ProxyAdminAPIClient{
			proxyEndpoint: proxyEndpoint,
		},
	}
}

// Update the proxy's memstore config to start returning 503 errors
// Note: we have to GetConfig, update it and then UpdateConfig because the client doesn't implement a "patch" method,
// even though the API does support it.
func (c *ProxyClients) Failover(ctx context.Context) error {
	memConfig, err := c.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("GetConfig: %w", err)
	}
	memConfig.PutReturnsFailoverError = true
	_, err = c.UpdateConfig(ctx, memConfig)
	if err != nil {
		return fmt.Errorf("UpdateConfig: %w", err)
	}
	return nil
}
func (c *ProxyClients) Failback(ctx context.Context) error {
	memConfig, err := c.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("GetConfig: %w", err)
	}
	memConfig.PutReturnsFailoverError = false
	_, err = c.UpdateConfig(ctx, memConfig)
	if err != nil {
		return fmt.Errorf("UpdateConfig: %w", err)
	}
	return nil
}

type EigenDACertVersion string

const (
	EigenDACertVersionV1 EigenDACertVersion = "V1"
	EigenDACertVersionV2 EigenDACertVersion = "V2"
)

// Simple REST client for the proxy's admin API routes:
// https://github.com/Layr-Labs/eigenda-proxy?tab=readme-ov-file#admin-routes
// TODO: this should prob live in proxy repo?
type ProxyAdminAPIClient struct {
	proxyEndpoint string // e.g. http://localhost:3100
}

func (c *ProxyAdminAPIClient) GetDispersalBackend(ctx context.Context) (EigenDACertVersion, error) {
	// URL to send the request to
	url := c.proxyEndpoint + "/admin/eigenda-dispersal-backend"

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Create HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from server: %s", resp.Status)
	}

	// Read and parse the response body
	var response struct {
		EigenDADispersalBackend string `json:"eigenDADispersalBackend"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %w", err)
	}

	switch response.EigenDADispersalBackend {
	case "V1":
		return EigenDACertVersionV1, nil
	case "V2":
		return EigenDACertVersionV2, nil
	default:
		return "", fmt.Errorf("unknown backend version received from proxy: %s", response.EigenDADispersalBackend)
	}
}

func (c *ProxyAdminAPIClient) SetDispersalBackend(ctx context.Context, version EigenDACertVersion) error {
	// URL to send the request to
	url := c.proxyEndpoint + "/admin/eigenda-dispersal-backend"

	// body is json containing backend version
	jsonData := []byte(fmt.Sprintf(`{"eigenDADispersalBackend": "%s"}`, version))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client
	client := &http.Client{}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from server: %s", resp.Status)
	}
	return nil
}
