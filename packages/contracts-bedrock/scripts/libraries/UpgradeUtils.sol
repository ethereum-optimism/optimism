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
    struct GasLimits {
        uint64 l2cmDeployment;
        uint64 upgradeExecution;
    }

    /// @notice Returns the gas limits for all upgrade transaction types.
    /// @dev Calibration: see `_buildImplementationDeploymentConfigs` in GenerateNUTBundle.s.sol.
    /// @return Gas limits struct.
    function gasLimits() internal pure returns (GasLimits memory) {
        return GasLimits({ l2cmDeployment: 8_700_000, upgradeExecution: 3_500_000 });
    }

    /// @notice Computes the intrinsic gas cost for a NUT bundle transaction.
    /// @dev Replicates op-geth's IntrinsicGas formula (core/state_transition.go) for
    ///      non-contract-creation transactions post-EIP-2028:
    ///      21,000 base + 16 gas per non-zero byte + 4 gas per zero byte.
    ///      NUT bundle entries are never contract creations (all have a non-zero `to` address),
    ///      so the contract-creation base cost and EIP-3860 init code cost are not included.
    /// @param _data The transaction calldata.
    /// @return gas_ The intrinsic gas cost.
    function computeIntrinsicGas(bytes memory _data) internal pure returns (uint64 gas_) {
        gas_ = 21_000;
        for (uint256 i = 0; i < _data.length; i++) {
            gas_ += _data[i] == 0 ? 4 : 16;
        }
    }

    /// @notice Computes the recommended gas limit for a transaction.
    /// @dev Recommended gas limit = 1.5x max(intrinsic + body, EIP-7623 floor). The EIP-7623
    ///      floor dominates when execution is cheap relative to calldata,
    ///      since op-geth charges the floor as gasUsed.
    /// @param _intrinsicGas The intrinsic gas cost.
    /// @param _bodyGasUsed The body gas used.
    /// @param _floorGas The EIP-7623 floor gas.
    /// @return gas_ The recommended gas limit.
    function computeRecommendedGasLimit(
        uint64 _intrinsicGas,
        uint64 _bodyGasUsed,
        uint64 _floorGas
    )
        internal
        pure
        returns (uint64 gas_)
    {
        uint64 exec = _intrinsicGas + _bodyGasUsed;
        uint64 effective = exec > _floorGas ? exec : _floorGas;
        gas_ = uint64((uint256(effective) * 150) / 100); // 1.5x
    }

    /// @notice Computes the EIP-7623 floor gas for a transaction.
    /// @dev Replicates op-geth's FloorDataGas (core/state_transition.go) active on Prague/Isthmus+:
    ///      floor = 21_000 + 10 * tokens, where tokens = zero_bytes + 4 * non_zero_bytes.
    ///      op-geth applies this twice: pre-execution it rejects the tx (ErrFloorDataGas) when
    ///      gasLimit < floor; post-execution it sets receipt.gasUsed = max(execution_gas, floor).
    /// @param _data The transaction calldata.
    /// @return gas_ The EIP-7623 floor gas (includes the 21_000 base).
    function computeFloorDataGas(bytes memory _data) internal pure returns (uint64 gas_) {
        uint64 tokens;
        for (uint256 i = 0; i < _data.length; i++) {
            tokens += _data[i] == 0 ? 1 : 4;
        }
        gas_ = 21_000 + tokens * 10;
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
