package sysgo

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	w3eth "github.com/lmittmann/w3/module/eth"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/contracts/bindings/delegatecallproxy"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// V2 structs for OPCM >= 7.0.0 (using IOPContractsManagerMigrator interface)
type DisputeGameConfigV2 struct {
	Enabled  bool
	InitBond *big.Int
	GameType uint32
	GameArgs []byte
}

// Proposal matches the Solidity Proposal struct: (bytes32 root, uint256 l2SequenceNumber)
type Proposal struct {
	Root             common.Hash
	L2SequenceNumber *big.Int
}

type MigrateInputV2 struct {
	ChainSystemConfigs        []common.Address
	DisputeGameConfigs        []DisputeGameConfigV2
	StartingAnchorRoot        Proposal
	StartingRespectedGameType uint32
}

func deployDelegateCallProxy(t devtest.CommonT, transactOpts *bind.TransactOpts, client *ethclient.Client, owner common.Address) (common.Address, *delegatecallproxy.Delegatecallproxy) {
	deployAddress, tx, proxyContract, err := delegatecallproxy.DeployDelegatecallproxy(transactOpts, client, owner)
	t.Require().NoError(err, "DelegateCallProxy deployment failed")
	// Make sure the transaction actually got included rather than just being sent
	_, err = wait.ForReceiptOK(t.Ctx(), client, tx.Hash())
	t.Require().NoError(err, "DelegateCallProxy deployment tx was not included successfully")
	return deployAddress, proxyContract
}

// delegateCallWithSetCode executes a delegatecall to target with the given data
// by delegating the sender's code to a DelegateCallProxy via EIP-7702 SetCode.
// The sender EOA temporarily gets DelegateCallProxy code, calls executeDelegateCall
// on itself, and the target runs via delegatecall in the sender's context.
// This avoids transferring ownership of contracts to a temporary proxy.
//
// Uses txplan for nonce management, gas pricing, signing, and retry — the same
// infrastructure that dsl.EOA.PlanAuth uses, without requiring a dsl.EOA.
func delegateCallWithSetCode(
	t devtest.CommonT,
	privateKey *ecdsa.PrivateKey,
	client *ethclient.Client,
	target common.Address,
	data []byte,
) {
	require := t.Require()
	ctx := t.Ctx()
	sender := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Deploy DelegateCallProxy as code reference for SetCode delegation
	chainID, err := client.ChainID(ctx)
	require.NoError(err)
	transactOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	require.NoError(err)
	transactOpts.Context = ctx
	proxyAddr, _ := deployDelegateCallProxy(t, transactOpts, client, sender)

	// Encode executeDelegateCall(target, data) calldata
	proxyABI, err := delegatecallproxy.DelegatecallproxyMetaData.GetAbi()
	require.NoError(err)
	calldata, err := proxyABI.Pack("executeDelegateCall", target, data)
	require.NoError(err)

	// Build a SetCode tx using txplan — same options PlanAuth uses,
	// plus the delegatecall payload as tx data.
	toAddr := sender
	tx := txplan.NewPlannedTx(
		txplan.WithChainID(client),
		txplan.WithPrivateKey(privateKey),
		txplan.WithPendingNonce(client),
		txplan.WithAgainstLatestBlockEthClient(client),
		txplan.WithType(types.SetCodeTxType),
		txplan.WithTo(&toAddr),
		txplan.WithAuthorizationTo(proxyAddr),
		txplan.WithData(calldata),
		// Fixed gas limit because eth_estimateGas doesn't handle EIP-7702 authorizations.
		// Use the EIP-7825 max transaction gas limit to give the migration maximum room.
		txplan.WithGasLimit(params.MaxTxGas),
		txplan.WithRetrySubmission(client, 5, retry.Exponential()),
		txplan.WithRetryInclusion(client, 10, retry.Exponential()),
	)

	receipt, err := tx.Included.Eval(ctx)
	require.NoError(err)
	require.Equal(types.ReceiptStatusSuccessful, receipt.Status, "delegatecall via SetCode failed")
}

func awaitSuperrootTime(t devtest.T, cls ...L2CLNode) uint64 {
	t.Require().NotEmpty(cls, "at least one L2 CL is required")

	var superrootTime uint64
	for _, l2CL := range cls {
		rollupClient, err := dial.DialRollupClientWithTimeout(t.Ctx(), t.Logger(), l2CL.UserRPC())
		t.Require().NoError(err)
		defer rollupClient.Close()

		ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
		err = wait.For(ctx, time.Second, func() (bool, error) {
			status, err := rollupClient.SyncStatus(ctx)
			if err != nil {
				return false, err
			}
			if status == nil || status.SafeL2.Number == 0 {
				return false, nil
			}
			superrootTime = status.SafeL2.Time
			return true, nil
		})
		cancel()
		t.Require().NoError(err, "waiting for chain safe head to advance failed")
	}
	return superrootTime
}

