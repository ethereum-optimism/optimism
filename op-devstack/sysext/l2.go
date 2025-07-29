package sysext

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"math/big"

	"github.com/ethereum-optimism/optimism/devnet-sdk/descriptors"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/compat"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
)

func getL2ID(net *descriptors.L2Chain) stack.L2NetworkID {
	return stack.L2NetworkID(eth.ChainIDFromBig(net.Config.ChainID))
}

func (o *Orchestrator) hydrateL2(net *descriptors.L2Chain, system stack.ExtensibleSystem) {
	t := system.T()
	commonConfig := shim.NewCommonConfig(t)

	env := o.env
	l2ID := getL2ID(net)

	l1 := system.L1Network(stack.L1NetworkID(eth.ChainIDFromBig(env.Env.L1.Config.ChainID)))

	cfg := shim.L2NetworkConfig{
		NetworkConfig: shim.NetworkConfig{
			CommonConfig: commonConfig,
			ChainConfig:  net.Config,
		},
		ID:           l2ID,
		RollupConfig: net.RollupConfig,
		Deployment:   newL2AddressBook(t, net.L1Addresses),
		Keys:         o.defineSystemKeys(t, net),
		Superchain:   system.Superchain(stack.SuperchainID(env.Env.Name)),
		L1:           l1,
	}
	if o.isInterop() {
		cfg.Cluster = system.Cluster(stack.ClusterID(env.Env.Name))
	}

	opts := []client.RPCOption{}

	if o.compatType == compat.Persistent {
		// Increase the timeout by default for persistent devnets, but not for kurtosis
		opts = append(opts, client.WithCallTimeout(time.Minute*5), client.WithBatchCallTimeout(time.Minute*10))
	}

	l2 := shim.NewL2Network(cfg)

	for _, node := range net.Nodes {
		o.hydrateL2ELCL(&node, l2, opts)
		o.hydrateConductors(&node, l2)
		o.hydrateFlashblocksBuilderIfPresent(&node, l2, opts)
	}
	o.hydrateBatcherMaybe(net, l2)
	o.hydrateProposerMaybe(net, l2)
	o.hydrateChallengerMaybe(net, l2)
	o.hydrateL2ProxydMaybe(net, l2)
	o.hydrateFlashblocksWebsocketProxyMaybe(net, l2)

	if faucet, ok := net.Services["faucet"]; ok {
		for _, instance := range faucet {
			l2.AddFaucet(shim.NewFaucet(shim.FaucetConfig{
				CommonConfig: commonConfig,
				Client:       o.rpcClient(t, instance, RPCProtocol, fmt.Sprintf("/chain/%s", l2.ChainID().String()), opts...),
				ID:           stack.NewFaucetID(instance.Name, l2.ChainID()),
			}))
		}
	}

	system.AddL2Network(l2)
}

func (o *Orchestrator) hydrateL2ELCL(node *descriptors.Node, l2Net stack.ExtensibleL2Network, opts []client.RPCOption) {
	require := l2Net.T().Require()
	l2ID := l2Net.ID()

	txTimeout := 30 * time.Second
	if o.compatType == compat.Persistent {
		txTimeout = 5 * time.Minute
	}

	elService, ok := node.Services[ELServiceName]
	require.True(ok, "need L2 EL service for chain", l2ID)
	elClient := o.rpcClient(l2Net.T(), elService, RPCProtocol, "/", opts...)
	l2EL := shim.NewL2ELNode(shim.L2ELNodeConfig{
		RollupCfg: l2Net.RollupConfig(),
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig:       shim.NewCommonConfig(l2Net.T()),
			Client:             elClient,
			ChainID:            l2ID.ChainID(),
			TransactionTimeout: txTimeout,
		},
		ID: stack.NewL2ELNodeID(elService.Name, l2ID.ChainID()),
	})
	if strings.Contains(node.Name, "geth") {
		l2EL.SetLabel(match.LabelVendor, string(match.OpGeth))
	}
	if strings.Contains(node.Name, "reth") {
		l2EL.SetLabel(match.LabelVendor, string(match.OpReth))
	}
	l2Net.AddL2ELNode(l2EL)

	clService, ok := node.Services[CLServiceName]
	require.True(ok, "need L2 CL service for chain", l2ID)

	clClient := o.rpcClient(l2Net.T(), clService, RPCProtocol, "/", opts...)
	l2CL := shim.NewL2CLNode(shim.L2CLNodeConfig{
		ID:           stack.NewL2CLNodeID(clService.Name, l2ID.ChainID()),
		CommonConfig: shim.NewCommonConfig(l2Net.T()),
		Client:       clClient,
	})
	l2Net.AddL2CLNode(l2CL)
	l2CL.(stack.LinkableL2CLNode).LinkEL(l2EL)
}

