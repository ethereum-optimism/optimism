// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { Deployer } from "scripts/deploy/Deployer.sol";

// Libraries
import { Config } from "scripts/libraries/Config.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console2 as console } from "forge-std/console2.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ForkL2Live
/// @notice Sets up L2 fork tests by reading predeploy implementations from forked state
///         and optionally reading chain metadata from superchain-registry.
contract ForkL2Live is Deployer {
    /// @notice Whether Custom Gas Token is detected on the forked chain.
    bool public isCustomGasToken;

    /// @notice Whether interop features are detected on the forked chain.
    bool public isInteropEnabled;

    /// @notice Main entry point for L2 fork setup.
    function run() public {
        console.log("ForkL2Live: Starting L2 fork setup");

        // Detect chain features from forked state
        _detectChainFeatures();

        // Read and save predeploy implementation addresses
        _savePredeployImplementations();

        // Optionally read chain metadata from superchain-registry
        _readL2ChainMetadata();

        console.log("ForkL2Live: L2 fork setup complete");
    }

    /// @notice Detects chain features from the forked L2 state.
    function _detectChainFeatures() internal {
        // Detect Custom Gas Token
        try IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken() returns (bool isCGT_) {
            isCustomGasToken = isCGT_;
            if (isCGT_) {
                console.log("ForkL2Live: Custom Gas Token detected");
            }
        } catch {
            isCustomGasToken = false;
        }

        // Detect interop by checking if CrossL2Inbox implementation has code
        address crossL2InboxImpl = EIP1967Helper.getImplementation(Predeploys.CROSS_L2_INBOX);
        isInteropEnabled = crossL2InboxImpl.code.length > 0;
        if (isInteropEnabled) {
            console.log("ForkL2Live: Interop features detected");
        }

        // Log contract versions for key predeploys
        _logPredeployVersion("L2CrossDomainMessenger", Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        _logPredeployVersion("L2StandardBridge", Predeploys.L2_STANDARD_BRIDGE);
        _logPredeployVersion("L1Block", Predeploys.L1_BLOCK_ATTRIBUTES);
    }

    /// @notice Logs the version of a predeploy contract if available.
    function _logPredeployVersion(string memory _name, address _proxy) internal view {
        // Try to call version() on the proxy
        (bool success, bytes memory data) = _proxy.staticcall(abi.encodeCall(ISemver.version, ()));
        if (success && data.length > 0) {
            string memory version = abi.decode(data, (string));
            console.log("ForkL2Live: %s version %s", _name, version);
        }
    }

    /// @notice Reads and saves implementation addresses for all L2 predeploys.
    function _savePredeployImplementations() internal {
        console.log("ForkL2Live: Reading predeploy implementations");

        // Core predeploys
        _saveImpl("L2CrossDomainMessenger", Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        _saveImpl("L2StandardBridge", Predeploys.L2_STANDARD_BRIDGE);
        _saveImpl("L2ToL1MessagePasser", Predeploys.L2_TO_L1_MESSAGE_PASSER);
        _saveImpl("SequencerFeeVault", Predeploys.SEQUENCER_FEE_WALLET);
        _saveImpl("OptimismMintableERC20Factory", Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
        _saveImpl("L2ERC721Bridge", Predeploys.L2_ERC721_BRIDGE);
        _saveImpl("OptimismMintableERC721Factory", Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        _saveImpl("L1Block", Predeploys.L1_BLOCK_ATTRIBUTES);
        _saveImpl("GasPriceOracle", Predeploys.GAS_PRICE_ORACLE);
        _saveImpl("ProxyAdmin", Predeploys.PROXY_ADMIN);

        // Fee vaults
        _saveImpl("BaseFeeVault", Predeploys.BASE_FEE_VAULT);
        _saveImpl("L1FeeVault", Predeploys.L1_FEE_VAULT);
        _saveImpl("OperatorFeeVault", Predeploys.OPERATOR_FEE_VAULT);

        // EAS
        _saveImplIfExists("SchemaRegistry", Predeploys.SCHEMA_REGISTRY);
        _saveImplIfExists("EAS", Predeploys.EAS);

        // Interop contracts (may not exist on all chains)
        _saveImplIfExists("CrossL2Inbox", Predeploys.CROSS_L2_INBOX);
        _saveImplIfExists("L2ToL2CrossDomainMessenger", Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        _saveImplIfExists("SuperchainETHBridge", Predeploys.SUPERCHAIN_ETH_BRIDGE);
        _saveImplIfExists("ETHLiquidity", Predeploys.ETH_LIQUIDITY);
        _saveImplIfExists("OptimismSuperchainERC20Factory", Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY);
        _saveImplIfExists("OptimismSuperchainERC20Beacon", Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        _saveImplIfExists("SuperchainTokenBridge", Predeploys.SUPERCHAIN_TOKEN_BRIDGE);

        // CGT contracts (may not exist on all chains)
        _saveImplIfExists("NativeAssetLiquidity", Predeploys.NATIVE_ASSET_LIQUIDITY);
        _saveImplIfExists("LiquidityController", Predeploys.LIQUIDITY_CONTROLLER);

        // Revenue share
        _saveImplIfExists("FeeSplitter", Predeploys.FEE_SPLITTER);

        // ConditionalDeployer (may not exist on older chains)
        _saveImplIfExists("ConditionalDeployer", Predeploys.CONDITIONAL_DEPLOYER);

        console.log("ForkL2Live: Saved predeploy implementations");
    }

    /// @notice Saves implementation address for a predeploy proxy.
    function _saveImpl(string memory _name, address _proxy) internal {
        address impl = EIP1967Helper.getImplementation(_proxy);
        artifacts.save(string.concat(_name, "Impl"), impl);
        console.log("ForkL2Live: %s impl at %s", _name, impl);
    }

    /// @notice Saves implementation address only if the predeploy has code.
    function _saveImplIfExists(string memory _name, address _proxy) internal {
        if (_proxy.code.length > 0) {
            _saveImpl(_name, _proxy);
        }
    }

    /// @notice Reads chain metadata from superchain-registry (optional).
    function _readL2ChainMetadata() internal view {
        string memory l2Chain = Config.l2ForkChain();
        string memory baseChain = Config.forkBaseChain();

        // Path to chain config in superchain-registry
        string memory tomlPath =
            string.concat("./lib/superchain-registry/superchain/configs/", baseChain, "/", l2Chain, ".toml");

        // Try to read the TOML file for additional metadata
        try vm.readFile(tomlPath) returns (string memory) {
            console.log("ForkL2Live: Read chain metadata from %s", tomlPath);
            // Could parse additional chain-specific config here if needed
        } catch {
            console.log("ForkL2Live: No registry config found for %s/%s", baseChain, l2Chain);
        }
    }
}
