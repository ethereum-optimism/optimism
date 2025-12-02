// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";

/// @title Constants_Test
/// @notice Tests the constant values defined in the `Constants` library.
contract Constants_Test is Test {
    /// @notice Verify ESTIMATION_ADDRESS constant value.
    function test_estimationAddress_succeeds() external pure {
        assertEq(Constants.ESTIMATION_ADDRESS, address(1));
    }

    /// @notice Verify DEFAULT_L2_SENDER constant value.
    function test_defaultL2Sender_succeeds() external pure {
        assertEq(Constants.DEFAULT_L2_SENDER, 0x000000000000000000000000000000000000dEaD);
    }

    /// @notice Verify EIP1967 proxy implementation storage slot.
    function test_proxyImplementationAddress_succeeds() external pure {
        assertEq(
            bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1), Constants.PROXY_IMPLEMENTATION_ADDRESS
        );
    }

    /// @notice Verify EIP1967 proxy admin storage slot.
    function test_proxyOwnerAddress_succeeds() external pure {
        assertEq(bytes32(uint256(keccak256("eip1967.proxy.admin")) - 1), Constants.PROXY_OWNER_ADDRESS);
    }

    /// @notice Verify GUARD_STORAGE_SLOT constant value.
    function test_guardStorageSlot_succeeds() external pure {
        assertEq(keccak256("guard_manager.guard.address"), Constants.GUARD_STORAGE_SLOT);
    }

    /// @notice Verify ETHER constant value.
    function test_ether_succeeds() external pure {
        assertEq(Constants.ETHER, 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE);
    }

    /// @notice Verify DEPOSITOR_ACCOUNT constant value.
    function test_depositorAccount_succeeds() external pure {
        assertEq(Constants.DEPOSITOR_ACCOUNT, 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001);
    }

    /// @notice Verify DEFAULT_RESOURCE_CONFIG returns expected values.
    function test_defaultResourceConfig_succeeds() external pure {
        IResourceMetering.ResourceConfig memory config = Constants.DEFAULT_RESOURCE_CONFIG();
        assertEq(config.maxResourceLimit, 20_000_000);
        assertEq(config.elasticityMultiplier, 10);
        assertEq(config.baseFeeMaxChangeDenominator, 8);
        assertEq(config.minimumBaseFee, 1 gwei);
        assertEq(config.systemTxMaxGas, 1_000_000);
        assertEq(config.maximumBaseFee, type(uint128).max);
    }
}
