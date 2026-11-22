package sources

import (
	"encoding/json"
	"io/fs"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// l2FixturesDir holds op-reth's eth_getBlockByNumber(n, true) responses, as
// l2_<chainID>_<number>_block.json, each optionally with the block's eth_getBlockReceipts(n)
// response as l2_<chainID>_<number>_receipts.json, pretty-printed with every value unchanged.
// They pin the RPC wire format of the OP transaction and receipt types independently of this
// package's own codec. Every block file is checked; a new fixture needs no test change.
//
// l2_901_8 is a Lagoon block with a trailing post-exec tx, from the SDM acceptance system
// (newFixedPolicySDMRethSystem in op-acceptance-tests/tests/sdm) on op-reth: the block
// sdmtest.MustFindRepeatedSlotBlock returns, read from the sequencer's op-reth.
//
// l2_901_12 holds three user deposits after the L1-info deposit, all from one L1 block of the
// minimal system without fault proofs (presets.NewMinimalNoFaultProofs) on op-reth, each sent
// through OptimismPortal2.depositTransaction with a nonzero mint and value: a contract creation,
// a plain value transfer to an EOA, and a call that reverts on L2.
const l2FixturesDir = "testdata/l2"

// wireTx holds the transaction-object fields the fixtures are checked against.
type wireTx struct {
	Type       hexutil.Uint64  `json:"type"`
	Hash       common.Hash     `json:"hash"`
	SourceHash common.Hash     `json:"sourceHash"`
	From       common.Address  `json:"from"`
	To         *common.Address `json:"to"`
	Mint       *hexutil.Big    `json:"mint"`
	Value      *hexutil.Big    `json:"value"`
	Gas        hexutil.Uint64  `json:"gas"`
	IsSystemTx bool            `json:"isSystemTx"`
	Input      hexutil.Bytes   `json:"input"`
}

// TestRPCBlockDecodesOpRethFixtures decodes op-reth L2 blocks: every tx re-encodes to the hash
// op-reth reports, the block verifies against its transactions root, deposits and post-exec txs
// carry op-reth's field values, and the receipts verify against the receipts root with their OP
// fields set.
func TestRPCBlockDecodesOpRethFixtures(t *testing.T) {
	entries, err := fs.Glob(blocksTestdata, l2FixturesDir+"/*_block.json")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	seen := make(map[byte]bool)
	var seenMint, seenCreation, seenRevertedDeposit bool
	ran := 0
	for _, path := range entries {
		name := strings.TrimSuffix(strings.TrimPrefix(path, l2FixturesDir+"/"), "_block.json")
		t.Run(name, func(t *testing.T) {
			ran++
			data, err := blocksTestdata.ReadFile(path)
			require.NoError(t, err)

			var block RPCBlock
			require.NoError(t, json.Unmarshal(data, &block))
			require.NoError(t, block.Verify(), "tx root and block hash must verify")

			var wire struct {
				Transactions []wireTx `json:"transactions"`
			}
			require.NoError(t, json.Unmarshal(data, &wire))
			require.Len(t, block.Transactions, len(wire.Transactions))
			require.NotEmpty(t, block.Transactions)
			require.Equal(t, byte(optypes.DepositTxType), block.Transactions[0].Type(), "the L1-info deposit comes first")

			userCount := 0
			for i, raw := range block.Transactions {
				w := wire.Transactions[i]
				require.Equal(t, byte(w.Type), raw.Type(), "tx %d type", i)
				require.Equal(t, w.Hash, raw.Hash(), "tx %d hash", i)
				seen[raw.Type()] = true

				switch raw.Type() {
				case optypes.DepositTxType:
					dep, err := optypes.UnmarshalDepositTx(raw)
					require.NoError(t, err, "tx %d", i)
					require.Equal(t, w.SourceHash, dep.SourceHash, "tx %d sourceHash", i)
					require.Equal(t, w.From, dep.From, "tx %d from", i)
					require.Equal(t, w.To, dep.To, "tx %d to", i)
					require.Equal(t, bigOrZero(w.Mint).String(), bigOrZero((*hexutil.Big)(dep.Mint)).String(), "tx %d mint", i)
					require.Equal(t, bigOrZero(w.Value).String(), bigOrZero((*hexutil.Big)(dep.Value)).String(), "tx %d value", i)
					require.Equal(t, uint64(w.Gas), dep.Gas, "tx %d gas", i)
					require.Equal(t, w.IsSystemTx, dep.IsSystemTransaction, "tx %d isSystemTx", i)
					require.Equal(t, []byte(w.Input), dep.Data, "tx %d input", i)
					seenMint = seenMint || bigOrZero(w.Mint).Sign() != 0
					seenCreation = seenCreation || dep.To == nil
				case optypes.PostExecTxType:
					require.Equal(t, len(block.Transactions)-1, i, "the post-exec tx comes last")
					pe, err := optypes.UnmarshalPostExecTx(raw)
					require.NoError(t, err, "tx %d", i)
					require.Equal(t, []byte(w.Input), pe.Data, "tx %d input", i)
				default:
					userCount++
				}
			}

			userTxs, err := block.Transactions.UserTxs()
			require.NoError(t, err)
			require.Len(t, userTxs, userCount)

			envelope, err := block.ExecutionPayloadEnvelope(false)
			require.NoError(t, err)
			require.Len(t, envelope.ExecutionPayload.Transactions, len(block.Transactions))
			for i := range block.Transactions {
				require.Equal(t, []byte(block.Transactions[i]), []byte(envelope.ExecutionPayload.Transactions[i]), "payload tx %d passthrough", i)
			}

			receiptsData, err := blocksTestdata.ReadFile(l2FixturesDir + "/" + name + "_receipts.json")
			if err != nil {
				require.ErrorIs(t, err, fs.ErrNotExist)
				return
			}
			var receipts optypes.Receipts
			require.NoError(t, json.Unmarshal(receiptsData, &receipts))
			id := eth.BlockID{Hash: block.Hash, Number: uint64(block.Number)}
			require.NoError(t, validateReceipts(id, block.ReceiptHash, block.Transactions.Hashes(), receipts))
			// Re-encode the receipts and compare each OP field with op-reth's object, so a misnamed
			// JSON tag fails here instead of decoding to nil.
			var wireReceipts, ourReceipts []map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(receiptsData, &wireReceipts))
			reencoded, err := json.Marshal(receipts)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(reencoded, &ourReceipts))
			for i := range wireReceipts {
				for _, field := range []string{"depositNonce", "depositReceiptVersion", "l1GasPrice", "l1GasUsed", "l1Fee", "l1BaseFeeScalar", "l1BlobBaseFee", "l1BlobBaseFeeScalar", "operatorFeeScalar", "operatorFeeConstant", "daFootprintGasScalar"} {
					require.Equal(t, string(wireReceipts[i][field]), string(ourReceipts[i][field]), "receipt %d %s", i, field)
				}
			}
			for i, r := range receipts {
				switch block.Transactions[i].Type() {
				case optypes.DepositTxType:
					require.NotNil(t, r.DepositNonce, "deposit receipt %d depositNonce", i)
					seenRevertedDeposit = seenRevertedDeposit || r.Status == types.ReceiptStatusFailed
				case optypes.PostExecTxType:
				default:
					require.NotNil(t, r.L1Fee, "user receipt %d l1Fee", i)
					require.NotNil(t, r.L1GasUsed, "user receipt %d l1GasUsed", i)
					require.NotNil(t, r.L1GasPrice, "user receipt %d l1GasPrice", i)
				}
			}
		})
	}

	// The checks below hold for the whole fixture set, not for a -run filter that selects a subset.
	if ran < len(entries) {
		return
	}
	require.True(t, seen[optypes.DepositTxType], "the fixtures must hold a deposit tx")
	require.True(t, seen[optypes.PostExecTxType], "the fixtures must hold a post-exec tx")
	require.True(t, seenMint, "the fixtures must hold a deposit with a nonzero mint")
	require.True(t, seenCreation, "the fixtures must hold a contract-creation deposit")
	require.True(t, seenRevertedDeposit, "the fixtures must hold a deposit whose receipt has status failed")
}

func bigOrZero(b *hexutil.Big) *big.Int {
	if b == nil {
		return new(big.Int)
	}
	return b.ToInt()
}
