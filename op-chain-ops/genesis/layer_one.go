package genesis

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-chain-ops/state"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var proxies = []string{
	"L2OutputOracleProxy",
	"L1CrossDomainMessengerProxy",
	"L1StandardBridgeProxy",
	"OptimismPortalProxy",
	"OptimismMintableERC20FactoryProxy",
}

var portalMeteringSlot = common.Hash{31: 0x01}

var zeroHash common.Hash

func BuildL1DeveloperGenesis(config *DeployConfig) (*core.Genesis, error) {
	if config.L2OutputOracleStartingTimestamp != -1 {
		return nil, errors.New("l2oo starting timestamp must be -1")
	}

	if config.L1GenesisBlockTimestamp == 0 {
		return nil, errors.New("must specify l1 genesis block timestamp")
	}

	genesis, err := NewL1Genesis(config)
	if err != nil {
		return nil, err
	}

	backend := deployer.NewBackendWithGenesisTimestamp(uint64(config.L1GenesisBlockTimestamp))

	deployments, err := deployL1Contracts(config, backend)
	if err != nil {
		return nil, err
	}

	depsByName := make(map[string]deployer.Deployment)
	depsByAddr := make(map[common.Address]deployer.Deployment)
	for _, dep := range deployments {
		depsByName[dep.Name] = dep
		depsByAddr[dep.Address] = dep
	}

	opts, err := bind.NewKeyedTransactorWithChainID(deployer.TestKey, deployer.ChainID)
	if err != nil {
		return nil, err
	}

	l2ooABI, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err := l2ooABI.Pack(
		"initialize",
		config.L2OutputOracleGenesisL2Output,
		config.L2OutputOracleProposer,
		config.L2OutputOracleOwner,
	)
	if err != nil {
		return nil, err
	}
	if _, err := upgradeProxy(
		backend,
		opts,
		depsByName["L2OutputOracleProxy"].Address,
		depsByName["L2OutputOracle"].Address,
		data,
	); err != nil {
		return nil, err
	}

	portalABI, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err = portalABI.Pack("initialize")
	if err != nil {
		return nil, err
	}
	if _, err := upgradeProxy(
		backend,
		opts,
		depsByName["OptimismPortalProxy"].Address,
		depsByName["OptimismPortal"].Address,
		data,
	); err != nil {
		return nil, err
	}
	l1XDMABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	data, err = l1XDMABI.Pack("initialize")
	if err != nil {
		return nil, err
	}
	if _, err := upgradeProxy(
		backend,
		opts,
		depsByName["L1CrossDomainMessengerProxy"].Address,
		depsByName["L1CrossDomainMessenger"].Address,
		data,
	); err != nil {
		return nil, err
	}

	if _, err := upgradeProxy(
		backend,
		opts,
		depsByName["L1StandardBridgeProxy"].Address,
		depsByName["L1StandardBridge"].Address,
		nil,
	); err != nil {
		return nil, err
	}

	var lastUpgradeTx *types.Transaction
	if lastUpgradeTx, err = upgradeProxy(
		backend,
		opts,
		depsByName["OptimismMintableERC20FactoryProxy"].Address,
		depsByName["OptimismMintableERC20Factory"].Address,
		nil,
	); err != nil {
		return nil, err
	}

	// Commit all the upgrades at once, then wait for the last
	// transaction to be mined. The simulator performs async
	// processing, and as such we need to wait for the transaction
	// receipt to appear before considering the above transactions
	// committed to the chain.

	backend.Commit()
	if _, err := bind.WaitMined(context.Background(), backend, lastUpgradeTx); err != nil {
		return nil, err
	}

	memDB := state.NewMemoryStateDB(genesis)
	if err := SetL1Proxies(memDB, predeploys.DevProxyAdminAddr); err != nil {
		return nil, err
	}
	FundDevAccounts(memDB)
	SetPrecompileBalances(memDB)

	for name, proxyAddr := range predeploys.DevPredeploys {
		memDB.SetState(*proxyAddr, ImplementationSlot, depsByName[name].Address.Hash())

		// Special case for WETH since it was not designed to be behind a proxy
		if name == "WETH9" {
			name, _ := state.EncodeStringValue("Wrapped Ether", 0)
			symbol, _ := state.EncodeStringValue("WETH", 0)
			decimals, _ := state.EncodeUintValue(18, 0)
			memDB.SetState(*proxyAddr, common.Hash{}, name)
			memDB.SetState(*proxyAddr, common.Hash{31: 0x01}, symbol)
			memDB.SetState(*proxyAddr, common.Hash{31: 0x02}, decimals)
		}
	}

	stateDB, err := backend.Blockchain().State()
	if err != nil {
		return nil, err
	}

	for _, dep := range deployments {
		st := stateDB.StorageTrie(dep.Address)
		iter := trie.NewIterator(st.NodeIterator(nil))

		depAddr := dep.Address
		if strings.HasSuffix(dep.Name, "Proxy") {
			depAddr = *predeploys.DevPredeploys[strings.TrimSuffix(dep.Name, "Proxy")]
		}

		memDB.CreateAccount(depAddr)
		memDB.SetCode(depAddr, dep.Bytecode)

		for iter.Next() {
			_, data, _, err := rlp.Split(iter.Value)
			if err != nil {
				return nil, err
			}

			key := common.BytesToHash(st.GetKey(iter.Key))
			value := common.BytesToHash(data)

			if depAddr == predeploys.DevOptimismPortalAddr && key == portalMeteringSlot {
				// We need to manually set the block number in the resource
				// metering storage slot to zero. Otherwise, deposits will
				// revert.
				copy(value[:24], zeroHash[:])
			}

			memDB.SetState(depAddr, key, value)
		}
	}
	return memDB.Genesis(), nil
}

