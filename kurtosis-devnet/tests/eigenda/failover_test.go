package eigenda_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Layr-Labs/eigenda-proxy/clients/memconfig_client"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/stretchr/testify/require"
)

// All tests are run in the context of the eigenda-memstore-devnet enclave.
// We assume that this enclave is already running.
const enclaveName = "eigenda-memstore-devnet"

// TestFailover tests the failover behavior of the batcher, in response to the proxy returning 503 errors.
// See https://github.com/Layr-Labs/eigenda-proxy?tab=readme-ov-file#failover-signals for proxy behavior.
// The proxy's memstore's failover behavior is toggled on and off by this test via a REST api.
// We then check that the batcher correctly interprets the 503 signals and starts submitting batches to EthDACalldata instead.
// The test then toggles the failover back off and checks that the batcher starts submitting EigenDA batches again.
// The batches inbox transactions are queried via geth's GraphQL API.
//
// Note: because this test relies on modifying the proxy's memstore config, it should be run in isolation.
// That is, if we ever implement more kurtosis tests, they would currently need to be run sequentially.
func TestFailoverToEthDACalldata(t *testing.T) {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Minute)
	}
	ctxWithDeadline, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	harness := newHarness(t)
	t.Cleanup(func() {
		// switch proxy back to normal mode, in case test gets cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := harness.clients.proxyMemconfigClient.Failback(ctx)
		if err != nil {
			t.Logf("Error failing back... you might need to reset proxy to normal mode manually: %v", err)
		}
	})

	// Number of blocks to queried for batcher txs, for each of the initial/failover/failback stages
	// Test will look at batcher txs between blocks:
	// - initial altda stage: [testStartL1BlockNum - l1BlocksQueriedForBatcherTxs, testStartL1BlockNum]
	// - ethDACalldata stage: [afterFailoverFromBlockNum, afterFailoverFromBlockNum + l1BlocksQueriedForBatcherTxs]
	// - altDA stage: [afterFailbackFromBlockNum, afterFailbackFromBlockNum + l1BlocksQueriedForBatcherTxs]
	//
	// After Failover/Failback, will wait for 10 L1 blocks to make sure failover/failback has happened.
	// Assumption is that a cert is being posted every 2 blocks (hardcoded in batcher config)
	// TODO: read max-channel-duration from batcher's config instead of assuming 2 blocks
	l1BlocksQueriedForBatcherTxs := uint64(10)

	// assume kurtosis is running and is at least at block numBlocksBetweenStages
	require.GreaterOrEqual(t, harness.testStartL1BlockNum, l1BlocksQueriedForBatcherTxs, "Test started too early in the chain")
	fromBlock := harness.testStartL1BlockNum - l1BlocksQueriedForBatcherTxs

	// 1. Check that the original commitments are EigenDA
	harness.requireBatcherTxsToBeFromLayer(t, fromBlock, fromBlock+l1BlocksQueriedForBatcherTxs, DALayerEigenDA)

	// 2. Failover and check that the commitments are now EthDACalldata
	t.Logf("Failing over... changing proxy's config to return 503 errors")
	err := harness.clients.proxyMemconfigClient.Failover(ctxWithDeadline)
	require.NoError(t, err)

	afterFailoverFromBlockNum, err := harness.clients.gethL1Client.BlockNumber(ctxWithDeadline)
	require.NoError(t, err)
	afterFailoverToBlockNum := afterFailoverFromBlockNum + l1BlocksQueriedForBatcherTxs
	_, err = geth.WaitForBlock(big.NewInt(int64(afterFailoverToBlockNum)), harness.clients.gethL1Client)
	require.NoError(t, err)

	harness.requireBatcherTxsToBeFromLayer(t, afterFailoverFromBlockNum, afterFailoverToBlockNum, DALayerEthCalldata)

	// We also check that the op-node is still finalizing blocks after the failover
	syncStatus, err := harness.clients.opNodeClient.SyncStatus(ctxWithDeadline)
	require.NoError(t, err)
	afterFailoverFinalizedL2 := syncStatus.FinalizedL2
	t.Logf("Current finalized L2 block: %d. Waiting for next block to finalize to make sure finalization is still happening.", afterFailoverFinalizedL2.Number)
	// On average would expect this to take half an epoch, aka 16 L1 blocks, which at 6 sec/block means 1.5 minutes.
	// This generally takes longer (3-6 minutes), but I'm not quite sure why.
	_, err = geth.WaitForBlockToBeFinalized(new(big.Int).SetUint64(afterFailoverFinalizedL2.Number+1), harness.clients.opGethClient, 6*time.Minute)
	require.NoError(t, err, "op-node should still be finalizing blocks after failover")

	// 3. Failback and check that the commitments are EigenDA again
	t.Logf("Failing back... changing proxy's config to start processing PUT requests normally again")
	err = harness.clients.proxyMemconfigClient.Failback(ctxWithDeadline)
	require.NoError(t, err)

	afterFailbackFromBlockNum, err := harness.clients.gethL1Client.BlockNumber(ctxWithDeadline)
	require.NoError(t, err)
	afterFailbackToBlockNum := afterFailbackFromBlockNum + l1BlocksQueriedForBatcherTxs
	_, err = geth.WaitForBlock(big.NewInt(int64(afterFailbackToBlockNum)), harness.clients.gethL1Client)
	require.NoError(t, err)

	harness.requireBatcherTxsToBeFromLayer(t, afterFailbackFromBlockNum, afterFailbackToBlockNum, DALayerEigenDA)

}

