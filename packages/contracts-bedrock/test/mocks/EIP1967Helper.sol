// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Vm } from "forge-std/Vm.sol";

/// @title EIP1967Helper
/// @dev Testing library to help with reading EIP 1967 variables from state
library EIP1967Helper {
    /// @notice The storage slot that holds the address of a proxy implementation.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)`
    bytes32 internal constant PROXY_IMPLEMENTATION_SLOT =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    /// @notice The storage slot that holds the address of the owner.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)`
    bytes32 internal constant PROXY_ADMIN_SLOT = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;

    Vm internal constant vm = Vm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

    function getAdmin(address _proxy) internal view returns (address) {
        return address(uint160(uint256(vm.load(address(_proxy), PROXY_ADMIN_SLOT))));
    }

    function setAdmin(address _addr, address _admin) internal {
        vm.store(_addr, PROXY_ADMIN_SLOT, bytes32(uint256(uint160(_admin))));
    }

    function getImplementation(address _proxy) internal view returns (address) {
        return address(uint160(uint256(vm.load(address(_proxy), PROXY_IMPLEMENTATION_SLOT))));
    }

    function setImplementation(address _addr, address _impl) internal {
        vm.store(_addr, PROXY_IMPLEMENTATION_SLOT, bytes32(uint256(uint160(_impl))));
    }
}
