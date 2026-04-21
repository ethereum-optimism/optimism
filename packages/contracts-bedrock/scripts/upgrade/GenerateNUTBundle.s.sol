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
    /// @param deploymentGasLimit Gas limit for the deployment transaction.
    /// @param implementation Expected implementation address after deployment.
    /// @param name Human-readable name for the contract.
    /// @param artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
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

    /// @notice Ordered list of all implementation configurations.
    /// @dev This is the single source of truth for what implementations exist and which ones
    ///      are deployed.
    ImplementationConfig[] internal _implementationConfigs;

    /// @notice Array of generated transactions.
    NetworkUpgradeTxns.NetworkUpgradeTxn[] internal _txns;

    function setUp() public {
        // Clear previous state: dynamic arrays must be deleted to avoid pushing duplicates
        // across multiple setUp/run calls.
        delete _txns;
        delete _implementationConfigs;

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
        uint256 txnsLength = _txns.length;
        output_.txns = new NetworkUpgradeTxns.NetworkUpgradeTxn[](txnsLength);
        for (uint256 i = 0; i < txnsLength; i++) {
            output_.txns[i] = _txns[i];
        }

        _assertValidOutput(output_);

        // Write transactions to artifact with metadata
        NetworkUpgradeTxns.BundleMetadata memory metadata =
            NetworkUpgradeTxns.BundleMetadata({ version: BUNDLE_VERSION });
        NetworkUpgradeTxns.writeArtifact(_txns, metadata, Constants.CURRENT_BUNDLE_PATH);
    }

    /// @notice Returns the names of implementations scheduled for standard deployment.
    function getStandardDeploymentNames() public view returns (string[] memory names_) {
        uint256 count = _implementationConfigs.length;
        names_ = new string[](count);
        for (uint256 i = 0; i < count; i++) {
            names_[i] = _implementationConfigs[i].name;
        }
    }

    /// @notice Asserts the output is valid.
    /// @param _output The output to assert.
    function _assertValidOutput(Output memory _output) internal view {
        // Total expected transactions:
        //   - standard implementation deployments
        //   - [KARST] ConditionalDeployer deployment + ConditionalDeployer upgrade + L2ProxyAdmin upgrade
        //   - L2ContractsManager deployment
        //   - L2ProxyAdmin.upgradePredeploys() call
        uint256 transactionCount = _implementationConfigs.length + 2;
        if (keccak256(abi.encodePacked(UPGRADE_NAME)) == keccak256(abi.encodePacked("karst"))) {
            transactionCount += 3;
        }
        require(_output.txns.length == transactionCount, "GenerateNUTBundle: invalid transaction count");

        for (uint256 i = 0; i < transactionCount; i++) {
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

        _txns.push(
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
        _txns.push(
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
        _txns.push(
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
        for (uint256 i = 0; i < _implementationConfigs.length; i++) {
            ImplementationConfig memory config = _implementationConfigs[i];

            _assertValidImplementationConfig(config);

            _txns.push(
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
        _txns.push(
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
        _txns.push(
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

    /// @notice Looks up an implementation address by contract name.
    /// @dev Iterates implementationConfigs linearly; acceptable cost for a script.
    /// @param _name The human-readable name used in _buildImplementationDeploymentConfigs.
    /// @return impl_ The implementation address, or reverts if not found.
    function findImplByName(string memory _name) public view returns (address impl_) {
        for (uint256 i = 0; i < _implementationConfigs.length; i++) {
            if (keccak256(bytes(_implementationConfigs[i].name)) == keccak256(bytes(_name))) {
                return _implementationConfigs[i].implementation;
            }
        }
        revert(string.concat("GenerateNUTBundle: implementation not found: ", _name));
    }

    /// @notice Retrieves all expected implementation addresses for the upgrade.
    /// @dev All addresses are looked up from the implementationConfigs array, which contains
    ///      deterministically computed CREATE2 addresses using the hardcoded salt. This ensures
    ///      identical addresses across all chains executing the upgrade.
    /// @return implementations_ Struct containing all implementation addresses.
    function _getImplementations()
        internal
        view
        returns (L2ContractsManagerTypes.Implementations memory implementations_)
    {
        implementations_ = L2ContractsManagerTypes.Implementations({
            storageSetterImpl: findImplByName("StorageSetter"),
            l2CrossDomainMessengerImpl: findImplByName("L2CrossDomainMessenger"),
            gasPriceOracleImpl: findImplByName("GasPriceOracle"),
            l2StandardBridgeImpl: findImplByName("L2StandardBridge"),
            sequencerFeeWalletImpl: findImplByName("SequencerFeeVault"),
            optimismMintableERC20FactoryImpl: findImplByName("OptimismMintableERC20Factory"),
            l2ERC721BridgeImpl: findImplByName("L2ERC721Bridge"),
            l1BlockImpl: findImplByName("L1Block"),
            l1BlockCGTImpl: findImplByName("L1BlockCGT"),
            l2ToL1MessagePasserImpl: findImplByName("L2ToL1MessagePasser"),
            l2ToL1MessagePasserCGTImpl: findImplByName("L2ToL1MessagePasserCGT"),
            optimismMintableERC721FactoryImpl: findImplByName("OptimismMintableERC721Factory"),
            proxyAdminImpl: findImplByName("L2ProxyAdmin"),
            baseFeeVaultImpl: findImplByName("BaseFeeVault"),
            l1FeeVaultImpl: findImplByName("L1FeeVault"),
            operatorFeeVaultImpl: findImplByName("OperatorFeeVault"),
            schemaRegistryImpl: findImplByName("SchemaRegistry"),
            easImpl: findImplByName("EAS"),
            crossL2InboxImpl: findImplByName("CrossL2Inbox"),
            l2ToL2CrossDomainMessengerImpl: findImplByName("L2ToL2CrossDomainMessenger"),
            superchainETHBridgeImpl: findImplByName("SuperchainETHBridge"),
            ethLiquidityImpl: findImplByName("ETHLiquidity"),
            nativeAssetLiquidityImpl: findImplByName("NativeAssetLiquidity"),
            liquidityControllerImpl: findImplByName("LiquidityController"),
            conditionalDeployerImpl: findImplByName("ConditionalDeployer"),
            l2DevFeatureFlagsImpl: findImplByName("L2DevFeatureFlags")
        });
    }

    /// @notice Builds the implementation configurations for all contracts to be deployed.
    /// @dev Iterates the predeploy registry as the single source of truth.
    ///      All records are deployed unconditionally. L2CM selects the correct variant at runtime.
    ///      StorageSetter is prepended first; it is a utility impl, not a predeploy.
    function _buildImplementationDeploymentConfigs() public {
        _implementationConfigs.push(_makeConfig("StorageSetter", "StorageSetter.sol:StorageSetter", 500_000));

        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();
        for (uint256 i = 0; i < records.length; i++) {
            _implementationConfigs.push(
                _makeConfig(records[i].name, records[i].artifactPath, records[i].deployGasLimit)
            );
        }
    }

    /// @notice Builds a single ImplementationConfig from name, artifact path, and gas limit.
    /// @param _name The name of the implementation.
    /// @param _artifactPath The artifact path of the implementation.
    /// @param _gasLimit The gas limit for the implementation deployment.
    /// @return config_ The implementation configuration.
    function _makeConfig(
        string memory _name,
        string memory _artifactPath,
        uint64 _gasLimit
    )
        internal
        view
        returns (ImplementationConfig memory config_)
    {
        config_ = ImplementationConfig({
            name: _name,
            artifactPath: _artifactPath,
            deploymentGasLimit: _gasLimit,
            implementation: UpgradeUtils.computeCreate2Address(DeployUtils.getCode(_artifactPath), SALT)
        });
    }

    /// @notice Returns the implementation configurations.
    /// @dev This function is used to get the implementation configurations for testing.
    /// @return implementationConfigs_ The implementation configurations.
    function implementationConfigs() public view returns (ImplementationConfig[] memory implementationConfigs_) {
        implementationConfigs_ = _implementationConfigs;
    }
}
