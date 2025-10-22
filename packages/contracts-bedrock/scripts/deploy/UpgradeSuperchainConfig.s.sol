// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

contract UpgradeSuperchainConfig is Script {
    struct Input {
        address prank;
        IOPContractsManager opcm;
        ISuperchainConfig superchainConfig;
    }

    /// @notice Delegate calls upgradeSuperchainConfig on the OPCM from the input.prank address.
    function run(Input memory _input) external {
        // Make sure the input is valid
        assertValidInput(_input);

        IOPContractsManager opcm = _input.opcm;

        // Etch DummyCaller contract. This contract is used to mimic the contract that is used
        // as the source of the delegatecall to the OPCM. In practice this will be the governance
        // 2/2 or similar.
        address prank = _input.prank;

        bytes memory code = vm.getDeployedCode("UpgradeSuperchainConfig.s.sol:DummyCaller");
        vm.etch(prank, code);

        vm.store(prank, bytes32(0), bytes32(uint256(uint160(address(opcm)))));
        vm.label(prank, "DummyCaller");

        ISuperchainConfig superchainConfig = _input.superchainConfig;

        // Call into the DummyCaller to perform the delegatecall
        vm.broadcast(msg.sender);

        (bool success,) = DummyCaller(prank).upgradeSuperchainConfig(superchainConfig);
        require(success, "UpgradeSuperchainConfig: upgradeSuperchainConfig failed");
    }

    /// @notice Asserts that the input is valid.
    function assertValidInput(Input memory _input) internal pure {
        require(_input.prank != address(0), "UpgradeSuperchainConfig: prank not set");
        require(address(_input.opcm) != address(0), "UpgradeSuperchainConfig: opcm not set");
        require(address(_input.superchainConfig) != address(0), "UpgradeSuperchainConfig: superchainConfig not set");
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
