package script

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
)

// DependencyMetadata lists the inputs (contract sources and build settings)
// that were involved in building a certain Call dependency.
type DependencyMetadata struct {
	OptimizerSettings json.RawMessage        `json:"optimizerSettings"`
	MetadataSettings  json.RawMessage        `json:"metadataSettings"`
	EVMVersion        string                 `json:"evmVersion"`
	CompilerVersion   string                 `json:"compilerVersion"`
	Sources           map[string]common.Hash `json:"sources"`
}

// Call is an invocation of a foundry script that applies a modification on top of a pre-state
// to produce a post-state. The diff created by the call is cacheable, see CallDiff.
type Call struct {
	Prestate common.Hash `json:"prestate"`

	Target string `json:"target"`
	Sig    string `json:"sig"`

	Dependencies map[string]DependencyMetadata `json:"dependencies"`

	Addrs map[string]common.Address `json:"addrs"`

	Args map[string]json.RawMessage `json:"args"`
}

// Hash computes a commitment that uniquely identifies a call, to be used as cache key.
func (sc *Call) Hash() common.Hash {
	// hash of version, pre-state hash, dependencies, call-args, script target
	data, err := json.Marshal(sc)
	if err != nil {
		panic(fmt.Errorf("invalid ScriptCall: %w", err))
	}
	return crypto.Keccak256Hash(data)
}

// ConstructCall turns a pre-state and a Script into an actionable call that can be executed and cached.
func ConstructCall(log log.Logger, preScriptHash common.Hash, s Script) (*Call, error) {
	// Scripts define addresses statically as struct of address fields.
	// But we want to pre-validate that it's a valid JSON map of addresses, before invoking the script.
	// So we decode again into the type we expect.
	encodedAddrs, err := json.Marshal(s.ScriptAddresses())
	if err != nil {
		return nil, fmt.Errorf("failed to JSON-encode script addresses: %w", err)
	}
	var addrsMap map[string]common.Address
	if err := json.Unmarshal(encodedAddrs, &addrsMap); err != nil {
		return nil, fmt.Errorf("failed to decode addresses JSON into addresses map: %w", err)
	}
	log.Info("Loaded script artifact addresses", "count", len(addrsMap))

	// Similar to addresses, the config must be a map with string keys.
	encodedArgs, err := json.Marshal(s.ScriptArgs())
	if err != nil {
		return nil, fmt.Errorf("failed to JSON-encode script args: %w", err)
	}
	var argsMap map[string]json.RawMessage
	if err := json.Unmarshal(encodedArgs, &argsMap); err != nil {
		return nil, fmt.Errorf("failed to decode args JSON into args map: %w", err)
	}
	log.Info("Loaded script arguments", "count", len(argsMap))

	scriptCall := &Call{
		Prestate:     preScriptHash,
		Target:       s.ScriptTarget(),
		Sig:          s.ScriptSig(),
		Dependencies: make(map[string]DependencyMetadata),
		Addrs:        addrsMap,
		Args:         argsMap,
	}

	dependencies := s.ScriptDependencies()
	log.Info("Immediate script dependencies", "dependencies", dependencies)

	artifactsPath := "forge-artifacts"

	// Retrieve full list of dependencies (inspect `sources` of the Dependencies() that don't end with ".s.sol")
	for _, dep := range dependencies {
		p := filepath.Join(artifactsPath, dep)
		artifact, err := foundry.ReadArtifact(p)
		if err != nil {
			return nil, fmt.Errorf("failed to read artifact %q: %w", dep, err)
		}
		data := DependencyMetadata{
			OptimizerSettings: artifact.Metadata.Settings.Optimizer,
			MetadataSettings:  artifact.Metadata.Settings.Metadata,
			EVMVersion:        artifact.Metadata.Settings.EVMVersion,
			CompilerVersion:   artifact.Metadata.Compiler.Version,
			Sources:           make(map[string]common.Hash),
		}

		// add each of the sources to the dependency set
		for sourcePath, sourceEntry := range artifact.Metadata.Sources {
			sourcePath = filepath.ToSlash(sourcePath) // normalize the filepath (replace separator with slash)
			data.Sources[sourcePath] = sourceEntry.Keccak256
			log.Info("dependency source", "path", sourcePath)
		}
		scriptCall.Dependencies[dep] = data

		log.Info("Loaded dependency", "dep", dep,
			"optimizer", data.OptimizerSettings,
			"metadata", data.MetadataSettings,
			"evmVersion", data.EVMVersion,
			"compiler", data.CompilerVersion,
			"sources", data.Sources,
		)
	}

	return scriptCall, nil
}
