package signer

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

func TestBlockPayloadArgs(t *testing.T) {
	l2ChainID := big.NewInt(100)
	payloadBytes := []byte("arbitraryData")
	addr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	exampleDomain := [32]byte{0: 123, 1: 42}
	args := NewBlockPayloadArgs(exampleDomain, l2ChainID, payloadBytes, &addr)
	out, err := json.MarshalIndent(args, "  ", "  ")
	require.NoError(t, err)
	content := string(out)
	// previously erroneously included in every request. Not used on server-side. Should be dropped now.
	require.NotContains(t, content, "PayloadBytes")
	// mistyped as list of ints in v0
	require.Contains(t, content, `"domain": [`)
	require.Contains(t, content, ` 123,`)
	require.Contains(t, content, ` 42,`)
	require.Contains(t, content, `"chainId": 100`)
	// mistyped as standard Go bytes, hence base64
	require.Contains(t, content, `"payloadHash": "7qa7ZZHSC1LytldPsgv3J5zQPVgWE9jqHojIK4QAFEs="`)
	require.Contains(t, content, `"senderAddress": "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"`)

	// Assert signing hash, to detect breaking changes to hashing
	msg, err := args.Message()
	require.NoError(t, err)
	h := msg.ToSigningHash()
	require.Equal(t, common.HexToHash("0xc724de7aefa316a12cbbd5edc39d512830598cce5d6ac81d1b518ef69450ad77"), h)

	argsV2 := BlockPayloadArgsV2{
		Domain:        eth.Bytes32(exampleDomain),
		ChainID:       eth.ChainIDFromUInt64(100),
		PayloadHash:   PayloadHash(payloadBytes),
		SenderAddress: &addr,
	}
	msg2, err := argsV2.Message()
	require.NoError(t, err)
	require.Equal(t, msg, msg2)
	h2 := msg2.ToSigningHash()
	require.NoError(t, err)
	require.Equal(t, h, h2, "signing hash still the same in V2 API")
}

func TestSigningHash_DifferentDomain(t *testing.T) {
	chainID := big.NewInt(100)

	payloadBytes := []byte("arbitraryData")
	msg, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID, payloadBytes, nil).Message()
	require.NoError(t, err, "creating first signing hash")

	msg2, err := NewBlockPayloadArgs([32]byte{3}, chainID, payloadBytes, nil).Message()
	require.NoError(t, err, "creating second signing hash")

	hash := msg.ToSigningHash()
	hash2 := msg2.ToSigningHash()
	require.NotEqual(t, hash, hash2, "signing hash should be different when domain is different")
}

func TestSigningHash_DifferentChainID(t *testing.T) {
	chainID1 := big.NewInt(100)
	chainID2 := big.NewInt(101)

	payloadBytes := []byte("arbitraryData")
	msg, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID1, payloadBytes, nil).Message()
	require.NoError(t, err, "creating first signing hash")

	msg2, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID2, payloadBytes, nil).Message()
	require.NoError(t, err, "creating second signing hash")

	hash := msg.ToSigningHash()
	hash2 := msg2.ToSigningHash()
	require.NotEqual(t, hash, hash2, "signing hash should be different when chain ID is different")
}

func TestSigningHash_DifferentPayload(t *testing.T) {
	chainID := big.NewInt(100)

	msg, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID, []byte("payload1"), nil).Message()
	require.NoError(t, err, "creating first signing hash")

	msg2, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID, []byte("payload2"), nil).Message()
	require.NoError(t, err, "creating second signing hash")

	hash := msg.ToSigningHash()
	hash2 := msg2.ToSigningHash()
	require.NotEqual(t, hash, hash2, "signing hash should be different when payload is different")
}

func TestSigningHash_LimitChainID(t *testing.T) {
	// ChainID with bitlen 257
	chainID := big.NewInt(1)
	chainID = chainID.SetBit(chainID, 256, 1)
	_, err := NewBlockPayloadArgs(SigningDomainBlocksV1, chainID, []byte("arbitraryData"), nil).Message()
	require.ErrorContains(t, err, "chain_id is too large")
}
