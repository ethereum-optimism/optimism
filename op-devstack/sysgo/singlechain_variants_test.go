package sysgo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleChainTwoVerifiersCLKinds(t *testing.T) {
	clKinds := func(sequencer, derivingVerifier, followSourceVerifier MixedL2CLKind) singleChainTwoVerifiersCLKinds {
		return singleChainTwoVerifiersCLKinds{
			DerivingVerifier:     derivingVerifier,
			FollowSourceVerifier: followSourceVerifier,
			Sequencer:            sequencer,
		}
	}
	for _, testCase := range []struct {
		name          string
		requestedKind MixedL2CLKind
		want          singleChainTwoVerifiersCLKinds
	}{
		{name: "empty", requestedKind: "", want: clKinds(MixedL2CLOpNode, MixedL2CLOpNode, MixedL2CLOpNode)},
		{name: "op-node", requestedKind: MixedL2CLOpNode, want: clKinds(MixedL2CLOpNode, MixedL2CLOpNode, MixedL2CLOpNode)},
		{name: "kona-node", requestedKind: MixedL2CLKona, want: clKinds(MixedL2CLOpNode, MixedL2CLOpNode, MixedL2CLKona)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("DEVSTACK_L2CL_KIND", string(testCase.requestedKind))
			require.Equal(t, testCase.want, newSingleChainTwoVerifiersCLKinds(ResolveMixedL2CLKind()))
		})
	}
}