// Test Harness, which contains all the state needed to run the tests.
// harness also defines some higher-level "require" methods that are used in the tests.
type harness struct {
	logger              log.Logger
	endpoints           *EnclaveServicePublicEndpoints
	clients             *EnclaveServiceClients
	batchInboxAddr      common.Address
	testStartL1BlockNum uint64
}

func newHarness(t *testing.T) *harness {
	logger := testlog.Logger(t, slog.LevelInfo)

	// We leave 20 seconds to build the entire testHarness.
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create a Kurtosis context
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	require.NoError(t, err)

	// Get the eigenda-memstore-devnet enclave (assuming it's already running)
	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctxWithTimeout, enclaveName)
	require.NoError(t, err, "Error getting enclave context: is enclave %v running?", enclaveName)

	endpoints, err := getPublicEndpointsFromKurtosis(enclaveCtx)
	require.NoError(t, err)
	t.Logf("Endpoints: %+v", endpoints)

	clients, err := getClientsFromEndpoints(ctxWithTimeout, logger, endpoints)
	require.NoError(t, err)

	// Get the batch inbox address from the rollup config
	rollupConfig, err := clients.opNodeClient.RollupConfig(ctxWithTimeout)
	require.NoError(t, err)

	// Get the current L1 block number
	testStartL1BlockNum, err := clients.gethL1Client.BlockNumber(ctxWithTimeout)
	require.NoError(t, err)

	return &harness{
		logger:              logger,
		endpoints:           endpoints,
		clients:             clients,
		batchInboxAddr:      rollupConfig.BatchInboxAddress,
		testStartL1BlockNum: testStartL1BlockNum,
	}
}