func getSupernodeSuperRoot(t devtest.T, supernode *SuperNode, timestamp uint64) eth.Bytes32 {
	client, err := dial.DialSuperNodeClientWithTimeout(t.Ctx(), t.Logger(), supernode.UserRPC())
	t.Require().NoError(err)

	ctx, cancel := context.WithTimeout(t.Ctx(), 2*time.Minute)
	err = wait.For(ctx, time.Second, func() (bool, error) {
		resp, err := client.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			t.Logf("DEBUG: Failed to get super root at timestamp %d: err: %v", timestamp, err)
			return false, err
		}
		return resp.Data != nil, nil
	})
	cancel()
	t.Require().NoError(err, "waiting for supernode superroot to be ready failed")

	resp, err := client.SuperRootAtTimestamp(t.Ctx(), timestamp)
	t.Require().NoError(err, "super root at timestamp failed")
	t.Require().NotNil(resp.Data, "super root data must be present")
	return resp.Data.SuperRoot
}

func migrateSuperRoots(
	t devtest.T,
	keys devkeys.Keys,
	migration *interopMigrationState,
	l1ChainID eth.ChainID,
	l1EL L1ELNode,
	superRoot eth.Bytes32,
	superrootTime uint64,
	primaryL2 eth.ChainID,
) common.Address {
	return migrateSuperRootsWithProposal(
		t,
		keys,
		migration,
		l1ChainID,
		l1EL,
		Proposal{
			Root:             common.Hash(superRoot),
			L2SequenceNumber: new(big.Int).SetUint64(superrootTime),
		},
		primaryL2,
	)
}

func migrateSuperRootsWithProposal(
	t devtest.T,
	keys devkeys.Keys,
	migration *interopMigrationState,
	l1ChainID eth.ChainID,
	l1EL L1ELNode,
	startingAnchorRoot Proposal,
	primaryL2 eth.ChainID,
) common.Address {
	require := t.Require()
	require.NotNil(migration, "interop migration state is required")
	require.NotEmpty(migration.opcmImpl, "must have an OPCM implementation")
	require.NotEmpty(migration.superchainConfigAddr, "must have a superchain deployment")
	require.NotEmpty(migration.l2Deployments, "must have L2 deployments for interop migration")

	rpcClient, err := rpc.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err)
	client := ethclient.NewClient(rpcClient)
	w3Client := w3.NewClient(rpcClient)

	absoluteCannonPrestate := getInteropCannonAbsolutePrestate(t)
	absoluteCannonKonaPrestate := getCannonKonaAbsolutePrestate(t)

	permissionedChainOps := devkeys.ChainOperatorKeys(primaryL2.ToBig())
	proposer, err := keys.Address(permissionedChainOps(devkeys.ProposerRole))
	require.NoError(err, "must have configured proposer")
	challenger, err := keys.Address(permissionedChainOps(devkeys.ChallengerRole))
	require.NoError(err, "must have configured challenger")

	var chainSystemConfigs []common.Address
	for _, l2Deployment := range migration.l2Deployments {
		chainSystemConfigs = append(chainSystemConfigs, l2Deployment.SystemConfigProxyAddr())
	}

	// Use the v2 migrator ABI directly (v1 OPCM is deleted, bindings are stale)
	migratorABI, err := OPContractsManagerMigratorABI()
	require.NoError(err, "invalid migrator ABI")
	contract := batching.NewBoundContract(migratorABI, migration.opcmImpl)

	// ABI-encode permissioned game args: (bytes32 absolutePrestate, address proposer, address challenger)
	bytes32Ty, _ := abi.NewType("bytes32", "", nil)
	addressTy, _ := abi.NewType("address", "", nil)
	permGameArgs, err := abi.Arguments{
		{Type: bytes32Ty},
		{Type: addressTy},
		{Type: addressTy},
	}.Pack(absoluteCannonPrestate, proposer, challenger)
	require.NoError(err, "failed to encode permissioned game args")

	migrateInputV2 := MigrateInputV2{
		ChainSystemConfigs: chainSystemConfigs,
		DisputeGameConfigs: []DisputeGameConfigV2{
			{
				Enabled:  true,
				InitBond: new(big.Int).Set(defaultInitBond),
				GameType: superPermissionedCannonGameType,
				GameArgs: permGameArgs,
			},
			{
				Enabled:  true,
				InitBond: new(big.Int).Set(defaultInitBond),
				GameType: superCannonKonaGameType,
				GameArgs: absoluteCannonKonaPrestate[:],
			},
		},
		StartingAnchorRoot:        startingAnchorRoot,
		StartingRespectedGameType: superCannonKonaGameType,
	}
	migrateCall := contract.Call("migrate", migrateInputV2)
	migrateCallData, err := migrateCall.Pack()
	require.NoError(err)

	_, l1PAOKey := resolveL1ProxyAdminOwner(t, keys, l1ChainID)

	t.Log("Executing OPCM migration via SetCode delegatecall")
	delegateCallWithSetCode(t, l1PAOKey, client, migration.opcmImpl, migrateCallData)

	var sharedDGF common.Address
	for _, l2Deployment := range migration.l2Deployments {
		portal := getOptimismPortal(t, w3Client, l2Deployment.SystemConfigProxyAddr())
		addr := getDisputeGameFactory(t, w3Client, portal)
		if sharedDGF == (common.Address{}) {
			sharedDGF = addr
		} else {
			require.Equal(sharedDGF, addr, "dispute game factory address is not the same for all deployments")
		}
	}
	require.NotEmpty(getSuperGameImpl(t, w3Client, sharedDGF))

	for chainID, l2Deployment := range migration.l2Deployments {
		l2Deployment.disputeGameFactoryProxy = sharedDGF
		migration.l2Deployments[chainID] = l2Deployment
	}
	t.Log("Interop migration complete")
	return sharedDGF
}

