package sysgo

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts/gameargs"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// containerImplementations mirrors OPContractsManagerContainer.Implementations.
// Field order MUST match the Solidity declaration.
type containerImplementations struct {
	SuperchainConfigImpl             common.Address
	ProtocolVersionsImpl             common.Address
	L1ERC721BridgeImpl               common.Address
	OptimismPortalImpl               common.Address
	EthLockboxImpl                   common.Address
	SystemConfigImpl                 common.Address
	OptimismMintableERC20FactoryImpl common.Address
	L1CrossDomainMessengerImpl       common.Address
	L1StandardBridgeImpl             common.Address
	DisputeGameFactoryImpl           common.Address
	AnchorStateRegistryImpl          common.Address
	DelayedWETHImpl                  common.Address
	MipsImpl                         common.Address
	FaultDisputeGameImpl             common.Address
	PermissionedDisputeGameImpl      common.Address
	SuperFaultDisputeGameImpl        common.Address
	SuperPermissionedDisputeGameImpl common.Address
	ZkDisputeGameImpl                common.Address
	StorageSetterImpl                common.Address
}

func opContractsManagerContainerABI() (*abi.ABI, error) {
	root, err := findMonorepoRoot("packages/contracts-bedrock/forge-artifacts")
	if err != nil {
		return nil, fmt.Errorf("failed to find monorepo root: %w", err)
	}
	artifactPath := path.Join(root, "packages", "contracts-bedrock", "forge-artifacts",
		"OPContractsManagerContainer.sol", "OPContractsManagerContainer.json")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read container artifact: %w", err)
	}
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse container artifact: %w", err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse container ABI: %w", err)
	}
	return &parsed, nil
}

func readContainerImplementations(t devtest.CommonT, client *ethclient.Client, container common.Address) containerImplementations {
	require := t.Require()
	containerABI, err := opContractsManagerContainerABI()
	require.NoError(err, "failed to load OPContractsManagerContainer ABI")

	method, ok := containerABI.Methods["implementations"]
	require.True(ok, "ABI missing implementations() method")

	callMsg := ethereum.CallMsg{To: &container, Data: method.ID}
	result, err := client.CallContract(t.Ctx(), callMsg, nil)
	require.NoError(err, "failed to call implementations() on OPCMContainer %s", container)

	out, err := method.Outputs.Unpack(result)
	require.NoError(err, "failed to unpack implementations() output")
	require.Len(out, 1, "implementations() returned unexpected shape")

	// The ABI decoder produces an anonymous struct. Marshal+unmarshal via JSON
	// into our named struct so field order stays explicit in the Go source.
	b, err := json.Marshal(out[0])
	require.NoError(err, "failed to marshal implementations tuple")
	var decoded containerImplementations
	require.NoError(json.Unmarshal(b, &decoded), "failed to decode implementations into named struct")
	return decoded
}

func readAnchorStateRegistry(t devtest.CommonT, client *ethclient.Client, portal common.Address) common.Address {
	require := t.Require()
	// keccak("anchorStateRegistry()")[:4] = 0xf6bd2505. Build selector via ABI to stay robust.
	sigABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"anchorStateRegistry","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]}]`))
	require.NoError(err, "failed to build anchorStateRegistry ABI")
	method := sigABI.Methods["anchorStateRegistry"]
	result, err := client.CallContract(t.Ctx(), ethereum.CallMsg{To: &portal, Data: method.ID}, nil)
	require.NoError(err, "failed to call anchorStateRegistry() on portal %s", portal)
	out, err := method.Outputs.Unpack(result)
	require.NoError(err, "failed to unpack anchorStateRegistry() output")
	require.Len(out, 1, "anchorStateRegistry() returned unexpected shape")
	addr, ok := out[0].(common.Address)
	require.True(ok, "anchorStateRegistry() returned non-address")
	return addr
}

