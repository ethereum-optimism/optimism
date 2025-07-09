// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Test } from "forge-std/Test.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

/// @title DeployUtils_TestInit
/// @notice Reusable test initialization for `DeployUtils` tests.
contract DeployUtils_TestInit is Test {
    /// @notice Helper function to test the revert message of assertUniqueAddresses with duplicate
    ///         addresses.
    /// @dev This function only exists because expectRevert only accepts a calldata argument but
    ///      string concatenation (required to create the revert message) is not possible in
    ///      calldata. See testFuzz_assertUniqueAddresses_withDuplicateAddress_reverts
    function helper_assertUniqueAddresses_withDuplicateAddress_reverts(
        string calldata _message,
        address[] calldata _addresses
    )
        external
    {
        vm.expectRevert(bytes(_message));
        DeployUtils.assertUniqueAddresses(_addresses);
    }
}

/// @title DeployUtils_AssertUniqueAddresses_Test
/// @notice Tests the `assertUniqueAddresses` function of the `DeployUtils` library.
contract DeployUtils_AssertUniqueAddresses_Test is DeployUtils_TestInit {
    function test_assertUniqueAddresses_withEmptyArray_succeeds() public pure {
        DeployUtils.assertUniqueAddresses(new address[](0));
    }

    /// @param value The address to be tested.
    function testFuzz_assertUniqueAddresses_withOneAddress_succeeds(address value) public pure {
        DeployUtils.assertUniqueAddresses(Solarray.addresses(value));
    }

    /// @param _length The length of the array of addresses.
    /// @param _seed The seed for generating the addresses.
    function testFuzz_assertUniqueAddresses_withUniqueAddresses_succeeds(uint8 _length, bytes32 _seed) public pure {
        vm.assume(_length != 0);

        address[] memory addresses = new address[](_length);
        for (uint256 i = 0; i < _length; i++) {
            addresses[i] = address(uint160(uint256(keccak256(abi.encode(_seed, i)))));
        }

        DeployUtils.assertUniqueAddresses(addresses);
    }

    /// @param _length The length of the array of addresses.
    /// @param _duplicateIndex The index of the address to be duplicated.
    /// @param _seed The seed for generating the addresses.
    /// forge-config: default.allow_internal_expect_revert = true
    function testFuzz_assertUniqueAddresses_withDuplicateAddress_reverts(
        uint8 _length,
        uint8 _duplicateIndex,
        bytes32 _seed
    )
        public
    {
        vm.assume(_length != 0);
        vm.assume(_duplicateIndex < _length);

        address[] memory addresses = new address[](uint16(_length) + 1);
        for (uint256 i = 0; i < _length; i++) {
            addresses[i] = address(uint160(uint256(keccak256(abi.encode(_seed, i)))));
        }

        // Insert a duplicate address at the end of the array
        addresses[_length] = addresses[_duplicateIndex];

        // Unfortunately it's not possible to use vm.expectRevert() here because the revert
        // message is not a calldata argument so we need to externalize the call
        DeployUtils_AssertUniqueAddresses_Test(this).helper_assertUniqueAddresses_withDuplicateAddress_reverts(
            string.concat(
                "DeployUtils: check failed, duplicates at ", vm.toString(_duplicateIndex), ",", vm.toString(_length)
            ),
            addresses
        );
    }
}
