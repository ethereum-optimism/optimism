// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { DummyCaller } from "scripts/libraries/DummyCaller.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Constants } from "src/libraries/Constants.sol";

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

        address opcm = _input.opcm;

        bool isOPCMv2 = SemverComp.gte(IOPContractsManager(opcm).version(), Constants.OPCM_V2_MIN_VERSION);

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _input.prank;

        bytes memory code = _getDummyCallerCode();
        vm.etch(prank, code);

        vm.store(prank, bytes32(0), bytes32(uint256(uint160(opcm))));
        vm.label(prank, "DummyCaller");

        // Call into the DummyCaller. This will perform the delegatecall under the hood.
        // The DummyCaller uses a fallback that reverts on failure, so no need to check success.
        vm.broadcast(msg.sender);
        _upgrade(prank, isOPCMv2, _input);
    }

    /// @notice Asserts that the input is valid.
    function assertValidInput(Input memory _input) internal pure {
        // Note: Intentionally not checking extra instructions for OPCM v2 as they are not required in some upgrades.
        // This responsibility is delegated to the OPCM v2 contract.
        require(_input.prank != address(0), "UpgradeSuperchainConfig: prank not set");
        require(address(_input.opcm) != address(0), "UpgradeSuperchainConfig: opcm not set");
        require(address(_input.superchainConfig) != address(0), "UpgradeSuperchainConfig: superchainConfig not set");
    }

    /// @notice Helper function to get the dummy caller code.
    /// @return code The code of the dummy caller.
    function _getDummyCallerCode() internal pure returns (bytes memory) {
        return type(DummyCaller).runtimeCode;
    }

    /// @notice Helper function to upgrade the OPCM based on the OPCM version. Performs the decoding of the upgrade
    /// input and the delegatecall to the OPCM.
    /// @param _prank The address of the dummy caller contract.
    /// @param _isOPCMv2 Whether to use OPCM v2.
    /// @param _input The input.
    function _upgrade(address _prank, bool _isOPCMv2, Input memory _input) internal {
        bytes memory data;
        if (_isOPCMv2) {
            data = abi.encodeCall(
                IOPContractsManagerV2.upgradeSuperchain,
                IOPContractsManagerV2.SuperchainUpgradeInput({
                    superchainConfig: _input.superchainConfig,
                    extraInstructions: _input.extraInstructions
                })
            );
        } else {
            data = abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, _input.superchainConfig);
        }
        (bool success, bytes memory returnData) = _prank.call(data);
        if (!success) {
            assembly {
                revert(add(returnData, 0x20), mload(returnData))
            }
        }
    }
}