func (o *Orchestrator) hydrateConductors(node *descriptors.Node, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := l2Net.ID()

	conductorService, ok := node.Services[ConductorServiceName]
	if !ok {
		l2Net.Logger().Debug("L2 net node is missing a conductor service", "node", node.Name, "l2", l2ID)
		return
	}

	endpoint, header, err := o.findProtocolService(conductorService, RPCProtocol)
	require.NoError(err, "failed to find RPC service for conductor")

	opts := make([]rpc.ClientOption, 0)

	if o.env.Env.ReverseProxyURL != "" && len(header) > 0 && !o.useDirectCnx {
		opts = append(opts,
			rpc.WithHeaders(header),
			rpc.WithHTTPClient(&http.Client{
				Transport: hostAwareRoundTripper(header),
			}))
	}
	conductorClient, err := rpc.DialOptions(l2Net.T().Ctx(), endpoint, opts...)
	require.NoError(err, "failed to dial conductor endpoint")
	l2Net.T().Cleanup(func() { conductorClient.Close() })

	conductor := shim.NewConductor(shim.ConductorConfig{
		CommonConfig: shim.NewCommonConfig(l2Net.T()),
		Client:       conductorClient,
		ID:           stack.ConductorID(conductorService.Name),
	})

	l2Net.AddConductor(conductor)
}

func (o *Orchestrator) hydrateFlashblocksBuilderIfPresent(node *descriptors.Node, l2Net stack.ExtensibleL2Network, opts []client.RPCOption) {
	require := l2Net.T().Require()
	l2ID := l2Net.ID()

	rbuilderService, ok := node.Services[RBuilderServiceName]
	if !ok {
		l2Net.Logger().Debug("L2 net node is missing the flashblocksBuilder service", "node", node.Name, "l2", l2ID)
		return
	}

	associatedConductorService, ok := node.Services[ConductorServiceName]
	require.True(ok, "L2 rbuilder service must have an associated conductor service", l2ID)

	flashblocksWsUrl, _, err := o.findProtocolService(rbuilderService, WebsocketFlashblocksProtocol)
	require.NoError(err, "failed to find websocket service for rbuilder")

	flashblocksBuilder := shim.NewFlashblocksBuilderNode(shim.FlashblocksBuilderNodeConfig{
		ID: stack.NewFlashblocksBuilderID(rbuilderService.Name, l2ID.ChainID()),
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			Client:       o.rpcClient(l2Net.T(), rbuilderService, RPCProtocol, "/", opts...),
			ChainID:      l2ID.ChainID(),
		},
		Conductor:        l2Net.Conductor(stack.ConductorID(associatedConductorService.Name)),
		FlashblocksWsUrl: flashblocksWsUrl,
	})

	l2Net.AddFlashblocksBuilder(flashblocksBuilder)
}

func (o *Orchestrator) hydrateL2ProxydMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	proxydService, ok := net.Services["proxyd"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a proxyd service")
		return
	}

	for _, instance := range proxydService {
		l2Proxyd := shim.NewL2ELNode(shim.L2ELNodeConfig{
			ELNodeConfig: shim.ELNodeConfig{
				CommonConfig: shim.NewCommonConfig(l2Net.T()),
				Client:       o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
				ChainID:      l2ID.ChainID(),
			},
			RollupCfg: l2Net.RollupConfig(),
			ID:        stack.NewL2ELNodeID(instance.Name, l2ID.ChainID()),
		})
		l2Proxyd.SetLabel(match.LabelVendor, string(match.Proxyd))
		l2Net.AddL2ELNode(l2Proxyd)
	}
}

func (o *Orchestrator) hydrateFlashblocksWebsocketProxyMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	fbWsProxyService, ok := net.Services["flashblocks-websocket-proxy"]
	if !ok {
		return
	}

	for _, instance := range fbWsProxyService {
		wsUrl, _, err := o.findProtocolService(instance, WebsocketFlashblocksProtocol)
		require.NoError(err, "failed to get the websocket url for the flashblocks websocket proxy", "service", instance.Name)

		fbWsProxyShim := shim.NewFlashblocksWebsocketProxy(shim.FlashblocksWebsocketProxyConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID:           stack.NewFlashblocksWebsocketProxyID(instance.Name, l2ID.ChainID()),
			WsUrl:        wsUrl,
		})
		fbWsProxyShim.SetLabel(match.LabelVendor, string(match.FlashblocksWebsocketProxy))
		l2Net.AddFlashblocksWebsocketProxy(fbWsProxyShim)
	}
}

