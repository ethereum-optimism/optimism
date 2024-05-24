package test

import (
	"context"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var DefaultClaimant = common.Address{0xba, 0xdb, 0xad, 0xba, 0xdb, 0xad}

type claimCfg struct {
	value          common.Hash
	invalidValue   bool
	claimant       common.Address
	parentIdx      int
	clockTimestamp time.Time
	clockDuration  time.Duration
}

func newClaimCfg(opts ...ClaimOpt) *claimCfg {
	cfg := &claimCfg{
		clockTimestamp: time.Unix(math.MaxInt64-1, 0),
	}
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return cfg
}

type ClaimOpt interface {
	Apply(cfg *claimCfg)
}

type claimOptFn func(cfg *claimCfg)

func (c claimOptFn) Apply(cfg *claimCfg) {
	c(cfg)
}

func WithValue(value common.Hash) ClaimOpt {
	return claimOptFn(func(cfg *claimCfg) {
		cfg.value = value
	})
}

func WithInvalidValue(invalid bool) ClaimOpt {
	return claimOptFn(func(cfg *claimCfg) {
		cfg.invalidValue = invalid
	})
}

func WithClaimant(claimant common.Address) ClaimOpt {
	return claimOptFn(func(cfg *claimCfg) {
		cfg.claimant = claimant
	})
}

func WithParent(claim types.Claim) ClaimOpt {
	return claimOptFn(func(cfg *claimCfg) {
		cfg.parentIdx = claim.ContractIndex
	})
}

func WithClock(timestamp time.Time, duration time.Duration) ClaimOpt {
	return claimOptFn(func(cfg *claimCfg) {
		cfg.clockTimestamp = timestamp
		cfg.clockDuration = duration
	})
}

// ClaimBuilder is a test utility to enable creating claims in a wide range of situations
type ClaimBuilder struct {
	require  *require.Assertions
	maxDepth types.Depth
	correct  types.TraceProvider
}

// NewClaimBuilder creates a new [ClaimBuilder].
func NewClaimBuilder(t *testing.T, maxDepth types.Depth, provider types.TraceProvider) *ClaimBuilder {
	return &ClaimBuilder{
		require:  require.New(t),
		maxDepth: maxDepth,
		correct:  provider,
	}
}

// CorrectTraceProvider returns a types.TraceProvider that provides the canonical trace.
func (c *ClaimBuilder) CorrectTraceProvider() types.TraceProvider {
	return c.correct
}

// CorrectClaimAtPosition returns the canonical claim at a specified position
func (c *ClaimBuilder) CorrectClaimAtPosition(pos types.Position) common.Hash {
	value, err := c.correct.Get(context.Background(), pos)
	c.require.NoError(err)
	return value
}

// CorrectPreState returns the pre-state (not hashed) required to execute the valid step at the specified trace index
func (c *ClaimBuilder) CorrectPreState(idx *big.Int) []byte {
	pos := types.NewPosition(c.maxDepth, idx)
	preimage, _, _, err := c.correct.GetStepData(context.Background(), pos)
	c.require.NoError(err)
	return preimage
}

// CorrectProofData returns the proof-data required to execute the valid step at the specified trace index
func (c *ClaimBuilder) CorrectProofData(idx *big.Int) []byte {
	pos := types.NewPosition(c.maxDepth, idx)
	_, proof, _, err := c.correct.GetStepData(context.Background(), pos)
	c.require.NoError(err)
	return proof
}

func (c *ClaimBuilder) CorrectOracleData(idx *big.Int) *types.PreimageOracleData {
	pos := types.NewPosition(c.maxDepth, idx)
	_, _, data, err := c.correct.GetStepData(context.Background(), pos)
	c.require.NoError(err)
	return data
}

func (c *ClaimBuilder) incorrectClaim(pos types.Position) common.Hash {
	return common.BigToHash(pos.TraceIndex(c.maxDepth))
}

func (c *ClaimBuilder) claim(pos types.Position, opts ...ClaimOpt) types.Claim {
	cfg := newClaimCfg(opts...)
	claim := types.Claim{
		ClaimData: types.ClaimData{
			Position: pos,
		},
		Claimant: DefaultClaimant,
		Clock: types.Clock{
			Duration:  cfg.clockDuration,
			Timestamp: cfg.clockTimestamp,
		},
	}
	if cfg.claimant != (common.Address{}) {
		claim.Claimant = cfg.claimant
	}
	if cfg.value != (common.Hash{}) {
		claim.Value = cfg.value
	} else if cfg.invalidValue {
		claim.Value = c.incorrectClaim(pos)
	} else {
		claim.Value = c.CorrectClaimAtPosition(pos)
	}
	claim.ParentContractIndex = cfg.parentIdx
	return claim
}

func (c *ClaimBuilder) CreateRootClaim(opts ...ClaimOpt) types.Claim {
	pos := types.NewPositionFromGIndex(big.NewInt(1))
	return c.claim(pos, opts...)
}

func (c *ClaimBuilder) CreateLeafClaim(traceIndex *big.Int, opts ...ClaimOpt) types.Claim {
	pos := types.NewPosition(c.maxDepth, traceIndex)
	return c.claim(pos, opts...)
}

func (c *ClaimBuilder) AttackClaim(claim types.Claim, opts ...ClaimOpt) types.Claim {
	pos := claim.Position.Attack()
	return c.claim(pos, append([]ClaimOpt{WithParent(claim)}, opts...)...)
}

func (c *ClaimBuilder) DefendClaim(claim types.Claim, opts ...ClaimOpt) types.Claim {
	pos := claim.Position.Defend()
	return c.claim(pos, append([]ClaimOpt{WithParent(claim)}, opts...)...)
}
