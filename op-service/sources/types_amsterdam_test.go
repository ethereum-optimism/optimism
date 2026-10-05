package sources

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// amsterdamHeaderJSON is a post-Glamsterdam (Amsterdam) header exactly as served by
// go-ethereum v1.17.7 over eth_getBlockByNumber: it carries the EIP-7928
// blockAccessListHash and EIP-7843 slotNumber fields, and "hash" is the canonical hash
// computed by upstream over all 23 header fields.
const amsterdamHeaderJSON = `{"parentHash":"0x0000000000000000000000000000000000000000000000000000000000000001","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x0000000000000000000000000000000000000000","stateRoot":"0x0000000000000000000000000000000000000000000000000000000000000002","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","difficulty":"0x0","number":"0x989680","gasLimit":"0x3938700","gasUsed":"0x0","timestamp":"0x6ac4fd60","extraData":"0x","mixHash":"0x0000000000000000000000000000000000000000000000000000000000000000","nonce":"0x0000000000000000","baseFeePerGas":"0x7","withdrawalsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","blobGasUsed":"0x0","excessBlobGas":"0x0","parentBeaconBlockRoot":"0x0000000000000000000000000000000000000000000000000000000000000000","requestsHash":"0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","blockAccessListHash":"0xba01000000000000000000000000000000000000000000000000000000000000","slotNumber":"0xac6000","hash":"0x8a5873e9569d658c029c64934fb5136ae61f51824ad0658499010864ae39b9c1"}`

const amsterdamHeaderHash = "0x8a5873e9569d658c029c64934fb5136ae61f51824ad0658499010864ae39b9c1"

// TestRPCHeaderAmsterdamFields checks that an L1 header carrying the Glamsterdam fields
// is parsed, converted and hash-verified: this is the check that would otherwise fail on
// every Sepolia block after the activation at timestamp 1791294816.
func TestRPCHeaderAmsterdamFields(t *testing.T) {
	var hdr RPCHeader
	require.NoError(t, json.Unmarshal([]byte(amsterdamHeaderJSON), &hdr))
	require.NotNil(t, hdr.BlockAccessListHash)
	require.NotNil(t, hdr.SlotNumber)
	require.Equal(t, uint64(11296768), uint64(*hdr.SlotNumber))

	want := common.HexToHash(amsterdamHeaderHash)
	require.Equal(t, want, hdr.Hash, "fixture must carry the canonical hash")

	// The geth header carries both fields and hashes to the canonical value.
	geth := hdr.CreateGethHeader()
	require.NotNil(t, geth.BlockAccessListHash)
	require.Equal(t, *hdr.BlockAccessListHash, *geth.BlockAccessListHash)
	require.NotNil(t, geth.SlotNumber)
	require.Equal(t, uint64(*hdr.SlotNumber), *geth.SlotNumber)
	require.Equal(t, want, geth.Hash())

	// Info with trustCache=false recomputes and compares the hash.
	info, err := hdr.Info(false, true)
	require.NoError(t, err)
	require.Equal(t, want, info.Hash())

	// Dropping either field must make the hash check fail loudly, which is how a
	// missing mapping would surface at the fork block.
	for _, mutate := range []func(h *RPCHeader){
		func(h *RPCHeader) { h.BlockAccessListHash = nil },
		func(h *RPCHeader) { h.SlotNumber = nil },
	} {
		broken := hdr
		mutate(&broken)
		_, err := broken.Info(false, true)
		require.ErrorContains(t, err, "failed to verify block hash")
	}
}

// TestRPCHeaderPreAmsterdamUnchanged checks that a pre-fork header (no Glamsterdam keys in
// the JSON) still maps to a geth header without those fields and keeps its hash.
func TestRPCHeaderPreAmsterdamUnchanged(t *testing.T) {
	var hdr RPCHeader
	require.NoError(t, json.Unmarshal([]byte(amsterdamHeaderJSON), &hdr))
	hdr.BlockAccessListHash = nil
	hdr.SlotNumber = nil
	geth := hdr.CreateGethHeader()
	require.Nil(t, geth.BlockAccessListHash)
	require.Nil(t, geth.SlotNumber)
	// Re-marshalling must not emit the keys for a pre-fork header (omitempty).
	out, err := json.Marshal(&hdr)
	require.NoError(t, err)
	require.NotContains(t, string(out), "blockAccessListHash")
	require.NotContains(t, string(out), "slotNumber")
	// And it must hash exactly as a 21-field (Prague) header: the same fixture minus the two
	// Amsterdam fields, as computed by upstream go-ethereum.
	want := common.HexToHash("0x674ef048b9dc0acfb0837431fddf9bf19b689effe7207df912cbf774899338be")
	require.Equal(t, want, geth.Hash())
	hdr.Hash = want
	info, err := hdr.Info(false, true)
	require.NoError(t, err)
	require.Equal(t, want, info.Hash())
}

// TestRPCBlockExecutionPayloadEnvelopeRejectsAmsterdamFields checks that a block carrying the
// Glamsterdam header fields is refused by ExecutionPayloadEnvelope instead of being converted
// into a payload that silently drops them (and whose block hash could then never be verified).
// Only L2 blocks are converted to payloads, and the L2 does not activate Amsterdam, so this is a
// guard against misuse rather than a live path.
func TestRPCBlockExecutionPayloadEnvelopeRejectsAmsterdamFields(t *testing.T) {
	var hdr RPCHeader
	require.NoError(t, json.Unmarshal([]byte(amsterdamHeaderJSON), &hdr))

	for _, tc := range []struct {
		name   string
		mutate func(h *RPCHeader)
	}{
		{"both fields", func(h *RPCHeader) {}},
		{"blockAccessListHash only", func(h *RPCHeader) { h.SlotNumber = nil }},
		{"slotNumber only", func(h *RPCHeader) { h.BlockAccessListHash = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := hdr
			tc.mutate(&h)
			block := RPCBlock{RPCHeader: h, Withdrawals: &types.Withdrawals{}}
			for _, trust := range []bool{true, false} {
				_, err := block.ExecutionPayloadEnvelope(trust)
				require.ErrorContains(t, err, "Amsterdam header fields", "trustCache=%v", trust)
			}
		})
	}

	// Without the Amsterdam fields the same block converts as before (trusted hash, no
	// re-verification, since clearing the fields changes the canonical hash).
	h := hdr
	h.BlockAccessListHash = nil
	h.SlotNumber = nil
	block := RPCBlock{RPCHeader: h, Withdrawals: &types.Withdrawals{}}
	envelope, err := block.ExecutionPayloadEnvelope(true)
	require.NoError(t, err)
	require.Equal(t, h.Hash, envelope.ExecutionPayload.BlockHash)
}
