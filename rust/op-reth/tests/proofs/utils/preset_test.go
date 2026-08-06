package utils

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func TestResolveELSpec(t *testing.T) {
	const envVar = "OP_DEVSTACK_TEST_EL"
	defaultKind := sysgo.MixedL2ELOpReth

	tests := []struct {
		name    string
		value   string
		want    sysgo.MixedL2ELKind
		wantErr bool
	}{
		{name: "default", want: defaultKind},
		{name: "op reth", value: "op-reth", want: sysgo.MixedL2ELOpReth},
		{name: "op reth proof v2", value: "op-reth-proof-v2", want: sysgo.MixedL2ELOpRethV2},
		{name: "op geth", value: "op-geth", want: sysgo.MixedL2ELOpGeth},
		{name: "removed v1", value: "op-reth-proof-v1", wantErr: true},
		{name: "unknown", value: "op-reht", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(envVar, test.value)
			got, err := resolveELSpec(envVar, defaultKind)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}
