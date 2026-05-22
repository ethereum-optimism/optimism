package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// Manifest describes deployer-facing schemas for one contracts ref.
type Manifest struct {
	SchemaHash string               `json:"schemaHash"`
	Structs    map[string]StructDef `json:"structs"`
}

// StructDef describes one generated struct shape.
type StructDef struct {
	Fields []Field `json:"fields"`
}

// Field describes one generated struct field.
type Field struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Struct   string `json:"struct,omitempty"`
}

// Hash returns the canonical schema hash of the manifest, excluding SchemaHash.
func (m Manifest) Hash() (string, error) {
	canonical, err := m.CanonicalJSON()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// MustHash returns the canonical schema hash or panics.
func (m Manifest) MustHash() string {
	hash, err := m.Hash()
	if err != nil {
		panic(err)
	}
	return hash
}

// WithHash returns a copy of the manifest with SchemaHash filled in.
func (m Manifest) WithHash() (Manifest, error) {
	hash, err := m.Hash()
	if err != nil {
		return Manifest{}, err
	}
	m.SchemaHash = hash
	return m, nil
}

// CanonicalJSON returns a deterministic JSON representation used for hashing.
func (m Manifest) CanonicalJSON() ([]byte, error) {
	structNames := make([]string, 0, len(m.Structs))
	for name := range m.Structs {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	canonical := canonicalManifest{
		Structs: make([]canonicalStruct, 0, len(structNames)),
	}
	for _, name := range structNames {
		def := m.Structs[name]
		canonical.Structs = append(canonical.Structs, canonicalStruct{
			Name:   name,
			Fields: def.Fields,
		})
	}

	out, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical manifest: %w", err)
	}
	return out, nil
}

type canonicalManifest struct {
	Structs []canonicalStruct `json:"structs"`
}

type canonicalStruct struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}
