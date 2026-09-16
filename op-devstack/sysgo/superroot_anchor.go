package sysgo

import (
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// bootstrapSuperRootAnchor advances the fixture's placeholder anchor through a
// finalized SuperPermissioned game. It returns an assertion to run after the
// upgrade, checking that both the anchor game and its proposal were preserved.
func bootstrapSuperRootAnchor(t devtest.T, keys devkeys.Keys, l1RPC string, l2Net *L2Network, root *eth.SuperV1) func() {
	t.Helper()
	require := t.Require()
	rpcClient, err := rpc.DialContext(t.Ctx(), l1RPC)
	require.NoError(err)
	t.Cleanup(rpcClient.Close)
	client := ethclient.NewClient(rpcClient)
	ethClient, err := sources.NewEthClient(opclient.NewBaseRPCClient(rpcClient), t.Logger(), nil, sources.DefaultEthClientConfig(10))
	require.NoError(err)

	dgf := bindings.NewBindings[bindings.DisputeGameFactory](
		bindings.WithClient(ethClient), bindings.WithTo(l2Net.deployment.DisputeGameFactoryProxyAddr()), bindings.WithTest(t),
	)
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(ethClient), bindings.WithTo(l2Net.rollupCfg.DepositContractAddress), bindings.WithTest(t),
	)
	registry, err := contractio.Read(portal.AnchorStateRegistry(), t.Ctx())
	require.NoError(err)
	asr := bindings.NewBindings[bindings.AnchorStateRegistry](
		bindings.WithClient(ethClient), bindings.WithTo(registry), bindings.WithTest(t),
	)
	// Fixture construction cannot rely on time travel performed later by the test.
	finalityDelay, err := contractio.Read(portal.DisputeGameFinalityDelaySeconds(), t.Ctx())
	require.NoError(err)
	require.LessOrEqual(finalityDelay.Cmp(big.NewInt(30)), 0,
		"permissioned anchor bootstrap requires a finality delay of at most 30 seconds (got %s); configure WithDisputeGameFinalityDelaySeconds(2)", finalityDelay)

	// Only games respected at creation can later become the anchor.
	respected, err := contractio.Read(asr.RespectedGameType(), t.Ctx())
	require.NoError(err)
	require.EqualValues(superPermissionedGameType, respected, "bootstrap game must be respected when created")

	// SuperPermissioned only accepts proposals from the configured proposer.
	proposerKey, err := keys.Secret(devkeys.ChainOperatorKeys(l2Net.ChainID().ToBig())(devkeys.ProposerRole))
	require.NoError(err)
	txOpts := txplan.Combine(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(proposerKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithEstimator(client, true),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(txplan.FromGethReceipts(client), 5, retry.Exponential()),
	)
	// The game checks that the full proof in extraData hashes to the claimed root.
	rootClaim, extraData := common.Hash(eth.SuperRoot(root)), root.Marshal()
	receipt, err := contractio.Write(dgf.Create(superPermissionedGameType, rootClaim, extraData), t.Ctx(), txOpts, txplan.WithGasRatio(2))
	require.NoError(err, "create permissioned anchor game")
	require.EqualValues(types.ReceiptStatusSuccessful, receipt.Status)
	game, err := contractio.Read(dgf.Games(superPermissionedGameType, rootClaim, extraData), t.Ctx())
	require.NoError(err)
	require.NotEqual(common.Address{}, game.Proxy)

	// SuperPermissioned resolves at creation; only the registry's finality delay remains.
	require.Eventually(func() bool {
		finalized, err := contractio.Read(asr.IsGameFinalized(game.Proxy), t.Ctx())
		if err != nil {
			t.Logger().Debug("Retrying anchor game finality read", "err", err)
			return false
		}
		return finalized
	}, contextTimeout(t), 250*time.Millisecond, "waiting for permissioned anchor game finality")
	// Advance anchorGame so new games use this root instead of the placeholder.
	receipt, err = contractio.Write(asr.SetAnchorState(game.Proxy), t.Ctx(), txOpts)
	require.NoError(err, "set finalized permissioned anchor")
	require.EqualValues(types.ReceiptStatusSuccessful, receipt.Status)

	assertAnchor := func() {
		anchorGame, err := contractio.Read(asr.AnchorGame(), t.Ctx())
		require.NoError(err)
		require.Equal(game.Proxy, anchorGame, "permissioned anchor game must survive the upgrade")
		anchor, err := contractio.Read(asr.GetAnchorRoot(), t.Ctx())
		require.NoError(err)
		require.Equal(rootClaim, anchor.Root)
		require.Equal(root.Timestamp, bigs.Uint64Strict(anchor.L2SequenceNumber))
	}
	// Check the anchor now, then let the caller repeat the check after upgrading.
	assertAnchor()
	return assertAnchor
}
