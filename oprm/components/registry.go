package components

import (
	"fmt"
	"sort"
)

type Kind string

const (
	KindMonorepoGo   Kind = "monorepo-go"
	KindMonorepoRust Kind = "monorepo-rust"
	KindExternalGo   Kind = "external-go"
)

type VersionPolicy struct {
	SupportsMajor   bool
	SupportsMinor   bool
	SupportsPatch   bool
	AutoIncrementRC bool
}

type ComponentSpec struct {
	ID                string
	Kind              Kind
	DisplayName       string
	GitHubOwner       string
	GitHubRepo        string
	BaseBranch        string
	TagPrefix         string
	ChangeScope       []string
	ReleaseNotesScope []string
	Versioning        VersionPolicy
}

type Registry struct {
	components map[string]ComponentSpec
}

func NewRegistry() *Registry {
	items := []ComponentSpec{
		{
			ID:          "op-geth",
			Kind:        KindExternalGo,
			DisplayName: "op-geth",
			GitHubOwner: "ethereum-optimism",
			GitHubRepo:  "op-geth",
			BaseBranch:  "optimism",
			TagPrefix:   "v",
			ChangeScope: []string{"**/*"},
			Versioning: VersionPolicy{
				SupportsMajor:   true,
				SupportsMinor:   true,
				SupportsPatch:   true,
				AutoIncrementRC: true,
			},
		},
		{
			ID:                "op-node",
			Kind:              KindMonorepoGo,
			DisplayName:       "op-node",
			GitHubOwner:       "ethereum-optimism",
			GitHubRepo:        "optimism",
			BaseBranch:        "develop",
			TagPrefix:         "op-node",
			ChangeScope:       []string{"op-node/**", "go.mod", "go.sum", "op-core/**", "op-service/**", "op-chain-ops/**"},
			ReleaseNotesScope: []string{"op-node/**", "go.*", "op-core/**", "op-service/**"},
			Versioning:        VersionPolicy{SupportsMajor: true, SupportsMinor: true, SupportsPatch: true, AutoIncrementRC: true},
		},
		{
			ID:                "op-batcher",
			Kind:              KindMonorepoGo,
			DisplayName:       "op-batcher",
			GitHubOwner:       "ethereum-optimism",
			GitHubRepo:        "optimism",
			BaseBranch:        "develop",
			TagPrefix:         "op-batcher",
			ChangeScope:       []string{"op-batcher/**", "go.mod", "go.sum", "op-core/**", "op-service/**", "op-chain-ops/**"},
			ReleaseNotesScope: []string{"op-batcher/**", "go.*", "op-core/**", "op-service/**"},
			Versioning:        VersionPolicy{SupportsMajor: true, SupportsMinor: true, SupportsPatch: true, AutoIncrementRC: true},
		},
		{
			ID:                "kona-node",
			Kind:              KindMonorepoRust,
			DisplayName:       "kona-node",
			GitHubOwner:       "ethereum-optimism",
			GitHubRepo:        "optimism",
			BaseBranch:        "develop",
			TagPrefix:         "kona-node",
			ChangeScope:       []string{"rust/kona/**", "rust/Cargo.toml", "rust/op-alloy/**", "rust/alloy-op*/**"},
			ReleaseNotesScope: []string{"rust/kona/**", "rust/Cargo.toml", "rust/op-alloy/**", "rust/alloy-op*/**"},
			Versioning:        VersionPolicy{SupportsMajor: true, SupportsMinor: true, SupportsPatch: true, AutoIncrementRC: true},
		},
		{
			ID:                "op-reth",
			Kind:              KindMonorepoRust,
			DisplayName:       "op-reth",
			GitHubOwner:       "ethereum-optimism",
			GitHubRepo:        "optimism",
			BaseBranch:        "develop",
			TagPrefix:         "op-reth",
			ChangeScope:       []string{"rust/op-reth/**", "rust/Cargo.toml", "rust/op-alloy/**", "rust/alloy-op*/**"},
			ReleaseNotesScope: []string{"rust/op-reth/**", "rust/Cargo.toml", "rust/op-alloy/**", "rust/alloy-op*/**"},
			Versioning:        VersionPolicy{SupportsMajor: true, SupportsMinor: true, SupportsPatch: true, AutoIncrementRC: true},
		},
	}
	byID := make(map[string]ComponentSpec, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	return &Registry{components: byID}
}

func (r *Registry) Get(id string) (ComponentSpec, error) {
	item, ok := r.components[id]
	if !ok {
		return ComponentSpec{}, fmt.Errorf("unsupported component %q", id)
	}
	return item, nil
}

func (r *Registry) MustGet(id string) ComponentSpec {
	item, err := r.Get(id)
	if err != nil {
		panic(err)
	}
	return item
}

func (r *Registry) IDs() []string {
	ids := make([]string, 0, len(r.components))
	for id := range r.components {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