// readGameArgsForType reads the packed gameArgs registered on the factory for
// the given game type. Returns empty bytes if no impl is registered.
func readGameArgsForType(t devtest.CommonT, client *ethclient.Client, dgf common.Address, gameType gameTypes.GameType) []byte {
	require := t.Require()
	sigABI, err := abi.JSON(strings.NewReader(`[{"type":"function","name":"gameArgs","stateMutability":"view","inputs":[{"type":"uint32"}],"outputs":[{"type":"bytes"}]}]`))
	require.NoError(err, "failed to build gameArgs ABI")
	method := sigABI.Methods["gameArgs"]
	data, err := method.Inputs.Pack(uint32(gameType))
	require.NoError(err, "failed to pack gameArgs input")
	calldata := append(method.ID, data...)
	result, err := client.CallContract(t.Ctx(), ethereum.CallMsg{To: &dgf, Data: calldata}, nil)
	require.NoError(err, "failed to call gameArgs(%s) on factory %s", gameType, dgf)
	out, err := method.Outputs.Unpack(result)
	require.NoError(err, "failed to unpack gameArgs() output")
	require.Len(out, 1, "gameArgs() returned unexpected shape")
	raw, ok := out[0].([]byte)
	require.True(ok, "gameArgs() returned non-bytes")
	return raw
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
	l1EL L1ELNode,
	l2Net *L2Network,
	gameType gameTypes.GameType,
	absolutePrestate common.Hash,
) {
	require := t.Require()
	require.NotNil(l2Net, "l2 network must exist")
	require.NotNil(l2Net.deployment, "l2 deployment must exist")
	require.NotEqual(common.Address{}, l2Net.opcmContainer, "missing OPCMContainer address")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err, "failed to dial L1 EL")
	defer rpcClient.Close()
	client := ethclient.NewClient(rpcClient)

	impls := readContainerImplementations(t, client, l2Net.opcmContainer)

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

	asrAddr := readAnchorStateRegistry(t, client, l2Net.rollupCfg.DepositContractAddress)

	chainOps := devkeys.ChainOperatorKeys(l1ChainID.ToBig())
	proposer, err := keys.Address(chainOps(devkeys.ProposerRole))
	require.NoError(err, "failed to get proposer address")
	challenger, err := keys.Address(chainOps(devkeys.ChallengerRole))
	require.NoError(err, "failed to get challenger address")

	// Super games reuse the WETH configured on SUPER_PERMISSIONED_CANNON at
	// initial deploy (super types share WETH configuration with the
	// permissioned slot).
	permArgsRaw := readGameArgsForType(t, client, l2Net.deployment.disputeGameFactoryProxy, gameTypes.SuperPermissionedGameType)
	require.NotEmpty(permArgsRaw,
		"SUPER_PERMISSIONED_CANNON gameArgs missing from factory - registerSuperDisputeGameForRuntime must run after initial deploy")
	permArgs, err := gameargs.Parse(permArgsRaw)
	require.NoError(err, "failed to parse SUPER_PERMISSIONED_CANNON gameArgs")

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

	setImpl3ABI, err := disputeGameFactorySetImplementationABI()
	require.NoError(err, "failed to build setImplementation ABI")
	method := setImpl3ABI.Methods["setImplementation"]
	inputData, err := method.Inputs.Pack(uint32(gameType), gameImpl, packedArgs)
	require.NoError(err, "failed to pack setImplementation inputs")
	calldata := append(method.ID, inputData...)

	l1PAOKey, err := keys.Secret(chainOps(devkeys.L1ProxyAdminOwnerRole))
	require.NoError(err, "failed to get L1 proxy admin owner key")

	to := l2Net.deployment.disputeGameFactoryProxy
	tx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(l1PAOKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithEstimator(client, true),
		txplan.WithTo(&to),
		txplan.WithData(calldata),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(client, 5, retry.Exponential()),
	)
	receipt, err := tx.Included.Eval(t.Ctx())
	require.NoError(err, "setImplementation tx failed to include")
	require.Equal(gethTypes.ReceiptStatusSuccessful, receipt.Status,
		"setImplementation reverted for %s", gameType)

	t.Logger().Info("registered super dispute game",
		"gameType", gameType, "impl", gameImpl, "dgf", l2Net.deployment.disputeGameFactoryProxy)
}

func disputeGameFactorySetImplementationABI() (*abi.ABI, error) {
	// Minimal ABI for the 3-arg overload; txintent/bindings only exposes the
	// 2-arg form. We need the 3-arg form to register gameArgs atomically.
	const spec = `[
		{
			"type": "function",
			"name": "setImplementation",
			"stateMutability": "nonpayable",
			"inputs": [
				{"name": "_gameType", "type": "uint32"},
				{"name": "_impl", "type": "address"},
				{"name": "_args", "type": "bytes"}
			],
			"outputs": []
		}
	]`
	parsed, err := abi.JSON(strings.NewReader(spec))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
