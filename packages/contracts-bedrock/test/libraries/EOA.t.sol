// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { EOA } from "src/libraries/EOA.sol";

/// @title EOA_Harness
/// @notice A helper contract to test the EOA library.
contract EOA_Harness {
    /// @notice Returns true if the sender is an EOA.
    /// @return isEOA_ True if the sender is an EOA.
    function isSenderEOA() external view returns (bool isEOA_) {
        return EOA.isSenderEOA();
    }
}

/// @title EOA_TestInit
/// @notice Reusable test initialization for `EOA` tests.
abstract contract EOA_TestInit is Test {
    EOA_Harness harness;

    /// @notice Sets up the test.
    function setUp() public {
        harness = new EOA_Harness();
    }
}

/// @title EOA_isSenderEOA_Test
/// @notice Tests the `isSenderEOA` function of the `EOA` library.
contract EOA_isSenderEOA_Test is EOA_TestInit {
    /// @notice Tests that a standard EOA is detected as an EOA.
    /// @param _privateKey The private key of the sender.
    function testFuzz_isSenderEOA_isStandardEOA_succeeds(uint256 _privateKey) external {
        // Make sure that the private key is in the range of a valid secp256k1 private key.
        _privateKey = boundPrivateKey(_privateKey);

        // Make sure that the sender is a standard EOA with no code.
        address sender = vm.addr(_privateKey);
        vm.assume(sender.code.length == 0);

        // Should be considered an EOA
        vm.prank(sender, sender);
        assertEq(harness.isSenderEOA(), true);
    }

    /// @notice Tests that a 7702 EOA is detected as an EOA.
    /// @param _privateKey The private key of the sender.
    /// @param _7702Target The target of the 7702 EOA.
    function testFuzz_isSenderEOA_is7702EOA_succeeds(uint256 _privateKey, address _7702Target) external {
        // Make sure that the private key is in the range of a valid secp256k1 private key.
        _privateKey = boundPrivateKey(_privateKey);

        // Make sure that the sender is a 7702 EOA.
        address sender = vm.addr(_privateKey);
        vm.etch(sender, abi.encodePacked(hex"EF0100", _7702Target));

        // Should be considered a 7702 EOA.
        vm.prank(sender, sender);
        assertEq(harness.isSenderEOA(), true);

        // Should still be considered an EOA even if origin is different.
        vm.prank(sender, address(0x0420));
        assertEq(harness.isSenderEOA(), true);
    }

    /// @notice Tests that a contract is not detected as an EOA.
    /// @param _privateKey The private key of the sender.
    /// @param _code The code of the sender.
    function testFuzz_isSenderEOA_isContract_succeeds(uint256 _privateKey, bytes memory _code) external {
        // Make sure that the private key is in the range of a valid secp256k1 private key.
        _privateKey = boundPrivateKey(_privateKey);

        // If code is empty or starts with EF, change it.
        if (_code.length == 0 || _code[0] == 0xEF) {
            _code = bytes.concat(hex"FFFFFF", _code);
        }

        // Make sure that the sender is a contract.
        address sender = vm.addr(_privateKey);
        vm.etch(sender, _code);

        // Should not be considered an EOA.
        vm.prank(sender);
        assertEq(harness.isSenderEOA(), false);
    }

    /// @notice Tests that a contract with 23 bytes of code is not detected as an EOA.
    /// @param _privateKey The private key of the sender.
    function testFuzz_isSenderEOA_isContract23BytesNot7702_succeeds(uint256 _privateKey) external {
        // Make sure that the private key is in the range of a valid secp256k1 private key.
        _privateKey = boundPrivateKey(_privateKey);

        // Generate a random 23 byte code.
        bytes memory code = vm.randomBytes(23);

        // If the code happens to be EOF code, change it!
        if (code[0] == 0xEF) {
            code[0] = 0xFE; // Anything but EF!
        }

        // Make sure that the sender is a contract.
        address sender = vm.addr(_privateKey);
        vm.etch(sender, code);

        // Should not be considered an EOA.
        vm.prank(sender);
        assertEq(harness.isSenderEOA(), false);
    }
}
