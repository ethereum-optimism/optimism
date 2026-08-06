package nuts

import (
	"encoding/json"
	"fmt"
)

const (
	// BundleVersion is the current NUT bundle schema version.
	BundleVersion = "1.1.0"
	// BundleVersionNoExtraGas is the schema version predating the extraGas field. It stays
	// supported because a bundle locked at this version is regenerated, for provenance
	// verification, by the generator of its own locked commit — which cannot emit anything newer.
	BundleVersionNoExtraGas = "1.0.0"
)

// BundleMetadata is the metadata block of a NUT bundle.
type BundleMetadata struct {
	Version string `json:"version"`
	// ExtraGas is added to the activation block's gas limit on top of the bundle's own
	// transactions, covering activation deposits the consensus layer emits outside the bundle —
	// deposits that cannot be expressed in the bundle format, such as one carrying a non-zero
	// mint. It is reserved unconditionally, so that the amount does not depend on anything the
	// system config reconstructed from the activation block cannot see.
	ExtraGas uint64 `json:"extraGas"`
}

// Validate reports whether the metadata declares a schema version this build understands, and
// whether its extraGas usage is consistent with that version.
func (m BundleMetadata) Validate() error {
	switch m.Version {
	case BundleVersion:
		return nil
	case BundleVersionNoExtraGas:
		if m.ExtraGas != 0 {
			return fmt.Errorf("NUT bundle version %s must not declare extraGas, got %d",
				BundleVersionNoExtraGas, m.ExtraGas)
		}
		return nil
	default:
		return fmt.Errorf("unsupported NUT bundle version: got %q, want %q or %q",
			m.Version, BundleVersion, BundleVersionNoExtraGas)
	}
}

// ReadBundleMetadata parses the metadata block out of a NUT bundle's JSON contents, ignoring the
// transactions. Tooling uses it to inspect a bundle without decoding megabytes of calldata.
func ReadBundleMetadata(content []byte) (BundleMetadata, error) {
	var bundle struct {
		Metadata BundleMetadata `json:"metadata"`
	}
	if err := json.Unmarshal(content, &bundle); err != nil {
		return BundleMetadata{}, fmt.Errorf("parsing NUT bundle metadata: %w", err)
	}
	return bundle.Metadata, nil
}