// requireBatcherTxsToBeFromLayer checks that the batcher transactions since startingFromBlockNum are all from the expectedLayer.
// It allows for up to 3 initial commitments to be of the wrong type, as the failover/failback might not have taken effect yet.
// It requires that at least 2 commitments of the expected type are present after the failover/failback.
func (h *harness) requireBatcherTxsToBeFromLayer(t *testing.T, fromBlockNum, toBlockNum uint64, expectedLayer DALayer) {
	batcherTxs, err := fetchBatcherTxs(h.endpoints.GethL1Endpoint, h.batchInboxAddr.String(), fromBlockNum, toBlockNum)
	require.NoError(t, err)
	t.Logf("Fetched %d batcher transactions since block %d", len(batcherTxs), fromBlockNum)

	// We allow first 3 commitments to be of the wrong DA layer, as the failover/failback might not have taken effect yet.
	wrongCommitmentsToDiscard := 0
	for _, batcherTx := range batcherTxs {
		if batcherTx.daLayer != expectedLayer {
			wrongCommitmentsToDiscard++
		}
		// as soon as we see a commitment from expectedLayer, or 3 from the other layer, we stop discarding.
		if wrongCommitmentsToDiscard > 2 || batcherTx.daLayer == expectedLayer {
			break
		}
	}
	batcherTxs = batcherTxs[wrongCommitmentsToDiscard:]
	t.Logf("Discarded %d commitments. %d left which should all be %v", wrongCommitmentsToDiscard, len(batcherTxs), expectedLayer)

	// After potentially discarding up to 3 commitments, we expect all future commitments (at least 2) to be of the expectedLayer
	require.GreaterOrEqual(t, len(batcherTxs), 2, "Expected at least 2 %v commitments after failover/failback", expectedLayer)
	for _, batcherTx := range batcherTxs {
		require.Equal(t, expectedLayer, batcherTx.daLayer,
			"Invalid commitment in block %d: expected %v, received commitment %s", batcherTx.block, expectedLayer, batcherTx.commitment)
	}
}

// See https://specs.optimism.io/experimental/alt-da.html#example-commitments
// Batcher only supports failing over to calldata txs right now, so this test doesn't test 4844 failover.
// Note that 4844 txs are completely different and don't use normal txs with a prefix in the calldata,
// see https://github.com/ethereum-optimism/optimism/blob/develop/op-node/rollup/derive/blob_data_source.go#L134-L137
const ethDACalldataCommitmentPrefix = "0x00"
const eigenDACommitmentPrefix = "0x010100"

type DALayer string

const (
	DALayerEthCalldata DALayer = "ethda-calldata"
	DALayerEigenDA     DALayer = "eigenda"
)

type BatcherTx struct {
	commitment string
	daLayer    DALayer // commitment starts with respective prefix
	block      uint64
}

// HexUint64 is a custom type that can unmarshal from a hex string
type HexUint64 uint64

// UnmarshalJSON implements the json.Unmarshaler interface
func (h *HexUint64) UnmarshalJSON(data []byte) error {
	// Remove quotes from the JSON string
	hexStr := string(data)
	hexStr = strings.Trim(hexStr, "\"")

	// Check if it's a hex string
	if !strings.HasPrefix(hexStr, "0x") {
		return fmt.Errorf("not a hex string: %s", hexStr)
	}

	// Parse the hex string (without the 0x prefix)
	val, err := strconv.ParseUint(hexStr[2:], 16, 64)
	if err != nil {
		return err
	}

	*h = HexUint64(val)
	return nil
}

