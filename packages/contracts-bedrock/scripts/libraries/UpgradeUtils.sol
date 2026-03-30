// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Vm } from "forge-std/Vm.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";
import { Preinstalls } from "src/libraries/Preinstalls.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

// Contracts
import { ConditionalDeployer } from "src/L2/ConditionalDeployer.sol";

/// @title UpgradeUtils
/// @notice Utility library for L2 hardfork upgrade transaction generation.
library UpgradeUtils {
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

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

    /// @notice Returns the gas limits for all upgrade transaction types.
    /// @dev Gas limits are chosen to provide sufficient headroom while being
    ///      conservative enough to fit within the upgrade block gas allocation.
    ///      All values based on gas profiling of actual mainnet fork execution.
    ///      Rationale for each limit:
    ///      - l2cmDeployment: L2ContractsManager deployment measured at 2,824,780 gas
    ///        Recommended 4,237,170 (1.5x). Set to 4.5M for safety.
    ///      - upgradeExecution: L2ProxyAdmin.upgradePredeploys() measured at 1,602,448 gas
    ///        Recommended 2,403,672 (1.5x). Set to 3M for safety.
    ///      - conditionalDeployerDeployment: ConditionalDeployer deployment measured at 339,403 gas
    ///        Recommended 509,104 (1.5x). Set to 600K for safety.
    ///      - conditionalDeployerUpgrade: ConditionalDeployer upgrade measured at 29,169 gas
    ///        Recommended 43,753 (1.5x). Set to 50K for safety.
    ///      - proxyAdminUpgrade: ProxyAdmin upgrade measured at 12,069 gas
    ///        Recommended 18,103 (1.5x). Set to 50K for safety.
    /// @return Gas limits struct.
    function gasLimits() internal pure returns (GasLimits memory) {
        return GasLimits({
            // Fixed
            l2cmDeployment: 4_500_000,
            upgradeExecution: 3_000_000,
            // Karst
            conditionalDeployerDeployment: 600_000,
            conditionalDeployerUpgrade: 50_000,
            proxyAdminUpgrade: 50_000
        });
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

    /// @notice Extracts a revert reason from return data.
    /// @param _returnData The return data from a failed call.
    /// @return reason_ The revert reason string, or a default message if unavailable.
    function getRevertReason(bytes memory _returnData) internal pure returns (string memory reason_) {
        // If the return data is at least 68 bytes, it might contain a revert reason
        // 4 bytes for Error(string) selector + 32 bytes for offset + 32 bytes for length
        if (_returnData.length >= 68) {
            // Check if it's an Error(string) revert
            bytes4 errorSelector = bytes4(_returnData);
            if (errorSelector == 0x08c379a0) {
                // Decode the string
                assembly {
                    // Skip the first 68 bytes (4 byte selector + 32 byte offset + 32 byte length)
                    // to get to the actual string data
                    reason_ := add(_returnData, 0x44)
                }
                return reason_;
            }
        }

        // If we can't decode a revert reason, return hex representation
        if (_returnData.length > 0) {
            return string(abi.encodePacked(LibString.toHexString(_returnData)));
        }

        return "Unknown error";
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
