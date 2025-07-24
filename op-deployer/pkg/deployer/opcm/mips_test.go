package opcm

import (
	"testing"

	"github.com/HashKeyChain/verse/op-deployer/pkg/deployer/broadcaster"
	"github.com/HashKeyChain/verse/op-deployer/pkg/deployer/standard"
	"github.com/HashKeyChain/verse/op-deployer/pkg/deployer/testutil"
	"github.com/HashKeyChain/verse/op-deployer/pkg/env"
	"github.com/HashKeyChain/verse/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDeployMIPS(t *testing.T) {
	t.Parallel()

	_, artifacts := testutil.LocalArtifacts(t)

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		artifacts,
	)
	require.NoError(t, err)

	input := DeployMIPSInput{
		MipsVersion:    uint64(standard.MIPSVersion),
		PreimageOracle: common.Address{0xab},
	}

	output, err := DeployMIPS(host, input)
	require.NoError(t, err)

	require.NotEmpty(t, output.MipsSingleton)
}
