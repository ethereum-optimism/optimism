// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import {
    L1Block_SetL1BlockValues_Test,
    L1Block_SetL1BlockValuesEcotone_Test,
    L1Block_SetL1BlockValuesIsthmus_Test
} from "test/L2/L1Block.t.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";

// Libraries
import "src/libraries/L1BlockErrors.sol";

// Interfaces
import { IL1BlockCGT } from "interfaces/L2/IL1BlockCGT.sol";

/// @title L1BlockCGT_TestInit
/// @notice Reusable test initialization for `L1Block` tests with custom gas token enabled.
contract L1BlockCGT_TestInit is CommonTest {
    address depositor;
    IL1BlockCGT l1BlockCGT;

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.enableCustomGasToken();
        super.setUp();
        l1BlockCGT = IL1BlockCGT(address(l1Block));
        depositor = l1BlockCGT.DEPOSITOR_ACCOUNT();
    }
}

/// @title L1BlockCGT_GasPayingTokenName_Test
/// @notice Tests the `gasPayingTokenName` function of the `L1Block` contract with custom gas
///         token enabled.
contract L1BlockCGT_GasPayingTokenName_Test is L1BlockCGT_TestInit {
    /// @notice Tests that the `gasPayingTokenName` function returns the correct token name.
    function test_gasPayingTokenName_succeeds() external view {
        assertEq(liquidityController.gasPayingTokenName(), l1BlockCGT.gasPayingTokenName());
    }
}

/// @title L1BlockCGT_GasPayingTokenSymbol_Test
/// @notice Tests the `gasPayingTokenSymbol` function of the `L1Block` contract with custom gas
///         token enabled.
contract L1BlockCGT_GasPayingTokenSymbol_Test is L1BlockCGT_TestInit {
    /// @notice Tests that the `gasPayingTokenSymbol` function returns the correct token symbol.
    function test_gasPayingTokenSymbol_succeeds() external view {
        assertEq(liquidityController.gasPayingTokenSymbol(), l1BlockCGT.gasPayingTokenSymbol());
    }
}

/// @title L1BlockCGT_IsCustomGasToken_Test
/// @notice Tests the `isCustomGasToken` function of the `L1Block` contract with custom gas token
///         enabled.
contract L1BlockCGT_IsCustomGasToken_Test is L1BlockCGT_TestInit {
    /// @notice Tests that the `isCustomGasToken` function returns false when no custom gas token
    ///         is used.
    function test_isCustomGasToken_succeeds() external view {
        assertTrue(l1BlockCGT.isCustomGasToken());
    }
}

/// @title L1BlockCGT_SetL1BlockValues_Test
/// @notice Tests the `setL1BlockValues` function of the `L1Block` contract with custom gas token
///         enabled.
contract L1BlockCGT_SetL1BlockValues_Test is L1Block_SetL1BlockValues_Test {
    // Override setUp to enable custom gas token
    // Re-use the test from L1Block.t.sol
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title L1BlockCGT_SetL1BlockValuesEcotone_Test
/// @notice Tests the `setL1BlockValuesEcotone` function of the `L1Block` contract with custom gas
///         token enabled.
contract L1BlockCGT_SetL1BlockValuesEcotone_Test is L1Block_SetL1BlockValuesEcotone_Test {
    // Override setUp to enable custom gas token
    // Re-use the test from L1Block.t.sol
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title L1BlockCGTSetL1BlockValuesIsthmus_Test
/// @notice Tests the `setL1BlockValuesIsthmus` function of the `L1Block` contract with custom gas
///         token enabled.
contract L1BlockCGT_SetL1BlockValuesIsthmus_Test is L1Block_SetL1BlockValuesIsthmus_Test {
    // Override setUp to enable custom gas token
    // Re-use the test from L1Block.t.sol
    function setUp() public override {
        super.enableCustomGasToken();
        super.setUp();
    }
}

/// @title L1BlockCGT_SetCustomGasToken_Test
/// @notice Tests the `setCustomGasToken` function of the `L1Block` contract.
contract L1BlockCGT_SetCustomGasToken_Test is L1BlockCGT_TestInit {
    using stdStorage for StdStorage;

    /// @notice Tests that `setCustomGasToken` reverts if called twice.
    function test_setCustomGasToken_alreadyActive_reverts() external {
        // This test uses the setUp that already activates custom gas token
        assertTrue(l1BlockCGT.isCustomGasToken());

        vm.expectRevert("L1Block: CustomGasToken already active");
        vm.prank(depositor);
        IL1BlockCGT(address(l1BlockCGT)).setCustomGasToken();
    }

    /// @notice Tests that `setCustomGasToken` updates the flag correctly when called by depositor.
    function test_setCustomGasToken_succeeds() external {
        stdstore.target(address(l1BlockCGT)).sig("isCustomGasToken()").checked_write(false);
        // This test uses the setUp that already activates custom gas token
        assertFalse(l1BlockCGT.isCustomGasToken());

        vm.prank(depositor);
        l1BlockCGT.setCustomGasToken();

        assertTrue(l1BlockCGT.isCustomGasToken());
    }

    /// @notice Tests that `setCustomGasToken` reverts if sender address is not the depositor.
    function test_setCustomGasToken_notDepositor_reverts(address nonDepositor) external {
        stdstore.target(address(l1BlockCGT)).sig("isCustomGasToken()").checked_write(false);
        vm.assume(nonDepositor != depositor);
        vm.expectRevert("L1Block: only the depositor account can set isCustomGasToken flag");
        vm.prank(nonDepositor);
        l1BlockCGT.setCustomGasToken();
    }
}
