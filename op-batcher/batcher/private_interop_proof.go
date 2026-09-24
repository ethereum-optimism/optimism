package batcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// PublicationRequestVersion is the version of the proof command's stdin protocol
// (`<ProofCommand> --publication-request`, spec-sound-profile §G.2).
const PublicationRequestVersion = 2

// MaxPublicationRequestBytes is the proof command's stdin cap. The request carries only
// span-bounded data (the public span, private data, configs and the dependency set); recovery
// blocks and witnesses are fetched by the child over RPC. A larger request fails before the child
// is spawned.
const MaxPublicationRequestBytes = 128 << 20

// proofCommandWaitDelay bounds how long a timed-out proof command's pipes are drained.
const proofCommandWaitDelay = 2 * time.Second

// publicationBlock is one public span block of the request.
type publicationBlock struct {
	Timestamp    uint64          `json:"timestamp"`
	Epoch        uint64          `json:"epoch"`
	Transactions []hexutil.Bytes `json:"transactions"`
}

// publicationRequest is the v2 request. The field set is normative (§G.2).
type publicationRequest struct {
	Version              uint64             `json:"version"`
	Prover               string             `json:"prover"`
	PrivateRPC           string             `json:"private_rpc"`
	ProjectionRPC        string             `json:"projection_rpc"`
	L1RPC                string             `json:"l1_rpc"`
	PrivateConfig        hexutil.Bytes      `json:"private_config"`
	L1Config             hexutil.Bytes      `json:"l1_config"`
	ProjectionConfig     hexutil.Bytes      `json:"projection_config"`
	DependencySet        hexutil.Bytes      `json:"dependency_set"`
	ParentHash           common.Hash        `json:"parent_hash"`
	Anchor               eth.BlockID        `json:"anchor"`
	AnchorOutput         common.Hash        `json:"anchor_output"`
	RecoveryHash         common.Hash        `json:"recovery_hash"`
	Blocks               []publicationBlock `json:"blocks"`
	PrivateData          hexutil.Bytes      `json:"private_data"`
	L1Head               common.Hash        `json:"l1_head"`
	ExpectedPublicValues hexutil.Bytes      `json:"expected_public_values"`
	// ExpectedDigest is set for execution-mock-v1 only.
	ExpectedDigest *common.Hash `json:"expected_digest,omitempty"`
}

// projectionProducerConfig is the operator-side producer configuration.
type projectionProducerConfig struct {
	// Command is the proof command, run as `<Command> --publication-request`.
	Command string
	// Prover is the request's prover field (native for execution-mock-v1).
	Prover                           string
	PrivateRPC, ProjectionRPC, L1RPC string
	// PublicRollup is the projection rollup config, carrying the deployed private_projection.
	PublicRollup *rollup.Config
	// PrivateConfig, L1Config, ProjectionConfig and DependencySet are the request's config bytes.
	PrivateConfig, L1Config, ProjectionConfig, DependencySet []byte
	// Timeout and TimeoutPerBlock scale the producer timeout with the proven interval.
	Timeout, TimeoutPerBlock time.Duration
}

// effectiveProofTimeout is base + perBlock × (lastBlock − anchorBlock) (§G.3a): a span that must
// also prove a long recovery interval since its anchor gets proportionally longer.
func effectiveProofTimeout(base, perBlock time.Duration, lastBlock, anchorBlock uint64) time.Duration {
	if lastBlock <= anchorBlock || perBlock <= 0 {
		return base
	}
	blocks := lastBlock - anchorBlock
	if blocks > uint64(math.MaxInt64-base)/uint64(perBlock) {
		return math.MaxInt64
	}
	return base + time.Duration(blocks)*perBlock
}

// projectionPreflight computes the statement the producer's proof must commit to: the admission
// statement of the candidate span, from this batcher's derivation view (§C.5).
func projectionPreflight(cfg *rollup.Config, candidate *builder.BuiltRange, start RangeStart) (*projection.Statement, error) {
	if len(candidate.Blocks) == 0 {
		return nil, errors.New("empty candidate range")
	}
	return projection.ValidateProjectionRange(cfg.PrivateProjection, projection.Context{
		ChainID: cfg.L2ChainID, GenesisNumber: cfg.Genesis.L2.Number,
		GenesisTime: cfg.Genesis.L2Time, BlockTime: cfg.BlockTime,
		GenesisHash: cfg.Genesis.L2.Hash, ParentHash: start.PrevTerminalRenderingHash,
		L1Head:       candidate.Blocks[len(candidate.Blocks)-1].Origin.Hash,
		Continuation: start.Continuation,
	}, candidate.SpanBatch, projection.StubVerifier{})
}

