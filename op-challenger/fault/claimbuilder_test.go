package fault

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// ClaimBuilder is a test utility to enable creating claims in a wide range of situations
type ClaimBuilder struct {
	require  *require.Assertions
	maxDepth int
	correct  types.TraceProvider
}

func NewClaimBuilder(t *testing.T, maxDepth int) *ClaimBuilder {
	return &ClaimBuilder{
		require:  require.New(t),
		maxDepth: maxDepth,
		correct:  &alphabetWithProofProvider{NewAlphabetProvider("abcdefghijklmnopqrstuvwxyz", uint64(maxDepth))},
	}
}

// CorrectTraceProvider returns a types.TraceProvider that provides the canonical trace.
func (c *ClaimBuilder) CorrectTraceProvider() types.TraceProvider {
	return c.correct
}

// CorrectClaim returns the canonical claim at a specified trace index
func (c *ClaimBuilder) CorrectClaim(idx uint64) common.Hash {
	value, err := c.correct.Get(idx)
	c.require.NoError(err)
	return value
}

// CorrectPreState returns the pre-image of the canonical claim at the specified trace index
func (c *ClaimBuilder) CorrectPreState(idx uint64) []byte {
	preimage, _, err := c.correct.GetPreimage(idx)
	c.require.NoError(err)
	return preimage
}

// CorrectProofData returns the proof-data for the canonical claim at the specified trace index
func (c *ClaimBuilder) CorrectProofData(idx uint64) []byte {
	_, proof, err := c.correct.GetPreimage(idx)
	c.require.NoError(err)
	return proof
}

func (c *ClaimBuilder) incorrectClaim(idx uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(idx))
}

func (c *ClaimBuilder) claim(idx uint64, correct bool) common.Hash {
	if correct {
		return c.CorrectClaim(idx)
	} else {
		return c.incorrectClaim(idx)
	}
}

func (c *ClaimBuilder) CreateRootClaim(correct bool) types.Claim {
	value := c.claim((1<<c.maxDepth)-1, correct)
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    value,
			Position: types.NewPosition(0, 0),
		},
	}
}

func (c *ClaimBuilder) CreateLeafClaim(traceIndex uint64, correct bool) types.Claim {
	parentPos := types.NewPosition(c.maxDepth-1, 0)
	pos := types.NewPosition(c.maxDepth, int(traceIndex))
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    c.claim(pos.TraceIndex(c.maxDepth), correct),
			Position: pos,
		},
		Parent: types.ClaimData{
			Value:    c.claim(parentPos.TraceIndex(c.maxDepth), !correct),
			Position: parentPos,
		},
	}
}

func (c *ClaimBuilder) AttackClaim(claim types.Claim, correct bool) types.Claim {
	pos := claim.Position.Attack()
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    c.claim(pos.TraceIndex(c.maxDepth), correct),
			Position: pos,
		},
		Parent: claim.ClaimData,
	}
}

func (c *ClaimBuilder) DefendClaim(claim types.Claim, correct bool) types.Claim {
	pos := claim.Position.Defend()
	return types.Claim{
		ClaimData: types.ClaimData{
			Value:    c.claim(pos.TraceIndex(c.maxDepth), correct),
			Position: pos,
		},
		Parent: claim.ClaimData,
	}
}

type SequenceBuilder struct {
	builder   *ClaimBuilder
	lastClaim types.Claim
}

// Seq starts building a claim by following a sequence of attack and defend moves from the root
// The returned SequenceBuilder can be used to add additional moves. e.g:
// claim := Seq(true).Attack(false).Attack(true).Defend(true).Get()
func (c *ClaimBuilder) Seq(rootCorrect bool) *SequenceBuilder {
	claim := c.CreateRootClaim(rootCorrect)
	return &SequenceBuilder{
		builder:   c,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Attack(correct bool) *SequenceBuilder {
	claim := s.builder.AttackClaim(s.lastClaim, correct)
	return &SequenceBuilder{
		builder:   s.builder,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Defend(correct bool) *SequenceBuilder {
	claim := s.builder.DefendClaim(s.lastClaim, correct)
	return &SequenceBuilder{
		builder:   s.builder,
		lastClaim: claim,
	}
}

func (s *SequenceBuilder) Get() types.Claim {
	return s.lastClaim
}

type alphabetWithProofProvider struct {
	*AlphabetProvider
}

func (a *alphabetWithProofProvider) GetPreimage(i uint64) ([]byte, []byte, error) {
	preimage, _, err := a.AlphabetProvider.GetPreimage(i)
	if err != nil {
		return nil, nil, err
	}
	return preimage, []byte{byte(i)}, nil
}
