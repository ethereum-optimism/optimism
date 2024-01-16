package actions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestL2Interop(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	interopAtGenesis := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisInteropTimeOffset = &interopAtGenesis

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, engine, sequencer := setupSequencerTest(t, sd, log)
	cl := engine.EthClient()
	sequencer.ActL2PipelineFull(t)

	sequencer.ActL2StartBlock(t)
	sequencer.ActL2EndBlock(t)

	head := engine.l2Chain.CurrentBlock()
	require.Less(t, uint64(0), head.Number.Uint64())

	inbox, err := bindings.NewCrossL2Inbox(predeploys.CrossL2InboxAddr, cl)
	require.NoError(t, err)
	inboxVersion, err := inbox.Version(nil)
	require.NoError(t, err)
	require.Equal(t, "0.0.1", inboxVersion, "CrossL2Inbox contract is available")

	cdm, err := bindings.NewInteropL2CrossDomainMessenger(predeploys.InteropL2CrossDomainMessengerAddr, cl)
	require.NoError(t, err)
	cdmVersion, err := cdm.Version(nil)
	require.NoError(t, err)
	require.Equal(t, "0.0.1", cdmVersion, "Interop CDM contract is available")

	sb, err := bindings.NewInteropL2StandardBridge(predeploys.InteropL2StandardBridgeAddr, cl)
	require.NoError(t, err)
	sbVersion, err := sb.Version(nil)
	require.NoError(t, err)
	require.Equal(t, "0.0.1", sbVersion, "Interop SB contract is available")

	messenger, err := sb.MESSENGER(nil)
	require.NoError(t, err)
	require.Equal(t, predeploys.InteropL2CrossDomainMessengerAddr, messenger, "Interop SB Messenger contract misconfigured")

	// This test can be extended with batch-submission, and replication by a verifier
}
