package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	ProofsDir      = "proofs"
	diskStateCache = "state.json.gz"
)

type ProofData struct {
	ClaimValue   common.Hash   `json:"post"`
	StateData    hexutil.Bytes `json:"state-data"`
	ProofData    hexutil.Bytes `json:"proof-data"`
	OracleKey    hexutil.Bytes `json:"oracle-key,omitempty"`
	OracleValue  hexutil.Bytes `json:"oracle-value,omitempty"`
	OracleOffset uint32        `json:"oracle-offset,omitempty"`
}

type ProofGenerator interface {
	// GenerateProof executes FPVM binary to generate a proof at the specified trace index in dataDir.
	GenerateProof(ctx context.Context, dataDir string, proofAt uint64) error
}

type diskStateCacheObj struct {
	Step uint64 `json:"step"`
}

// ReadLastStep reads the tracked last step from disk.
func ReadLastStep(dir string) (uint64, error) {
	state := diskStateCacheObj{}
	file, err := ioutil.OpenDecompressed(filepath.Join(dir, diskStateCache))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&state)
	if err != nil {
		return 0, err
	}
	return state.Step, nil
}

// WriteLastStep writes the last step and proof to disk as a persistent cache.
func WriteLastStep(dir string, proof *ProofData, step uint64) error {
	state := diskStateCacheObj{Step: step}
	lastStepFile := filepath.Join(dir, diskStateCache)
	if err := ioutil.WriteCompressedJson(lastStepFile, state); err != nil {
		return fmt.Errorf("failed to write last step to %v: %w", lastStepFile, err)
	}
	if err := ioutil.WriteCompressedJson(filepath.Join(dir, ProofsDir, fmt.Sprintf("%d.json.gz", step)), proof); err != nil {
		return fmt.Errorf("failed to write proof: %w", err)
	}
	return nil
}

// below methods and definitions are only to be used for testing
type preimageOpts []string

type PreimageOpt func() preimageOpts

func PreimageLoad(key preimage.Key, offset uint32) PreimageOpt {
	return func() preimageOpts {
		return []string{"--stop-at-preimage", fmt.Sprintf("%v@%v", common.Hash(key.PreimageKey()).Hex(), offset)}
	}
}

func FirstPreimageLoadOfType(preimageType string) PreimageOpt {
	return func() preimageOpts {
		return []string{"--stop-at-preimage-type", preimageType}
	}
}

func FirstKeccakPreimageLoad() PreimageOpt {
	return FirstPreimageLoadOfType("keccak")
}

func FirstPrecompilePreimageLoad() PreimageOpt {
	return FirstPreimageLoadOfType("precompile")
}

func PreimageLargerThan(size int) PreimageOpt {
	return func() preimageOpts {
		return []string{"--stop-at-preimage-larger-than", strconv.Itoa(size)}
	}
}