func (o *Orchestrator) hydrateBatcherMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	batcherService, ok := net.Services["batcher"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a batcher service")
		return
	}

	for _, instance := range batcherService {
		l2Net.AddL2Batcher(shim.NewL2Batcher(shim.L2BatcherConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID:           stack.NewL2BatcherID(instance.Name, l2ID.ChainID()),
			Client:       o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
		}))
	}
}

func (o *Orchestrator) hydrateProposerMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	proposerService, ok := net.Services["proposer"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a proposer service")
		return
	}

	for _, instance := range proposerService {
		l2Net.AddL2Proposer(shim.NewL2Proposer(shim.L2ProposerConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID:           stack.NewL2ProposerID(instance.Name, l2ID.ChainID()),
			Client:       o.rpcClient(l2Net.T(), instance, HTTPProtocol, "/"),
		}))
	}
}

func (o *Orchestrator) hydrateChallengerMaybe(net *descriptors.L2Chain, l2Net stack.ExtensibleL2Network) {
	require := l2Net.T().Require()
	l2ID := getL2ID(net)
	require.Equal(l2ID, l2Net.ID(), "must match L2 chain descriptor and target L2 net")

	challengerService, ok := net.Services["challenger"]
	if !ok {
		l2Net.Logger().Warn("L2 net is missing a challenger service")
		return
	}

	for _, instance := range challengerService {
		l2Net.AddL2Challenger(shim.NewL2Challenger(shim.L2ChallengerConfig{
			CommonConfig: shim.NewCommonConfig(l2Net.T()),
			ID:           stack.NewL2ChallengerID(instance.Name, l2ID.ChainID()),
		}))
	}
}

func (o *Orchestrator) defineSystemKeys(t devtest.T, net *descriptors.L2Chain) stack.Keys {
	devnetKeys := o.getActualSystemKeys(t, net)
	t.Require().NotNil(devnetKeys, "sysext backend requires actual system keys from devnet descriptor, but none were found. "+
		"Ensure the devnet environment contains the required wallet configurations.")

	return devnetKeys
}

func (o *Orchestrator) getActualSystemKeys(t devtest.T, net *descriptors.L2Chain) stack.Keys {
	env := o.env
	if env == nil || env.Env == nil {
		return nil
	}

	if net == nil {
		t.Logf("No L2 chain provided")
		return nil
	}

	l1Wallets := net.L1Wallets
	if l1Wallets == nil {
		t.Logf("No L1 wallets found in L2 chain config")
		return nil
	}

	chainID := net.Config.ChainID

	keyMap := make(map[string]*ecdsa.PrivateKey)
	loadedL1Keys := 0
	for walletRole, keySpec := range o.getWalletMappings(l1Wallets) {
		if wallet, exists := l1Wallets[walletRole]; exists {
			t.Require().NotEmpty(wallet.PrivateKey, "Private key for wallet role '%s' is empty", walletRole)

			privateKey := o.parsePrivateKey(wallet.PrivateKey)
			t.Require().NotNil(privateKey, "Failed to parse private key for wallet role '%s'", walletRole)

			var keyPath string
			switch keyType := keySpec.(type) {
			case devkeys.Role:
				keyPath = keyType.Key(chainID).String()
			case devkeys.UserKey:
				keyPath = keyType.String()
			case *FaucetKey:
				keyPath = keyType.String()
			default:
				t.Errorf("Unknown key type for wallet role '%s'", walletRole)
				continue
			}

			keyMap[keyPath] = privateKey
			loadedL1Keys++
		}
	}

	// Also check L2 wallets for chain-specific user keys
	loadedL2Keys := 0
	if net.Wallets != nil {
		l2ChainID := net.Config.ChainID

		for walletRole, wallet := range net.Wallets {
			if strings.HasPrefix(walletRole, "dev-account-") {
				indexStr := strings.TrimPrefix(walletRole, "dev-account-")
				index := 0
				if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
					continue
				}

				t.Require().NotEmpty(wallet.PrivateKey, "Private key for L2 wallet '%s' is empty", walletRole)

				privateKey := o.parsePrivateKey(wallet.PrivateKey)
				t.Require().NotNil(privateKey, "Failed to parse private key for L2 wallet '%s'", walletRole)

				chainUserKey := devkeys.ChainUserKey{ChainID: l2ChainID, Index: uint64(index)}
				keyPath := chainUserKey.String()
				keyMap[keyPath] = privateKey
				loadedL2Keys++
			}
		}
	} else {
		t.Logger().Warn("No L2 wallets found in devnet config")
	}

	if loadedL1Keys > 0 || loadedL2Keys > 0 {
		t.Logf("Loaded devnet keys: %d L1 system keys, %d L2 user keys", loadedL1Keys, loadedL2Keys)
	}

	return &devnetKeyring{
		devnetKeys: keyMap,
		chainID:    chainID,
	}
}

