package cannon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-program/client/l1"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var kzg4844Ctx *gokzg4844.Context

func kzg() *gokzg4844.Context {
	if kzg4844Ctx == nil {
		ctx, err := gokzg4844.NewContext4096Secure()
		if err != nil {
			panic("unable to load kzg trusted setup")
		}
		kzg4844Ctx = ctx
	}
	return kzg4844Ctx
}

type preimageSource func(key common.Hash) ([]byte, error)

type preimageLoader struct {
	getPreimage preimageSource
}

func newPreimageLoader(getPreimage preimageSource) *preimageLoader {
	return &preimageLoader{
		getPreimage: getPreimage,
	}
}

func (l *preimageLoader) LoadPreimage(proof *proofData) (*types.PreimageOracleData, error) {
	if len(proof.OracleKey) == 0 {
		return nil, nil
	}
	switch preimage.KeyType(proof.OracleKey[0]) {
	case preimage.BlobKeyType:
		return l.loadBlobPreimage(proof)
	default:
		return types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset), nil
	}
}

func (l *preimageLoader) loadBlobPreimage(proof *proofData) (*types.PreimageOracleData, error) {
	hint, hintBytes, err := parseHint(string(proof.LastHint))
	if err != nil {
		return nil, fmt.Errorf("failed to parse last hint: %w", err)
	}
	if hint != l1.HintL1Blob {
		return nil, fmt.Errorf("invalid last hint for blob type: %v", hint)
	}

	blobVersionHash := common.Hash(hintBytes[:32])

	// Start by using the last hint to load the blob commitment
	commitmentKey := preimage.Sha256Key(blobVersionHash)
	commitment, err := l.getPreimage(commitmentKey.PreimageKey())
	if err != nil {
		return nil, fmt.Errorf("failed to get blob commitment preimage (%v): %w", commitmentKey, err)
	}

	// Load the full blob data
	// Reconstruct the full blob from the 4096 field elements.
	blob := eth.Blob{}
	fieldElemKey := make([]byte, 80)
	copy(fieldElemKey[:48], commitment)
	requiredFieldElement := -1
	for i := 0; i < params.BlobTxFieldElementsPerBlob; i++ {
		binary.BigEndian.PutUint64(fieldElemKey[72:], uint64(i))
		key := preimage.BlobKey(crypto.Keccak256(fieldElemKey)).PreimageKey()
		if bytes.Equal(key[:], proof.OracleKey) {
			requiredFieldElement = i
		}
		fieldElement, err := l.getPreimage(key)
		if err != nil {
			return nil, fmt.Errorf("failed to load field element %v with key %v", i, common.Hash(key))
		}
		copy(blob[i<<5:(i+1)<<5], fieldElement[:])
	}
	if requiredFieldElement == -1 {
		return nil, fmt.Errorf("no field element key matched: %v", proof.OracleKey)
	}

	blobCommitment, err := blob.ComputeKZGCommitment()
	if err != nil || !bytes.Equal(blobCommitment[:], commitment[:]) {
		return nil, fmt.Errorf("invalid blob commitment: %w", err)
	}

	kzgProof, _, err := kzg().ComputeKZGProof(gokzg4844.Blob(blob), gokzg4844.Scalar(blob[requiredFieldElement<<5:(requiredFieldElement+1)<<5]), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to compute kzg proof: %w", err)
	}

	return types.NewPreimageOracleBlobData(proof.OracleKey, proof.OracleValue, proof.OracleOffset, uint64(requiredFieldElement), commitment, kzgProof[:]), nil
}

// parseHint parses a hint string in wire protocol. Returns the hint type, requested hash and error (if any).
func parseHint(hint string) (string, []byte, error) {
	hintType, bytesStr, found := strings.Cut(hint, " ")
	if !found {
		return "", nil, fmt.Errorf("unsupported hint: %s", hint)
	}

	hintBytes, err := hexutil.Decode(bytesStr)
	if err != nil {
		return "", make([]byte, 0), fmt.Errorf("invalid bytes: %s", bytesStr)
	}
	return hintType, hintBytes, nil
}
