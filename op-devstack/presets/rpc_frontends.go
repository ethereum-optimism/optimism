package presets

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	conductorRpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum-optimism/optimism/op-sync-tester/synctester"
)

type keyringImpl struct {
	keys    devkeys.Keys
	require *testreq.Assertions
}

var _ stack.Keys = (*keyringImpl)(nil)

func newKeyring(keys devkeys.Keys, req *testreq.Assertions) *keyringImpl {
	return &keyringImpl{
		keys:    keys,
		require: req,
	}
}

func (k *keyringImpl) Secret(key devkeys.Key) *ecdsa.PrivateKey {
	pk, err := k.keys.Secret(key)
	k.require.NoError(err)
	return pk
}

func (k *keyringImpl) Address(key devkeys.Key) common.Address {
	addr, err := k.keys.Address(key)
	k.require.NoError(err)
	return addr
}

type rpcELNode struct {
	presetCommon

	client    opclient.RPC
	ethClient *sources.EthClient
	chainID   eth.ChainID
	txTimeout time.Duration
}

var _ stack.ELNode = (*rpcELNode)(nil)

func newRPCELNode(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC, timeout time.Duration) rpcELNode {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	ethCl, err := sources.NewEthClient(rpcCl, t.Logger(), nil, sources.DefaultEthClientConfig(10))
	t.Require().NoError(err)
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return rpcELNode{
		presetCommon: newPresetCommon(t, name),
		client:       rpcCl,
		ethClient:    ethCl,
		chainID:      chainID,
		txTimeout:    timeout,
	}
}

func (r *rpcELNode) ChainID() eth.ChainID {
	return r.chainID
}

func (r *rpcELNode) EthClient() apis.EthClient {
	return r.ethClient
}

func (r *rpcELNode) TransactionTimeout() time.Duration {
	return r.txTimeout
}

type l1ELFrontend struct {
	rpcELNode
}

var _ stack.L1ELNode = (*l1ELFrontend)(nil)

func newPresetL1ELNode(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC) *l1ELFrontend {
	return &l1ELFrontend{
		rpcELNode: newRPCELNode(t, name, chainID, rpcCl, 0),
	}
}

type l1CLFrontend struct {
	presetCommon
	chainID   eth.ChainID
	client    apis.BeaconClient
	lifecycle stack.Lifecycle
}

var _ stack.L1CLNode = (*l1CLFrontend)(nil)

func newPresetL1CLNode(t devtest.T, name string, chainID eth.ChainID, httpCl opclient.HTTP) *l1CLFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &l1CLFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		client:       sources.NewBeaconHTTPClient(httpCl),
	}
}

func (r *l1CLFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *l1CLFrontend) BeaconClient() apis.BeaconClient {
	return r.client
}

