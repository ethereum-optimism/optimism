package batchconsensus

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	VerifyBatchConsensusMethod = "verifyBatchConsensus"
	MockProof                  = "op-batcher-consensus-poc"
)

var verifierABI = mustParseVerifierABI()

func mustParseVerifierABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{
		"type":"function",
		"name":"verifyBatchConsensus",
		"inputs":[
			{"name":"l1ChainId","type":"uint256"},
			{"name":"l2ChainId","type":"uint256"},
			{"name":"batchInbox","type":"address"},
			{"name":"batcher","type":"address"},
			{"name":"blobVersionedHashes","type":"bytes32[]"},
			{"name":"proof","type":"bytes"}
		],
		"outputs":[{"name":"ok","type":"bool"}],
		"stateMutability":"view"
	}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}

// BuildMockVerifyCalldata builds the POC verifier calldata. The final proof bytes are intentionally
// opaque so the mock proof can be replaced by a Commonware certificate without changing derivation.
func BuildMockVerifyCalldata(l1ChainID, l2ChainID *big.Int, batchInbox common.Address, batcher common.Address, blobHashes []common.Hash) ([]byte, error) {
	if l1ChainID == nil {
		return nil, fmt.Errorf("missing L1 chain ID")
	}
	if l2ChainID == nil {
		return nil, fmt.Errorf("missing L2 chain ID")
	}
	return verifierABI.Pack(
		VerifyBatchConsensusMethod,
		l1ChainID,
		l2ChainID,
		batchInbox,
		batcher,
		blobHashes,
		[]byte(MockProof),
	)
}

func DecodeVerifyResult(out []byte) (bool, error) {
	values, err := verifierABI.Unpack(VerifyBatchConsensusMethod, out)
	if err != nil {
		return false, err
	}
	if len(values) != 1 {
		return false, fmt.Errorf("expected one verifier return value, got %d", len(values))
	}
	ok, valid := values[0].(bool)
	if !valid {
		return false, fmt.Errorf("expected bool verifier return value, got %T", values[0])
	}
	return ok, nil
}
