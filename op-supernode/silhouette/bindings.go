package silhouette

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// CanonicalArtifactHash hashes the compact JSON encoding of a parsed artifact. Parsing before
// hashing removes whitespace, object-key order and alternate numeric spellings from the operator's
// input. Struct field order and encoding/json's sorted map keys make the resulting bytes stable.
func CanonicalArtifactHash(artifact any) (common.Hash, error) {
	if artifact == nil {
		return common.Hash{}, fmt.Errorf("cannot hash a nil artifact")
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		return common.Hash{}, fmt.Errorf("encode parsed artifact: %w", err)
	}
	return common.Hash(sha256.Sum256(raw)), nil
}

// BindingHashes returns the two artifact commitments carried by every proof batch. This is the
// authoritative Go API used by the verifier fixtures, devstack and the silhouette-bindings CLI.
func BindingHashes(rollupCfg *rollup.Config, depSet *depset.StaticConfigDependencySet) (common.Hash, common.Hash, error) {
	if rollupCfg == nil {
		return common.Hash{}, common.Hash{}, fmt.Errorf("rollup config is required")
	}
	if depSet == nil {
		return common.Hash{}, common.Hash{}, fmt.Errorf("dependency set is required")
	}
	rollupHash, err := CanonicalArtifactHash(rollupCfg)
	if err != nil {
		return common.Hash{}, common.Hash{}, fmt.Errorf("hash rollup config: %w", err)
	}
	depSetHash, err := CanonicalArtifactHash(depSet)
	if err != nil {
		return common.Hash{}, common.Hash{}, fmt.Errorf("hash dependency set: %w", err)
	}
	return rollupHash, depSetHash, nil
}
