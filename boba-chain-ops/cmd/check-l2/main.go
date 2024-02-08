package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	"github.com/ledgerwatch/erigon/accounts/abi/bind"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/bindings"
	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/clients"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/genesis"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/log/v3"
)

var defaultCrossDomainMessageSender = libcommon.HexToAddress("0x000000000000000000000000000000000000dead")

// Default script for checking that L2 has been configured correctly. This should be extended in the future
// to pull in L1 deploy artifacts and assert that the L2 state is consistent with the L1 state.
func main() {
	log.Root().SetHandler(log.StreamHandler(os.Stderr, log.TerminalFormat()))

	app := &cli.App{
		Name:  "check-l2",
		Usage: "Check that an OP Stack L2 has been configured correctly",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "l1-rpc-url",
				Value:   "http://127.0.0.1:8545",
				Usage:   "L1 RPC URL",
				EnvVars: []string{"L1_RPC_URL"},
			},
			&cli.StringFlag{
				Name:    "l2-rpc-url",
				Value:   "http://127.0.0.1:9545",
				Usage:   "L2 RPC URL",
				EnvVars: []string{"L2_RPC_URL"},
			},
			&cli.Int64Flag{
				Name:    "l2-block-number",
				Value:   0,
				Usage:   "L2 block number",
				EnvVars: []string{"L2_BLOCK_NUMBER"},
			},
		},
		Action: entrypoint,
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("error checking l2", "err", err)
	}
}

