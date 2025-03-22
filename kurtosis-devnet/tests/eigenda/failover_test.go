package eigenda

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TODO: add test which sets other properties of memstore like latency, out of order, etc.

// TestFailover tests the failover behavior of the batcher, in response to the proxy returning 503 errors.
// See https://github.com/Layr-Labs/eigenda-proxy?tab=readme-ov-file#failover-signals for proxy behavior.
// The proxy's memstore's failover behavior is toggled on and off by this test via a REST api.
// We then check that the batcher correctly interprets the 503 signals and starts submitting batches to EthDACalldata instead.
// The test then toggles the failover back off and checks that the batcher starts submitting EigenDA batches again.
// The batches inbox transactions are queried via geth's GraphQL API.
//
// Note: because this test relies on modifying the proxy's memstore config, it should be run in isolation.
// That is, if we ever implement more kurtosis tests, they would currently need to be run sequentially.
func TestFailoverToEthDACalldata_Memstore(t *testing.T) {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Minute)
	}
	ctxWithDeadline, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	harness := NewHarness(t)
	t.Cleanup(func() {
		// switch proxy back to normal mode, in case test gets cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		t.Logf("Cleanup; Failing back to eigenda... resetting enclave proxy to start posting to eigenda again")
		err := harness.Clients.ProxyMemconfigClient.Failback(ctx)
		if err != nil {
			t.Logf("Error failing back... you might need to reset proxy to normal mode manually: %v", err)
		}
	})

	// Number of blocks to query for batcher txs, for each of the initial/failover/failback stages
	// Test will look at batcher txs between blocks:
	// - initial altda stage: [testStartL1BlockNum - l1BlocksQueriedForBatcherTxs, testStartL1BlockNum]
	// - ethDACalldata stage: [afterFailoverFromBlockNum, afterFailoverFromBlockNum + l1BlocksQueriedForBatcherTxs]
	// - altDA stage: [afterFailbackFromBlockNum, afterFailbackFromBlockNum + l1BlocksQueriedForBatcherTxs]
	//
	// Need to make sure each stage contains at least 2 commitments of the correct type. This can only happen if channels
	// are being closed in time, which requires either: sending traffic with traffic-generator, or setting a low (e.g. 2 L1 blocks) channel-timeout.
	l1BlocksQueriedForBatcherTxs := uint64(15)

	// assume kurtosis is running and is at least at block numBlocksBetweenStages
	require.GreaterOrEqual(t, harness.TestStartL1BlockNum, l1BlocksQueriedForBatcherTxs, "Test started too early in the chain")
	fromBlock := harness.TestStartL1BlockNum - l1BlocksQueriedForBatcherTxs

	// 1. Check that the original commitments are EigenDA
	t.Logf("[Stage1: EigenDA] Checking that the initial commitments are EigenDA")
	requireBatcherTxsToBeFromLayer(t, fromBlock, fromBlock+l1BlocksQueriedForBatcherTxs, DALayerEigenDA, harness.Endpoints.GethL1Endpoint, harness.BatchInboxAddr)

	// 2. Failover and check that the commitments are now EthDACalldata
	t.Logf("[Stage2: EthDA-Calldata] Failing over... changing proxy's config to return 503 errors")
	err := harness.Clients.ProxyMemconfigClient.Failover(ctxWithDeadline)
	require.NoError(t, err)

	afterFailoverFromBlockNum, err := harness.Clients.GethL1Client.BlockNumber(ctxWithDeadline)
	require.NoError(t, err)
	afterFailoverToBlockNum := afterFailoverFromBlockNum + l1BlocksQueriedForBatcherTxs
	_, err = geth.WaitForBlock(big.NewInt(int64(afterFailoverToBlockNum)), harness.Clients.GethL1Client)
	require.NoError(t, err)

	requireBatcherTxsToBeFromLayer(t, afterFailoverFromBlockNum, afterFailoverToBlockNum, DALayerEthCalldata, harness.Endpoints.GethL1Endpoint, harness.BatchInboxAddr)

	// We also check that the op-node is still finalizing blocks after the failover
	syncStatus, err := harness.Clients.OpNodeClient.SyncStatus(ctxWithDeadline)
	require.NoError(t, err)
	afterFailoverFinalizedL2 := syncStatus.FinalizedL2
	t.Logf("[Finalization] Current finalized L2 block: %d. Waiting for next block to finalize to make sure finalization is still happening.", afterFailoverFinalizedL2.Number)
	// On average would expect this to take half an epoch, aka 16 L1 blocks, which at 6 sec/block means 1.5 minutes.
	// This generally takes longer (3-6 minutes), but I'm not quite sure why.
	_, err = geth.WaitForBlockToBeFinalized(new(big.Int).SetUint64(afterFailoverFinalizedL2.Number+1), harness.Clients.OpGethClient, 6*time.Minute)
	require.NoError(t, err, "op-node should still be finalizing blocks after failover")

	// 3. Failback and check that the commitments are EigenDA again
	t.Logf("[Stage3: EigenDA] Failing back... changing proxy's config to start processing PUT requests normally again")
	err = harness.Clients.ProxyMemconfigClient.Failback(ctxWithDeadline)
	require.NoError(t, err)

	afterFailbackFromBlockNum, err := harness.Clients.GethL1Client.BlockNumber(ctxWithDeadline)
	require.NoError(t, err)
	afterFailbackToBlockNum := afterFailbackFromBlockNum + l1BlocksQueriedForBatcherTxs
	_, err = geth.WaitForBlock(big.NewInt(int64(afterFailbackToBlockNum)), harness.Clients.GethL1Client)
	require.NoError(t, err)

	requireBatcherTxsToBeFromLayer(t, afterFailbackFromBlockNum, afterFailbackToBlockNum, DALayerEigenDA, harness.Endpoints.GethL1Endpoint, harness.BatchInboxAddr)

}

// requireBatcherTxsToBeFromLayer checks that the batcher transactions since startingFromBlockNum are all from the expectedLayer.
// It allows for up to 3 initial commitments to be of the wrong type, as the failover/failback might not have taken effect yet.
// It requires that at least 2 commitments of the expected type are present after the failover/failback.
func requireBatcherTxsToBeFromLayer(t *testing.T, fromBlockNum, toBlockNum uint64, expectedLayer DALayer, gethL1Endpoint string, batchInboxAddr common.Address) {
	batcherTxs, err := fetchBatcherTxs(gethL1Endpoint, batchInboxAddr.String(), fromBlockNum, toBlockNum)
	require.NoError(t, err)
	t.Logf("Fetched %d batcher transactions since block %d", len(batcherTxs), fromBlockNum)

	// We allow first 3 commitments to be of the wrong DA layer, as the failover/failback might not have taken effect yet.
	wrongCommitmentsToDiscard := 0
	for _, batcherTx := range batcherTxs {
		if batcherTx.daLayer != expectedLayer {
			wrongCommitmentsToDiscard++
		}
		// as soon as we see a commitment from expectedLayer, we stop discarding.
		if batcherTx.daLayer == expectedLayer {
			break
		}
	}
	batcherTxs = batcherTxs[wrongCommitmentsToDiscard:]
	t.Logf("Discarded %d commitments. %d left which should all be %v", wrongCommitmentsToDiscard, len(batcherTxs), expectedLayer)

	// After potentially discarding some commitments from wrong da layer, we expect all future commitments (at least 2) to be of the expectedLayer
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
