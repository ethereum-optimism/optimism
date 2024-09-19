// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Proxy } from "src/universal/Proxy.sol";
import { LibString } from "@solady/utils/LibString.sol";
import { Vm } from "forge-std/Vm.sol";

library DeployUtils {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    // This takes a sender and an identifier and returns a deterministic address based on the two.
    // The result is used to etch the input and output contracts to a deterministic address based on
    // those two values, where the identifier represents the input or output contract, such as
    // `optimism.DeploySuperchainInput` or `optimism.DeployOPChainOutput`.
    // Example: `toIOAddress(msg.sender, "optimism.DeploySuperchainInput")`
    function toIOAddress(address _sender, string memory _identifier) internal pure returns (address) {
        return address(uint160(uint256(keccak256(abi.encode(_sender, _identifier)))));
    }

    function assertValidContractAddress(address _who) internal view {
        require(_who != address(0), "DeployUtils: zero address");
        require(_who.code.length > 0, string.concat("DeployUtils: no code at ", LibString.toHexStringChecksummed(_who)));
    }

    function assertImplementationSet(address _proxy) internal {
        // We prank as the zero address due to the Proxy's `proxyCallIfNotAdmin` modifier.
        // Pranking inside this function also means it can no longer be considered `view`.
        vm.prank(address(0));
        address implementation = Proxy(payable(_proxy)).implementation();
        assertValidContractAddress(implementation);
    }

    function assertValidContractAddresses(address[] memory _addrs) internal view {
        // Assert that all addresses are non-zero and have code.
        // We use LibString to avoid the need for adding cheatcodes to this contract.
        for (uint256 i = 0; i < _addrs.length; i++) {
            address who = _addrs[i];
            assertValidContractAddress(who);
        }

        // All addresses should be unique.
        for (uint256 i = 0; i < _addrs.length; i++) {
            for (uint256 j = i + 1; j < _addrs.length; j++) {
                string memory err =
                    string.concat("check failed: duplicates at ", LibString.toString(i), ",", LibString.toString(j));
                require(_addrs[i] != _addrs[j], err);
            }
        }
    }

    // Asserts that for a given contract the value of a storage slot at an offset is 1 or
    // `type(uint8).max`. The value is set to 1 when a contract is initialized, and set to
    // `type(uint8).max` when `_disableInitializers` is called.
    function assertInitialized(address _contractAddress, uint256 _slot, uint256 _offset) internal view {
        bytes32 slotVal = vm.load(_contractAddress, bytes32(_slot));
        uint8 value = uint8((uint256(slotVal) >> (_offset * 8)) & 0xFF);
        require(
            value == 1 || value == type(uint8).max,
            "Value at the given slot and offset does not indicate initialization"
        );
    }
}
