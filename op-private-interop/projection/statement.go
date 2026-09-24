package projection

import (
	"encoding/binary"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	// PublicValuesWords is the number of 32-byte words in PublicValuesV1.
	PublicValuesWords = 21
	// PublicValuesLength is the byte length of PublicValuesV1 (the static ABI tuple of 21 words).
	PublicValuesLength = PublicValuesWords * 32
)

// PublicValuesMagic is word 0 of PublicValuesV1.
var PublicValuesMagic = crypto.Keccak256Hash([]byte("optimism.private-projection.public-values.v1"))

// Word indices of PublicValuesV1 (§C.2).
const (
	WordMagic = iota
	WordChainID
	WordProjectionConfigHash
	WordPrivateConfigHash
	WordDepSetHash
	WordParentHash
	WordAnchorNumber
	WordAnchorHash
	WordAnchorOutputRoot
	WordRecoveryHash
	WordFirstBlock
	WordLastBlock
	WordParentOutputRoot
	WordPrivateTerminalBlockHash
	WordPrivateTerminalParentHash
	WordL1Head
	WordPrivateDataHash
	WordProjectionHash
	WordOutputsRoot
	WordMessagesRoot
	WordTerminalOutput
)

func u256(n uint64) common.Hash {
	var w common.Hash
	binary.BigEndian.PutUint64(w[24:], n)
	return w
}

// PublicValues is the 672-byte PublicValuesV1 the sp1-private-projection-v1 relation commits.
// Every word is taken from the admission-computed statement.
func PublicValues(s *Statement) [PublicValuesLength]byte {
	words := [PublicValuesWords]common.Hash{
		WordMagic:                     PublicValuesMagic,
		WordChainID:                   s.ChainID,
		WordProjectionConfigHash:      s.ProjectionConfigHash,
		WordPrivateConfigHash:         s.PrivateConfigHash,
		WordDepSetHash:                s.Claim.DepSetHash,
		WordParentHash:                s.ParentHash,
		WordAnchorNumber:              u256(s.Continuation.Anchor.Number),
		WordAnchorHash:                s.Continuation.Anchor.Hash,
		WordAnchorOutputRoot:          s.Continuation.OutputRoot,
		WordRecoveryHash:              s.Continuation.RecoveryHash,
		WordFirstBlock:                u256(s.Claim.FirstBlock),
		WordLastBlock:                 u256(s.Claim.LastBlock),
		WordParentOutputRoot:          s.Claim.ParentOutputRoot,
		WordPrivateTerminalBlockHash:  s.Claim.PrivateTerminalBlockHash,
		WordPrivateTerminalParentHash: s.Claim.PrivateTerminalParentHash,
		WordL1Head:                    s.Claim.L1Head,
		WordPrivateDataHash:           s.Claim.PrivateDataHash,
		WordProjectionHash:            s.ProjectionHash,
		WordOutputsRoot:               s.OutputsRoot,
		WordMessagesRoot:              s.MessagesRoot,
		WordTerminalOutput:            s.TerminalOutput,
	}
	var out [PublicValuesLength]byte
	for i, w := range words {
		copy(out[32*i:], w[:])
	}
	return out
}

// ConfigHash is the claim's rollupConfigHash (§C.3.3): a canonical binary hash of the projection
// chain geometry and its private_projection consensus constants.
func ConfigHash(c *Config, ctx Context) common.Hash {
	var chainID common.Hash
	if ctx.ChainID != nil {
		chainID = common.BigToHash(ctx.ChainID)
	}
	u64 := func(n uint64) []byte { var b [8]byte; binary.BigEndian.PutUint64(b[:], n); return b[:] }
	flag := func(b bool) []byte {
		if b {
			return []byte{1}
		}
		return []byte{0}
	}
	verifier := crypto.Keccak256Hash([]byte(c.Verifier))
	return crypto.Keccak256Hash(
		[]byte(ConfigDomain),
		chainID[:],
		u64(ctx.GenesisNumber), ctx.GenesisHash[:], u64(ctx.GenesisTime), u64(ctx.BlockTime),
		verifier[:],
		c.GenesisOutputRoot[:], c.ProgramVKey[:], c.PrivateConfigHash[:], c.DependencySetHash[:],
		flag(c.AllowEvents), flag(c.MockProofs),
	)
}

// PrivateConfigHash (§B.2) hashes the exact deployed artifact bytes; neither side re-serialises.
func PrivateConfigHash(privateRollupJSON, l1ChainConfigJSON []byte) common.Hash {
	rollupHash := crypto.Keccak256Hash(privateRollupJSON)
	l1Hash := crypto.Keccak256Hash(l1ChainConfigJSON)
	return crypto.Keccak256Hash([]byte(PrivateConfigDomain), rollupHash[:], l1Hash[:])
}

// DependencySetHash (§C.3.4) hashes the distinct chain IDs of the private chain's dependency set,
// as uint256 big-endian words sorted ascending, prefixed by their count.
func DependencySetHash(ids []eth.ChainID) common.Hash {
	sorted := slices.Clone(ids)
	slices.SortFunc(sorted, func(a, b eth.ChainID) int { return a.Cmp(b) })
	sorted = slices.CompactFunc(sorted, func(a, b eth.ChainID) bool { return a.Cmp(b) == 0 })
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(len(sorted)))
	h := crypto.NewKeccakState()
	_, _ = h.Write([]byte(DepSetDomain))
	_, _ = h.Write(n[:])
	for _, id := range sorted {
		w := id.Bytes32()
		_, _ = h.Write(w[:])
	}
	var out common.Hash
	_, _ = h.Read(out[:])
	return out
}
