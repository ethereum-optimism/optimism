package sdm

import (
	"encoding/json"
	"strings"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/sdm/sdmtest"
	sdmpkg "github.com/ethereum-optimism/optimism/op-chain-ops/pkg/sdm"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

// getOPGasRefund reads the opGasRefund field from a transaction receipt via
// raw JSON RPC. The boolean return value reports whether the field was present.
func getOPGasRefund(t devtest.T, l2EL *dsl.L2ELNode, txHash common.Hash) (uint64, bool) {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var raw json.RawMessage
	err := rpcClient.CallContext(t.Ctx(), &raw, "eth_getTransactionReceipt", txHash)
	t.Require().NoError(err, "eth_getTransactionReceipt RPC failed for tx %s", txHash)
	t.Require().NotNil(raw, "receipt %s not found", txHash)

	var result struct {
		OPGasRefund *hexutil.Uint64 `json:"opGasRefund"`
	}
	err = json.Unmarshal(raw, &result)
	t.Require().NoError(err, "failed to unmarshal receipt %s", txHash)
	if result.OPGasRefund == nil {
		return 0, false
	}
	return uint64(*result.OPGasRefund), true
}

func getReceiptGasUsed(t devtest.T, l2EL *dsl.L2ELNode, txHash common.Hash) uint64 {
	rpcClient := l2EL.Escape().L2EthClient().RPC()
	var result struct {
		GasUsed hexutil.Uint64 `json:"gasUsed"`
	}
	err := rpcClient.CallContext(t.Ctx(), &result, "eth_getTransactionReceipt", txHash)
	t.Require().NoError(err, "eth_getTransactionReceipt RPC failed for tx %s", txHash)
	return uint64(result.GasUsed)
}

func assertFixtureBlockOracle(t devtest.T, sys *sdmtest.RethSystem, block *sdmpkg.RPCBlock, blockNum uint64) {
	postExecTx, postExecPos := sdmpkg.FindPostExecTransaction(block)
	t.Require().NotNil(postExecTx, "fixture block must contain a post-exec tx")
	t.Require().Equal(len(block.Transactions)-1, postExecPos, "fixture post-exec tx must be trailing")

	postExecCount := 0
	expectedIndexes := make([]uint64, 0, len(block.Transactions))
	for i, tx := range block.Transactions {
		switch uint64(tx.Type) {
		case uint64(optypes.PostExecTxType):
			postExecCount++
		case uint64(gethtypes.DepositTxType):
		default:
			expectedIndexes = append(expectedIndexes, uint64(i))
		}
	}
	t.Require().Equal(1, postExecCount, "fixture block must contain exactly one post-exec tx")
	assertPostExecTxHashIsCanonical(t, sys.L2EL, postExecTx)

	payload, err := optypes.DecodePostExecPayload(postExecTx.Input)
	t.Require().NoError(err, "fixture post-exec payload must decode")
	t.Require().Equal(blockNum, payload.BlockNumber, "fixture payload must anchor to its block")
	t.Require().Equal(optypes.PostExecPayloadVersion, payload.Version, "fixture payload version must match")
	t.Require().Len(payload.GasRefundEntries, len(expectedIndexes),
		"fixture must emit one entry per committed normal transaction")

	for i, entry := range payload.GasRefundEntries {
		t.Require().Equal(expectedIndexes[i], entry.Index, "fixture entry indexes must match normal tx order")
		t.Require().Equal(uint64(1), entry.GasRefund, "fixture refund must be exactly one gas")
		target := block.Transactions[entry.Index]
		refund, present := getOPGasRefund(t, sys.L2EL, target.Hash)
		t.Require().True(present, "fixture target receipt %s must expose opGasRefund", target.Hash)
		t.Require().Equal(uint64(1), refund, "fixture target receipt must expose one gas")
	}

	for _, tx := range block.Transactions {
		if uint64(tx.Type) != uint64(gethtypes.DepositTxType) && uint64(tx.Type) != uint64(optypes.PostExecTxType) {
			continue
		}
		refund, present := getOPGasRefund(t, sys.L2EL, tx.Hash)
		t.Require().False(present, "deposit and post-exec receipts must omit opGasRefund")
		t.Require().Zero(refund, "non-target receipt refund must decode to zero")
	}

	// Structural replay is independent of the fixture policy. It re-executes without rebates and
	// must reconcile the embedded one-gas claims with receipts and block gas accounting.
	replay := sdmtest.ReplayBlockWithSDM(t, sys.L2EL, blockNum)
	t.Require().Equal(block.Hash, replay.BlockHash, "replay block hash must match fixture block")
	t.Require().True(replay.PostExecTxPresent, "structural replay must observe the trailing post-exec tx")
	t.Require().NotNil(replay.EmbeddedPayload, "structural replay must report the embedded payload")
	t.Require().Equal(payload, replay.EmbeddedPayload, "replay payload must match the canonical tx bytes")
	t.Require().Empty(replay.Mismatches, "valid fixture payload must have no structural mismatches")
	t.Require().Zero(replay.Summary.MismatchCount, "valid fixture replay mismatch count must be zero")
	t.Require().Equal(uint64(block.GasUsed), replay.Summary.BlockGasUsed,
		"replay canonical block gas must match the block header")
	t.Require().Equal(uint64(len(expectedIndexes)), replay.Summary.PayloadRefundTotal,
		"one-gas fixture payload total must equal the number of normal transactions")
	t.Require().Equal(uint64(block.GasUsed)+uint64(len(expectedIndexes)), replay.Summary.BlockRawGasUsed,
		"raw block gas must equal canonical gas plus fixture refunds")

	rows := make(map[uint64]sdmpkg.ReplaySDMTx, len(replay.Txs))
	for _, row := range replay.Txs {
		_, duplicate := rows[row.TxIndex]
		t.Require().False(duplicate, "replay must not return duplicate tx index %d", row.TxIndex)
		rows[row.TxIndex] = row
	}
	for i, tx := range block.Transactions {
		if uint64(tx.Type) == uint64(optypes.PostExecTxType) {
			continue
		}
		row, ok := rows[uint64(i)]
		t.Require().True(ok, "replay must include non-post-exec tx index %d", i)
		t.Require().Equal(tx.Hash, row.TxHash, "replay tx hash at index %d must match", i)
		t.Require().Equal(getReceiptGasUsed(t, sys.L2EL, tx.Hash), row.CanonicalGasUsed,
			"receipt gas and replay canonical gas at index %d must match", i)
		if uint64(tx.Type) == uint64(gethtypes.DepositTxType) {
			t.Require().Nil(row.OPGasRefundPayload, "deposit replay row must not have a refund claim")
			t.Require().Equal(row.RawGasUsed, row.CanonicalGasUsed,
				"deposit raw and canonical gas must match")
			continue
		}
		t.Require().NotNil(row.OPGasRefundPayload, "normal tx replay row must have a refund claim")
		t.Require().Equal(uint64(1), *row.OPGasRefundPayload, "fixture replay refund must be one gas")
		t.Require().Equal(row.RawGasUsed, row.CanonicalGasUsed+1,
			"normal tx raw gas must equal canonical gas plus one")
	}
}

func assertFixtureVerifierReceipts(t devtest.T, sys *sdmtest.RethSystem, block *sdmpkg.RPCBlock) {
	verifierBlock := sdmtest.GetBlockWithTxs(t, sys.L2ELVerifier, uint64(block.Number))
	t.Require().Equal(block.Hash, verifierBlock.Hash, "stock verifier fixture block hash must match")
	t.Require().Len(verifierBlock.Transactions, len(block.Transactions),
		"stock verifier fixture transaction count must match")
	for i, tx := range block.Transactions {
		t.Require().Equal(tx.Hash, verifierBlock.Transactions[i].Hash,
			"stock verifier transaction hash at index %d must match", i)
		producerRefund, producerPresent := getOPGasRefund(t, sys.L2EL, tx.Hash)
		verifierRefund, verifierPresent := getOPGasRefund(t, sys.L2ELVerifier, tx.Hash)
		t.Require().Equal(producerPresent, verifierPresent,
			"stock verifier opGasRefund field presence for tx %s must match", tx.Hash)
		t.Require().Equal(producerRefund, verifierRefund,
			"stock verifier opGasRefund for tx %s must match", tx.Hash)
	}
}

// assertPostExecTxHashIsCanonical asserts op-reth serves and resolves the post-exec tx (and its
// receipt) under the hash Go's PostExecTx.Hash() produces — keccak256(0x7D || Data). Hashing via
// Hash() rather than a hand-rolled keccak checks the Go hasher and op-reth agree — the cross-client
// guarantee op-service/sources relies on.
func assertPostExecTxHashIsCanonical(t devtest.T, l2EL *dsl.L2ELNode, postExecTx *sdmpkg.RPCTransaction) {
	wantHash := gethtypes.NewTx(&gethtypes.PostExecTx{Data: []byte(postExecTx.Input)}).Hash()

	t.Require().Equal(wantHash, postExecTx.Hash,
		"op-reth-served post-exec tx hash %s must equal PostExecTx.Hash() %s (keccak256(0x7D || Data), matching TxDeposit)",
		postExecTx.Hash, wantHash)

	rpcClient := l2EL.Escape().L2EthClient().RPC()

	var txRaw json.RawMessage
	err := rpcClient.CallContext(t.Ctx(), &txRaw, "eth_getTransactionByHash", wantHash)
	t.Require().NoError(err, "eth_getTransactionByHash RPC failed for canonical hash %s", wantHash)
	t.Require().False(isNullJSONResult(txRaw),
		"op-reth must resolve the post-exec tx under its canonical hash %s", wantHash)

	var receiptRaw json.RawMessage
	err = rpcClient.CallContext(t.Ctx(), &receiptRaw, "eth_getTransactionReceipt", wantHash)
	t.Require().NoError(err, "eth_getTransactionReceipt RPC failed for canonical hash %s", wantHash)
	t.Require().False(isNullJSONResult(receiptRaw),
		"op-reth must resolve the post-exec receipt under its canonical hash %s", wantHash)
}

// isNullJSONResult reports whether a raw JSON-RPC result is absent (null or empty) — i.e. not found.
func isNullJSONResult(raw json.RawMessage) bool {
	return len(raw) == 0 || strings.TrimSpace(string(raw)) == "null"
}

// assertPostExecDAFootprint verifies Jovian's blobGasUsed accounting for a block with a 0x7D.
// Deposits are omitted because block accounting excludes them while their receipts do not.
func assertPostExecDAFootprint(t devtest.T, l2EL *dsl.L2ELNode, block *sdmpkg.RPCBlock) {
	rpcClient := l2EL.Escape().L2EthClient().RPC()

	var header struct {
		BlobGasUsed *hexutil.Uint64 `json:"blobGasUsed"`
	}
	err := rpcClient.CallContext(t.Ctx(), &header, "eth_getBlockByHash", block.Hash, false)
	t.Require().NoError(err, "eth_getBlockByHash RPC failed for block %s", block.Hash)
	t.Require().NotNil(header.BlobGasUsed,
		"blobGasUsed must be set on block %s: SDM rides Lagoon, which is post-Jovian", block.Hash)
	blockDAFootprint := uint64(*header.BlobGasUsed)

	var (
		totalDAFootprint uint64
		postExecSeen     bool
	)
	for _, tx := range block.Transactions {
		if uint64(tx.Type) == uint64(gethtypes.DepositTxType) {
			continue
		}

		var receipt struct {
			BlobGasUsed *hexutil.Uint64 `json:"blobGasUsed"`
		}
		err := rpcClient.CallContext(t.Ctx(), &receipt, "eth_getTransactionReceipt", tx.Hash)
		t.Require().NoError(err, "eth_getTransactionReceipt RPC failed for tx %s", tx.Hash)
		t.Require().NotNil(receipt.BlobGasUsed, "nil receipt blobGasUsed for tx %s", tx.Hash)

		if uint64(tx.Type) == uint64(optypes.PostExecTxType) {
			postExecSeen = true
			t.Require().Zero(uint64(*receipt.BlobGasUsed),
				"post-exec tx %s must report a zero DA footprint on its receipt", tx.Hash)
		}
		totalDAFootprint += uint64(*receipt.BlobGasUsed)
	}

	t.Require().True(postExecSeen, "block %s must contain a post-exec tx", block.Hash)
	t.Require().Equal(blockDAFootprint, totalDAFootprint,
		"per-tx receipt DA footprints must sum to the header total on block %s", block.Hash)
	// Ensure the sum is not vacuously zero.
	t.Require().Positive(blockDAFootprint, "user txs must accrue a DA footprint for this to bite")
}
