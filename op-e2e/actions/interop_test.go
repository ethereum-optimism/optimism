package actions

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestL2Interop_CrossL2Inbox(gt *testing.T) {
	t := NewDefaultTesting(gt)
	dp := e2eutils.MakeDeployParams(t, defaultRollupTestParams)
	interopAtGenesis := hexutil.Uint64(0)
	dp.DeployConfig.L2GenesisInteropTimeOffset = &interopAtGenesis
	dp.DeployConfig.SuperchainPostie = &dp.Addresses.Alice

	sd := e2eutils.Setup(t, dp, defaultAlloc)
	log := testlog.Logger(t, log.LvlDebug)
	_, engine, sequencer := setupSequencerTest(t, sd, log)

	sequencer.ActL2PipelineFull(t)

	signer := types.LatestSigner(sd.L2Cfg.Config)
	cl := engine.EthClient()

	inboxAbi, err := bindings.CrossL2InboxMetaData.GetAbi()
	require.NoError(t, err)

	sendToInbox := func(mail []bindings.InboxEntry) common.Hash {
		n, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
		require.NoError(t, err)

		data, err := inboxAbi.Pack("deliverMail", mail)
		require.NoError(t, err)

		tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
			ChainID:   sd.L2Cfg.Config.ChainID,
			Nonce:     n,
			GasTipCap: big.NewInt(2 * params.GWei),
			GasFeeCap: new(big.Int).Add(engine.l2Chain.CurrentBlock().BaseFee, big.NewInt(2*params.GWei)),
			Gas:       1000_000,
			To:        &predeploys.CrossL2InboxAddr,
			Value:     big.NewInt(0),
			Data:      data,
		})
		require.NoError(gt, cl.SendTransaction(t.Ctx(), tx))
		return tx.Hash()
	}

	inboxTxHash := sendToInbox([]bindings.InboxEntry{
		{
			Chain:  [32]byte{10}, // TODO: how do we encode the chain?
			Output: [32]byte{42}, // TODO: an output root from another L2 chain
		},
	})
	sequencer.ActL2StartBlock(t)
	engine.ActL2IncludeTx(dp.Addresses.Alice)(t) // include the inbox tx from alice
	// Next up: include a tx from bob that proves a cross-chain message against the updated inbox
	sequencer.ActL2EndBlock(t)

	head := engine.l2Chain.CurrentBlock()
	require.Less(t, uint64(0), head.Number.Uint64())
	rec, err := cl.TransactionReceipt(t.Ctx(), inboxTxHash)
	require.NoError(t, err)
	require.Equal(t, rec.Status, types.ReceiptStatusSuccessful, "must update inbox")

	// This test can be extended with batch-submission, and replication by a verifier
}
