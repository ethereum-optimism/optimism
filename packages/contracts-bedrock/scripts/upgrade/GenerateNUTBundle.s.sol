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

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

/// @title GenerateNUTBundle
/// @notice Generates Network Upgrade Transaction (NUT) bundles for L2 hardfork upgrades.
/// @dev This script creates deterministic upgrade transaction bundles for L2 hardfork upgrades
///      using the L2ContractsManager (L2CM) system.
contract GenerateNUTBundle is Script {
    /// @notice CREATE2 salt for deterministic deployments.
    /// TODO: Define standard format for salts.
    bytes32 internal constant SALT = bytes32(uint256(keccak256("optimism.network-upgrade")));

    /// @notice Name of the upgrade.
    string internal constant UPGRADE_NAME = "jovian";

    /// @notice Path to the upgrade artifact.
    string public constant UPGRADE_ARTIFACT_PATH = "deployments/nut-jovian-upgrade.json";

    /// @notice Input parameters for bundle generation.
    /// @param l1ChainID The L1 chain ID.
    struct Input {
        uint256 l1ChainID;
    }

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
        bytes args;
    }

    /// @notice Gas limits for the upgrade.
    UpgradeUtils.GasLimits internal gasLimits;

    /// @notice Expected implementations for the upgrade.
    L2ContractsManagerTypes.Implementations internal implementations;

    /// @notice Implementation configurations.
    mapping(string => ImplementationConfig) internal implementationConfigs;

    /// @notice Array of generated transactions.
    NetworkUpgradeTxns.NetworkUpgradeTxn[] internal txns;

    function setUp() public {
        _resetScript();
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
    /// @param _input Input parameters including l1ChainID.
    /// @return output_ Output containing all generated transactions in execution order.
    function run(Input memory _input) public returns (Output memory output_) {
        setUp();
        _assertValidInput(_input);

        // Build implementation deployment configurations
        _buildImplementationDeploymentConfigs(_input);

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
        // TODO: Uncomment once L2ContractsManager is merged and ready for deployment
        // _generateL2CMDeployment();

        // Phase 5: Upgrade execution
        // TODO: Uncomment once L2ContractsManager is merged and upgrade flow is finalized
        // _generateUpgradeExecution();

        // Copy storage array to memory array for return
        uint256 txnsLength = txns.length;
        output_.txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](txnsLength);
        for (uint256 i = 0; i < txnsLength; i++) {
            output_.txns[i] = txns[i];
        }

        _assertValidOutput(output_);

        // Write transactions to artifact
        NetworkUpgradeTxns.writeArtifact(txns, UPGRADE_ARTIFACT_PATH);
    }

    /// @notice Asserts the input is valid.
    /// @param _input The input to assert.
    function _assertValidInput(Input memory _input) internal pure {
        require(_input.l1ChainID != 0, "GenerateNUTBundle: l1ChainID cannot be zero");
    }

    /// @notice Asserts the output is valid.
    /// @param _output The output to assert.
    function _assertValidOutput(Output memory _output) internal pure {
        uint256 transactionCount = UpgradeUtils.getTransactionCount();
        // TODO: Remove -2 once L2CM deployment and upgrade execution phases are added
        uint256 txnsLength = _output.txns.length;
        require(txnsLength == transactionCount - 2, "GenerateNUTBundle: invalid transaction count");

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

    /// @notice Resets the script state.
    /// @dev This function is used to reset the script state before running the script.
    function _resetScript() internal {
        // Clear previous txns: Transactions are pushed to a dynamic array, so we need
        // to delete the array to avoid pushing duplicates.
        delete txns;
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
        // ConditionalDeployer deployment + upgrade
        _generateConditionalDeployerTxns();
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
        // L2ProxyAdmin upgrade
        _generateL2ProxyAdminUpgrade(implementations.proxyAdminImpl);
    }

    // ========================================
    // JOVIAN-ONLY NUTs
    // ========================================

    /// @notice Generates ConditionalDeployer deployment and upgrade transactions.
    function _generateConditionalDeployerTxns() internal {
        // 1. Deploy ConditionalDeployer implementation
        bytes memory conditionalDeployerCode =
            abi.encodePacked(vm.getCode("ConditionalDeployer.sol:ConditionalDeployer"));

        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                intent: string.concat(UPGRADE_NAME, ": ConditionalDeployer Deployment"),
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
                UPGRADE_NAME,
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
    function _generateL2ProxyAdminUpgrade(address _proxyAdminImpl) internal {
        txns.push(
            UpgradeUtils.createUpgradeTxn(
                UPGRADE_NAME, "L2ProxyAdmin", Predeploys.PROXY_ADMIN, _proxyAdminImpl, gasLimits.proxyAdminUpgrade
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

            if (config.args.length > 0) {
                // Deploy implementation with constructor arguments
                txns.push(
                    UpgradeUtils.createDeploymentTxnWithArgs(
                        UPGRADE_NAME, config.name, config.artifactPath, config.args, SALT, config.deploymentGasLimit
                    )
                );
            } else {
                txns.push(
                    UpgradeUtils.createDeploymentTxn(
                        UPGRADE_NAME, config.name, config.artifactPath, SALT, config.deploymentGasLimit
                    )
                );
            }
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
                UPGRADE_NAME,
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
            abi.encodePacked(vm.getCode("L2ContractsManager.sol:L2ContractsManager"), l2cmArgs), SALT
        );

        // Create upgrade execution transaction
        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                intent: string.concat(UPGRADE_NAME, ": L2ProxyAdmin Upgrade Predeploys"),
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
            optimismSuperchainERC20FactoryImpl: implementationConfigs["OptimismSuperchainERC20Factory"].implementation,
            optimismSuperchainERC20BeaconImpl: implementationConfigs["OptimismSuperchainERC20Beacon"].implementation,
            superchainTokenBridgeImpl: implementationConfigs["SuperchainTokenBridge"].implementation,
            nativeAssetLiquidityImpl: implementationConfigs["NativeAssetLiquidity"].implementation,
            liquidityControllerImpl: implementationConfigs["LiquidityController"].implementation,
            feeSplitterImpl: implementationConfigs["FeeSplitter"].implementation,
            conditionalDeployerImpl: implementationConfigs["ConditionalDeployer"].implementation
        });
    }

    /// @notice Builds the implementation configuration mapping for all contracts to be deployed.
    /// @dev IMPORTANT: Only modify this function if you need to add or modify a deployment implementation
    /// configuration.
    function _buildImplementationDeploymentConfigs(Input memory _input) internal {
        implementationConfigs["StorageSetter"] = ImplementationConfig({
            name: "StorageSetter",
            artifactPath: "StorageSetter.sol:StorageSetter",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("StorageSetter.sol:StorageSetter"), SALT)
        });
        implementationConfigs["L2CrossDomainMessenger"] = ImplementationConfig({
            name: "L2CrossDomainMessenger",
            artifactPath: "L2CrossDomainMessenger.sol:L2CrossDomainMessenger",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("L2CrossDomainMessenger.sol:L2CrossDomainMessenger"), SALT
            )
        });
        implementationConfigs["GasPriceOracle"] = ImplementationConfig({
            name: "GasPriceOracle",
            artifactPath: "GasPriceOracle.sol:GasPriceOracle",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("GasPriceOracle.sol:GasPriceOracle"), SALT)
        });
        implementationConfigs["L2StandardBridge"] = ImplementationConfig({
            name: "L2StandardBridge",
            artifactPath: "L2StandardBridge.sol:L2StandardBridge",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L2StandardBridge.sol:L2StandardBridge"), SALT)
        });
        implementationConfigs["SequencerFeeVault"] = ImplementationConfig({
            name: "SequencerFeeVault",
            artifactPath: "SequencerFeeVault.sol:SequencerFeeVault",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("SequencerFeeVault.sol:SequencerFeeVault"), SALT)
        });
        implementationConfigs["OptimismMintableERC20Factory"] = ImplementationConfig({
            name: "OptimismMintableERC20Factory",
            artifactPath: "OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory"), SALT
            )
        });
        implementationConfigs["L2ERC721Bridge"] = ImplementationConfig({
            name: "L2ERC721Bridge",
            artifactPath: "L2ERC721Bridge.sol:L2ERC721Bridge",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L2ERC721Bridge.sol:L2ERC721Bridge"), SALT)
        });
        implementationConfigs["L1Block"] = ImplementationConfig({
            name: "L1Block",
            artifactPath: "L1Block.sol:L1Block",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L1Block.sol:L1Block"), SALT)
        });
        implementationConfigs["L1BlockCGT"] = ImplementationConfig({
            name: "L1BlockCGT",
            artifactPath: "L1BlockCGT.sol:L1BlockCGT",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L1BlockCGT.sol:L1BlockCGT"), SALT)
        });
        implementationConfigs["L2ToL1MessagePasser"] = ImplementationConfig({
            name: "L2ToL1MessagePasser",
            artifactPath: "L2ToL1MessagePasser.sol:L2ToL1MessagePasser",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("L2ToL1MessagePasser.sol:L2ToL1MessagePasser"), SALT
            )
        });
        implementationConfigs["L2ToL1MessagePasserCGT"] = ImplementationConfig({
            name: "L2ToL1MessagePasserCGT",
            artifactPath: "L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT"), SALT
            )
        });

        implementationConfigs["OptimismMintableERC721Factory"] = ImplementationConfig({
            name: "OptimismMintableERC721Factory",
            artifactPath: "OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory",
            args: abi.encode(Predeploys.L2_ERC721_BRIDGE, _input.l1ChainID),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                abi.encodePacked(
                    vm.getCode("OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory"),
                    abi.encode(Predeploys.L2_ERC721_BRIDGE, _input.l1ChainID)
                ),
                SALT
            )
        });
        implementationConfigs["L2ProxyAdmin"] = ImplementationConfig({
            name: "L2ProxyAdmin",
            artifactPath: "L2ProxyAdmin.sol:L2ProxyAdmin",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L2ProxyAdmin.sol:L2ProxyAdmin"), SALT)
        });
        implementationConfigs["BaseFeeVault"] = ImplementationConfig({
            name: "BaseFeeVault",
            artifactPath: "BaseFeeVault.sol:BaseFeeVault",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("BaseFeeVault.sol:BaseFeeVault"), SALT)
        });
        implementationConfigs["L1FeeVault"] = ImplementationConfig({
            name: "L1FeeVault",
            artifactPath: "L1FeeVault.sol:L1FeeVault",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("L1FeeVault.sol:L1FeeVault"), SALT)
        });
        implementationConfigs["OperatorFeeVault"] = ImplementationConfig({
            name: "OperatorFeeVault",
            artifactPath: "OperatorFeeVault.sol:OperatorFeeVault",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("OperatorFeeVault.sol:OperatorFeeVault"), SALT)
        });
        implementationConfigs["SchemaRegistry"] = ImplementationConfig({
            name: "SchemaRegistry",
            artifactPath: "SchemaRegistry.sol:SchemaRegistry",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("SchemaRegistry.sol:SchemaRegistry"), SALT)
        });
        implementationConfigs["EAS"] = ImplementationConfig({
            name: "EAS",
            artifactPath: "EAS.sol:EAS",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("EAS.sol:EAS"), SALT)
        });
        implementationConfigs["CrossL2Inbox"] = ImplementationConfig({
            name: "CrossL2Inbox",
            artifactPath: "CrossL2Inbox.sol:CrossL2Inbox",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("CrossL2Inbox.sol:CrossL2Inbox"), SALT)
        });
        implementationConfigs["L2ToL2CrossDomainMessenger"] = ImplementationConfig({
            name: "L2ToL2CrossDomainMessenger",
            artifactPath: "L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger"), SALT
            )
        });
        implementationConfigs["SuperchainETHBridge"] = ImplementationConfig({
            name: "SuperchainETHBridge",
            artifactPath: "SuperchainETHBridge.sol:SuperchainETHBridge",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("SuperchainETHBridge.sol:SuperchainETHBridge"), SALT
            )
        });
        implementationConfigs["ETHLiquidity"] = ImplementationConfig({
            name: "ETHLiquidity",
            artifactPath: "ETHLiquidity.sol:ETHLiquidity",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("ETHLiquidity.sol:ETHLiquidity"), SALT)
        });
        implementationConfigs["OptimismSuperchainERC20Factory"] = ImplementationConfig({
            name: "OptimismSuperchainERC20Factory",
            artifactPath: "OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory"), SALT
            )
        });
        implementationConfigs["OptimismSuperchainERC20Beacon"] = ImplementationConfig({
            name: "OptimismSuperchainERC20Beacon",
            artifactPath: "OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon"), SALT
            )
        });
        implementationConfigs["SuperchainTokenBridge"] = ImplementationConfig({
            name: "SuperchainTokenBridge",
            artifactPath: "SuperchainTokenBridge.sol:SuperchainTokenBridge",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("SuperchainTokenBridge.sol:SuperchainTokenBridge"), SALT
            )
        });
        implementationConfigs["NativeAssetLiquidity"] = ImplementationConfig({
            name: "NativeAssetLiquidity",
            artifactPath: "NativeAssetLiquidity.sol:NativeAssetLiquidity",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("NativeAssetLiquidity.sol:NativeAssetLiquidity"), SALT
            )
        });
        implementationConfigs["LiquidityController"] = ImplementationConfig({
            name: "LiquidityController",
            artifactPath: "LiquidityController.sol:LiquidityController",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("LiquidityController.sol:LiquidityController"), SALT
            )
        });
        implementationConfigs["FeeSplitter"] = ImplementationConfig({
            name: "FeeSplitter",
            artifactPath: "FeeSplitter.sol:FeeSplitter",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(vm.getCode("FeeSplitter.sol:FeeSplitter"), SALT)
        });
        implementationConfigs["ConditionalDeployer"] = ImplementationConfig({
            name: "ConditionalDeployer",
            artifactPath: "ConditionalDeployer.sol:ConditionalDeployer",
            args: bytes(""),
            deploymentGasLimit: UpgradeUtils.DEFAULT_DEPLOYMENT_GAS,
            implementation: UpgradeUtils.computeCreate2Address(
                vm.getCode("ConditionalDeployer.sol:ConditionalDeployer"), SALT
            )
        });
    }
}
