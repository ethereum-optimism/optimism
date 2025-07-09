package spec

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const (
	FeatureInterop = "interop"
	FeatureFaucet  = "faucet"
)

// ChainSpec represents the network parameters for a chain
type ChainSpec struct {
	Name      string
	NetworkID string
	Nodes     map[string]NodeConfig
}

// NodeConfig represents the configuration for a chain node
type NodeConfig struct {
	IsSequencer bool
	ELType      string
	CLType      string
}

type FeatureList []string

func (fl FeatureList) Contains(feature string) bool {
	for _, f := range fl {
		if f == feature {
			return true
		}
	}
	return false
}

// EnclaveSpec represents the parsed chain specifications from the YAML
type EnclaveSpec struct {
	Chains   []*ChainSpec
	Features FeatureList
}

// NetworkParams represents the network parameters section in the YAML
type NetworkParams struct {
	Name      string `yaml:"name"`
	NetworkID string `yaml:"network_id"`
}

// ChainConfig represents a chain configuration in the YAML
type ChainConfig struct {
	NetworkParams NetworkParams                `yaml:"network_params"`
	Participants  map[string]ParticipantConfig `yaml:"participants"`
}

// NodeConfig represents a node configuration in the YAML
type ParticipantConfig struct {
	Sequencer bool          `yaml:"sequencer"`
	EL        ComponentType `yaml:"el"`
	CL        ComponentType `yaml:"cl"`
}

// ComponentType represents a component type in the YAML
type ComponentType struct {
	Type string `yaml:"type"`
}

// InteropConfig represents the interop section in the YAML
type SuperchainConfig struct {
	Enabled bool `yaml:"enabled"`
}

// FaucetConfig represents the faucet section in the YAML
type FaucetConfig struct {
	Enabled bool `yaml:"enabled"`
}

// OptimismPackage represents the optimism_package section in the YAML
type OptimismPackage struct {
	Faucet      FaucetConfig                `yaml:"faucet"`
	Superchains map[string]SuperchainConfig `yaml:"superchains"`
	Chains      map[string]ChainConfig      `yaml:"chains"`
}

// YAMLSpec represents the root of the YAML document
type YAMLSpec struct {
	OptimismPackage OptimismPackage `yaml:"optimism_package"`
}

type Spec struct{}

type SpecOption func(*Spec)

func NewSpec(opts ...SpecOption) *Spec {
	s := &Spec{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type featureExtractor func(YAMLSpec, string) bool

var featuresMap = map[string]featureExtractor{
	FeatureInterop: interopExtractor,
	FeatureFaucet:  faucetExtractor,
}

func interopExtractor(yamlSpec YAMLSpec, _feature string) bool {
	for _, superchain := range yamlSpec.OptimismPackage.Superchains {
		if superchain.Enabled {
			return true
		}
	}
	return false
}

func faucetExtractor(yamlSpec YAMLSpec, _feature string) bool {
	return yamlSpec.OptimismPackage.Faucet.Enabled
}

// ExtractData parses a YAML document and returns the chain specifications
func (s *Spec) ExtractData(r io.Reader) (*EnclaveSpec, error) {
	var yamlSpec YAMLSpec
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&yamlSpec); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	var features []string
	for feature, extractor := range featuresMap {
		if extractor(yamlSpec, feature) {
			features = append(features, feature)
		}
	}

	result := &EnclaveSpec{
		Chains:   make([]*ChainSpec, 0, len(yamlSpec.OptimismPackage.Chains)),
		Features: features,
	}

	// Extract chain specifications
	for name, chain := range yamlSpec.OptimismPackage.Chains {

		nodes := make(map[string]NodeConfig, len(chain.Participants))
		for name, participant := range chain.Participants {
			elType := participant.EL.Type
			clType := participant.CL.Type
			if elType == "" {
				elType = "op-geth"
			}
			if clType == "" {
				clType = "op-node"
			}
			nodes[name] = NodeConfig{
				IsSequencer: participant.Sequencer,
				ELType:      elType,
				CLType:      clType,
			}
		}

		result.Chains = append(result.Chains, &ChainSpec{
			Name:      name,
			NetworkID: chain.NetworkParams.NetworkID,
			Nodes:     nodes,
		})
	}

	return result, nil
}
