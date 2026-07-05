package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleProposalOutputs() []proposalOutputRecord {
	idx := uint64(14231)
	return []proposalOutputRecord{
		{
			Index: &idx, Game: "0x3ddfB3C5EcB18fE5f4DcFfe052CbEC3c3853344e", Status: "In Progress",
			L2BlockNumber: 29541759,
			ProposedRoot:  "0xede109637900d069eab8a7dfcced76725b303df0cb0feb4c1890903173c9da62",
			OutputRoot:    "0xede109637900d069eab8a7dfcced76725b303df0cb0feb4c1890903173c9da62",
			RootMatch:     true,
			L1Head:        "0x6946284891a0fb032ee9e6050522dbc4c7354a3b6f7ffeed5020548b56bebdf4",
			L1HeadNumber:  11127878, SafeHead: 29542323, SafeHeadAtOrAboveBlock: true,
		},
		{
			// Explicit-game form: no factory index, and a node that disagrees with the proposal.
			Game: "0x0000000000000000000000000000000000001234", Status: "In Progress",
			L2BlockNumber: 100,
			ProposedRoot:  "0x1111111111111111111111111111111111111111111111111111111111111111",
			OutputRoot:    "0x2222222222222222222222222222222222222222222222222222222222222222",
			RootMatch:     false,
			L1Head:        "0x3333333333333333333333333333333333333333333333333333333333333333",
			L1HeadNumber:  50, SafeHead: 90, SafeHeadAtOrAboveBlock: false,
		},
	}
}

func TestRenderProposalOutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderProposalOutputsJSON(&buf, sampleProposalOutputs()))

	var got struct {
		Games []proposalOutputRecord `json:"games"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Games, 2)

	require.NotNil(t, got.Games[0].Index)
	require.Equal(t, uint64(14231), *got.Games[0].Index)
	require.True(t, got.Games[0].RootMatch)
	require.True(t, got.Games[0].SafeHeadAtOrAboveBlock)

	// Explicit-game records omit the factory index entirely (json omitempty).
	require.Nil(t, got.Games[1].Index)
	require.False(t, got.Games[1].RootMatch)
	require.False(t, got.Games[1].SafeHeadAtOrAboveBlock)
}

func TestRenderProposalOutputsText(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderProposalOutputsText(&buf, sampleProposalOutputs()))
	out := buf.String()
	require.Contains(t, out, "Output Root (ours)")
	require.Contains(t, out, "Safe Head")
	require.Contains(t, out, "0x3ddfB3C5EcB18fE5f4DcFfe052CbEC3c3853344e")
	require.Contains(t, out, "true")
	require.Contains(t, out, "false")
}
