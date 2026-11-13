// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { AddressManager } from "src/legacy/AddressManager.sol";

// Libraries
import { Blueprint } from "src/libraries/Blueprint.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// Interfaces
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";

/// @title DeployUtils_TestInit
/// @notice Reusable test initialization for `DeployUtils` tests.
abstract contract DeployUtils_TestInit is Test {
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

    /// @notice Calls createDeterministicBlueprint in a call of its own.
    /// @dev An expected revert applies to the next call. Called internally, that is the CREATE2 of the blueprint,
    ///      which succeeds, so forge makes it fail before DeployUtils can revert.
    function helper_createDeterministicBlueprint(bytes calldata _rawBytecode, bytes32 _salt) external {
        DeployUtils.createDeterministicBlueprint(_rawBytecode, _salt);
    }
}

/// @title DeployUtils_CreateDeterministic_Test
/// @notice Tests deployment and reuse of deterministic contracts.
contract DeployUtils_CreateDeterministic_Test is DeployUtils_TestInit {
    /// @notice Deploys at the computed address or reuses existing code without changing it.
    function testFuzz_createDeterministic_freshOrReused_succeeds(bool _reuse) public {
        bytes32 salt = keccak256("reuse logging");
        address expected = vm.computeCreate2Address(salt, keccak256(type(AddressManager).creationCode));
        if (_reuse) {
            vm.etch(expected, hex"00");
        }

        address deployed = DeployUtils.createDeterministic({
            _name: "AddressManager",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IAddressManager.__constructor__, ())),
            _salt: salt
        });

        assertEq(deployed, expected);
        if (_reuse) {
            assertEq(deployed.code, hex"00");
        } else {
            assertGt(deployed.code.length, 0);
        }
    }
}

/// @title DeployUtils_CreateDeterministicBlueprint_Test
/// @notice Tests deployment and reuse of single- and two-part blueprints.
contract DeployUtils_CreateDeterministicBlueprint_Test is DeployUtils_TestInit {
    /// @notice A single-part blueprint is deployed or reused without changing existing code.
    function testFuzz_createDeterministicBlueprint_singleFreshOrReused_succeeds(bool _reuse) public {
        bytes memory code = hex"6000";
        bytes32 salt = keccak256("single blueprint reuse logging");
        address expected = vm.computeCreate2Address(salt, keccak256(Blueprint.blueprintDeployerBytecode(code)));
        if (_reuse) {
            vm.etch(expected, hex"00");
        }

        vm.startBroadcast(msg.sender);
        (address first, address second) = DeployUtils.createDeterministicBlueprint(code, salt);
        vm.stopBroadcast();

        assertEq(first, expected);
        assertEq(second, address(0));
        assertEq(first.code, _reuse ? bytes(hex"00") : bytes.concat(hex"fe7100", code));
    }

    /// @notice Each part is deployed or reused independently, including when only one part exists.
    function testFuzz_createDeterministicBlueprint_splitFreshOrReused_succeeds(
        bool _reuseFirst,
        bool _reuseSecond
    )
        public
    {
        uint256 maxSize = Blueprint.maxInitCodeSize();
        bytes memory code = new bytes(maxSize + 1);
        bytes memory part1 = Bytes.slice(code, 0, maxSize);
        bytes memory part2 = Bytes.slice(code, maxSize, 1);
        bytes32 salt = keccak256("split blueprint reuse logging");
        address expected1 = vm.computeCreate2Address(salt, keccak256(Blueprint.blueprintDeployerBytecode(part1)));
        address expected2 = vm.computeCreate2Address(salt, keccak256(Blueprint.blueprintDeployerBytecode(part2)));
        if (_reuseFirst) {
            vm.etch(expected1, hex"00");
        }
        if (_reuseSecond) {
            vm.etch(expected2, hex"00");
        }

        vm.startBroadcast(msg.sender);
        (address first, address second) = DeployUtils.createDeterministicBlueprint(code, salt);
        vm.stopBroadcast();

        assertEq(first, expected1);
        assertEq(second, expected2);
        assertEq(first.code, _reuseFirst ? bytes(hex"00") : bytes.concat(hex"fe7100", part1));
        assertEq(second.code, _reuseSecond ? bytes(hex"00") : bytes.concat(hex"fe7100", part2));
    }

    // The tests below run without a broadcast, so CREATE2 runs from this contract instead of the CREATE2 factory and
    // deploys at an address other than the computed one. Each blueprint names itself in the revert with the label
    // that it also logs, which a test cannot read.

    /// @notice A single-part blueprint at an unexpected address reverts with its label.
    function test_createDeterministicBlueprint_unexpectedAddress_reverts() public {
        vm.expectRevert("DeployUtils: unexpected address for Blueprint");
        this.helper_createDeterministicBlueprint(hex"6000", keccak256("single blueprint unexpected address"));
    }

    /// @notice The first part of a split blueprint at an unexpected address reverts with its label.
    function test_createDeterministicBlueprint_unexpectedPart1Address_reverts() public {
        bytes memory code = new bytes(Blueprint.maxInitCodeSize() + 1);

        vm.expectRevert("DeployUtils: unexpected address for Blueprint part 1");
        this.helper_createDeterministicBlueprint(code, keccak256("split blueprint unexpected address"));
    }

    /// @notice The second part of a split blueprint at an unexpected address reverts with its label.
    function test_createDeterministicBlueprint_unexpectedPart2Address_reverts() public {
        uint256 maxSize = Blueprint.maxInitCodeSize();
        bytes memory code = new bytes(maxSize + 1);
        bytes32 salt = keccak256("split blueprint unexpected address");
        bytes memory part1 = Bytes.slice(code, 0, maxSize);
        vm.etch(vm.computeCreate2Address(salt, keccak256(Blueprint.blueprintDeployerBytecode(part1))), hex"00");

        vm.expectRevert("DeployUtils: unexpected address for Blueprint part 2");
        this.helper_createDeterministicBlueprint(code, salt);
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
        DeployUtils_AssertUniqueAddresses_Test(this)
            .helper_assertUniqueAddresses_withDuplicateAddress_reverts(
                string.concat(
                    "DeployUtils: check failed, duplicates at ", vm.toString(_duplicateIndex), ",", vm.toString(_length)
                ),
                addresses
            );
    }
}
