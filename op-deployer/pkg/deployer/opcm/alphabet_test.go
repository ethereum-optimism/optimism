package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeployAlphabetVM(t *testing.T) {
	env := newTestGenesisEnv(t)
	output, err := DeployAlphabetVM(env, DeployAlphabetVMInput{
		AbsolutePrestate: common.BigToHash(big.NewInt(1)),
		PreimageOracle:   common.BigToAddress(big.NewInt(2)),
	})
	require.NoError(t, err)
	requireCode(t, env, output.AlphabetVM)
}