// Fetches all the batch-inbox posted commitments from blockNum (inclusive) to current block.
// We rely on geth's GraphQL API to fetch the batcher transactions.
// We could possibly have reused op-node's L1Retriever, but the API felt very derivation-pipeline specific,
// and there doesn't seem to be a way to reuse it easily for constructing a custom derivation-pipeline with a subset of stages
// like what we need here. Could consider migrating in the future if we need more complex logic.
func fetchBatcherTxs(gethL1Endpoint string, batchInbox string, fromBlockNum, toBlockNum uint64) ([]BatcherTx, error) {
	// We use standard HTTP for GraphQL as it's not directly supported by the rpc package
	// Visit gethL1Endpoint/graphql/ui to see the schema and test queries
	query := fmt.Sprintf(`
	{
		"query": "query txInfo { blocks(from:%v, to:%v) { transactions { to { address } inputData block { number } } } }"
	}`, fromBlockNum, toBlockNum)

	// Make GraphQL request
	req, err := http.NewRequest("POST", gethL1Endpoint+"/graphql", strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response
	type GraphQLResponse struct {
		Data struct {
			Blocks []struct {
				Transactions []struct {
					To struct {
						Address string `json:"address"`
					} `json:"to"`
					InputData string `json:"inputData"`
					Block     struct {
						// we use HexUint64 to properly parse the hex strings returned
						Number HexUint64 `json:"number"`
					} `json:"block"`
				} `json:"transactions"`
			} `json:"blocks"`
		} `json:"data"`
	}
	var graphQLResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphQLResp); err != nil {
		return nil, err
	}
	if len(graphQLResp.Data.Blocks) == 0 {
		// Assume that this is a graphQL query error, that would have returned something like
		// "errors": [
		// 	{
		// 	  "message": "syntax error: unexpected \"\", expecting Ident",
		// 	}
		// ]
		// TODO: prob should just switch to a proper graphql client that can handle these properly
		return nil, fmt.Errorf("no blocks returned in GraphQL response")
	}

	// Filter transactions to the batcher address
	var batcherTxs []BatcherTx
	for _, block := range graphQLResp.Data.Blocks {
		for _, tx := range block.Transactions {
			if strings.EqualFold(tx.To.Address, batchInbox) {
				var daLayer DALayer
				if strings.HasPrefix(tx.InputData, eigenDACommitmentPrefix) {
					daLayer = DALayerEigenDA
				} else if strings.HasPrefix(tx.InputData, ethDACalldataCommitmentPrefix) {
					daLayer = DALayerEthCalldata
				} else {
					return nil, fmt.Errorf("unknown commitment prefix: %s", tx.InputData)
				}
				batcherTxs = append(batcherTxs, BatcherTx{
					commitment: tx.InputData,
					daLayer:    daLayer,
					block:      uint64(tx.Block.Number),
				})
			}
		}
	}

	return batcherTxs, nil
}

// Localhost endpoints for the different services in the enclave
// that we need to interact with. We store the public localhost endpoints instead
// of the private enclave endpoints because we need to interact with the services
// using external shell commands like `cast rpc ...` and `cast geth ...`.
// The public endpoints are the ones that are exposed to the host machine.
type EnclaveServicePublicEndpoints struct {
	OpNodeEndpoint       string `kurtosis:"op-cl-1-op-node-op-geth-op-kurtosis,http"`
	OpGethEndpoint       string `kurtosis:"op-el-1-op-geth-op-node-op-kurtosis,rpc"`
	GethL1Endpoint       string `kurtosis:"el-1-geth-teku,rpc"`
	EigendaProxyEndpoint string `kurtosis:"da-server-op-kurtosis,http"`
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
	opNodeClient *sources.RollupClient
	// opGeth is the client for the L2 execution layer client.
	opGethClient *ethclient.Client
	// gethL1 is the client for the L1 chain execution layer client.
	gethL1Client *ethclient.Client
	// proxyMemconfigClient is the client for the eigenda-proxy's memstore config API.
	// It allows us to toggle the proxy's failover behavior.
	proxyMemconfigClient *ProxyMemconfigClient
}

func getClientsFromEndpoints(ctx context.Context, logger log.Logger, endpoints *EnclaveServicePublicEndpoints) (*EnclaveServiceClients, error) {
	opNodeClient, err := dial.DialRollupClientWithTimeout(ctx, 10*time.Second, logger, endpoints.OpNodeEndpoint)
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

	proxyMemconfigClient := &ProxyMemconfigClient{
		Client: memconfig_client.New(&memconfig_client.Config{URL: endpoints.EigendaProxyEndpoint}),
	}

	return &EnclaveServiceClients{
		opNodeClient:         opNodeClient,
		opGethClient:         opGethClient,
		gethL1Client:         gethL1Client,
		proxyMemconfigClient: proxyMemconfigClient,
	}, nil
}

// ProxyMemconfigClient is a wrapper around the memconfig client that adds a Failover method
// TODO: we should upstream this to eigenda-proxy repo
type ProxyMemconfigClient struct {
	*memconfig_client.Client
}

// Update the proxy's memstore config to start returning 503 errors
// Note: we have to GetConfig, update it and then UpdateConfig because the client doesn't implement a "patch" method,
// even though the API does support it.
func (c *ProxyMemconfigClient) Failover(ctx context.Context) error {
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
func (c *ProxyMemconfigClient) Failback(ctx context.Context) error {
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
