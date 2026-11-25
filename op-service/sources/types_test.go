package sources

import (
	"embed"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

//go:embed testdata
var blocksTestdata embed.FS

type testMetadata struct {
	Name   string `json:"name"`
	Fail   bool   `json:"fail,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func readJsonTestdata(t *testing.T, name string, dest any) {
	f, err := blocksTestdata.Open(name)
	require.NoError(t, err, "must open %q", name)
	require.NoError(t, json.NewDecoder(f).Decode(dest), "must json-decode %q", name)
	require.NoError(t, f.Close(), "must close %q", name)
}

func TestBlockHeaderJSON(t *testing.T) {
	headersDir, err := blocksTestdata.ReadDir("testdata/data/headers")
	require.NoError(t, err)

	for _, entry := range headersDir {
		if !strings.HasSuffix(entry.Name(), "_metadata.json") {
			continue
		}

		var metadata testMetadata
		readJsonTestdata(t, "testdata/data/headers/"+entry.Name(), &metadata)
		t.Run(metadata.Name, func(t *testing.T) {
			var header RPCHeader
			readJsonTestdata(t, "testdata/data/headers/"+strings.Replace(entry.Name(), "_metadata.json", "_data.json", 1), &header)

			h := header.computeBlockHash()
			if metadata.Fail {
				require.NotEqual(t, h, header.Hash, "expecting verification error")
			} else {
				require.Equal(t, h, header.Hash, "blockhash should verify ok")
			}
		})
	}
}

func TestRPCHeaderAmsterdamFields(t *testing.T) {
	zero := uint64(0)
	slotNumber := uint64(123)
	blockAccessListHash := randHash()
	hdr := &types.Header{
		ParentHash:          randHash(),
		UncleHash:           types.EmptyUncleHash,
		Coinbase:            common.Address{},
		Root:                randHash(),
		TxHash:              types.EmptyTxsHash,
		ReceiptHash:         randHash(),
		Bloom:               types.Bloom{},
		Difficulty:          big.NewInt(0),
		Number:              big.NewInt(1234),
		GasLimit:            30_000_000,
		GasUsed:             0,
		Time:                123456,
		Extra:               nil,
		MixDigest:           randHash(),
		Nonce:               types.BlockNonce{},
		BaseFee:             big.NewInt(100),
		WithdrawalsHash:     &types.EmptyWithdrawalsHash,
		ExcessBlobGas:       &zero,
		BlobGasUsed:         &zero,
		ParentBeaconRoot:    &common.Hash{},
		RequestsHash:        &types.EmptyRequestsHash,
		BlockAccessListHash: &blockAccessListHash,
		SlotNumber:          &slotNumber,
	}

	data, err := json.Marshal(hdr)
	require.NoError(t, err)

	var rpcHeader RPCHeader
	require.NoError(t, json.Unmarshal(data, &rpcHeader))
	require.Equal(t, hdr.BlockAccessListHash, rpcHeader.BlockAccessListHash)
	require.Equal(t, hdr.SlotNumber, (*uint64)(rpcHeader.SlotNumber))
	require.Equal(t, hdr.Hash(), rpcHeader.Hash)
	require.Equal(t, hdr.Hash(), rpcHeader.computeBlockHash())
}

// TestBlockJSON verifies L1 blocks as an RPC endpoint serves them. The mainnet-<number> fixtures are
// geth's eth_getBlockByNumber(n, true) responses for Ethereum mainnet; reth and erigon serve the same
// objects for them, up to key order. Every transaction of a passing fixture must re-encode to the
// hash the endpoint reports.
func TestBlockJSON(t *testing.T) {
	blocksDir, err := blocksTestdata.ReadDir("testdata/data/blocks")
	require.NoError(t, err)

	seen := make(map[byte]bool)
	var seenNonCanonicalAuthYParity bool
	ran, total := 0, 0
	for _, entry := range blocksDir {
		if !strings.HasSuffix(entry.Name(), "_metadata.json") {
			continue
		}
		total++

		var metadata testMetadata
		readJsonTestdata(t, "testdata/data/blocks/"+entry.Name(), &metadata)
		t.Run(metadata.Name, func(t *testing.T) {
			ran++
			dataFile := "testdata/data/blocks/" + strings.Replace(entry.Name(), "_metadata.json", "_data.json", 1)
			var block RPCBlock
			readJsonTestdata(t, dataFile, &block)

			err := block.Verify()
			if metadata.Fail {
				require.NotNil(t, err, "expecting verification error")
				require.ErrorContains(t, err, metadata.Reason, "validation failed for incorrect reason")
				return
			}
			require.NoError(t, err, "verification should pass")

			var wire struct {
				Transactions []struct {
					Hash common.Hash `json:"hash"`
				} `json:"transactions"`
			}
			readJsonTestdata(t, dataFile, &wire)
			require.Len(t, block.Transactions, len(wire.Transactions))
			for i, raw := range block.Transactions {
				require.Equal(t, wire.Transactions[i].Hash, raw.Hash(), "tx %d hash", i)
				seen[raw.Type()] = true
			}
			txs, err := block.Transactions.Geth()
			require.NoError(t, err)
			for _, tx := range txs {
				for _, auth := range tx.SetCodeAuthorizations() {
					seenNonCanonicalAuthYParity = seenNonCanonicalAuthYParity || auth.V > 1
				}
			}
		})
	}

	// The checks below hold for the whole fixture set, not for a -run filter that selects a subset.
	if ran < total {
		return
	}
	for _, typ := range []byte{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType} {
		require.True(t, seen[typ], "the fixtures must hold a tx of type %#x", typ)
	}
	// EIP-7702 bounds an authorization's yParity only to a byte, and mainnet holds larger values.
	require.True(t, seenNonCanonicalAuthYParity, "the fixtures must hold an authorization with a yParity above 1")
}

func TestBlockToExecutionPayloadIncludesEcotoneProperties(t *testing.T) {
	zero := uint64(0)

	hdr := &types.Header{
		ParentHash:       randHash(),
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         common.Address{},
		Root:             randHash(),
		TxHash:           types.EmptyTxsHash,
		ReceiptHash:      randHash(),
		Bloom:            types.Bloom{},
		Difficulty:       big.NewInt(0),
		Number:           big.NewInt(1234),
		GasLimit:         0,
		GasUsed:          0,
		Time:             123456,
		Extra:            make([]byte, 0),
		MixDigest:        randHash(),
		Nonce:            types.BlockNonce{},
		BaseFee:          big.NewInt(100),
		WithdrawalsHash:  &types.EmptyWithdrawalsHash,
		ExcessBlobGas:    &zero,
		BlobGasUsed:      &zero,
		ParentBeaconRoot: &common.Hash{},
	}
	rhdr := RPCHeader{
		ParentBeaconRoot: hdr.ParentBeaconRoot,
		ParentHash:       hdr.ParentHash,
		WithdrawalsRoot:  hdr.WithdrawalsHash,
		UncleHash:        hdr.UncleHash,
		Coinbase:         hdr.Coinbase,
		Root:             hdr.Root,
		TxHash:           hdr.TxHash,
		ReceiptHash:      hdr.ReceiptHash,
		Bloom:            eth.Bytes256(hdr.Bloom),
		Difficulty:       *(*hexutil.Big)(hdr.Difficulty),
		Number:           hexutil.Uint64(bigs.Uint64Strict(hdr.Number)),
		GasLimit:         hexutil.Uint64(hdr.GasLimit),
		GasUsed:          hexutil.Uint64(hdr.GasUsed),
		Time:             hexutil.Uint64(hdr.Time),
		Extra:            hdr.Extra,
		MixDigest:        hdr.MixDigest,
		Nonce:            hdr.Nonce,
		BaseFee:          (*hexutil.Big)(hdr.BaseFee),
		Hash:             hdr.Hash(),
		BlobGasUsed:      (*hexutil.Uint64)(hdr.BlobGasUsed),
		ExcessBlobGas:    (*hexutil.Uint64)(hdr.ExcessBlobGas),
	}

	block := RPCBlock{
		RPCHeader:    rhdr,
		Transactions: RawTransactions{},
		Withdrawals:  &types.Withdrawals{},
	}

	envelope, err := block.ExecutionPayloadEnvelope(false)
	require.NoError(t, err)

	require.NotNil(t, envelope.ParentBeaconBlockRoot)
	require.Equal(t, *envelope.ParentBeaconBlockRoot, *hdr.ParentBeaconRoot)
	require.NotNil(t, envelope.ExecutionPayload.ExcessBlobGas)
	require.Equal(t, *envelope.ExecutionPayload.ExcessBlobGas, *rhdr.ExcessBlobGas)
	require.NotNil(t, envelope.ExecutionPayload.BlobGasUsed)
	require.Equal(t, *envelope.ExecutionPayload.BlobGasUsed, *rhdr.BlobGasUsed)
}

func TestValidateL2Withdrawals(t *testing.T) {
	var block RPCBlock
	root := types.EmptyWithdrawalsHash
	require.NoError(t, block.validateL2Withdrawals(nil, nil))
	require.NoError(t, block.validateL2Withdrawals(&types.Withdrawals{}, &root))
	require.ErrorContains(t, block.validateL2Withdrawals(nil, &root), "expected empty withdrawals")
	require.ErrorContains(t, block.validateL2Withdrawals(&types.Withdrawals{{}}, &root), "expected empty withdrawals")
	// The block hash does not cover the list, so only this check catches one on a pre-Canyon block.
	require.ErrorContains(t, block.validateL2Withdrawals(&types.Withdrawals{}, nil), "expected no withdrawals")
	require.ErrorContains(t, block.validateL2Withdrawals(&types.Withdrawals{{}}, nil), "expected no withdrawals")
}