func (o *Orchestrator) getWalletMappings(l1Wallets descriptors.WalletMap) map[string]interface{} {
	mappings := make(map[string]interface{})

	// System role mappings
	systemRoles := map[string]devkeys.Role{
		"systemConfigOwner":          devkeys.SystemConfigOwner,
		"l1ProxyAdmin":               devkeys.L1ProxyAdminOwnerRole,
		"l2ProxyAdmin":               devkeys.L2ProxyAdminOwnerRole,
		"batcher":                    devkeys.BatcherRole,
		"proposer":                   devkeys.ProposerRole,
		"challenger":                 devkeys.ChallengerRole,
		"sequencer":                  devkeys.SequencerP2PRole,
		"sequencerFeeVaultRecipient": devkeys.SequencerFeeVaultRecipientRole,
		"baseFeeVaultRecipient":      devkeys.BaseFeeVaultRecipientRole,
		"l1FeeVaultRecipient":        devkeys.L1FeeVaultRecipientRole,
	}

	for walletRole, devkeyRole := range systemRoles {
		mappings[walletRole] = devkeyRole
	}

	o.addFaucetMappings(mappings, l1Wallets)

	// Dynamically discover user-key-* mappings from available L1 wallets
	for walletRole := range l1Wallets {
		if strings.HasPrefix(walletRole, "user-key-") {
			if _, alreadyMapped := mappings[walletRole]; !alreadyMapped {
				indexStr := strings.TrimPrefix(walletRole, "user-key-")
				index := 0
				if _, err := fmt.Sscanf(indexStr, "%d", &index); err == nil {
					mappings[walletRole] = devkeys.UserKey(index)
				}
			}
		}
	}

	return mappings
}

// addFaucetMappings implements the faucet wallet fallback logic described in op-acceptor's faucet.go
func (o *Orchestrator) addFaucetMappings(mappings map[string]interface{}, l1Wallets descriptors.WalletMap) {
	// L1 faucet logic following op-acceptor convention:
	// - prefer l1Faucet if available (store under its own name)
	// - fallback to user-key-20 only if l1Faucet doesn't exist (use devkeys mapping)
	if _, hasL1Faucet := l1Wallets["l1Faucet"]; hasL1Faucet {
		mappings["l1Faucet"] = &FaucetKey{name: "l1Faucet"}
	} else if _, hasUserKey20 := l1Wallets["user-key-20"]; hasUserKey20 {
		// Only use user-key-20 as fallback when l1Faucet doesn't exist
		mappings["user-key-20"] = devkeys.UserKey(20)
	}

	// L2 faucet logic: use l2Faucet if present (store under its own name)
	if _, hasL2Faucet := l1Wallets["l2Faucet"]; hasL2Faucet {
		mappings["l2Faucet"] = &FaucetKey{name: "l2Faucet"}
	}
}

type FaucetKey struct {
	name string
}

func (f *FaucetKey) String() string {
	return f.name
}

func (o *Orchestrator) parsePrivateKey(keyStr string) *ecdsa.PrivateKey {
	keyStr = strings.TrimPrefix(keyStr, "0x")

	keyBytes, err := hex.DecodeString(keyStr)
	if err != nil {
		return nil
	}

	privateKey, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil
	}

	return privateKey
}

type devnetKeyring struct {
	devnetKeys map[string]*ecdsa.PrivateKey
	chainID    *big.Int
}

func (d *devnetKeyring) getPrivateKey(key devkeys.Key) *ecdsa.PrivateKey {
	keyPath := key.String()
	privateKey, exists := d.devnetKeys[keyPath]

	if !exists {
		// If it's a UserKey, try to map it to a ChainUserKey for L2 dev-account
		if userKey, ok := key.(devkeys.UserKey); ok {
			chainUserKey := devkeys.ChainUserKey{ChainID: d.chainID, Index: uint64(userKey)}
			chainKeyPath := chainUserKey.String()
			if chainPrivateKey, chainExists := d.devnetKeys[chainKeyPath]; chainExists {
				return chainPrivateKey
			}
		}
		panic(fmt.Sprintf("devnet key not found for %s - ensure all required keys are present in devnet configuration", keyPath))
	}

	return privateKey
}

func (d *devnetKeyring) Secret(key devkeys.Key) *ecdsa.PrivateKey {
	return d.getPrivateKey(key)
}

func (d *devnetKeyring) Address(key devkeys.Key) common.Address {
	privateKey := d.getPrivateKey(key)
	return crypto.PubkeyToAddress(privateKey.PublicKey)
}