func deployL1Contracts(config *DeployConfig, backend *backends.SimulatedBackend) ([]deployer.Deployment, error) {
	constructors := make([]deployer.Constructor, 0)
	for _, proxy := range proxies {
		constructors = append(constructors, deployer.Constructor{
			Name: proxy,
		})
	}
	constructors = append(constructors, []deployer.Constructor{
		{
			Name: "L2OutputOracle",
			Args: []interface{}{
				uint642Big(config.L2OutputOracleSubmissionInterval),
				[32]byte(config.L2OutputOracleGenesisL2Output),
				big.NewInt(0),
				big.NewInt(0),
				uint642Big(uint64(config.L1GenesisBlockTimestamp)),
				uint642Big(config.L2BlockTime),
				config.L2OutputOracleProposer,
				config.L2OutputOracleOwner,
			},
		},
		{
			Name: "OptimismPortal",
			Args: []interface{}{
				uint642Big(config.FinalizationPeriodSeconds),
			},
		},
		{
			Name: "L1CrossDomainMessenger",
		},
		{
			Name: "L1StandardBridge",
		},
		{
			Name: "L1ERC721Bridge",
		},
		{
			Name: "OptimismMintableERC20Factory",
		},
		{
			Name: "AddressManager",
		},
		{
			Name: "ProxyAdmin",
			Args: []interface{}{
				common.Address{19: 0x01},
			},
		},
		{
			Name: "WETH9",
		},
	}...)
	return deployer.Deploy(backend, constructors, l1Deployer)
}

func l1Deployer(backend *backends.SimulatedBackend, opts *bind.TransactOpts, deployment deployer.Constructor) (*types.Transaction, error) {
	var tx *types.Transaction
	var err error

	switch deployment.Name {
	case "L2OutputOracle":
		_, tx, _, err = bindings.DeployL2OutputOracle(
			opts,
			backend,
			deployment.Args[0].(*big.Int),
			deployment.Args[1].([32]byte),
			deployment.Args[2].(*big.Int),
			deployment.Args[3].(*big.Int),
			deployment.Args[4].(*big.Int),
			deployment.Args[5].(*big.Int),
			deployment.Args[6].(common.Address),
			deployment.Args[7].(common.Address),
		)
	case "OptimismPortal":
		_, tx, _, err = bindings.DeployOptimismPortal(
			opts,
			backend,
			predeploys.DevL2OutputOracleAddr,
			deployment.Args[0].(*big.Int),
		)
	case "L1CrossDomainMessenger":
		_, tx, _, err = bindings.DeployL1CrossDomainMessenger(
			opts,
			backend,
			predeploys.DevOptimismPortalAddr,
		)
	case "L1StandardBridge":
		_, tx, _, err = bindings.DeployL1StandardBridge(
			opts,
			backend,
			predeploys.DevL1CrossDomainMessengerAddr,
		)
	case "OptimismMintableERC20Factory":
		_, tx, _, err = bindings.DeployOptimismMintableERC20Factory(
			opts,
			backend,
			predeploys.DevL1StandardBridgeAddr,
		)
	case "AddressManager":
		_, tx, _, err = bindings.DeployAddressManager(
			opts,
			backend,
		)
	case "ProxyAdmin":
		_, tx, _, err = bindings.DeployProxyAdmin(
			opts,
			backend,
			common.Address{},
		)
	case "WETH9":
		_, tx, _, err = bindings.DeployWETH9(
			opts,
			backend,
		)
	case "L1ERC721Bridge":
		_, tx, _, err = bindings.DeployL1ERC721Bridge(
			opts,
			backend,
			predeploys.DevL1CrossDomainMessengerAddr,
			predeploys.L2ERC721BridgeAddr,
		)
	default:
		if strings.HasSuffix(deployment.Name, "Proxy") {
			_, tx, _, err = bindings.DeployProxy(opts, backend, deployer.TestAddress)
		} else {
			err = fmt.Errorf("unknown contract %s", deployment.Name)
		}
	}

	return tx, err
}

func upgradeProxy(backend *backends.SimulatedBackend, opts *bind.TransactOpts, proxyAddr common.Address, implAddr common.Address, callData []byte) (*types.Transaction, error) {
	var tx *types.Transaction
	proxy, err := bindings.NewProxy(proxyAddr, backend)
	if err != nil {
		return nil, err
	}
	if callData == nil {
		tx, err = proxy.UpgradeTo(opts, implAddr)
	} else {
		tx, err = proxy.UpgradeToAndCall(
			opts,
			implAddr,
			callData,
		)
	}
	return tx, err
}