func (r *l1CLFrontend) Start() {
	r.require().NotNil(r.lifecycle, "L1CL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *l1CLFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "L1CL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type l2ELFrontend struct {
	rpcELNode
	l2Client       *sources.L2Client
	l2EngineClient *sources.EngineClient
	lifecycle      stack.Lifecycle
}

var _ stack.L2ELNode = (*l2ELFrontend)(nil)

func newPresetL2ELNode(t devtest.T, name string, chainID eth.ChainID, userRPCCl opclient.RPC, engineRPCCl opclient.RPC, rollupCfg *rollup.Config) *l2ELFrontend {
	t.Require().NotNil(rollupCfg, "rollup config must be configured")
	l2Client, err := sources.NewL2Client(userRPCCl, t.Logger(), nil, sources.L2ClientSimpleConfig(rollupCfg, false, 10, 10))
	t.Require().NoError(err)
	engineClientCfg := &sources.EngineClientConfig{
		L2ClientConfig: *sources.L2ClientSimpleConfig(rollupCfg, false, 10, 10),
	}
	engineClient, err := sources.NewEngineClient(engineRPCCl, t.Logger(), nil, engineClientCfg)
	t.Require().NoError(err)
	return &l2ELFrontend{
		rpcELNode:      newRPCELNode(t, name, chainID, userRPCCl, 0),
		l2Client:       l2Client,
		l2EngineClient: engineClient,
	}
}

func (r *l2ELFrontend) L2EthClient() apis.L2EthClient {
	return r.l2Client
}

func (r *l2ELFrontend) L2EngineClient() apis.EngineClient {
	return r.l2EngineClient.EngineAPIClient
}

func (r *l2ELFrontend) Start() {
	r.require().NotNil(r.lifecycle, "L2EL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *l2ELFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "L2EL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type l2CLFrontend struct {
	presetCommon
	chainID          eth.ChainID
	client           opclient.RPC
	rollupClient     apis.RollupClient
	p2pClient        apis.P2PClient
	els              locks.RWMap[string, *l2ELFrontend]
	rollupBoostNodes locks.RWMap[string, *rollupBoostFrontend]
	oprBuilderNodes  locks.RWMap[string, *oprBuilderFrontend]
	userRPC          string
	interopEndpoint  string
	interopJWTSecret eth.Bytes32
	lifecycle        stack.Lifecycle
}

var _ stack.L2CLNode = (*l2CLFrontend)(nil)

func newPresetL2CLNode(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC, userRPC, interopEndpoint string, interopJWTSecret eth.Bytes32) *l2CLFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &l2CLFrontend{
		presetCommon:     newPresetCommon(t, name),
		chainID:          chainID,
		client:           rpcCl,
		rollupClient:     sources.NewRollupClient(rpcCl),
		p2pClient:        sources.NewP2PClient(rpcCl),
		userRPC:          userRPC,
		interopEndpoint:  interopEndpoint,
		interopJWTSecret: interopJWTSecret,
	}
}

func (r *l2CLFrontend) ClientRPC() opclient.RPC {
	return r.client
}

func (r *l2CLFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *l2CLFrontend) RollupAPI() apis.RollupClient {
	return r.rollupClient
}

func (r *l2CLFrontend) P2PAPI() apis.P2PClient {
	return r.p2pClient
}

func (r *l2CLFrontend) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return r.interopEndpoint, r.interopJWTSecret
}

func (r *l2CLFrontend) UserRPC() string {
	return r.userRPC
}

func (r *l2CLFrontend) attachEL(el *l2ELFrontend) {
	r.els.Set(el.Name(), el)
}

func (r *l2CLFrontend) attachRollupBoostNode(node *rollupBoostFrontend) {
	r.rollupBoostNodes.Set(node.Name(), node)
}

func (r *l2CLFrontend) attachOPRBuilderNode(node *oprBuilderFrontend) {
	r.oprBuilderNodes.Set(node.Name(), node)
}

func (r *l2CLFrontend) ELs() []stack.L2ELNode {
	return mapSlice(sortByNameFunc(r.els.Values()), func(v *l2ELFrontend) stack.L2ELNode { return v })
}

func (r *l2CLFrontend) RollupBoostNodes() []stack.RollupBoostNode {
	return mapSlice(sortByNameFunc(r.rollupBoostNodes.Values()), func(v *rollupBoostFrontend) stack.RollupBoostNode { return v })
}

func (r *l2CLFrontend) OPRBuilderNodes() []stack.OPRBuilderNode {
	return mapSlice(sortByNameFunc(r.oprBuilderNodes.Values()), func(v *oprBuilderFrontend) stack.OPRBuilderNode { return v })
}

func (r *l2CLFrontend) ELClient() apis.EthClient {
	if els := sortByNameFunc(r.els.Values()); len(els) > 0 {
		return els[0].EthClient()
	}
	if nodes := sortByNameFunc(r.rollupBoostNodes.Values()); len(nodes) > 0 {
		return nodes[0].EthClient()
	}
	if nodes := sortByNameFunc(r.oprBuilderNodes.Values()); len(nodes) > 0 {
		return nodes[0].EthClient()
	}
	return nil
}

func (r *l2CLFrontend) Start() {
	r.require().NotNil(r.lifecycle, "L2CL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *l2CLFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "L2CL node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type l2BatcherFrontend struct {
	presetCommon
	chainID eth.ChainID
	client  *sources.BatcherAdminClient
}

var _ stack.L2Batcher = (*l2BatcherFrontend)(nil)

func newPresetL2Batcher(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC) *l2BatcherFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &l2BatcherFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		client:       sources.NewBatcherAdminClient(rpcCl),
	}
}

func (r *l2BatcherFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *l2BatcherFrontend) ActivityAPI() apis.BatcherActivity {
	return r.client
}

type l2ProposerFrontend struct {
	presetCommon
	chainID eth.ChainID
}

var _ stack.L2Proposer = (*l2ProposerFrontend)(nil)

func (r *l2ProposerFrontend) ChainID() eth.ChainID {
	return r.chainID
}

type l2ChallengerFrontend struct {
	presetCommon
	chainID eth.ChainID
	config  *challengerConfig.Config
}

var _ stack.L2Challenger = (*l2ChallengerFrontend)(nil)

func newPresetL2Challenger(t devtest.T, name string, chainID eth.ChainID, cfg *challengerConfig.Config) *l2ChallengerFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &l2ChallengerFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		config:       cfg,
	}
}

