package cannon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

const (
	proofsDir = "proofs"
)

type proofData struct {
	ClaimValue  hexutil.Bytes `json:"post"`
	StateData   hexutil.Bytes `json:"state-data"`
	ProofData   hexutil.Bytes `json:"proof-data"`
	OracleKey   hexutil.Bytes `json:"oracle-key,omitempty"`
	OracleValue hexutil.Bytes `json:"oracle-value,omitempty"`
}

type ProofGenerator interface {
	// GenerateProof executes cannon to generate a proof at the specified trace index in dataDir.
	GenerateProof(ctx context.Context, dataDir string, proofAt uint64) error
}

type CannonTraceProvider struct {
	dir       string
	prestate  string
	generator ProofGenerator
}

func NewTraceProvider(logger log.Logger, cfg *config.Config) *CannonTraceProvider {
	return &CannonTraceProvider{
		dir:       cfg.CannonDatadir,
		prestate:  cfg.CannonAbsolutePreState,
		generator: NewExecutor(logger, cfg),
	}
}

func (p *CannonTraceProvider) GetOracleData(ctx context.Context, i uint64) (*types.PreimageOracleData, error) {
	proof, err := p.loadProof(ctx, i)
	if err != nil {
		return nil, err
	}
	data := types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue)
	return &data, nil
}

func (p *CannonTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	proof, err := p.loadProof(ctx, i)
	if err != nil {
		return common.Hash{}, err
	}
	value := common.BytesToHash(proof.ClaimValue)

	if value == (common.Hash{}) {
		return common.Hash{}, errors.New("proof missing post hash")
	}
	return value, nil
}

func (p *CannonTraceProvider) GetPreimage(ctx context.Context, i uint64) ([]byte, []byte, error) {
	proof, err := p.loadProof(ctx, i)
	if err != nil {
		return nil, nil, err
	}
	value := ([]byte)(proof.StateData)
	if len(value) == 0 {
		return nil, nil, errors.New("proof missing state data")
	}
	data := ([]byte)(proof.ProofData)
	if len(data) == 0 {
		return nil, nil, errors.New("proof missing proof data")
	}
	return value, data, nil
}

func (p *CannonTraceProvider) AbsolutePreState(ctx context.Context) ([]byte, error) {
	path := filepath.Join(p.dir, p.prestate)
	file, err := os.Open(path)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot open state file (%v): %w", path, err)
	}
	defer file.Close()
	var state mipsevm.State
	err = json.NewDecoder(file).Decode(&state)
	if err != nil {
		return []byte{}, fmt.Errorf("invalid mipsevm state (%v): %w", path, err)
	}
	return state.EncodeWitness(), nil
}

func (p *CannonTraceProvider) loadProof(ctx context.Context, i uint64) (*proofData, error) {
	path := filepath.Join(p.dir, proofsDir, fmt.Sprintf("%d.json", i))
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := p.generator.GenerateProof(ctx, p.dir, i); err != nil {
			return nil, fmt.Errorf("generate cannon trace with proof at %v: %w", i, err)
		}
		// Try opening the file again now and it should exist.
		file, err = os.Open(path)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open proof file (%v): %w", path, err)
	}
	defer file.Close()
	var proof proofData
	err = json.NewDecoder(file).Decode(&proof)
	if err != nil {
		return nil, fmt.Errorf("failed to read proof (%v): %w", path, err)
	}
	return &proof, nil
}
