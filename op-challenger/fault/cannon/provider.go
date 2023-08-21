package cannon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	// lastProof stores the proof data to use for all steps extended beyond lastStep
	lastProof *proofData
}

func NewTraceProvider(ctx context.Context, logger log.Logger, cfg *config.Config, l1Client bind.ContractCaller) (*CannonTraceProvider, error) {
	l2Client, err := ethclient.DialContext(ctx, cfg.CannonL2)
	if err != nil {
		return nil, fmt.Errorf("dial l2 client %v: %w", cfg.CannonL2, err)
	}
	defer l2Client.Close() // Not needed after fetching the inputs
	gameCaller, err := bindings.NewFaultDisputeGameCaller(cfg.GameAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("create caller for game %v: %w", cfg.GameAddress, err)
	}
	localInputs, err := fetchLocalInputs(ctx, cfg.GameAddress, gameCaller, l2Client)
	if err != nil {
		return nil, fmt.Errorf("fetch local game inputs: %w", err)
	}
	return NewTraceProviderFromInputs(logger, cfg, localInputs), nil
}

func NewTraceProviderFromInputs(logger log.Logger, cfg *config.Config, localInputs LocalGameInputs) *CannonTraceProvider {
	return &CannonTraceProvider{
		logger:    logger,
		dir:       cfg.CannonDatadir,
		prestate:  cfg.CannonAbsolutePreState,
		generator: NewExecutor(logger, cfg, localInputs),
	}
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

func (p *CannonTraceProvider) GetStepData(ctx context.Context, i uint64) ([]byte, []byte, *types.PreimageOracleData, error) {
	proof, err := p.loadProof(ctx, i)
	if err != nil {
		return nil, nil, nil, err
	}
	value := ([]byte)(proof.StateData)
	if len(value) == 0 {
		return nil, nil, nil, errors.New("proof missing state data")
	}
	data := ([]byte)(proof.ProofData)
	if data == nil {
		return nil, nil, nil, errors.New("proof missing proof data")
	}
	var oracleData *types.PreimageOracleData
	if len(proof.OracleKey) > 0 {
		oracleData = types.NewPreimageOracleData(proof.OracleKey, proof.OracleValue, proof.OracleOffset)
	}
	return value, data, oracleData, nil
}

func (p *CannonTraceProvider) AbsolutePreState(ctx context.Context) ([]byte, error) {
	state, err := parseState(p.prestate)
	if err != nil {
		return nil, fmt.Errorf("cannot load absolute pre-state: %w", err)
	}
	return state.EncodeWitness(), nil
}

// loadProof will attempt to load or generate the proof data at the specified index
// If the requested index is beyond the end of the actual trace it is extended with no-op instructions.
func (p *CannonTraceProvider) loadProof(ctx context.Context, i uint64) (*proofData, error) {
	if p.lastProof != nil && i > p.lastStep {
		// If the requested index is after the last step in the actual trace, extend the final no-op step
		return p.lastProof, nil
	}
	path := filepath.Join(p.dir, proofsDir, fmt.Sprintf("%d.json", i))
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := p.generator.GenerateProof(ctx, p.dir, i); err != nil {
			return nil, fmt.Errorf("generate cannon trace with proof at %v: %w", i, err)
		}
		// Try opening the file again now and it should exist.
		file, err = os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			// Expected proof wasn't generated, check if we reached the end of execution
			state, err := parseState(filepath.Join(p.dir, finalState))
			if err != nil {
				return nil, fmt.Errorf("cannot read final state: %w", err)
			}
			if state.Exited && state.Step <= i {
				p.logger.Warn("Requested proof was after the program exited", "proof", i, "last", state.Step)
				// The final instruction has already been applied to this state, so the last step we can execute
				// is one before its Step value.
				p.lastStep = state.Step - 1
				// Extend the trace out to the full length using a no-op instruction that doesn't change any state
				// No execution is done, so no proof-data or oracle values are required.
				witness := state.EncodeWitness()
				proof := &proofData{
					ClaimValue:   crypto.Keccak256(witness),
					StateData:    witness,
					ProofData:    []byte{},
					OracleKey:    nil,
					OracleValue:  nil,
					OracleOffset: 0,
				}
				p.lastProof = proof
				return proof, nil
			} else {
				return nil, fmt.Errorf("expected proof not generated but final state was not exited, requested step %v, final state at step %v", i, state.Step)
			}
		}
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
