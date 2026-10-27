package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestDeployAltDA(t *testing.T) {
	env := newTestGenesisEnv(t)
	output, err := DeployAltDA(env, DeployAltDAInput{
		Salt:                     common.BigToHash(big.NewInt(1)),
		ProxyAdmin:               common.BigToAddress(big.NewInt(2)),
		ChallengeContractOwner:   common.BigToAddress(big.NewInt(3)),
		ChallengeWindow:          big.NewInt(4),
		ResolveWindow:            big.NewInt(5),
		BondSize:                 big.NewInt(6),
		ResolverRefundPercentage: big.NewInt(7),
	})
	require.NoError(t, err)
	requireCode(t, env, output.DataAvailabilityChallengeProxy)
	requireCode(t, env, output.DataAvailabilityChallengeImpl)
}
