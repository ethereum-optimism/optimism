// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Proxy } from "src/universal/Proxy.sol";
import { LibString } from "@solady/utils/LibString.sol";

library DeployUtils {
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

    function assertEIP1967Implementation(address _proxy) internal {
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
}
