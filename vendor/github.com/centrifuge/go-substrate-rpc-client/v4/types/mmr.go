package types

import (
	"encoding/json"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types/codec"
)

// GenerateMMRProofResponse contains the generate proof rpc response
type GenerateMMRProofResponse struct {
	BlockHash H256
	Leaf      MMRLeaf
	Proof     MMRProof
}

// UnmarshalJSON fills u with the JSON encoded byte array given by b
func (d *GenerateMMRProofResponse) UnmarshalJSON(bz []byte) error {
	var tmp struct {
		BlockHash string `json:"blockHash"`
		Leaf      string `json:"leaf"`
		Proof     string `json:"proof"`
	}
	if err := json.Unmarshal(bz, &tmp); err != nil {
		return err
	}
	err := codec.DecodeFromHex(tmp.BlockHash, &d.BlockHash)
	if err != nil {
		return err
	}
	var encodedLeaf MMREncodableOpaqueLeaf
	err = codec.DecodeFromHex(tmp.Leaf, &encodedLeaf)
	if err != nil {
		return err
	}
	err = codec.Decode(encodedLeaf, &d.Leaf)
	if err != nil {
		return err
	}
	err = codec.DecodeFromHex(tmp.Proof, &d.Proof)
	if err != nil {
		return err
	}
	return nil
}

type MMREncodableOpaqueLeaf Bytes

// MMRProof is a MMR proof
type MMRProof struct {
	// The index of the leaf the proof is for.
	LeafIndex U64
	// Number of leaves in MMR, when the proof was generated.
	LeafCount U64
	// Proof elements (hashes of siblings of inner nodes on the path to the leaf).
	Items []H256
}

type MMRLeaf struct {
	Version               MMRLeafVersion
	ParentNumberAndHash   ParentNumberAndHash
	BeefyNextAuthoritySet BeefyNextAuthoritySet
	ParachainHeads        H256
}

type MMRLeafVersion U8

type ParentNumberAndHash struct {
	ParentNumber U32
	Hash         Hash
}

type BeefyNextAuthoritySet struct {
	// ID
	ID U64
	// Number of validators in the set.
	Len U32
	// Merkle Root Hash build from BEEFY uncompressed AuthorityIds.
	Root H256
}