// entrypoint is the entrypoint for the check-l2 script
func entrypoint(ctx *cli.Context) error {
	clients, err := clients.NewClients(ctx)
	if err != nil {
		return err
	}

	log.Info("Checking predeploy proxy config")
	g := new(errgroup.Group)

	// Check that all proxies are configured correctly
	// Do this in parallel but not too quickly to allow for
	// querying against rate limiting RPC backends
	count := uint64(2048)
	for i := uint64(0); i < count; i++ {
		i := i
		if i%4 == 0 {
			log.Info("Checking proxy", "index", i, "total", count)
			if err := g.Wait(); err != nil {
				return err
			}
		}
		g.Go(func() error {
			return checkPredeploy(clients.L2RpcClient, i)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	log.Info("All predeploy proxies are set correctly")

	// Check that all of the defined predeploys are set up correctly
	for name, addr := range predeploys.Predeploys {
		log.Info("Checking predeploy", "name", name, "address", addr.Hex())
		if err := checkPredeployConfig(clients.L2RpcClient, name); err != nil {
			return err
		}
	}
	return nil
}

// checkPredeploy ensures that the predeploy at index i has the correct proxy admin set
func checkPredeploy(client *clients.RpcClient, i uint64) error {
	bigAddr := new(big.Int).Or(genesis.BigL2PredeployNamespace, new(big.Int).SetUint64(i))
	addr := libcommon.BigToAddress(bigAddr)
	if !predeploys.IsProxied(addr) {
		return nil
	}
	admin, err := getEIP1967AdminAddress(client, addr)
	if err != nil {
		return err
	}
	if admin != predeploys.ProxyAdminAddr {
		return fmt.Errorf("%s does not have correct proxy admin set", addr)
	}
	return nil
}

// checkPredeployConfig checks that the defined predeploys are configured correctly
func checkPredeployConfig(client *clients.RpcClient, name string) error {
	predeploy := predeploys.Predeploys[name]
	if predeploy == nil {
		return fmt.Errorf("unknown predeploy %s", name)
	}
	p := *predeploy

	g := new(errgroup.Group)
	if predeploys.IsProxied(p) {
		// Check that an implementation is set. If the implementation has been upgraded,
		// it will be considered non-standard. Ensure that there is code set at the implementation.
		g.Go(func() error {
			impl, err := getEIP1967ImplementationAddress(client, p)
			if err != nil {
				return err
			}
			log.Info("checking contract", "name", name, "implementation", impl.Hex())
			standardImpl, err := genesis.AddressToCodeNamespace(p)
			if err != nil {
				return err
			}
			if impl != standardImpl {
				log.Warn("contract does not have the standard implementation", "name", name)
			}
			implCode, err := client.CodeAt(context.Background(), impl, nil)
			if err != nil {
				return err
			}
			if len(implCode) == 0 {
				return fmt.Errorf("%s implementation is not deployed", name)
			}
			return nil
		})

		// Ensure that the code is set to the proxy bytecode as expected
		g.Go(func() error {
			proxyCode, err := client.CodeAt(context.Background(), p, nil)
			if err != nil {
				return err
			}
			proxy, err := bindings.GetDeployedBytecode("Proxy")
			if err != nil {
				return err
			}
			if !bytes.Equal(proxyCode, proxy) {
				log.Warn("contract does not have the standard proxy bytecode", "name", name)
			}
			return nil
		})
	}

	// Check the predeploy specific config is correct
	g.Go(func() error {
		switch p {
		case predeploys.LegacyMessagePasserAddr:
			if err := checkLegacyMessagePasser(p, client); err != nil {
				return err
			}

		case predeploys.DeployerWhitelistAddr:
			if err := checkDeployerWhitelist(p, client); err != nil {
				return err
			}

		case predeploys.L2CrossDomainMessengerAddr:
			if err := checkL2CrossDomainMessenger(p, client); err != nil {
				return err
			}

		case predeploys.GasPriceOracleAddr:
			if err := checkGasPriceOracle(p, client); err != nil {
				return err
			}

		case predeploys.L2StandardBridgeAddr:
			if err := checkL2StandardBridge(p, client); err != nil {
				return err
			}

		case predeploys.SequencerFeeVaultAddr:
			if err := checkSequencerFeeVault(p, client); err != nil {
				return err
			}

		case predeploys.OptimismMintableERC20FactoryAddr:
			if err := checkOptimismMintableERC20Factory(p, client); err != nil {
				return err
			}

		case predeploys.L1BlockNumberAddr:
			if err := checkL1BlockNumber(p, client); err != nil {
				return err
			}

		case predeploys.L1BlockAddr:
			if err := checkL1Block(p, client); err != nil {
				return err
			}

		case predeploys.WETH9Addr:
			if err := checkWETH9(p, client); err != nil {
				return err
			}

		case predeploys.L2ERC721BridgeAddr:
			if err := checkL2ERC721Bridge(p, client); err != nil {
				return err
			}

		case predeploys.OptimismMintableERC721FactoryAddr:
			if err := checkOptimismMintableERC721Factory(p, client); err != nil {
				return err
			}

		case predeploys.ProxyAdminAddr:
			if err := checkProxyAdmin(p, client); err != nil {
				return err
			}

		case predeploys.BaseFeeVaultAddr:
			if err := checkBaseFeeVault(p, client); err != nil {
				return err
			}

		case predeploys.L1FeeVaultAddr:
			if err := checkL1FeeVault(p, client); err != nil {
				return err
			}

		case predeploys.L2ToL1MessagePasserAddr:
			if err := checkL2ToL1MessagePasser(p, client); err != nil {
				return err
			}

		case predeploys.SchemaRegistryAddr:
			if err := checkSchemaRegistry(p, client); err != nil {
				return err
			}

		case predeploys.EASAddr:
			if err := checkEAS(p, client); err != nil {
				return err
			}

		case predeploys.BobaL2Addr:
			if err := checkBobaL2(p, client); err != nil {
				return err
			}

		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func checkL2ToL1MessagePasser(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL2ToL1MessagePasser(addr, client)
	if err != nil {
		return err
	}
	messageVersion, err := contract.MESSAGEVERSION(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2ToL1MessagePasser", "MESSAGE_VERSION", messageVersion)

	messageNonce, err := contract.MessageNonce(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2ToL1MessagePasser", "MESSAGE_NONCE", messageNonce)
	return nil
}

func checkL1FeeVault(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL1FeeVault(addr, client)
	if err != nil {
		return err
	}
	recipient, err := contract.RECIPIENT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L1FeeVault", "RECIPIENT", recipient.Hex())
	if recipient == (libcommon.Address{}) {
		return errors.New("RECIPIENT should not be address(0)")
	}

	minWithdrawalAmount, err := contract.MINWITHDRAWALAMOUNT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L1FeeVault", "MIN_WITHDRAWAL_AMOUNT", minWithdrawalAmount)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L1FeeVault version", "version", version)
	return nil
}

func checkBaseFeeVault(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewBaseFeeVault(addr, client)
	if err != nil {
		return err
	}
	recipient, err := contract.RECIPIENT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("BaseFeeVault", "RECIPIENT", recipient.Hex())
	if recipient == (libcommon.Address{}) {
		return errors.New("RECIPIENT should not be address(0)")
	}

	minWithdrawalAmount, err := contract.MINWITHDRAWALAMOUNT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("BaseFeeVault", "MIN_WITHDRAWAL_AMOUNT", minWithdrawalAmount)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("BaseFeeVault version", "version", version)
	return nil
}

func checkProxyAdmin(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewProxyAdmin(addr, client)
	if err != nil {
		return err
	}

	owner, err := contract.Owner(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("ProxyAdmin", "owner", owner.Hex())
	if owner == (libcommon.Address{}) {
		return errors.New("ProxyAdmin.owner is zero address")
	}

	addressManager, err := contract.AddressManager(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("ProxyAdmin", "addressManager", addressManager.Hex())
	return nil
}

func checkOptimismMintableERC721Factory(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewOptimismMintableERC721Factory(addr, client)
	if err != nil {
		return err
	}
	bridge, err := contract.BRIDGE(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("OptimismMintableERC721Factory", "BRIDGE", bridge.Hex())
	if bridge == (libcommon.Address{}) {
		return errors.New("OptimismMintableERC721Factory.BRIDGE is zero address")
	}

	remoteChainID, err := contract.REMOTECHAINID(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("OptimismMintableERC721Factory", "REMOTE_CHAIN_ID", remoteChainID)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("OptimismMintableERC721Factory version", "version", version)
	return nil
}

func checkL2ERC721Bridge(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL2ERC721Bridge(addr, client)
	if err != nil {
		return err
	}
	messenger, err := contract.MESSENGER(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2ERC721Bridge", "MESSENGER", messenger.Hex())
	if messenger == (libcommon.Address{}) {
		return errors.New("L2ERC721Bridge.MESSENGER is zero address")
	}

	otherBridge, err := contract.OTHERBRIDGE(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2ERC721Bridge", "OTHERBRIDGE", otherBridge.Hex())
	if otherBridge == (libcommon.Address{}) {
		return errors.New("L2ERC721Bridge.OTHERBRIDGE is zero address")
	}

	initialized, err := getInitialized("L2ERC721Bridge", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2ERC721Bridge", "_initialized", initialized)

	initializing, err := getInitializing("L2ERC721Bridge", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2ERC721Bridge", "_initializing", initializing)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2ERC721Bridge version", "version", version)
	return nil
}

func checkWETH9(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewWETH9(addr, client)
	if err != nil {
		return err
	}
	name, err := contract.Name(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("WETH9", "name", name)
	if name != "Wrapped Ether" {
		return fmt.Errorf("WETH9 name should be 'Wrapped Ether', got %s", name)
	}

	symbol, err := contract.Symbol(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("WETH9", "symbol", symbol)
	if symbol != "WETH" {
		return fmt.Errorf("WETH9 symbol should be 'WETH', got %s", symbol)
	}

	decimals, err := contract.Decimals(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("WETH9", "decimals", decimals)
	if decimals != 18 {
		return fmt.Errorf("WETH9 decimals should be 18, got %d", decimals)
	}
	return nil
}

func checkL1Block(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL1Block(addr, client)
	if err != nil {
		return err
	}
	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L1Block version", "version", version)
	return nil
}

func checkL1BlockNumber(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL1BlockNumber(addr, client)
	if err != nil {
		return err
	}
	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L1BlockNumber version", "version", version)
	return nil
}

func checkOptimismMintableERC20Factory(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewOptimismMintableERC20Factory(addr, client)
	if err != nil {
		return err
	}

	bridgeLegacy, err := contract.BRIDGE(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("OptimismMintableERC20Factory", "BRIDGE", bridgeLegacy.Hex())
	if bridgeLegacy == (libcommon.Address{}) {
		return errors.New("OptimismMintableERC20Factory.BRIDGE is zero address")
	}

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("OptimismMintableERC20Factory version", "version", version)

	if version > "1.1.1" {
		bridge, err := contract.Bridge(&bind.CallOpts{})
		if err != nil {
			return err
		}
		if bridge == (libcommon.Address{}) {
			return errors.New("OptimismMintableERC20Factory.bridge is zero address")
		}
		log.Info("OptimismMintableERC20Factory", "bridge", bridge.Hex())
	}
	return nil
}

func checkSequencerFeeVault(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewSequencerFeeVault(addr, client)
	if err != nil {
		return err
	}
	recipient, err := contract.RECIPIENT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("SequencerFeeVault", "RECIPIENT", recipient.Hex())
	if recipient == (libcommon.Address{}) {
		return errors.New("RECIPIENT should not be address(0)")
	}

	l1FeeWallet, err := contract.L1FeeWallet(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("SequencerFeeVault", "l1FeeWallet", l1FeeWallet.Hex())

	minWithdrawalAmount, err := contract.MINWITHDRAWALAMOUNT(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("SequencerFeeVault", "MIN_WITHDRAWAL_AMOUNT", minWithdrawalAmount)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("SequencerFeeVault version", "version", version)
	return nil
}

func checkL2StandardBridge(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL2StandardBridge(addr, client)
	if err != nil {
		return err
	}
	otherBridge, err := contract.OTHERBRIDGE(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if otherBridge == (libcommon.Address{}) {
		return errors.New("OTHERBRIDGE should not be address(0)")
	}
	log.Info("L2StandardBridge", "OTHERBRIDGE", otherBridge.Hex())

	messenger, err := contract.MESSENGER(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2StandardBridge", "MESSENGER", messenger.Hex())
	if messenger != predeploys.L2CrossDomainMessengerAddr {
		return fmt.Errorf("L2StandardBridge MESSENGER should be %s, got %s", predeploys.L2CrossDomainMessengerAddr, messenger)
	}
	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}

	initialized, err := getInitialized("L2StandardBridge", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2StandardBridge", "_initialized", initialized)

	initializing, err := getInitializing("L2StandardBridge", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2StandardBridge", "_initializing", initializing)

	log.Info("L2StandardBridge version", "version", version)
	return nil
}

func checkGasPriceOracle(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewGasPriceOracle(addr, client)
	if err != nil {
		return err
	}
	decimals, err := contract.DECIMALS(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("GasPriceOracle", "DECIMALS", decimals)
	if decimals.Cmp(big.NewInt(6)) != 0 {
		return fmt.Errorf("GasPriceOracle decimals should be 6, got %v", decimals)
	}

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("GasPriceOracle version", "version", version)
	return nil
}

func checkL2CrossDomainMessenger(addr libcommon.Address, client *clients.RpcClient) error {
	slot, err := client.StorageAt(context.Background(), addr, libcommon.Hash{31: 0xcc}, nil)
	if err != nil {
		return err
	}
	if libcommon.BytesToAddress(slot) != defaultCrossDomainMessageSender {
		return fmt.Errorf("Expected xDomainMsgSender to be %s, got %s", defaultCrossDomainMessageSender, addr)
	}

	contract, err := bindings.NewL2CrossDomainMessenger(addr, client)
	if err != nil {
		return err
	}

	otherMessenger, err := contract.OTHERMESSENGER(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if otherMessenger == (libcommon.Address{}) {
		return errors.New("OTHERMESSENGER should not be address(0)")
	}
	log.Info("L2CrossDomainMessenger", "OTHERMESSENGER", otherMessenger.Hex())

	l1CrossDomainMessenger, err := contract.L1CrossDomainMessenger(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "l1CrossDomainMessenger", l1CrossDomainMessenger.Hex())

	messageVersion, err := contract.MESSAGEVERSION(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "MESSAGE_VERSION", messageVersion)
	minGasCallDataOverhead, err := contract.MINGASCALLDATAOVERHEAD(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "MIN_GAS_CALLDATA_OVERHEAD", minGasCallDataOverhead)

	relayConstantOverhead, err := contract.RELAYCONSTANTOVERHEAD(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "RELAY_CONSTANT_OVERHEAD", relayConstantOverhead)

	minGasDynamicsOverheadDenominator, err := contract.MINGASDYNAMICOVERHEADDENOMINATOR(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR", minGasDynamicsOverheadDenominator)

	minGasDynamicsOverheadNumerator, err := contract.MINGASDYNAMICOVERHEADNUMERATOR(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR", minGasDynamicsOverheadNumerator)

	relayCallOverhead, err := contract.RELAYCALLOVERHEAD(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "RELAY_CALL_OVERHEAD", relayCallOverhead)

	relayReservedGas, err := contract.RELAYRESERVEDGAS(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "RELAY_RESERVED_GAS", relayReservedGas)

	relayGasCheckBuffer, err := contract.RELAYGASCHECKBUFFER(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "RELAY_GAS_CHECK_BUFFER", relayGasCheckBuffer)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}

	initialized, err := getInitialized("L2CrossDomainMessenger", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "_initialized", initialized)

	initializing, err := getInitializing("L2CrossDomainMessenger", addr, client)
	if err != nil {
		return err
	}
	log.Info("L2CrossDomainMessenger", "_initializing", initializing)

	log.Info("L2CrossDomainMessenger version", "version", version)
	return nil
}

func checkLegacyMessagePasser(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewLegacyMessagePasser(addr, client)
	if err != nil {
		return err
	}
	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("LegacyMessagePasser version", "version", version)
	return nil
}

func checkDeployerWhitelist(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewDeployerWhitelist(addr, client)
	if err != nil {
		return err
	}
	owner, err := contract.Owner(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if owner != (libcommon.Address{}) {
		return fmt.Errorf("DeployerWhitelist owner should be set to address(0)")
	}
	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("DeployerWhitelist version", "version", version)
	return nil
}

func getEIP1967AdminAddress(client *clients.RpcClient, addr libcommon.Address) (libcommon.Address, error) {
	slot, err := client.StorageAt(context.Background(), addr, genesis.AdminSlot, nil)
	if err != nil {
		return libcommon.Address{}, err
	}
	admin := libcommon.BytesToAddress(slot)
	return admin, nil
}

func checkSchemaRegistry(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewSchemaRegistry(addr, client)
	if err != nil {
		return err
	}

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("SchemaRegistry version", "version", version)
	return nil
}

func checkEAS(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewEAS(addr, client)
	if err != nil {
		return err
	}

	registry, err := contract.GetSchemaRegistry(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if registry != predeploys.SchemaRegistryAddr {
		return fmt.Errorf("Incorrect registry address %s", registry)
	}
	log.Info("EAS", "registry", registry)

	version, err := contract.Version(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("EAS version", "version", version)
	return nil
}

func checkBobaL2(addr libcommon.Address, client *clients.RpcClient) error {
	contract, err := bindings.NewL2GovernanceERC20(addr, client)
	if err != nil {
		return err
	}
	l2Bridge, err := contract.L2Bridge(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if l2Bridge == (libcommon.Address{}) {
		return fmt.Errorf("BobaL2 l2Bridge should not be set to address(0)")
	}
	log.Info("BobaL2", "l2Bridge", l2Bridge.Hex())
	l1Token, err := contract.L1Token(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if l1Token == (libcommon.Address{}) {
		return fmt.Errorf("BobaL2 l1Token should not be set to address(0)")
	}
	log.Info("BobaL2", "l1Token", l1Token.Hex())
	name, err := contract.Name(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if name != "Boba Token" && name != "Boba Network" {
		return fmt.Errorf("BobaL2 name should be 'Boba Token' or 'Boba Network', got %s", name)
	}
	log.Info("BobaL2", "name", name)
	symbol, err := contract.Symbol(&bind.CallOpts{})
	if err != nil {
		return err
	}
	if symbol != "BOBA" {
		return fmt.Errorf("BobaL2 symbol should be 'BOBA', got %s", symbol)
	}
	decimals, err := contract.Decimals(&bind.CallOpts{})
	if err != nil {
		return err
	}
	log.Info("BobaL2", "decimals", decimals)

	return nil
}

func getEIP1967ImplementationAddress(client *clients.RpcClient, addr libcommon.Address) (libcommon.Address, error) {
	slot, err := client.StorageAt(context.Background(), addr, genesis.ImplementationSlot, nil)
	if err != nil {
		return libcommon.Address{}, err
	}
	impl := libcommon.BytesToAddress(slot)
	return impl, nil
}

// getInitialized will get the initialized value in storage of a contract.
// This is an incrementing number that starts at 1 and increments each time that
// the contract is upgraded.
func getInitialized(name string, addr libcommon.Address, client *clients.RpcClient) (*big.Int, error) {
	value, err := getStorageValue(name, "_initialized", addr, client)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(value), nil
}

// getInitializing will get the _initializing value in storage of a contract.
func getInitializing(name string, addr libcommon.Address, client *clients.RpcClient) (bool, error) {
	value, err := getStorageValue(name, "_initializing", addr, client)
	if err != nil {
		return false, err
	}
	if len(value) != 1 {
		return false, fmt.Errorf("Unexpected length for _initializing: %d", len(value))
	}
	return value[0] == 1, nil
}

// getStorageValue will get the value of a named storage slot in a contract. It isn't smart about
// automatically converting from a byte slice to a type, it is the caller's responsibility to do that.
func getStorageValue(name, entryName string, addr libcommon.Address, client *clients.RpcClient) ([]byte, error) {
	layout, err := bindings.GetStorageLayout(name)
	if err != nil {
		return nil, err
	}
	entry, err := layout.GetStorageLayoutEntry(entryName)
	if err != nil {
		return nil, err
	}
	typ, err := layout.GetStorageLayoutType(entry.Type)
	if err != nil {
		return nil, err
	}
	slot := libcommon.BigToHash(big.NewInt(int64(entry.Slot)))
	value, err := client.StorageAt(context.Background(), addr, slot, nil)
	if err != nil {
		return nil, err
	}
	if entry.Offset+typ.NumberOfBytes > uint(len(value)) {
		return nil, fmt.Errorf("value length is too short")
	}
	// Swap the endianness
	slice := libcommon.CopyBytes(value)
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice[entry.Offset : entry.Offset+typ.NumberOfBytes], nil
}
