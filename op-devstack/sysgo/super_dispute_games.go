package sysgo

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// dialL1EthClient builds the apis.EthClient-compatible wrapper that the
// txintent bindings expect for read calls.
func dialL1EthClient(t devtest.T, l1ELRPC string) (*ethclient.Client, *sources.EthClient, func()) {
	require := t.Require()

	rpcClient, err := rpc.DialContext(t.Ctx(), l1ELRPC)
	require.NoError(err, "failed to dial L1 EL")
	rawClient := ethclient.NewClient(rpcClient)

	opRPC, err := opclient.NewRPC(t.Ctx(), t.Logger(), l1ELRPC, opclient.WithLazyDial())
	require.NoError(err, "failed to build op-service RPC client")
	ethClient, err := sources.NewEthClient(opRPC, t.Logger(), nil, sources.DefaultEthClientConfig(10))
	require.NoError(err, "failed to build sources.EthClient")

	cleanup := func() {
		opRPC.Close()
		rpcClient.Close()
	}
	return rawClient, ethClient, cleanup
}

// registerSuperDisputeGameForRuntime wires a super game type onto the chain's
// DisputeGameFactory by calling setImplementation(gameType, impl, gameArgs)
// directly as the L1 ProxyAdmin owner. The super impl itself lives on the
// OPContractsManagerContainer and is shared across all chains, so no
// per-chain deploy is performed.
//
// Preconditions:
//   - The SuperRootGamesMigration dev flag was set at initial deploy, so the
//     container holds SuperFaultDisputeGame / SuperPermissionedDisputeGame impls.
//   - SUPER_PERMISSIONED_CANNON is already registered on the factory (the
//     initial-deploy super path installs it). Its gameArgs supplies the
//     DelayedWETH proxy that super games reuse.
func registerSuperDisputeGameForRuntime(
	t devtest.T,
	keys devkeys.Keys,
	l1ChainID eth.ChainID,
	l1ELRPC string,
	l2Net *L2Network,
	gameType gameTypes.GameType,
	absolutePrestate common.Hash,
) {
	require := t.Require()
	require.NotNil(l2Net, "l2 network must exist")
	require.NotNil(l2Net.deployment, "l2 deployment must exist")
	require.NotEqual(common.Address{}, l2Net.opcmContainer, "missing OPCMContainer address")

	rawClient, ethClient, cleanup := dialL1EthClient(t, l1ELRPC)
	defer cleanup()

	container := bindings.NewOPContractsManagerContainer(
		bindings.WithClient(ethClient),
		bindings.WithTo(l2Net.opcmContainer),
		bindings.WithTest(t),
	)
	impls, err := contractio.Read(container.Implementations(), t.Ctx())
	require.NoError(err, "failed to read implementations() from OPCMContainer %s", l2Net.opcmContainer)

	var gameImpl common.Address
	isPermissioned := false
	switch gameType {
	case gameTypes.SuperCannonGameType, gameTypes.SuperCannonKonaGameType:
		gameImpl = impls.SuperFaultDisputeGameImpl
	case gameTypes.SuperPermissionedGameType:
		gameImpl = impls.SuperPermissionedDisputeGameImpl
		isPermissioned = true
	default:
		require.Failf("unsupported super game type", "%s (%d)", gameType, uint32(gameType))
	}
	require.NotEqual(common.Address{}, gameImpl,
		"OPCMContainer has no impl for %s - was SuperRootGamesMigration flag set at deploy?", gameType)

	portalAddr := l2Net.rollupCfg.DepositContractAddress
	portal := bindings.NewBindings[bindings.OptimismPortal2](
		bindings.WithClient(ethClient),
		bindings.WithTo(portalAddr),
		bindings.WithTest(t),
	)
	asrAddr, err := contractio.Read(portal.AnchorStateRegistry(), t.Ctx())
	require.NoError(err, "failed to read AnchorStateRegistry from portal")

	factory := bindings.NewDisputeGameFactory(
		bindings.WithClient(ethClient),
		bindings.WithTo(l2Net.deployment.disputeGameFactoryProxy),
		bindings.WithTest(t),
	)
	// Super games reuse the WETH configured on SUPER_PERMISSIONED_CANNON at
	// initial deploy (super types share WETH configuration with the
	// permissioned slot).
	permArgsRaw, err := contractio.Read(factory.GameArgs(uint32(gameTypes.SuperPermissionedGameType)), t.Ctx())
	require.NoError(err, "failed to read SUPER_PERMISSIONED_CANNON gameArgs from factory")
	require.NotEmpty(permArgsRaw,
		"SUPER_PERMISSIONED_CANNON gameArgs missing from factory - registerSuperDisputeGameForRuntime must run after initial deploy")
	permArgs, err := gameargs.Parse(permArgsRaw)
	require.NoError(err, "failed to parse SUPER_PERMISSIONED_CANNON gameArgs")

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	proposer, err := keys.Address(chainOps(devkeys.ProposerRole))
	require.NoError(err, "failed to get proposer address")
	challenger, err := keys.Address(chainOps(devkeys.ChallengerRole))
	require.NoError(err, "failed to get challenger address")

	args := gameargs.GameArgs{
		AbsolutePrestate:    absolutePrestate,
		Vm:                  impls.MipsImpl,
		AnchorStateRegistry: asrAddr,
		Weth:                permArgs.Weth,
		// Super games encode chain ID in the super-root extraData, not in
		// game args; must be zero here.
		L2ChainID:  eth.ChainID{},
		Proposer:   proposer,
		Challenger: challenger,
	}
	var packedArgs []byte
	if isPermissioned {
		packedArgs = args.PackPermissioned()
	} else {
		packedArgs = args.PackPermissionless()
	}

	l1PAOKey, err := keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get L1 proxy admin owner key")

	txOpts := txplan.Combine(
		txplan.WithChainID(rawClient),
		txplan.WithPrivateKey(l1PAOKey),
		txplan.WithPendingNonce(rawClient),
		txplan.WithAgainstLatestBlockEthClient(rawClient),
		txplan.WithEstimator(rawClient, true),
		txplan.WithRetrySubmission(rawClient, 5, retry.Exponential()),
		txplan.WithRetryInclusion(rawClient, 5, retry.Exponential()),
	)
	rcpt, err := contractio.Write(
		factory.SetImplementationWithArgs(uint32(gameType), gameImpl, packedArgs),
		t.Ctx(),
		txOpts,
	)
	require.NoError(err, "setImplementation tx failed")
	require.Equal(types.ReceiptStatusSuccessful, rcpt.Status,
		"setImplementation reverted for %s", gameType)

	t.Logger().Info("registered super dispute game",
		"gameType", gameType, "impl", gameImpl, "dgf", l2Net.deployment.disputeGameFactoryProxy)
}