// buildPublicationRequest assembles the v2 request for a candidate range and enforces the stdin
// cap before any child is spawned.
func buildPublicationRequest(cfg projectionProducerConfig, statement *projection.Statement, candidate *builder.BuiltRange,
	privateData []byte, start RangeStart,
) ([]byte, error) {
	publicValues := projection.PublicValues(statement)
	request := publicationRequest{
		Version:              PublicationRequestVersion,
		Prover:               cfg.Prover,
		PrivateRPC:           cfg.PrivateRPC,
		ProjectionRPC:        cfg.ProjectionRPC,
		L1RPC:                cfg.L1RPC,
		PrivateConfig:        nonNil(cfg.PrivateConfig),
		L1Config:             nonNil(cfg.L1Config),
		ProjectionConfig:     nonNil(cfg.ProjectionConfig),
		DependencySet:        nonNil(cfg.DependencySet),
		ParentHash:           start.PrevTerminalRenderingHash,
		Anchor:               start.Continuation.Anchor,
		AnchorOutput:         start.Continuation.OutputRoot,
		RecoveryHash:         start.Continuation.RecoveryHash,
		Blocks:               make([]publicationBlock, 0, len(candidate.Blocks)),
		PrivateData:          nonNil(privateData),
		L1Head:               candidate.Blocks[len(candidate.Blocks)-1].Origin.Hash,
		ExpectedPublicValues: publicValues[:],
	}
	if cfg.PublicRollup.PrivateProjection.Verifier == projection.ExecutionMock {
		digest := projection.AdmissionDigest(*statement)
		request.ExpectedDigest = &digest
	}
	for _, b := range candidate.Blocks {
		txs := b.Txs
		if txs == nil {
			txs = []hexutil.Bytes{}
		}
		request.Blocks = append(request.Blocks, publicationBlock{Timestamp: b.Timestamp, Epoch: b.Origin.Number, Transactions: txs})
	}
	input, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if len(input) > MaxPublicationRequestBytes {
		return nil, fmt.Errorf("proof request for range %d-%d is %d bytes, above the %d-byte proof command stdin cap",
			candidate.FirstBlock, candidate.LastBlock, len(input), MaxPublicationRequestBytes)
	}
	return input, nil
}

func nonNil(b []byte) hexutil.Bytes {
	if b == nil {
		return hexutil.Bytes{}
	}
	return b
}

// newProjectionProducer is operator-side I/O. Consensus verification remains pure and does not
// call this command, an RPC, or the private execution client. The returned envelope has been
// verified with the deployment's own verifier against the preflight statement.
func newProjectionProducer(lifecycle context.Context, cfg projectionProducerConfig,
) func(context.Context, *builder.BuiltRange, []byte, RangeStart) ([]byte, error) {
	return func(jobCtx context.Context, candidate *builder.BuiltRange, privateData []byte, start RangeStart) ([]byte, error) {
		statement, err := projectionPreflight(cfg.PublicRollup, candidate, start)
		if err != nil {
			return nil, err
		}
		verifier, err := projection.VerifierFor(cfg.PublicRollup.PrivateProjection, cfg.PublicRollup.L2ChainID)
		if err != nil {
			return nil, err
		}
		input, err := buildPublicationRequest(cfg, statement, candidate, privateData, start)
		if err != nil {
			return nil, err
		}
		timeout := effectiveProofTimeout(cfg.Timeout, cfg.TimeoutPerBlock, candidate.LastBlock, start.Continuation.Anchor.Number)
		ctx, cancel := context.WithTimeout(jobCtx, timeout)
		stop := context.AfterFunc(lifecycle, cancel)
		defer stop()
		defer cancel()
		// Exact executable path, never shell interpolation. Sensitive witness
		// input is passed on stdin, never argv, logs, or a persistent file.
		cmd := exec.CommandContext(ctx, cfg.Command, "--publication-request")
		// A killed child's own children may still hold its pipes; stop waiting for them.
		cmd.WaitDelay = proofCommandWaitDelay
		cmd.Stdin = bytes.NewReader(input)
		var stdout, stderr proofCommandOutput
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				err = fmt.Errorf("timed out after %s: %w", timeout, err)
			}
			return nil, fmt.Errorf("proof command failed: %w: %s", err, stderr.String())
		}
		proof, err := hexutil.Decode(strings.TrimSpace(stdout.String()))
		if err != nil {
			return nil, fmt.Errorf("proof command returned a malformed envelope: %w", err)
		}
		if err := verifier.Verify(*statement, proof); err != nil {
			return nil, fmt.Errorf("proof command envelope does not verify: %w", err)
		}
		return proof, nil
	}
}

// Bound diagnostics from a failing child without blocking its pipe.
type proofCommandOutput struct{ bytes.Buffer }

func (b *proofCommandOutput) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 16384 - b.Len(); remaining > 0 {
		_, _ = b.Buffer.Write(p[:min(n, remaining)])
	}
	return n, nil
}
