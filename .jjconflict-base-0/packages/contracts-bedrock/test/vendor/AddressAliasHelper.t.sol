// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

/// @title AddressAliasHelper_ApplyL1ToL2Alias_Test
/// @notice Tests for the `applyL1ToL2Alias` function.
contract AddressAliasHelper_ApplyL1ToL2Alias_Test is Test {
    uint160 constant OFFSET = uint160(0x1111000000000000000000000000000000001111);

    /// @notice Tests that applyL1ToL2Alias correctly adds the offset to L1 address.
    /// @param _l1Address The L1 address to apply the alias to.
    function testFuzz_applyL1ToL2Alias_addsOffset_succeeds(address _l1Address) external pure {
        address l2Address = AddressAliasHelper.applyL1ToL2Alias(_l1Address);
        uint160 expected;
        unchecked {
            expected = uint160(_l1Address) + OFFSET;
        }
        assertEq(uint160(l2Address), expected);
    }
}

/// @title AddressAliasHelper_UndoL1ToL2Alias_Test
/// @notice Tests for the `undoL1ToL2Alias` function.
contract AddressAliasHelper_UndoL1ToL2Alias_Test is Test {
    uint160 constant OFFSET = uint160(0x1111000000000000000000000000000000001111);

    /// @notice Tests that undoL1ToL2Alias correctly subtracts offset from L2 address.
    /// @param _l2Address The L2 address to undo the alias from.
    function testFuzz_undoL1ToL2Alias_subtractsOffset_succeeds(address _l2Address) external pure {
        address l1Address = AddressAliasHelper.undoL1ToL2Alias(_l2Address);
        uint160 expected;
        unchecked {
            expected = uint160(_l2Address) - OFFSET;
        }
        assertEq(uint160(l1Address), expected);
    }
}

/// @title AddressAliasHelper_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `AddressAliasHelper`
///         contract or are testing multiple functions at once.
contract AddressAliasHelper_Uncategorized_Test is Test {
    /// @notice Tests that applying and then undoing an alias results in the original address.
    function testFuzz_applyAndUndo_succeeds(address _address) external pure {
        address aliased = AddressAliasHelper.applyL1ToL2Alias(_address);
        address unaliased = AddressAliasHelper.undoL1ToL2Alias(aliased);
        assertEq(_address, unaliased);
    }
}
