// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

contract PredeploysTest is CommonTest {
    /// @dev Tests that the predeploy addresses are set correctly. They have code
    ///      and the proxied accounts have the correct admin.
    function test_predeploysSet_succeeds() external {
        uint256 count = 2048;
        uint160 prefix = uint160(0x420) << 148;
        for (uint256 i = 0; i < count; i++) {
            address addr = address(prefix | uint160(i));
            bytes memory code = addr.code;
            assertTrue(code.length > 0);
            // Skip the accounts that do not have a proxy
            if (addr == Predeploys.LEGACY_ERC20_ETH || addr == Predeploys.GOVERNANCE_TOKEN || addr == Predeploys.WETH9)
            {
                continue;
            }
            assertTrue(EIP1967Helper.getAdmin(addr) == Predeploys.PROXY_ADMIN, "Admin mismatch");
        }
    }
}
