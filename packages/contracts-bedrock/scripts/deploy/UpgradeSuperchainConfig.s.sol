// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

contract UpgradeSuperchainConfig is Script {
    struct Input {
        address prank;
        address opcm;
        ISuperchainConfig superchainConfig;
        IOPContractsManagerUtils.ExtraInstruction[] extraInstructions;
    }

    /// @notice Delegate calls upgradeSuperchainConfig on the OPCM from the input.prank address.
    function run(Input memory _input) external {
        // Make sure the input is valid
        assertValidInput(_input);

        // Both OPCM v1 and v2 implement the isDevFeatureEnabled function.
        bool useOPCMv2 = IOPContractsManager(_input.opcm).isDevFeatureEnabled(DevFeatures.OPCM_V2);

        address opcm = _input.opcm;

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _input.prank;

        bytes memory code = _getDummyCallerCode(useOPCMv2);
        vm.etch(prank, code);

        vm.store(prank, bytes32(0), bytes32(uint256(uint160(opcm))));
        vm.label(prank, "DummyCaller");

        (bool success,) = _upgrade(prank, useOPCMv2, _input);
        require(success, "UpgradeSuperchainConfig: upgradeSuperchainConfig failed");
    }

    /// @notice Asserts that the input is valid.
    function assertValidInput(Input memory _input) internal pure {
        // Note: Intentionally not checking extra instructions for OPCM v2 as they are not required in some upgrades.
        // This responsibility is delegated to the OPCM v2 contract.
        require(_input.prank != address(0), "UpgradeSuperchainConfig: prank not set");
        require(address(_input.opcm) != address(0), "UpgradeSuperchainConfig: opcm not set");
        require(address(_input.superchainConfig) != address(0), "UpgradeSuperchainConfig: superchainConfig not set");
    }

    /// @notice Helper function to get the proper dummy caller code based on the OPCM version.
    /// @param _useOPCMv2 Whether to use OPCM v2.
    /// @return code The code of the dummy caller.
    function _getDummyCallerCode(bool _useOPCMv2) internal view returns (bytes memory) {
        if (_useOPCMv2) return vm.getDeployedCode("UpgradeSuperchainConfig.s.sol:DummyCallerV2");
        else return vm.getDeployedCode("UpgradeSuperchainConfig.s.sol:DummyCaller");
    }

    /// @notice Helper function to upgrade the OPCM based on the OPCM version. Performs the decoding of the upgrade
    /// input and the delegatecall to the OPCM.
    /// @param _prank The address of the dummy caller contract.
    /// @param _useOPCMv2 Whether to use OPCM v2.
    /// @param _input The input.
    /// @return success Whether the upgrade succeeded.
    /// @return result The result of the upgrade (bool, bytes memory).
    function _upgrade(address _prank, bool _useOPCMv2, Input memory _input) internal returns (bool, bytes memory) {
        // Call into the DummyCaller to perform the delegatecall
        vm.broadcast(msg.sender);
        if (_useOPCMv2) {
            return DummyCallerV2(_prank).upgradeSuperchain(
                IOPContractsManagerV2.SuperchainUpgradeInput({
                    superchainConfig: _input.superchainConfig,
                    extraInstructions: _input.extraInstructions
                })
            );
        } else {
            return DummyCaller(_prank).upgradeSuperchainConfig(_input.superchainConfig);
        }
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
contract DummyCaller {
    address internal _opcmAddr;

    function upgradeSuperchainConfig(ISuperchainConfig _superchainConfig) external returns (bool, bytes memory) {
        bytes memory data = abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (_superchainConfig));
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

/// @title DummyCallerV2
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM v2.
/// Uses IOPContractsManagerV2.SuperchainUpgradeInput type for the upgrade input.
contract DummyCallerV2 {
    address internal _opcmAddr;

    function upgradeSuperchain(IOPContractsManagerV2.SuperchainUpgradeInput memory _superchainUpgradeInput)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(IOPContractsManagerV2.upgradeSuperchain, (_superchainUpgradeInput));
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
