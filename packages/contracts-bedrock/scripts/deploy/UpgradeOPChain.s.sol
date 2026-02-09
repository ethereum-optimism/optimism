// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { OPContractsManagerV2 } from "src/L1/opcm/OPContractsManagerV2.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { DummyCaller } from "scripts/libraries/DummyCaller.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Constants } from "src/libraries/Constants.sol";

contract UpgradeOPChainInput is BaseDeployIO {
    address internal _prank;
    address internal _opcm;
    /// @notice The upgrade input is stored as opaque bytes to allow storing both OPCM v1 and v2 upgrade inputs.
    bytes _upgradeInput;

    // Setter for OPContractsManager type
    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "UpgradeOPCMInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = _value;
        else revert("UpgradeOPCMInput: unknown selector");
    }

    /// @notice Sets the upgrade input using the OPContractsManager.OpChainConfig[] type,
    ///         this is used when upgrading chains using OPCM v1.
    /// @param _sel The selector of the field to set.
    /// @param _value The value to set.
    function set(bytes4 _sel, OPContractsManager.OpChainConfig[] memory _value) public {
        if (SemverComp.gte(OPContractsManager(opcm()).version(), Constants.OPCM_V2_MIN_VERSION)) {
            revert("UpgradeOPCMInput: cannot set OPCM v1 upgrade input when OPCM v2 is enabled");
        }
        require(_value.length > 0, "UpgradeOPCMInput: cannot set empty array");

        if (_sel == this.upgradeInput.selector) _upgradeInput = abi.encode(_value);
        else revert("UpgradeOPCMInput: unknown selector");
    }

    /// @notice Sets the upgrade input using the OPContractsManagerV2.UpgradeInput type,
    ///         this is used when upgrading chains using OPCM v2.
    ///         Minimal validation is performed, relying on the OPCM v2 contract to perform the proper validation.
    ///         This is done to avoid duplicating the validation logic in the script.
    /// @param _sel The selector of the field to set.
    /// @param _value The value to set.
    function set(bytes4 _sel, OPContractsManagerV2.UpgradeInput memory _value) public {
        if (!SemverComp.gte(OPContractsManager(opcm()).version(), Constants.OPCM_V2_MIN_VERSION)) {
            revert("UpgradeOPCMInput: cannot set OPCM v2 upgrade input when OPCM v1 is enabled");
        }
        require(address(_value.systemConfig) != address(0), "UpgradeOPCMInput: cannot set zero address");
        require(_value.disputeGameConfigs.length > 0, "UpgradeOPCMInput: cannot set empty dispute game configs array");

        if (_sel == this.upgradeInput.selector) _upgradeInput = abi.encode(_value);
        else revert("UpgradeOPCMInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "UpgradeOPCMInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (address) {
        require(_opcm != address(0), "UpgradeOPCMInput: not set");
        return _opcm;
    }

    function upgradeInput() public view returns (bytes memory) {
        require(_upgradeInput.length > 0, "UpgradeOPCMInput: not set");
        return _upgradeInput;
    }
}

contract UpgradeOPChain is Script {
    function run(UpgradeOPChainInput _uoci) external {
        address opcm = _uoci.opcm();

        // First, we need to check what version of OPCM is being used.
        bool isOPCMv2 = SemverComp.gte(OPContractsManager(opcm).version(), Constants.OPCM_V2_MIN_VERSION);

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _uoci.prank();
        bytes memory code = _getDummyCallerCode();
        vm.etch(prank, code);
        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        // Get the upgrade input before broadcasting
        bytes memory upgradeInput = _uoci.upgradeInput();

        // Call into the DummyCaller. This will perform the delegatecall under the hood.
        // The DummyCaller uses a fallback that reverts on failure, so no need to check success.
        vm.broadcast(msg.sender);
        _upgrade(prank, isOPCMv2, upgradeInput);
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
    /// @param _upgradeInput The upgrade input.
    function _upgrade(address _prank, bool _isOPCMv2, bytes memory _upgradeInput) internal {
        bytes memory data;
        if (_isOPCMv2) {
            data = abi.encodeCall(
                OPContractsManagerV2.upgrade, abi.decode(_upgradeInput, (OPContractsManagerV2.UpgradeInput))
            );
        } else {
            data = abi.encodeCall(
                OPContractsManager.upgrade, abi.decode(_upgradeInput, (OPContractsManager.OpChainConfig[]))
            );
        }
        (bool success, bytes memory returnData) = _prank.call(data);
        if (!success) {
            assembly {
                revert(add(returnData, 0x20), mload(returnData))
            }
        }
    }
}
