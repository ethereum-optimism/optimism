// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Utilities
import { Script } from "forge-std/Script.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Constants } from "src/libraries/Constants.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { UpgradeUtils } from "scripts/libraries/UpgradeUtils.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

/// @title GenerateNUTBundle
/// @notice Generates Network Upgrade Transaction (NUT) bundles for L2 hardfork upgrades.
/// @dev This script creates deterministic upgrade transaction bundles for L2 hardfork upgrades
///      using the L2ContractsManager (L2CM) system.
contract GenerateNUTBundle is Script {
    /// @notice CREATE2 salt for deterministic deployments.
    bytes32 internal constant SALT = bytes32(uint256(keccak256("optimism.network-upgrade")));

    /// @notice Name of the upgrade.
    string internal constant UPGRADE_NAME = "karst";

    /// @notice Version of the upgrade bundle.
    string internal constant BUNDLE_VERSION = "1.0.0";

    /// @notice Output containing generated transactions.
    /// @param txns Array of Network Upgrade Transactions to execute.
    struct Output {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] txns;
    }

    /// @notice Configuration for a implementation contract deployment.
    /// @param implementation Expected implementation address after deployment.
    /// @param deploymentGasLimit Gas limit for the deployment transaction.
    /// @param artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
    /// @param name Human-readable name for the contract.
    /// @param args ABI-encoded constructor arguments.
    struct ImplementationConfig {
        address implementation;
        uint64 deploymentGasLimit;
        string name;
        string artifactPath;
    }

    /// @notice Gas limits for the upgrade.
    UpgradeUtils.GasLimits internal gasLimits;

    /// @notice Expected implementations for the upgrade.
    L2ContractsManagerTypes.Implementations internal implementations;

    /// @notice Implementation configurations.
    mapping(string => ImplementationConfig) public implementationConfigs;

    /// @notice Array of generated transactions.
    NetworkUpgradeTxns.NetworkUpgradeTxn[] internal txns;

    function setUp() public {
        // Clear previous txns: Transactions are pushed to a dynamic array, so we need
        // to delete the array to avoid pushing duplicates.
        delete txns;

        gasLimits = UpgradeUtils.gasLimits();
    }

    /// @notice Generates the complete upgrade transaction bundle.
    /// @dev Executes 5 phases in fixed order:
    ///      1. Pre-implementation deployments [CUSTOM]
    ///      2. Implementation deployments [FIXED]
    ///      3. Pre-L2CM deployment [CUSTOM]
    ///      4. L2CM deployment [FIXED]
    ///      5. Upgrade execution [FIXED]
    /// @dev Only modify phases 1 and 3 for fork-specific logic. Other phases must remain unchanged.
    /// @return output_ Output containing all generated transactions in execution order.
    function run() public returns (Output memory output_) {
        setUp();

        // Build implementation deployment configurations
        _buildImplementationDeploymentConfigs();

        // Phase 1: Pre-implementation deployments
        // Add fork-specific deployment or upgrade txns that must occur prior to the implementation deployments
        // phase.
        _preImplementationDeployments();

        // Phase 2: Implementation deployments
        _generateImplementationDeployments();

        // Build the implementations struct
        implementations = _getImplementations();

        // Phase 3: Pre-L2CM deployment
        // Add fork-specific deployment or upgrade logic that must occur between the implementation deployment
        // phase and the L2ContractsManager deployment phase.
        _preL2CMDeployment();

        // Phase 4: L2ContractsManager deployment
        _generateL2CMDeployment();

        // Phase 5: Upgrade execution
        _generateUpgradeExecution();

        // Copy storage array to memory array for return
        uint256 txnsLength = txns.length;
        output_.txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](txnsLength);
        for (uint256 i = 0; i < txnsLength; i++) {
            output_.txns[i] = txns[i];
        }

        _assertValidOutput(output_);

        // Write transactions to artifact with metadata
        NetworkUpgradeTxns.BundleMetadata memory metadata =
            NetworkUpgradeTxns.BundleMetadata({ version: BUNDLE_VERSION });
        NetworkUpgradeTxns.writeArtifact(txns, metadata, Constants.CURRENT_BUNDLE_PATH);
    }

    /// @notice Asserts the output is valid.
    /// @param _output The output to assert.
    function _assertValidOutput(Output memory _output) internal pure {
        uint256 transactionCount = UpgradeUtils.getTransactionCount();
        uint256 txnsLength = _output.txns.length;
        require(txnsLength == transactionCount, "GenerateNUTBundle: invalid transaction count");

        for (uint256 i = 0; i < txnsLength; i++) {
            require(_output.txns[i].data.length > 0, "GenerateNUTBundle: invalid transaction data");
            require(bytes(_output.txns[i].intent).length > 0, "GenerateNUTBundle: invalid transaction intent");
            require(_output.txns[i].to != address(0), "GenerateNUTBundle: invalid transaction to");
            require(_output.txns[i].gasLimit > 0, "GenerateNUTBundle: invalid transaction gasLimit");

            if (_output.txns[i].from == address(0)) {
                // Transactions must have a from address except for ProxyAdmin and ConditionalDeployer upgrades
                if (
                    _output.txns[i].to != Predeploys.PROXY_ADMIN
                        && _output.txns[i].to != Predeploys.CONDITIONAL_DEPLOYER
                ) {
                    revert("GenerateNUTBundle: invalid transaction from");
                }
            }
        }
    }

    /// @notice Asserts the implementation config is valid.
    /// @param _config The implementation config to assert.
    function _assertValidImplementationConfig(ImplementationConfig memory _config) internal pure {
        require(bytes(_config.name).length > 0, "GenerateNUTBundle: invalid implementation name");
        require(bytes(_config.artifactPath).length > 0, "GenerateNUTBundle: invalid implementation artifact path");
        require(_config.deploymentGasLimit > 0, "GenerateNUTBundle: invalid implementation deployment gas limit");
        require(_config.implementation != address(0), "GenerateNUTBundle: invalid implementation address");
    }

    // ========================================
    // CUSTOM NUT OPERATIONS
    // ========================================

    /// @notice Pre-implementation deployment phase for fork-specific setup.
    /// @dev Any transactions added to the txns array within this function will be executed BEFORE
    ///      any predeploy implementations are deployed. This is the designated location for adding
    ///      fork-specific deployment or upgrade logic that must occur prior to the standard
    ///      implementation deployment phase. The rest of the script follows a fixed structure and
    ///      should not be modified.
    function _preImplementationDeployments() internal {
        if (keccak256(abi.encodePacked(UPGRADE_NAME)) == keccak256(abi.encodePacked("karst"))) {
            // TODO(#19369): Remove these steps once Karst upgrade is deployed in all chains.
            // ConditionalDeployer deployment + upgrade
            _generateConditionalDeployerTxns();
        }
    }

    /// @notice Pre-L2CM deployment phase for fork-specific setup.
    /// @dev This function executes AFTER implementations are deployed but BEFORE the L2ContractsManager
    ///      is deployed. It is the designated location for adding fork-specific deployment or upgrade
    ///      logic that must occur between these two phases. The rest of the script follows a fixed
    ///      structure and should not be modified.
    /// @dev IMPORTANT: This is one of only TWO extension points in this script. Do not modify
    ///      the core deployment flow in _generateL2CMDeployment, _generateUpgradeExecution, or other
    ///      fixed phases.
    function _preL2CMDeployment() internal {
        if (keccak256(abi.encodePacked(UPGRADE_NAME)) == keccak256(abi.encodePacked("karst"))) {
            // TODO(#19369): Remove these steps once Karst upgrade is deployed in all chains.
            // L2ProxyAdmin upgrade
            _generateL2ProxyAdminUpgrade(implementations.proxyAdminImpl);
        }
    }

    // ========================================
    // KARST-ONLY NUTs
    // ========================================

    /// @notice Generates ConditionalDeployer deployment and upgrade transactions.
    /// @dev TODO(#19369): Remove this function once Karst upgrade is deployed in all chains.
    function _generateConditionalDeployerTxns() internal {
        // 1. Deploy ConditionalDeployer implementation
        bytes memory conditionalDeployerCode =
            abi.encodePacked(DeployUtils.getCode("ConditionalDeployer.sol:ConditionalDeployer"));

        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                intent: "ConditionalDeployer Deployment",
                from: Constants.DEPOSITOR_ACCOUNT,
                to: Preinstalls.DeterministicDeploymentProxy,
                gasLimit: gasLimits.conditionalDeployerDeployment,
                data: abi.encodePacked(SALT, conditionalDeployerCode)
            })
        );

        // 2. Upgrade ConditionalDeployer proxy
        address newConditionalDeployerImpl = UpgradeUtils.computeCreate2Address(conditionalDeployerCode, SALT);
        txns.push(
            UpgradeUtils.createUpgradeTxn(
                "ConditionalDeployer",
                Predeploys.CONDITIONAL_DEPLOYER,
                newConditionalDeployerImpl,
                gasLimits.conditionalDeployerUpgrade
            )
        );
    }

    /// @notice Generates L2ProxyAdmin upgrade transaction.
    /// @dev    It upgrades the L2ProxyAdmin to add the upgradePredeploys() function.
    /// @param _proxyAdminImpl Address of the new L2ProxyAdmin implementation.
    /// @dev TODO(#19369): Remove this function once Karst upgrade is deployed in all chains.
    function _generateL2ProxyAdminUpgrade(address _proxyAdminImpl) internal {
        txns.push(
            UpgradeUtils.createUpgradeTxn(
                "L2ProxyAdmin", Predeploys.PROXY_ADMIN, _proxyAdminImpl, gasLimits.proxyAdminUpgrade
            )
        );
    }

    // ========================================
    // FIXED NUT OPERATIONS
    // ========================================

    /// @notice Generates implementation deployment transactions for all the implementations to upgrade.
    /// @dev This function is called for all upgrades. It deploys implementation contracts
    ///      via ConditionalDeployer.deploy(), which ensures idempotent deployments.
    /// @dev IMPORTANT: Only modify this function if you need to add or modify a fixed implementation deployment.
    function _generateImplementationDeployments() internal {
        // Get all implementations to upgrade
        string[] memory implementationsToUpgrade = UpgradeUtils.getImplementationsNamesToUpgrade();

        for (uint256 i = 0; i < implementationsToUpgrade.length; i++) {
            // Get implementation config
            ImplementationConfig memory config = implementationConfigs[implementationsToUpgrade[i]];

            _assertValidImplementationConfig(config);

            txns.push(
                UpgradeUtils.createDeploymentTxn(config.name, config.artifactPath, SALT, config.deploymentGasLimit)
            );
        }
    }

    /// @notice Generates L2ContractsManager deployment transaction.
    /// @dev This function is called for all upgrades. The L2ContractsManager is deployed
    ///      with all implementation addresses encoded in its constructor.
    function _generateL2CMDeployment() internal {
        // Encode constructor arguments
        bytes memory l2cmArgs = abi.encode(implementations);

        // Deploy L2ContractsManager with encoded implementation addresses
        txns.push(
            UpgradeUtils.createDeploymentTxnWithArgs(
                "L2ContractsManager",
                "L2ContractsManager.sol:L2ContractsManager",
                l2cmArgs,
                SALT,
                gasLimits.l2cmDeployment
            )
        );
    }

    /// @notice Generates the final upgrade execution transaction.
    /// @dev This function is called for all upgrades. It creates the transaction that calls
    ///      L2ProxyAdmin.upgradePredeploys(l2cm), which executes a DELEGATECALL to the
    ///      L2ContractsManager.upgrade() function to perform the actual upgrades.
    function _generateUpgradeExecution() internal {
        // Encode constructor arguments
        bytes memory l2cmArgs = abi.encode(implementations);

        // Compute L2ContractsManager address
        address l2cm = UpgradeUtils.computeCreate2Address(
            abi.encodePacked(DeployUtils.getCode("L2ContractsManager.sol:L2ContractsManager"), l2cmArgs), SALT
        );

        // Create upgrade execution transaction
        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                intent: "L2ProxyAdmin Upgrade Predeploys",
                from: Constants.DEPOSITOR_ACCOUNT,
                to: Predeploys.PROXY_ADMIN,
                gasLimit: gasLimits.upgradeExecution,
                data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (l2cm))
            })
        );
    }

    // ========================================
    // HELPERS
    // ========================================

    /// @notice Retrieves all expected implementation addresses for the upgrade.
    /// @dev All addresses are looked up from the implementationConfigs mapping, which contains
    ///      deterministically computed CREATE2 addresses using the hardcoded salt. This ensures
    ///      identical addresses across all chains executing the upgrade.
    /// @return implementations_ Struct containing all implementation addresses.
    function _getImplementations()
        internal
        view
        returns (L2ContractsManagerTypes.Implementations memory implementations_)
    {
        implementations_ = L2ContractsManagerTypes.Implementations({
            storageSetterImpl: implementationConfigs["StorageSetter"].implementation,
            l2CrossDomainMessengerImpl: implementationConfigs["L2CrossDomainMessenger"].implementation,
            gasPriceOracleImpl: implementationConfigs["GasPriceOracle"].implementation,
            l2StandardBridgeImpl: implementationConfigs["L2StandardBridge"].implementation,
            sequencerFeeWalletImpl: implementationConfigs["SequencerFeeVault"].implementation,
            optimismMintableERC20FactoryImpl: implementationConfigs["OptimismMintableERC20Factory"].implementation,
            l2ERC721BridgeImpl: implementationConfigs["L2ERC721Bridge"].implementation,
            l1BlockImpl: implementationConfigs["L1Block"].implementation,
            l1BlockCGTImpl: implementationConfigs["L1BlockCGT"].implementation,
            l2ToL1MessagePasserImpl: implementationConfigs["L2ToL1MessagePasser"].implementation,
            l2ToL1MessagePasserCGTImpl: implementationConfigs["L2ToL1MessagePasserCGT"].implementation,
            optimismMintableERC721FactoryImpl: implementationConfigs["OptimismMintableERC721Factory"].implementation,
            proxyAdminImpl: implementationConfigs["L2ProxyAdmin"].implementation,
            baseFeeVaultImpl: implementationConfigs["BaseFeeVault"].implementation,
            l1FeeVaultImpl: implementationConfigs["L1FeeVault"].implementation,
            operatorFeeVaultImpl: implementationConfigs["OperatorFeeVault"].implementation,
            schemaRegistryImpl: implementationConfigs["SchemaRegistry"].implementation,
            easImpl: implementationConfigs["EAS"].implementation,
            crossL2InboxImpl: implementationConfigs["CrossL2Inbox"].implementation,
            l2ToL2CrossDomainMessengerImpl: implementationConfigs["L2ToL2CrossDomainMessenger"].implementation,
            superchainETHBridgeImpl: implementationConfigs["SuperchainETHBridge"].implementation,
            ethLiquidityImpl: implementationConfigs["ETHLiquidity"].implementation,
            nativeAssetLiquidityImpl: implementationConfigs["NativeAssetLiquidity"].implementation,
            liquidityControllerImpl: implementationConfigs["LiquidityController"].implementation,
            feeSplitterImpl: implementationConfigs["FeeSplitter"].implementation,
            conditionalDeployerImpl: implementationConfigs["ConditionalDeployer"].implementation,
            l2DevFeatureFlagsImpl: implementationConfigs["L2DevFeatureFlags"].implementation
        });
    }

    /// @notice Builds the implementation configuration mapping for all contracts to be deployed.
    /// @dev IMPORTANT: Only modify this function if you need to add or modify a deployment implementation
    /// configuration.
    /// @dev An array of strings is used to add contracts that are not predeploys (StorageSetter) or have
    /// feature-specific variants (e.g. CGT).
    /// @dev Gas limits are based on actual gas profiling of mainnet fork execution with 1.5x safety margin.
    function _buildImplementationDeploymentConfigs() internal {
        // Gas profiling: 280,600 gas used → 420,900 recommended → 500K with safety margin
        implementationConfigs["StorageSetter"] = ImplementationConfig({
            name: "StorageSetter",
            artifactPath: "StorageSetter.sol:StorageSetter",
            deploymentGasLimit: 500_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("StorageSetter.sol:StorageSetter"), SALT)
        });
        // Gas profiling: 1,708,099 gas used → 2,562,148 recommended → 2.6M with safety margin
        implementationConfigs["L2CrossDomainMessenger"] = ImplementationConfig({
            name: "L2CrossDomainMessenger",
            artifactPath: "L2CrossDomainMessenger.sol:L2CrossDomainMessenger",
            deploymentGasLimit: 2_600_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2CrossDomainMessenger.sol:L2CrossDomainMessenger"), SALT
            )
        });
        // Gas profiling: 1,681,024 gas used → 2,521,536 recommended → 2.6M with safety margin
        implementationConfigs["GasPriceOracle"] = ImplementationConfig({
            name: "GasPriceOracle",
            artifactPath: "GasPriceOracle.sol:GasPriceOracle",
            deploymentGasLimit: 2_600_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("GasPriceOracle.sol:GasPriceOracle"), SALT
            )
        });
        // Gas profiling: 2,358,092 gas used → 3,537,138 recommended → 3.6M with safety margin
        implementationConfigs["L2StandardBridge"] = ImplementationConfig({
            name: "L2StandardBridge",
            artifactPath: "L2StandardBridge.sol:L2StandardBridge",
            deploymentGasLimit: 3_600_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2StandardBridge.sol:L2StandardBridge"), SALT
            )
        });
        // Gas profiling: 841,152 gas used → 1,261,728 recommended → 1.3M with safety margin
        implementationConfigs["SequencerFeeVault"] = ImplementationConfig({
            name: "SequencerFeeVault",
            artifactPath: "SequencerFeeVault.sol:SequencerFeeVault",
            deploymentGasLimit: 1_300_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("SequencerFeeVault.sol:SequencerFeeVault"), SALT
            )
        });
        // Gas profiling: 2,347,504 gas used → 3,521,256 recommended → 3.6M with safety margin
        implementationConfigs["OptimismMintableERC20Factory"] = ImplementationConfig({
            name: "OptimismMintableERC20Factory",
            artifactPath: "OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory",
            deploymentGasLimit: 3_600_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory"), SALT
            )
        });
        // Gas profiling: 1,242,108 gas used → 1,863,162 recommended → 1.9M with safety margin
        implementationConfigs["L2ERC721Bridge"] = ImplementationConfig({
            name: "L2ERC721Bridge",
            artifactPath: "L2ERC721Bridge.sol:L2ERC721Bridge",
            deploymentGasLimit: 1_900_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2ERC721Bridge.sol:L2ERC721Bridge"), SALT
            )
        });
        // Gas profiling: 416,606 gas used → 624,909 recommended → 650K with safety margin
        implementationConfigs["L1Block"] = ImplementationConfig({
            name: "L1Block",
            artifactPath: "L1Block.sol:L1Block",
            deploymentGasLimit: 650_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("L1Block.sol:L1Block"), SALT)
        });
        // Gas profiling: 710,257 gas used → 1,065,385 recommended → 1.1M with safety margin
        implementationConfigs["L1BlockCGT"] = ImplementationConfig({
            name: "L1BlockCGT",
            artifactPath: "L1BlockCGT.sol:L1BlockCGT",
            deploymentGasLimit: 1_100_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("L1BlockCGT.sol:L1BlockCGT"), SALT)
        });
        // Gas profiling: 400,911 gas used → 601,366 recommended → 650K with safety margin
        implementationConfigs["L2ToL1MessagePasser"] = ImplementationConfig({
            name: "L2ToL1MessagePasser",
            artifactPath: "L2ToL1MessagePasser.sol:L2ToL1MessagePasser",
            deploymentGasLimit: 650_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2ToL1MessagePasser.sol:L2ToL1MessagePasser"), SALT
            )
        });
        // Gas profiling: 484,560 gas used → 726,840 recommended → 750K with safety margin
        implementationConfigs["L2ToL1MessagePasserCGT"] = ImplementationConfig({
            name: "L2ToL1MessagePasserCGT",
            artifactPath: "L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT",
            deploymentGasLimit: 750_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT"), SALT
            )
        });

        // Gas profiling: 3,248,395 gas used → 4,872,592 recommended → 4.9M with safety margin
        implementationConfigs["OptimismMintableERC721Factory"] = ImplementationConfig({
            name: "OptimismMintableERC721Factory",
            artifactPath: "OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory",
            deploymentGasLimit: 4_900_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory"), SALT
            )
        });
        // Gas profiling: 1,538,265 gas used → 2,307,397 recommended → 2.4M with safety margin
        implementationConfigs["L2ProxyAdmin"] = ImplementationConfig({
            name: "L2ProxyAdmin",
            artifactPath: "L2ProxyAdmin.sol:L2ProxyAdmin",
            deploymentGasLimit: 2_400_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("L2ProxyAdmin.sol:L2ProxyAdmin"), SALT)
        });
        // Gas profiling: 838,947 gas used → 1,258,420 recommended → 1.3M with safety margin
        implementationConfigs["BaseFeeVault"] = ImplementationConfig({
            name: "BaseFeeVault",
            artifactPath: "BaseFeeVault.sol:BaseFeeVault",
            deploymentGasLimit: 1_300_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("BaseFeeVault.sol:BaseFeeVault"), SALT)
        });
        // Gas profiling: 14,439 gas used → 21,658 recommended → 50K with safety margin
        implementationConfigs["L1FeeVault"] = ImplementationConfig({
            name: "L1FeeVault",
            artifactPath: "L1FeeVault.sol:L1FeeVault",
            deploymentGasLimit: 50_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("L1FeeVault.sol:L1FeeVault"), SALT)
        });
        // Gas profiling: 838,947 gas used → 1,258,420 recommended → 1.3M with safety margin
        implementationConfigs["OperatorFeeVault"] = ImplementationConfig({
            name: "OperatorFeeVault",
            artifactPath: "OperatorFeeVault.sol:OperatorFeeVault",
            deploymentGasLimit: 1_300_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("OperatorFeeVault.sol:OperatorFeeVault"), SALT
            )
        });
        // Gas profiling: 464,947 gas used → 697,420 recommended → 700K with safety margin
        implementationConfigs["SchemaRegistry"] = ImplementationConfig({
            name: "SchemaRegistry",
            artifactPath: "SchemaRegistry.sol:SchemaRegistry",
            deploymentGasLimit: 700_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("SchemaRegistry.sol:SchemaRegistry"), SALT
            )
        });
        // Gas profiling: 3,820,943 gas used → 5,731,414 recommended → 5.8M with safety margin
        implementationConfigs["EAS"] = ImplementationConfig({
            name: "EAS",
            artifactPath: "EAS.sol:EAS",
            deploymentGasLimit: 5_800_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("EAS.sol:EAS"), SALT)
        });
        // Gas profiling: 385,975 gas used → 578,962 recommended → 600K with safety margin
        implementationConfigs["CrossL2Inbox"] = ImplementationConfig({
            name: "CrossL2Inbox",
            artifactPath: "CrossL2Inbox.sol:CrossL2Inbox",
            deploymentGasLimit: 600_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("CrossL2Inbox.sol:CrossL2Inbox"), SALT)
        });
        // Gas profiling: 965,734 gas used → 1,448,601 recommended → 1.5M with safety margin
        implementationConfigs["L2ToL2CrossDomainMessenger"] = ImplementationConfig({
            name: "L2ToL2CrossDomainMessenger",
            artifactPath: "L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger",
            deploymentGasLimit: 1_500_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger"), SALT
            )
        });
        // Gas profiling: 441,198 gas used → 661,797 recommended → 700K with safety margin
        implementationConfigs["SuperchainETHBridge"] = ImplementationConfig({
            name: "SuperchainETHBridge",
            artifactPath: "SuperchainETHBridge.sol:SuperchainETHBridge",
            deploymentGasLimit: 700_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("SuperchainETHBridge.sol:SuperchainETHBridge"), SALT
            )
        });
        // Gas profiling: 230,857 gas used → 346,285 recommended → 400K with safety margin
        implementationConfigs["ETHLiquidity"] = ImplementationConfig({
            name: "ETHLiquidity",
            artifactPath: "ETHLiquidity.sol:ETHLiquidity",
            deploymentGasLimit: 400_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("ETHLiquidity.sol:ETHLiquidity"), SALT)
        });
        // Gas profiling: 215,592 gas used → 323,388 recommended → 400K with safety margin
        implementationConfigs["NativeAssetLiquidity"] = ImplementationConfig({
            name: "NativeAssetLiquidity",
            artifactPath: "NativeAssetLiquidity.sol:NativeAssetLiquidity",
            deploymentGasLimit: 400_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("NativeAssetLiquidity.sol:NativeAssetLiquidity"), SALT
            )
        });
        // Gas profiling: 914,648 gas used → 1,371,972 recommended → 1.4M with safety margin
        implementationConfigs["LiquidityController"] = ImplementationConfig({
            name: "LiquidityController",
            artifactPath: "LiquidityController.sol:LiquidityController",
            deploymentGasLimit: 1_400_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("LiquidityController.sol:LiquidityController"), SALT
            )
        });
        // Gas profiling: 1,077,380 gas used → 1,616,070 recommended → 1.7M with safety margin
        implementationConfigs["FeeSplitter"] = ImplementationConfig({
            name: "FeeSplitter",
            artifactPath: "FeeSplitter.sol:FeeSplitter",
            deploymentGasLimit: 1_700_000,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode("FeeSplitter.sol:FeeSplitter"), SALT)
        });
        // Gas profiling: 339,403 gas used → 509,104 recommended → 600K with safety margin
        implementationConfigs["ConditionalDeployer"] = ImplementationConfig({
            name: "ConditionalDeployer",
            artifactPath: "ConditionalDeployer.sol:ConditionalDeployer",
            deploymentGasLimit: 600_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("ConditionalDeployer.sol:ConditionalDeployer"), SALT
            )
        });
        implementationConfigs["L2DevFeatureFlags"] = ImplementationConfig({
            name: "L2DevFeatureFlags",
            artifactPath: "L2DevFeatureFlags.sol:L2DevFeatureFlags",
            deploymentGasLimit: 300_000,
            implementation: UpgradeUtils.computeCreate2Address(
                DeployUtils.getCode("L2DevFeatureFlags.sol:L2DevFeatureFlags"), SALT
            )
        });
    }
}
