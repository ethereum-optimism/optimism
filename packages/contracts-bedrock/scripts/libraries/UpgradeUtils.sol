// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Vm } from "forge-std/Vm.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

// Contracts
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";

/// @title UpgradeUtils
/// @notice Utility library for L2 hardfork upgrade transaction generation.
library UpgradeUtils {
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice The number of implementations deployed in every upgrade.
    ///         Includes:
    ///         - 1 StorageSetter
    ///         - 16 base predeploys
    ///         - 7 INTEROP predeploys
    ///         - 2 CGT predeploys (NativeAssetLiquidity, LiquidityController)
    ///         - 2 CGT variants (L1BlockCGT, L2ToL1MessagePasserCGT)
    ///         Total: 28 implementations
    uint256 internal constant IMPLEMENTATION_COUNT = 28;

    /// @notice The default gas limit for a deployment transaction.
    uint64 internal constant DEFAULT_DEPLOYMENT_GAS = 375_000;

    /// @notice The default gas limit for an upgrade transaction.
    uint64 internal constant DEFAULT_UPGRADE_GAS = 50_000;

    /// @notice Gas limits for different types of upgrade transactions.
    /// @param l2cmDeployment Gas for deploying L2ContractsManager
    /// @param upgradeExecution Gas for L2ProxyAdmin.upgradePredeploys() call
    /// @param conditionalDeployerDeployment Gas for deploying ConditionalDeployer
    /// @param conditionalDeployerUpgrade Gas for upgrading ConditionalDeployer proxy
    /// @param proxyAdminUpgrade Gas for upgrading ProxyAdmin implementation
    struct GasLimits {
        // Fixed
        uint64 l2cmDeployment;
        uint64 upgradeExecution;
        // Karst
        uint64 conditionalDeployerDeployment;
        uint64 conditionalDeployerUpgrade;
        uint64 proxyAdminUpgrade;
    }

    /// @notice Returns the total number of transactions for the current upgrade.
    /// @dev Total count:
    ///      - 28 implementation deployments
    ///      - [KARST] 2 ConditionalDeployer (deployment + upgrade)
    ///      - [KARST] 1 ProxyAdmin upgrade
    ///      - 1 L2CM deployment
    ///      - 1 Upgrade Predeploys call
    function getTransactionCount() internal pure returns (uint256 txnCount_) {
        if (IMPLEMENTATION_COUNT != 28) {
            revert(
                "UpgradeUtils: implementation count changed, ensure that the txnCount_ calculation is still correct."
            );
        }
        txnCount_ = IMPLEMENTATION_COUNT + 5;
    }

    /// @notice Returns the gas limits for all upgrade transaction types.
    /// @dev Gas limits are chosen to provide sufficient headroom while being
    ///      conservative enough to fit within the upgrade block gas allocation.
    ///      Rationale for each limit:
    ///      - TODO: Add rationale here
    /// @return Gas limits struct.
    function gasLimits() internal pure returns (GasLimits memory) {
        return GasLimits({
            // Fixed
            l2cmDeployment: DEFAULT_DEPLOYMENT_GAS,
            upgradeExecution: type(uint64).max,
            // Karst
            conditionalDeployerDeployment: DEFAULT_DEPLOYMENT_GAS,
            conditionalDeployerUpgrade: DEFAULT_UPGRADE_GAS,
            proxyAdminUpgrade: DEFAULT_UPGRADE_GAS
        });
    }

    /// @notice Returns the array of predeploy names to upgrade.
    /// @dev Exception: StorageSetter is not a predeploy, but is upgraded in L2CM too.
    /// @return implementations_ Array of implementation names to upgrade.
    function getImplementationsNamesToUpgrade() internal pure returns (string[] memory implementations_) {
        implementations_ = new string[](IMPLEMENTATION_COUNT);

        // StorageSetter
        implementations_[0] = "StorageSetter";

        // Base predeploys
        implementations_[1] = "L2CrossDomainMessenger";
        implementations_[2] = "GasPriceOracle";
        implementations_[3] = "L2StandardBridge";
        implementations_[4] = "SequencerFeeVault";
        implementations_[5] = "OptimismMintableERC20Factory";
        implementations_[6] = "L2ERC721Bridge";
        implementations_[7] = "L1Block";
        implementations_[8] = "L2ToL1MessagePasser";
        implementations_[9] = "OptimismMintableERC721Factory";
        implementations_[10] = "L2ProxyAdmin";
        implementations_[11] = "BaseFeeVault";
        implementations_[12] = "L1FeeVault";
        implementations_[13] = "OperatorFeeVault";
        implementations_[14] = "SchemaRegistry";
        implementations_[15] = "EAS";
        implementations_[16] = "FeeSplitter";

        // INTEROP predeploys
        implementations_[17] = "CrossL2Inbox";
        implementations_[18] = "L2ToL2CrossDomainMessenger";
        implementations_[19] = "SuperchainETHBridge";
        implementations_[20] = "OptimismSuperchainERC20Factory";
        implementations_[21] = "OptimismSuperchainERC20Beacon";
        implementations_[22] = "SuperchainTokenBridge";
        implementations_[23] = "ETHLiquidity";

        // CGT predeploys
        implementations_[24] = "L1BlockCGT";
        implementations_[25] = "L2ToL1MessagePasserCGT";
        implementations_[26] = "LiquidityController";
        implementations_[27] = "NativeAssetLiquidity";
    }

    /// @notice Uses vm.computeCreate2Address to compute the CREATE2 address for given initcode and salt.
    /// @dev Uses the DeterministicDeploymentProxy address as the deployer.
    /// @param _code The contract initcode (creation bytecode).
    /// @param _salt The CREATE2 salt.
    /// @return expected_ The computed contract address.
    function computeCreate2Address(bytes memory _code, bytes32 _salt) internal pure returns (address expected_) {
        expected_ = vm.computeCreate2Address(_salt, keccak256(_code), Preinstalls.DeterministicDeploymentProxy);
    }

    /// @notice Creates a deployment transaction via ConditionalDeployer.
    /// @dev The transaction calls ConditionalDeployer.deploy(salt, code) which performs
    ///      idempotent CREATE2 deployment via the DeterministicDeploymentProxy.
    /// @param _name Human-readable name for the contract being deployed.
    /// @param _artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
    /// @param _salt CREATE2 salt for address computation.
    /// @param _gasLimit Gas limit for the deployment transaction.
    /// @return txn_ The constructed deployment transaction.
    function createDeploymentTxn(
        string memory _name,
        string memory _artifactPath,
        bytes32 _salt,
        uint64 _gasLimit
    )
        internal
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        return createDeploymentTxnWithArgs(_name, _artifactPath, "", _salt, _gasLimit);
    }

    /// @notice Creates a deployment transaction via ConditionalDeployer with constructor arguments.
    /// @dev The transaction calls ConditionalDeployer.deploy(salt, code) which performs
    ///      idempotent CREATE2 deployment via the DeterministicDeploymentProxy.
    /// @param _name Human-readable name for the contract being deployed.
    /// @param _artifactPath Forge artifact path (e.g., "MyContract.sol:MyContract").
    /// @param _args ABI-encoded constructor arguments.
    /// @param _salt CREATE2 salt for address computation.
    /// @param _gasLimit Gas limit for the deployment transaction.
    /// @return txn_ The constructed deployment transaction.
    function createDeploymentTxnWithArgs(
        string memory _name,
        string memory _artifactPath,
        bytes memory _args,
        bytes32 _salt,
        uint64 _gasLimit
    )
        internal
        view
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        bytes memory code = abi.encodePacked(DeployUtils.getCode(_artifactPath), _args);
        txn_ = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: string.concat("Deploy ", _name, " Implementation"),
            from: Constants.DEPOSITOR_ACCOUNT,
            to: Predeploys.CONDITIONAL_DEPLOYER,
            gasLimit: _gasLimit,
            data: abi.encodeCall(ConditionalDeployer.deploy, (_salt, code))
        });
    }

    /// @notice Creates an upgrade transaction for a proxy contract.
    /// @dev The transaction calls IProxy(proxy).upgradeTo(implementation).
    ///      For the ProxyAdmin upgrade, the sender must be address(0) to use the
    ///      zero-address upgrade path in the Proxy.sol implementation.
    /// @param _name Human-readable name for the contract being upgraded.
    /// @param _proxy Address of the proxy contract.
    /// @param _implementation Address of the new implementation.
    /// @param _gasLimit Gas limit for the upgrade transaction.
    /// @return txn_ The constructed upgrade transaction.
    function createUpgradeTxn(
        string memory _name,
        address _proxy,
        address _implementation,
        uint64 _gasLimit
    )
        internal
        pure
        returns (NetworkUpgradeTxns.NetworkUpgradeTxn memory txn_)
    {
        txn_ = NetworkUpgradeTxns.NetworkUpgradeTxn({
            intent: string.concat("Upgrade ", _name, " Implementation"),
            from: address(0),
            to: _proxy,
            gasLimit: _gasLimit,
            data: abi.encodeCall(IProxy.upgradeTo, (_implementation))
        });
    }
}
