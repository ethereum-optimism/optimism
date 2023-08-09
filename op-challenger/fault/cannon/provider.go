package cannon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	proofsDir = "proofs"
)

type proofData struct {
	ClaimValue   hexutil.Bytes `json:"post"`
	StateData    hexutil.Bytes `json:"state-data"`
	ProofData    hexutil.Bytes `json:"proof-data"`
	OracleKey    hexutil.Bytes `json:"oracle-key,omitempty"`
	OracleValue  hexutil.Bytes `json:"oracle-value,omitempty"`
	OracleOffset uint32        `json:"oracle-offset,omitempty"`
}

type ProofGenerator interface {
	// GenerateProof executes cannon to generate a proof at the specified trace index in dataDir.
	GenerateProof(ctx context.Context, dataDir string, proofAt uint64) error
}

type CannonTraceProvider struct {
	logger    log.Logger
	dir       string
	prestate  string
	generator ProofGenerator

	// lastStep stores the last step in the actual trace if known. 0 indicates unknown.
	// Cached as an optimisation to avoid repeatedly attempting to execute beyond the end of the trace.
	lastStep uint64
}

func NewTraceProvider(ctx context.Context, logger log.Logger, cfg *config.Config, l1Client bind.ContractCaller) (*CannonTraceProvider, error) {
	l2Client, err := ethclient.DialContext(ctx, cfg.CannonL2)
	if err != nil {
		return nil, fmt.Errorf("dial l2 cleint %v: %w", cfg.CannonL2, err)
	}
	defer l2Client.Close() // Not needed after fetching the inputs
	gameCaller, err := bindings.NewFaultDisputeGameCaller(cfg.GameAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("create caller for game %v: %w", cfg.GameAddress, err)
	}
	l1Head, err := fetchLocalInputs(ctx, cfg.GameAddress, gameCaller, l2Client)
	if err != nil {
		return nil, fmt.Errorf("fetch local game inputs: %w", err)
	}
	return &CannonTraceProvider{
		logger:    logger,
		dir:       cfg.CannonDatadir,
		prestate:  cfg.CannonAbsolutePreState,
		generator: NewExecutor(logger, cfg, l1Head),
	}, nil
}

func (p *CannonTraceProvider) GetOracleData(ctx context.Context, i uint64) (*types.PreimageOracleData, error) {
	proof, err := p.loadProofData(ctx, i)
	if err != nil {
		return nil, err
	}
	data := types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset)
	return &data, nil
}

func (p *CannonTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	proof, state, err := p.loadProof(ctx, i)
	if err != nil {
		return common.Hash{}, err
	}
	if proof == nil && state != nil {
		// Use the hash from the final state
		return crypto.Keccak256Hash(state.EncodeWitness()), nil
	}
	value := common.BytesToHash(proof.ClaimValue)

	if value == (common.Hash{}) {
		return common.Hash{}, errors.New("proof missing post hash")
	}
	return value, nil
}

func (p *CannonTraceProvider) GetPreimage(ctx context.Context, i uint64) ([]byte, []byte, error) {
	proof, err := p.loadProofData(ctx, i)
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
	state, err := parseState(path)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	return state.EncodeWitness(), nil
}

// loadProofData loads the proof data for the specified step.
// If the requested index is beyond the end of the actual trace, the proof data from the last step is returned.
// Cannon will be executed a second time if required to generate the full proof data.
func (p *CannonTraceProvider) loadProofData(ctx context.Context, i uint64) (*proofData, error) {
	proof, state, err := p.loadProof(ctx, i)
	if err != nil {
		return nil, err
	} else if proof == nil && state != nil {
		p.logger.Info("Re-executing to generate proof for last step", "step", state.Step)
		proof, _, err = p.loadProof(ctx, state.Step)
		if err != nil {
			return nil, err
		}
		if proof == nil {
			return nil, fmt.Errorf("proof at step %v was not generated", i)
		}
		return proof, nil
	}
	return proof, nil
}

// loadProof will attempt to load or generate the proof data at the specified index
// If the requested index is beyond the end of the actual trace:
//   - When the actual trace length is known, the proof data from the last step is returned with nil state
//   - When the actual trace length is not yet know, the state from after the last step is returned with nil proofData
//     and the actual trace length is cached for future runs
func (p *CannonTraceProvider) loadProof(ctx context.Context, i uint64) (*proofData, *mipsevm.State, error) {
	if p.lastStep != 0 && i > p.lastStep {
		// If the requested index is after the last step in the actual trace, use the last step
		i = p.lastStep
	}
	path := filepath.Join(p.dir, proofsDir, fmt.Sprintf("%d.json", i))
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := p.generator.GenerateProof(ctx, p.dir, i); err != nil {
			return nil, nil, fmt.Errorf("generate cannon trace with proof at %v: %w", i, err)
		}
		// Try opening the file again now and it should exist.
		file, err = os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			// Expected proof wasn't generated, check if we reached the end of execution
			state, err := parseState(filepath.Join(p.dir, finalState))
			if err != nil {
				return nil, nil, fmt.Errorf("cannot read final state: %w", err)
			}
			if state.Exited && state.Step < i {
				p.logger.Warn("Requested proof was after the program exited", "proof", i, "last", state.Step)
				// The final instruction has already been applied to this state, so the last step we can execute
				// is one before its Step value.
				p.lastStep = state.Step - 1
				return nil, state, nil
			} else {
				return nil, nil, fmt.Errorf("expected proof not generated but final state was not exited, requested step %v, final state at step %v", i, state.Step)
			}
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open proof file (%v): %w", path, err)
	}
	defer file.Close()
	var proof proofData
	err = json.NewDecoder(file).Decode(&proof)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read proof (%v): %w", path, err)
	}
	return &proof, nil, nil
}