func (r *l2ChallengerFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *l2ChallengerFrontend) Config() *challengerConfig.Config {
	return r.config
}

type oprBuilderFrontend struct {
	rpcELNode
	engineClient      *sources.EngineClient
	flashblocksClient *opclient.WSClient
	lifecycle         stack.Lifecycle
	updateRuleSet     func(rulesYaml string) error
}

var _ stack.OPRBuilderNode = (*oprBuilderFrontend)(nil)

func newPresetOPRBuilderNode(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC, rollupCfg *rollup.Config, flashblocksCl *opclient.WSClient, updateRuleSet func(string) error) *oprBuilderFrontend {
	engineClient, err := sources.NewEngineClient(rpcCl, t.Logger(), nil, sources.EngineClientDefaultConfig(rollupCfg))
	t.Require().NoError(err)
	return &oprBuilderFrontend{
		rpcELNode:         newRPCELNode(t, name, chainID, rpcCl, 0),
		engineClient:      engineClient,
		flashblocksClient: flashblocksCl,
		updateRuleSet:     updateRuleSet,
	}
}

func (r *oprBuilderFrontend) L2EthClient() apis.L2EthClient {
	return r.engineClient.L2Client
}

func (r *oprBuilderFrontend) L2EngineClient() apis.EngineClient {
	return r.engineClient.EngineAPIClient
}

func (r *oprBuilderFrontend) FlashblocksClient() *opclient.WSClient {
	return r.flashblocksClient
}

func (r *oprBuilderFrontend) UpdateRuleSet(rulesYaml string) error {
	return r.updateRuleSet(rulesYaml)
}

