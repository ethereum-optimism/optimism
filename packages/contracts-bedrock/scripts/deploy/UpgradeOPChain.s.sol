// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { IOPContractsManager, IOPContractsManagerPre4_1_0 } from "interfaces/L1/IOPContractsManager.sol";
import { BaseDeployIO } from "scripts/deploy/BaseDeployIO.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

contract UpgradeOPChainInput is BaseDeployIO {
    address internal _prank;
    IOPContractsManager internal _opcm;
    bytes _opChainConfigs;

    // Setter for OPContractsManager type
    function set(bytes4 _sel, address _value) public {
        require(address(_value) != address(0), "UpgradeOPChainInput: cannot set zero address");

        if (_sel == this.prank.selector) _prank = _value;
        else if (_sel == this.opcm.selector) _opcm = IOPContractsManager(_value);
        else revert("UpgradeOPChainInput: unknown selector");
    }

    function set(bytes4 _sel, bytes memory _value) public {
        require(_value.length > 0, "UpgradeOPChainInput: cannot set empty array");

        if (_sel == this.opChainConfigs.selector) _opChainConfigs = _value;
        else revert("UpgradeOPChainInput: unknown selector");
    }

    function prank() public view returns (address) {
        require(address(_prank) != address(0), "UpgradeOPChainInput: prank not set");
        return _prank;
    }

    function opcm() public view returns (IOPContractsManager) {
        require(address(_opcm) != address(0), "UpgradeOPChainInput: opcm not set");
        return _opcm;
    }

    function opChainConfigs() public view returns (bytes memory) {
        require(_opChainConfigs.length > 0, "UpgradeOPChainInput: opChainConfigs not set");

        return _opChainConfigs;
    }
}

contract UpgradeOPChain is Script {
    function run(UpgradeOPChainInput _uoci) external {
        IOPContractsManager opcm = _uoci.opcm();

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _uoci.prank();

        // From OPCM version 4.1.0, the proxyAdmin was removed from the OpChainConfig struct so we do create support for
        // both interface variants.
        if (SemverComp.lt(opcm.version(), "4.1.0")) {
            bytes memory code = vm.getDeployedCode("UpgradeOPChain.s.sol:DummyCallerPreOPCM4_1_0");
            vm.etch(prank, code);

            vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
            vm.label(prank, "DummyCaller");

            bytes memory encoded = _uoci.opChainConfigs();
            IOPContractsManagerPre4_1_0.OpChainConfig[] memory opChainConfigs =
                abi.decode(encoded, (IOPContractsManagerPre4_1_0.OpChainConfig[]));

            // apart from the offset and length that take up 64 bytes, the rest should be a multiple of 96 bytes
            // (systemConfigProxy, proxyAdmin and absolutePrestate)
            require(
                (((encoded.length - 64) / 96) == opChainConfigs.length) && (((encoded.length - 64) % 96) == 0),
                "UpgradeOPChain: opChainConfigsPre410 Unexpected encoding"
            );

            // Call into the DummyCaller to perform the delegatecall
            vm.broadcast(msg.sender);

            (bool success,) = DummyCallerPreOPCM4_1_0(prank).upgrade(opChainConfigs);
            require(success, "UpgradeOPChain: Pre4_1_0 upgrade failed");
        } else {
            bytes memory code = vm.getDeployedCode("UpgradeOPChain.s.sol:DummyCaller");
            vm.etch(prank, code);

            vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
            vm.label(prank, "DummyCaller");

            bytes memory encoded = _uoci.opChainConfigs();
            IOPContractsManager.OpChainConfig[] memory opChainConfigs =
                abi.decode(encoded, (IOPContractsManager.OpChainConfig[]));

            // apart from the offset and length that take up 64 bytes, the rest should be a multiple of 64 bytes
            // (systemConfigProxy and absolutePrestate)
            require(
                ((encoded.length - 64) / 64 == opChainConfigs.length) && (((encoded.length - 64) % 64) == 0),
                "UpgradeOPChain: opChainConfigs Unexpected encoding"
            );

            // Call into the DummyCaller to perform the delegatecall
            vm.broadcast(msg.sender);

            (bool success,) = DummyCaller(prank).upgrade(opChainConfigs);
            require(success, "UpgradeOPChain: upgrade failed");
        }
    }
}

/// @title DummyCaller
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and above.
contract DummyCaller {
    address internal _opcmAddr;

    function upgrade(IOPContractsManager.OpChainConfig[] memory _opChainConfigs)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCaller.upgrade, _opChainConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}

/// @title DummyCallerPreOPCM4_1_0
/// @notice This contract is used to mimic the contract that is used as the source of the delegatecall to the OPCM.
/// @dev This contract is used for OPCM versions 4.1.0 and below.
contract DummyCallerPreOPCM4_1_0 {
    address internal _opcmAddr;

    function upgrade(IOPContractsManagerPre4_1_0.OpChainConfig[] memory _opChainConfigs)
        external
        returns (bool, bytes memory)
    {
        bytes memory data = abi.encodeCall(DummyCallerPreOPCM4_1_0.upgrade, _opChainConfigs);
        (bool success, bytes memory result) = _opcmAddr.delegatecall(data);
        return (success, result);
    }
}