func getInteropCannonAbsolutePrestate(t devtest.CommonT) common.Hash {
	return getAbsolutePrestate(t, "op-program/bin/prestate-proof-interop.json")
}

func getCannonKonaAbsolutePrestate(t devtest.CommonT) common.Hash {
	return getAbsolutePrestate(t, "rust/kona/prestate-artifacts-cannon/prestate-proof.json")
}

func getAbsolutePrestate(t devtest.CommonT, prestatePath string) common.Hash {
	root, err := findMonorepoRoot(prestatePath)
	t.Require().NoError(err)
	p := path.Join(root, prestatePath)
	file, err := os.Open(p)
	t.Require().NoError(err)
	decoder := json.NewDecoder(file)
	var prestate map[string]interface{}
	err = decoder.Decode(&prestate)
	t.Require().NoError(err)
	t.Require().NotEmpty(prestate, "prestate is empty")
	return common.HexToHash(prestate["pre"].(string))
}

const (
	superPermissionedCannonGameType = 5
	superCannonKonaGameType         = 9
)

var (
	optimismPortalFn     = w3.MustNewFunc("optimismPortal()", "address")
	disputeGameFactoryFn = w3.MustNewFunc("disputeGameFactory()", "address")
	gameImplsFn          = w3.MustNewFunc("gameImpls(uint32)", "address")
)

func getOptimismPortal(t devtest.CommonT, client *w3.Client, systemConfigProxy common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(systemConfigProxy, optimismPortalFn).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

func getDisputeGameFactory(t devtest.CommonT, client *w3.Client, portal common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(portal, disputeGameFactoryFn).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

func getSuperGameImpl(t devtest.CommonT, client *w3.Client, dgf common.Address) common.Address {
	var addr common.Address
	err := client.Call(w3eth.CallFunc(dgf, gameImplsFn, uint32(superCannonKonaGameType)).Returns(&addr))
	t.Require().NoError(err)
	return addr
}

// OPContractsManagerMigratorABI loads the ABI for the OPContractsManagerMigrator contract
// from the forge artifact file.
func OPContractsManagerMigratorABI() (*abi.ABI, error) {
	root, err := findMonorepoRoot("packages/contracts-bedrock/forge-artifacts")
	if err != nil {
		return nil, fmt.Errorf("failed to find monorepo root: %w", err)
	}
	artifactPath := path.Join(root, "packages", "contracts-bedrock", "forge-artifacts",
		"OPContractsManagerMigrator.sol", "OPContractsManagerMigrator.json")
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrator artifact: %w", err)
	}
	var artifact struct {
		ABI json.RawMessage `json:"abi"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		return nil, fmt.Errorf("failed to parse migrator artifact: %w", err)
	}
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse migrator ABI: %w", err)
	}
	return &parsed, nil
}