func (r *oprBuilderFrontend) Start() {
	r.require().NotNil(r.lifecycle, "OPR builder node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *oprBuilderFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "OPR builder node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type rollupBoostFrontend struct {
	rpcELNode
	engineClient      *sources.EngineClient
	flashblocksClient *opclient.WSClient
	lifecycle         stack.Lifecycle
}

var _ stack.RollupBoostNode = (*rollupBoostFrontend)(nil)

func newPresetRollupBoostNode(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC, rollupCfg *rollup.Config, flashblocksCl *opclient.WSClient) *rollupBoostFrontend {
	engineClient, err := sources.NewEngineClient(rpcCl, t.Logger(), nil, sources.EngineClientDefaultConfig(rollupCfg))
	t.Require().NoError(err)
	return &rollupBoostFrontend{
		rpcELNode:         newRPCELNode(t, name, chainID, rpcCl, 0),
		engineClient:      engineClient,
		flashblocksClient: flashblocksCl,
	}
}

func (r *rollupBoostFrontend) L2EthClient() apis.L2EthClient {
	return r.engineClient.L2Client
}

func (r *rollupBoostFrontend) L2EngineClient() apis.EngineClient {
	return r.engineClient.EngineAPIClient
}

func (r *rollupBoostFrontend) FlashblocksClient() *opclient.WSClient {
	return r.flashblocksClient
}

func (r *rollupBoostFrontend) Start() {
	r.require().NotNil(r.lifecycle, "rollup boost node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *rollupBoostFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "rollup boost node %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type supervisorFrontend struct {
	presetCommon
	api       apis.SupervisorAPI
	lifecycle stack.Lifecycle
}

var _ stack.Supervisor = (*supervisorFrontend)(nil)

func newPresetSupervisor(t devtest.T, name string, rpcCl opclient.RPC) *supervisorFrontend {
	return &supervisorFrontend{
		presetCommon: newPresetCommon(t, name),
		api:          sources.NewSupervisorClient(rpcCl),
	}
}

func (r *supervisorFrontend) AdminAPI() apis.SupervisorAdminAPI {
	return r.api
}

func (r *supervisorFrontend) QueryAPI() apis.SupervisorQueryAPI {
	return r.api
}

func (r *supervisorFrontend) Start() {
	r.require().NotNil(r.lifecycle, "supervisor %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Start()
}

func (r *supervisorFrontend) Stop() {
	r.require().NotNil(r.lifecycle, "supervisor %s is not lifecycle-controllable", r.Name())
	r.lifecycle.Stop()
}

type supernodeFrontend struct {
	presetCommon
	api apis.SupernodeQueryAPI
}

var _ stack.Supernode = (*supernodeFrontend)(nil)

func newPresetSupernode(t devtest.T, name string, rpcCl opclient.RPC) *supernodeFrontend {
	return &supernodeFrontend{
		presetCommon: newPresetCommon(t, name),
		api:          sources.NewSuperNodeClient(rpcCl),
	}
}

func (r *supernodeFrontend) QueryAPI() apis.SupernodeQueryAPI {
	return r.api
}

type conductorFrontend struct {
	presetCommon
	chainID eth.ChainID
	api     conductorRpc.API
}

var _ stack.Conductor = (*conductorFrontend)(nil)

func newPresetConductor(t devtest.T, name string, chainID eth.ChainID, rpcCl *gethrpc.Client) *conductorFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &conductorFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		api:          conductorRpc.NewAPIClient(rpcCl),
	}
}

func (r *conductorFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *conductorFrontend) RpcAPI() conductorRpc.API {
	return r.api
}

type faucetFrontend struct {
	presetCommon
	chainID eth.ChainID
	client  *sources.FaucetClient
}

var _ stack.Faucet = (*faucetFrontend)(nil)

func newPresetFaucet(t devtest.T, name string, chainID eth.ChainID, rpcCl opclient.RPC) *faucetFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &faucetFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		client:       sources.NewFaucetClient(rpcCl),
	}
}

func (r *faucetFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *faucetFrontend) API() apis.Faucet {
	return r.client
}

type testSequencerFrontend struct {
	presetCommon
	api      apis.TestSequencerAPI
	controls map[eth.ChainID]apis.TestSequencerControlAPI
}

var _ stack.TestSequencer = (*testSequencerFrontend)(nil)

func newPresetTestSequencer(t devtest.T, name string, adminRPCCl opclient.RPC, controlRPCs map[eth.ChainID]opclient.RPC) *testSequencerFrontend {
	s := &testSequencerFrontend{
		presetCommon: newPresetCommon(t, name),
		api:          sources.NewBuilderClient(adminRPCCl),
		controls:     make(map[eth.ChainID]apis.TestSequencerControlAPI, len(controlRPCs)),
	}
	for chainID, rpcCl := range controlRPCs {
		s.controls[chainID] = sources.NewControlClient(rpcCl)
	}
	return s
}

func (r *testSequencerFrontend) AdminAPI() apis.TestSequencerAdminAPI {
	return r.api
}

func (r *testSequencerFrontend) BuildAPI() apis.TestSequencerBuildAPI {
	return r.api
}

func (r *testSequencerFrontend) ControlAPI(chainID eth.ChainID) apis.TestSequencerControlAPI {
	return r.controls[chainID]
}

type syncTesterFrontend struct {
	presetCommon
	chainID eth.ChainID
	addr    string
	client  *sources.SyncTesterClient
}

var _ stack.SyncTester = (*syncTesterFrontend)(nil)

func newPresetSyncTester(t devtest.T, name string, chainID eth.ChainID, addr string, rpcCl opclient.RPC) *syncTesterFrontend {
	t = t.WithCtx(stack.ContextWithChainID(t.Ctx(), chainID))
	return &syncTesterFrontend{
		presetCommon: newPresetCommon(t, name),
		chainID:      chainID,
		addr:         addr,
		client:       sources.NewSyncTesterClient(rpcCl),
	}
}

func (r *syncTesterFrontend) ChainID() eth.ChainID {
	return r.chainID
}

func (r *syncTesterFrontend) API() apis.SyncTester {
	return r.client
}

func (r *syncTesterFrontend) APIWithSession(sessionID string) apis.SyncTester {
	require := r.T().Require()
	require.NoError(synctester.IsValidSessionID(sessionID))
	rpcCl, err := opclient.NewRPC(r.T().Ctx(), r.Logger(), r.addr+"/"+sessionID, opclient.WithLazyDial())
	require.NoError(err, "sync tester failed to initialize rpc per session")
	return sources.NewSyncTesterClient(rpcCl)
}
