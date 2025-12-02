package dsl

import (
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
)

func AssertAncestorDescendantRelationship(t helpers.Testing, chain *Chain, ancestor, descendant eth.BlockID) bool {
	assert.GreaterOrEqual(t, descendant.Number, ancestor.Number, "descendant block has a lower number than ancestor block")

	current := descendant
	result := true
	for current.Number > ancestor.Number && current.Number > 0 {
		header, err := chain.SequencerEngine.Eth.APIBackend.HeaderByNumber(t.Ctx(), rpc.BlockNumber(current.Number-1))
		result = result && assert.NoError(t, err)
		current = eth.BlockID{Hash: header.Hash(), Number: header.Number.Uint64()}
	}
	return result && assert.Equal(t, current, ancestor, "descendant block is not a descendant of the ancestor block")
}
