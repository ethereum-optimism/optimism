package opcm

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestCheckForgeScripts(t *testing.T) {
	artifact := func(abi string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(`{"abi":` + abi + `}`)}
	}
	withEntrypoint := artifact(`[{"type":"function","name":"runWithBytes","inputs":[{"name":"_input","type":"bytes"}],"outputs":[{"name":"","type":"bytes"}],"stateMutability":"nonpayable"}]`)
	withoutEntrypoint := artifact(`[{"type":"function","name":"run","inputs":[],"outputs":[],"stateMutability":"nonpayable"}]`)
	script := ForgeScript{"Deploy.s.sol", "Deploy"}

	require.NoError(t, CheckForgeScripts(fstest.MapFS{"Deploy.s.sol/Deploy.json": withEntrypoint}, []ForgeScript{script}))
	require.ErrorContains(t,
		CheckForgeScripts(fstest.MapFS{"Deploy.s.sol/Deploy.json": withoutEntrypoint}, []ForgeScript{script}),
		"the Deploy script of the contracts bundle has no runWithBytes entrypoint")
	require.ErrorContains(t,
		CheckForgeScripts(fstest.MapFS{}, []ForgeScript{script}),
		"the contracts bundle has no Deploy script")
}
