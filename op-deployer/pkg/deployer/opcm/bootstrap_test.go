package opcm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
)

// newTestGenesisEnv returns an env that runs the local scripts offline on an empty genesis state.
func newTestGenesisEnv(t *testing.T) *ForgeEnv {
	t.Helper()
	_, afs := testutil.LocalArtifacts(t)
	client, err := NewForgeClient(afs, "", append(L1ForgeScripts, L2ForgeScripts...))
	require.NoError(t, err)
	return &ForgeEnv{
		Client:  client,
		Context: context.Background(),
		Genesis: NewGenesisState(common.Address{'D'}, true),
	}
}

// requireCode checks that the genesis state of env has code at addr.
func requireCode(t *testing.T, env *ForgeEnv, addr common.Address) {
	t.Helper()
	require.NotEmpty(t, env.Genesis.Allocs.Accounts[addr].Code, "no code at %s", addr)
}
