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
import { Fork, ForkUtils } from "scripts/libraries/Config.sol";
import { UpgradeConfig } from "scripts/libraries/UpgradeConfig.sol";

// Interfaces
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

// Contracts
import { GenerateNUTBundleUtils } from "scripts/upgrade/GenerateNUTBundleUtils.s.sol";

/// @title GenerateNUTBundle
/// @notice Generates Network Upgrade Transaction (NUT) bundles for L2 hardfork upgrades.
/// @dev This script creates deterministic upgrade transaction bundles for L2 hardfork upgrades
///      using the L2ContractsManager (L2CM) system. The bundle structure varies based on fork.
contract GenerateNUTBundle is Script {
    /// @notice Input parameters for bundle generation.
    /// @param fork The hardfork this bundle is for (e.g., Fork.JOVIAN, Fork.KRYPTON).
    /// @param salt CREATE2 salt for deterministic address computation.
    struct Input {
        Fork fork;
        bytes32 salt;
        uint256 l1ChainID;
        bool useCustomGasToken;
    }

    /// @notice Output containing generated transactions.
    /// @param txns Array of Network Upgrade Transactions to execute.
    struct Output {
        NetworkUpgradeTxns.NetworkUpgradeTxn[] txns;
    }

    struct PredeployConfig {
        string name;
        string artifactPath;
        bytes args;
        uint64 deploymentGasLimit;
        address implementation;
    }

    /// @notice Current input parameters.
    Input internal input;

    /// @notice Gas limits for the upgrade.
    UpgradeConfig.GasLimits internal gasLimits;

    /// @notice Name of the upgrade.
    string internal upgradeName;

    /// @notice Expected implementations for the upgrade.
    L2ContractsManagerTypes.Implementations internal implementations;

    /// @notice Predeploy configurations.
    mapping(address => PredeployConfig) internal predeploysConfig;

    /// @notice Array of generated transactions.
    NetworkUpgradeTxns.NetworkUpgradeTxn[] internal txns;

    /// @notice BundleUtils contract instance.
    GenerateNUTBundleUtils internal bundleUtils;

    function setUp() public {
        gasLimits = UpgradeConfig.gasLimits();
    }

    /// @notice Generates the complete upgrade transaction bundle for the specified fork.
    /// @dev Executes 5 phases in fixed order:
    ///      1. Pre-implementation deployments [CUSTOM]
    ///      2. Implementation deployments [FIXED]
    ///      3. Pre-L2CM deployment [CUSTOM]
    ///      4. L2CM deployment [FIXED]
    ///      5. Upgrade execution [FIXED]
    /// @dev Only modify phases 1 and 3 for fork-specific logic. Other phases must remain unchanged.
    /// @param _input Input parameters including fork, salt, l1ChainID, and useCustomGasToken flag.
    /// @return output_ Output containing all generated transactions in execution order.
    function run(Input memory _input) public returns (Output memory output_) {
        setUp();
        _assertValidInput(_input);

        // Reset script state
        _resetScript();

        // Set input parameters
        input = _input;
        upgradeName = ForkUtils.toString(input.fork);

        bundleUtils = new GenerateNUTBundleUtils(input.fork, input.useCustomGasToken);
        _buildPredeployConfigs();

        // Phase 1: Pre-implementation deployments
        // Add fork-specific deployment or upgrade logic that must occur prior to the standard implementation deployment
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
        output_.txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](txns.length);
        for (uint256 i = 0; i < txns.length; i++) {
            output_.txns[i] = txns[i];
        }

        _assertValidOutput(output_);
        return output_;
    }

    /// @notice Asserts the input is valid.
    /// @param _input The input to assert.
    function _assertValidInput(Input memory _input) internal pure {
        require(_input.fork != Fork.NONE, "GenerateNUTBundle: invalid fork");
        require(_input.salt != bytes32(0), "GenerateNUTBundle: salt cannot be zero");
        require(_input.l1ChainID != 0, "GenerateNUTBundle: l1ChainID cannot be zero");
    }

    /// @notice Asserts the output is valid.
    /// @param _output The output to assert.
    function _assertValidOutput(Output memory _output) internal view {
        uint256 transactionCount = UpgradeConfig.calculateTransactionCount(input.fork, input.useCustomGasToken);
        // TODO: Remove -2 once L2CM deployment and upgrade execution phases are uncommented
        require(_output.txns.length == transactionCount - 2, "GenerateNUTBundle: invalid transaction count");

        for (uint256 i = 0; i < _output.txns.length; i++) {
            require(_output.txns[i].data.length > 0, "GenerateNUTBundle: invalid transaction data");
            // Note: from can be address(0) for certain upgrade transactions (e.g., ProxyAdmin upgrade)
            require(_output.txns[i].to != address(0), "GenerateNUTBundle: invalid transaction to");
            require(_output.txns[i].gas > 0, "GenerateNUTBundle: invalid transaction gas");
            require(
                _output.txns[i].isSystemTransaction == false,
                "GenerateNUTBundle: invalid transaction isSystemTransaction"
            );
        }
    }

    /// @notice Resets the script state.
    /// @dev This function is used to reset the script state before running the script.
    function _resetScript() internal {
        // Clear previous txns: Transactions are pushed to a dynamic array, so we need
        // to delete the array to avoid pushing duplicates.
        delete txns;
    }

    // ========================================
    // CUSTOM NUT OPERATIONS
    // ========================================

    /// @notice Pre-implementation deployment phase for fork-specific setup.
    /// @dev This function executes BEFORE any predeploy implementations are deployed. It is the
    ///      designated location for adding fork-specific deployment or upgrade logic that must
    ///      occur prior to the standard implementation deployment phase. The rest of the script
    ///      follows a fixed structure and should not be modified. Add new fork-specific logic
    ///      here by checking the input.fork value and calling the appropriate helper functions.
    /// @dev IMPORTANT: This is one of only TWO extension points in this script. Do not modify
    ///      the core deployment flow in _generateImplementationDeployments or other fixed phases.
    function _preImplementationDeployments() internal {
        if (input.fork == Fork.JOVIAN) {
            // ConditionalDeployer deployment + upgrade
            _generateConditionalDeployerTxns();
        }
    }

    /// @notice Pre-L2CM deployment phase for fork-specific setup.
    /// @dev This function executes AFTER implementations are deployed but BEFORE the L2ContractsManager
    ///      is deployed. It is the designated location for adding fork-specific deployment or upgrade
    ///      logic that must occur between these two phases. The rest of the script follows a fixed
    ///      structure and should not be modified. Add new fork-specific logic here by checking the
    ///      input.fork value and calling the appropriate helper functions.
    /// @dev IMPORTANT: This is one of only TWO extension points in this script. Do not modify
    ///      the core deployment flow in _generateL2CMDeployment, _generateUpgradeExecution, or other
    ///      fixed phases.
    function _preL2CMDeployment() internal {
        if (input.fork == Fork.JOVIAN) {
            // ProxyAdmin upgrade
            _generateProxyAdminUpgrade(implementations.proxyAdminImpl);
        }
    }

    // ========================================
    // JOVIAN-ONLY NUTs
    // ========================================

    /// @notice Generates ConditionalDeployer deployment and upgrade transactions (Jovian only).
    function _generateConditionalDeployerTxns() internal {
        bytes32 salt = input.salt;

        // 1. Deploy ConditionalDeployer implementation
        bytes memory conditionalDeployerCode =
            abi.encodePacked(vm.getCode("ConditionalDeployer.sol:ConditionalDeployer"));

        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                sourceHash: NetworkUpgradeTxns.sourceHash(string.concat(upgradeName, ": ConditionalDeployer Deployment")),
                from: Constants.DEPOSITOR_ACCOUNT,
                to: Preinstalls.DeterministicDeploymentProxy,
                mint: 0,
                value: 0,
                gas: gasLimits.conditionalDeployerDeployment,
                isSystemTransaction: false,
                data: abi.encodePacked(salt, conditionalDeployerCode)
            })
        );

        // 2. Upgrade ConditionalDeployer proxy
        address newConditionalDeployerImpl = bundleUtils.computeCreate2Address(conditionalDeployerCode, salt);
        txns.push(
            bundleUtils.createUpgradeTxn(
                upgradeName,
                "ConditionalDeployer",
                Predeploys.CONDITIONAL_DEPLOYER,
                newConditionalDeployerImpl,
                gasLimits.conditionalDeployerUpgrade
            )
        );
    }

    /// @notice Generates ProxyAdmin upgrade transaction.
    /// @dev    It upgrades the L2ProxyAdmin to add the upgradePredeploys() function.
    /// @param _proxyAdminImpl Address of the new ProxyAdmin implementation.
    function _generateProxyAdminUpgrade(address _proxyAdminImpl) internal {
        txns.push(
            bundleUtils.createUpgradeTxn(
                upgradeName, "ProxyAdmin", Predeploys.PROXY_ADMIN, _proxyAdminImpl, gasLimits.proxyAdminUpgrade
            )
        );
    }

    // ========================================
    // FIXED NUT OPERATIONS
    // ========================================

    /// @notice Generates implementation deployment transactions for all predeploys.
    /// @dev This function is called for all upgrades. It deploys implementation contracts
    ///      via ConditionalDeployer.deploy(), which ensures idempotent deployments.
    function _generateImplementationDeployments() internal {
        // Deploy StorageSetter first (not a predeploy, but needed for L2CM)
        txns.push(
            bundleUtils.createDeploymentTxn(
                upgradeName,
                "StorageSetter",
                "StorageSetter.sol:StorageSetter",
                input.salt,
                gasLimits.storageSetterDeployment
            )
        );

        // Deploy all predeploys
        address[] memory predeploysToUpgrade = bundleUtils.getPredeploysToUpgrade();

        for (uint256 i = 0; i < predeploysToUpgrade.length; i++) {
            // Get predeploy config
            PredeployConfig memory config = predeploysConfig[predeploysToUpgrade[i]];

            if (config.args.length > 0) {
                // Deploy predeploy with constructor arguments
                txns.push(
                    bundleUtils.createDeploymentTxnWithArgs(
                        upgradeName,
                        config.name,
                        config.artifactPath,
                        config.args,
                        input.salt,
                        config.deploymentGasLimit
                    )
                );
            } else {
                txns.push(
                    bundleUtils.createDeploymentTxn(
                        upgradeName, config.name, config.artifactPath, input.salt, config.deploymentGasLimit
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
            bundleUtils.createDeploymentTxnWithArgs(
                upgradeName,
                "L2ContractsManager",
                "L2ContractsManager.sol:L2ContractsManager",
                l2cmArgs,
                input.salt,
                gasLimits.l2cmDeployment
            )
        );
    }

    /// @notice Generates the final upgrade execution transaction.
    /// @dev This function is called for all upgrades. It creates the transaction that calls
    ///      L2ProxyAdmin.upgradePredeploys(l2cm), which executes a DELEGATECALL to the
    ///      L2ContractsManager.upgrade() function to perform the actual upgrades.
    function _generateUpgradeExecution() internal {
        bytes32 salt = input.salt;

        // Encode constructor arguments
        bytes memory l2cmArgs = abi.encode(implementations);

        // Compute L2ContractsManager address
        address l2cm = bundleUtils.computeCreate2Address(
            abi.encodePacked(vm.getCode("L2ContractsManager.sol:L2ContractsManager"), l2cmArgs), salt
        );

        // Create upgrade execution transaction
        txns.push(
            NetworkUpgradeTxns.NetworkUpgradeTxn({
                sourceHash: NetworkUpgradeTxns.sourceHash(string.concat(upgradeName, ": L2ProxyAdmin Upgrade Predeploys")),
                from: Constants.DEPOSITOR_ACCOUNT,
                to: Predeploys.PROXY_ADMIN,
                mint: 0,
                value: 0,
                gas: gasLimits.upgradeExecution,
                isSystemTransaction: false,
                data: abi.encodeCall(IL2ProxyAdmin.upgradePredeploys, (l2cm))
            })
        );
    }

    // ========================================
    // HELPERS
    // ========================================

    /// @notice Computes all expected implementation addresses for the upgrade.
    /// @dev All addresses are deterministically computed using CREATE2 with the provided salt.
    ///      This ensures identical addresses across all chains executing the upgrade.
    /// @return implementations_ Struct containing all implementation addresses.
    function _getImplementations()
        internal
        view
        returns (L2ContractsManagerTypes.Implementations memory implementations_)
    {
        implementations_ = L2ContractsManagerTypes.Implementations({
            storageSetterImpl: bundleUtils.computeCreate2Address(vm.getCode("StorageSetter.sol:StorageSetter"), input.salt),
            l2CrossDomainMessengerImpl: predeploysConfig[Predeploys.L2_CROSS_DOMAIN_MESSENGER].implementation,
            gasPriceOracleImpl: predeploysConfig[Predeploys.GAS_PRICE_ORACLE].implementation,
            l2StandardBridgeImpl: predeploysConfig[Predeploys.L2_STANDARD_BRIDGE].implementation,
            sequencerFeeWalletImpl: predeploysConfig[Predeploys.SEQUENCER_FEE_WALLET].implementation,
            optimismMintableERC20FactoryImpl: predeploysConfig[Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY].implementation,
            l2ERC721BridgeImpl: predeploysConfig[Predeploys.L2_ERC721_BRIDGE].implementation,
            l1BlockImpl: predeploysConfig[Predeploys.L1_BLOCK_ATTRIBUTES].implementation,
            l1BlockCGTImpl: input.useCustomGasToken
                ? predeploysConfig[Predeploys.L1_BLOCK_ATTRIBUTES].implementation
                : address(0),
            l2ToL1MessagePasserImpl: predeploysConfig[Predeploys.L2_TO_L1_MESSAGE_PASSER].implementation,
            l2ToL1MessagePasserCGTImpl: input.useCustomGasToken
                ? predeploysConfig[Predeploys.L2_TO_L1_MESSAGE_PASSER].implementation
                : address(0),
            optimismMintableERC721FactoryImpl: predeploysConfig[Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY].implementation,
            proxyAdminImpl: predeploysConfig[Predeploys.PROXY_ADMIN].implementation,
            baseFeeVaultImpl: predeploysConfig[Predeploys.BASE_FEE_VAULT].implementation,
            l1FeeVaultImpl: predeploysConfig[Predeploys.L1_FEE_VAULT].implementation,
            operatorFeeVaultImpl: predeploysConfig[Predeploys.OPERATOR_FEE_VAULT].implementation,
            schemaRegistryImpl: predeploysConfig[Predeploys.SCHEMA_REGISTRY].implementation,
            easImpl: predeploysConfig[Predeploys.EAS].implementation,
            crossL2InboxImpl: predeploysConfig[Predeploys.CROSS_L2_INBOX].implementation,
            l2ToL2CrossDomainMessengerImpl: predeploysConfig[Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER].implementation,
            superchainETHBridgeImpl: predeploysConfig[Predeploys.SUPERCHAIN_ETH_BRIDGE].implementation,
            ethLiquidityImpl: predeploysConfig[Predeploys.ETH_LIQUIDITY].implementation,
            optimismSuperchainERC20FactoryImpl: predeploysConfig[Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY]
                .implementation,
            optimismSuperchainERC20BeaconImpl: predeploysConfig[Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON].implementation,
            superchainTokenBridgeImpl: predeploysConfig[Predeploys.SUPERCHAIN_TOKEN_BRIDGE].implementation,
            nativeAssetLiquidityImpl: predeploysConfig[Predeploys.NATIVE_ASSET_LIQUIDITY].implementation,
            liquidityControllerImpl: predeploysConfig[Predeploys.LIQUIDITY_CONTROLLER].implementation,
            feeSplitterImpl: predeploysConfig[Predeploys.FEE_SPLITTER].implementation
        });
    }

    function _buildPredeployConfigs() internal {
        predeploysConfig[Predeploys.L2_CROSS_DOMAIN_MESSENGER] = PredeployConfig({
            name: "L2CrossDomainMessenger",
            artifactPath: "L2CrossDomainMessenger.sol:L2CrossDomainMessenger",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("L2CrossDomainMessenger.sol:L2CrossDomainMessenger"), input.salt
            )
        });
        predeploysConfig[Predeploys.GAS_PRICE_ORACLE] = PredeployConfig({
            name: "GasPriceOracle",
            artifactPath: "GasPriceOracle.sol:GasPriceOracle",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("GasPriceOracle.sol:GasPriceOracle"), input.salt)
        });
        predeploysConfig[Predeploys.L2_STANDARD_BRIDGE] = PredeployConfig({
            name: "L2StandardBridge",
            artifactPath: "L2StandardBridge.sol:L2StandardBridge",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("L2StandardBridge.sol:L2StandardBridge"), input.salt
            )
        });
        predeploysConfig[Predeploys.SEQUENCER_FEE_WALLET] = PredeployConfig({
            name: "SequencerFeeVault",
            artifactPath: "SequencerFeeVault.sol:SequencerFeeVault",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("SequencerFeeVault.sol:SequencerFeeVault"), input.salt
            )
        });
        predeploysConfig[Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY] = PredeployConfig({
            name: "OptimismMintableERC20Factory",
            artifactPath: "OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory"), input.salt
            )
        });
        predeploysConfig[Predeploys.L2_ERC721_BRIDGE] = PredeployConfig({
            name: "L2ERC721Bridge",
            artifactPath: "L2ERC721Bridge.sol:L2ERC721Bridge",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("L2ERC721Bridge.sol:L2ERC721Bridge"), input.salt)
        });
        predeploysConfig[Predeploys.L1_BLOCK_ATTRIBUTES] = PredeployConfig({
            name: "L1Block",
            artifactPath: input.useCustomGasToken ? "L1BlockCGT.sol:L1BlockCGT" : "L1Block.sol:L1Block",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode(input.useCustomGasToken ? "L1BlockCGT.sol:L1BlockCGT" : "L1Block.sol:L1Block"), input.salt
            )
        });
        predeploysConfig[Predeploys.L2_TO_L1_MESSAGE_PASSER] = PredeployConfig({
            name: "L2ToL1MessagePasser",
            artifactPath: input.useCustomGasToken
                ? "L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT"
                : "L2ToL1MessagePasser.sol:L2ToL1MessagePasser",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode(
                    input.useCustomGasToken
                        ? "L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT"
                        : "L2ToL1MessagePasser.sol:L2ToL1MessagePasser"
                ),
                input.salt
            )
        });
        predeploysConfig[Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY] = PredeployConfig({
            name: "OptimismMintableERC721Factory",
            artifactPath: "OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory",
            args: abi.encode(Predeploys.L2_ERC721_BRIDGE, input.l1ChainID),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                abi.encodePacked(
                    vm.getCode("OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory"),
                    abi.encode(Predeploys.L2_ERC721_BRIDGE, input.l1ChainID)
                ),
                input.salt
            )
        });
        predeploysConfig[Predeploys.PROXY_ADMIN] = PredeployConfig({
            name: "ProxyAdmin",
            artifactPath: "ProxyAdmin.sol:ProxyAdmin",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("ProxyAdmin.sol:ProxyAdmin"), input.salt)
        });
        predeploysConfig[Predeploys.BASE_FEE_VAULT] = PredeployConfig({
            name: "BaseFeeVault",
            artifactPath: "BaseFeeVault.sol:BaseFeeVault",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("BaseFeeVault.sol:BaseFeeVault"), input.salt)
        });
        predeploysConfig[Predeploys.L1_FEE_VAULT] = PredeployConfig({
            name: "L1FeeVault",
            artifactPath: "L1FeeVault.sol:L1FeeVault",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("L1FeeVault.sol:L1FeeVault"), input.salt)
        });
        predeploysConfig[Predeploys.OPERATOR_FEE_VAULT] = PredeployConfig({
            name: "OperatorFeeVault",
            artifactPath: "OperatorFeeVault.sol:OperatorFeeVault",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("OperatorFeeVault.sol:OperatorFeeVault"), input.salt
            )
        });
        predeploysConfig[Predeploys.SCHEMA_REGISTRY] = PredeployConfig({
            name: "SchemaRegistry",
            artifactPath: "SchemaRegistry.sol:SchemaRegistry",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("SchemaRegistry.sol:SchemaRegistry"), input.salt)
        });
        predeploysConfig[Predeploys.EAS] = PredeployConfig({
            name: "EAS",
            artifactPath: "EAS.sol:EAS",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("EAS.sol:EAS"), input.salt)
        });
        predeploysConfig[Predeploys.CROSS_L2_INBOX] = PredeployConfig({
            name: "CrossL2Inbox",
            artifactPath: "CrossL2Inbox.sol:CrossL2Inbox",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: input.fork >= Fork.INTEROP
                ? bundleUtils.computeCreate2Address(vm.getCode("CrossL2Inbox.sol:CrossL2Inbox"), input.salt)
                : address(0)
        });
        predeploysConfig[Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER] = PredeployConfig({
            name: "L2ToL2CrossDomainMessenger",
            artifactPath: "L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: input.fork >= Fork.INTEROP
                ? bundleUtils.computeCreate2Address(
                    vm.getCode("L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger"), input.salt
                )
                : address(0)
        });
        predeploysConfig[Predeploys.SUPERCHAIN_ETH_BRIDGE] = PredeployConfig({
            name: "SuperchainETHBridge",
            artifactPath: "SuperchainETHBridge.sol:SuperchainETHBridge",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("SuperchainETHBridge.sol:SuperchainETHBridge"), input.salt
            )
        });
        predeploysConfig[Predeploys.ETH_LIQUIDITY] = PredeployConfig({
            name: "ETHLiquidity",
            artifactPath: "ETHLiquidity.sol:ETHLiquidity",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("ETHLiquidity.sol:ETHLiquidity"), input.salt)
        });
        predeploysConfig[Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY] = PredeployConfig({
            name: "OptimismSuperchainERC20Factory",
            artifactPath: "OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory"), input.salt
            )
        });
        predeploysConfig[Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON] = PredeployConfig({
            name: "OptimismSuperchainERC20Beacon",
            artifactPath: "OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("OptimismSuperchainERC20Beacon.sol:OptimismSuperchainERC20Beacon"), input.salt
            )
        });
        predeploysConfig[Predeploys.SUPERCHAIN_TOKEN_BRIDGE] = PredeployConfig({
            name: "SuperchainTokenBridge",
            artifactPath: "SuperchainTokenBridge.sol:SuperchainTokenBridge",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(
                vm.getCode("SuperchainTokenBridge.sol:SuperchainTokenBridge"), input.salt
            )
        });
        predeploysConfig[Predeploys.NATIVE_ASSET_LIQUIDITY] = PredeployConfig({
            name: "NativeAssetLiquidity",
            artifactPath: "NativeAssetLiquidity.sol:NativeAssetLiquidity",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: input.useCustomGasToken
                ? bundleUtils.computeCreate2Address(vm.getCode("NativeAssetLiquidity.sol:NativeAssetLiquidity"), input.salt)
                : address(0)
        });
        predeploysConfig[Predeploys.LIQUIDITY_CONTROLLER] = PredeployConfig({
            name: "LiquidityController",
            artifactPath: "LiquidityController.sol:LiquidityController",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: input.useCustomGasToken
                ? bundleUtils.computeCreate2Address(vm.getCode("LiquidityController.sol:LiquidityController"), input.salt)
                : address(0)
        });
        predeploysConfig[Predeploys.FEE_SPLITTER] = PredeployConfig({
            name: "FeeSplitter",
            artifactPath: "FeeSplitter.sol:FeeSplitter",
            args: bytes(""),
            deploymentGasLimit: 375_000,
            implementation: bundleUtils.computeCreate2Address(vm.getCode("FeeSplitter.sol:FeeSplitter"), input.salt)
        });
    }
}
