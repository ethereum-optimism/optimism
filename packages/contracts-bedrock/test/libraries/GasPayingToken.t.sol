// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";
import { Constants } from "src/libraries/Constants.sol";
import { LibString } from "@solady/utils/LibString.sol";

contract GasPayingToken_Harness {
    function exposed_sanitize(string memory _str) public pure returns (bytes32) {
        return GasPayingToken.sanitize(_str);
    }
}

/// @title GasPayingToken_TestInit
/// @notice Reusable test initialization for `GasPayingToken` tests.
abstract contract GasPayingToken_TestInit is Test {
    GasPayingToken_Harness harness;

    function setUp() public {
        harness = new GasPayingToken_Harness();
    }
}

/// @title GasPayingToken_GetToken_Test
/// @notice Tests the `getToken` function of the `GasPayingToken` library.
contract GasPayingToken_GetToken_Test is GasPayingToken_TestInit {
    /// @notice Test that the gas paying token returns values associated with Ether when unset.
    function test_getToken_empty_succeeds() external view {
        (address token, uint8 decimals) = GasPayingToken.getToken();
        assertEq(Constants.ETHER, token);
        assertEq(18, decimals);

        assertEq("Ether", GasPayingToken.getName());

        assertEq("ETH", GasPayingToken.getSymbol());
    }

    /// @notice Test that the gas paying token correctly gets values from storage when set.
    function testFuzz_getToken_nonEmpty_succeeds(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        external
    {
        vm.assume(_token != address(0));
        vm.assume(_token != Constants.ETHER);

        GasPayingToken.set(_token, _decimals, _name, _symbol);

        (address token, uint8 decimals) = GasPayingToken.getToken();
        assertEq(_token, token);
        assertEq(_decimals, decimals);

        assertEq(LibString.fromSmallString(_name), GasPayingToken.getName());
        assertEq(LibString.fromSmallString(_symbol), GasPayingToken.getSymbol());
    }
}

/// @title GasPayingToken_Set_Test
/// @notice Tests the `set` function of the `GasPayingToken` library.
contract GasPayingToken_Set_Test is GasPayingToken_TestInit {
    /// @notice Test that the gas paying token correctly sets values in storage.
    function testFuzz_set_succeeds(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external {
        GasPayingToken.set(_token, _decimals, _name, _symbol);

        // Check the token address and decimals
        assertEq(
            bytes32(uint256(_decimals) << 160 | uint256(uint160(_token))),
            vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_SLOT)
        );

        // Check the token name
        assertEq(_name, vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_NAME_SLOT));

        // Check the token symbol
        assertEq(_symbol, vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_SYMBOL_SLOT));
    }
}

/// @title GasPayingToken_Sanitize_Test
/// @notice Tests the `sanitize` function of the `GasPayingToken` library.
contract GasPayingToken_Sanitize_Test is GasPayingToken_TestInit {
    /// @notice Differentially test `sanitize`.
    function testDiff_sanitize_succeeds(string memory _str) external pure {
        vm.assume(bytes(_str).length <= 32);
        vm.assume(bytes(_str).length > 0);

        bytes32 output;
        uint256 len = bytes(_str).length;

        assembly {
            output := mload(add(_str, 0x20))
        }

        output = (output >> 32 - len) << 32 - len;

        assertEq(output, GasPayingToken.sanitize(_str));
    }

    /// @notice Test that `sanitize` fails when the input string is too long.
    function test_sanitize_stringTooLong_fails(string memory _str) external {
        vm.assume(bytes(_str).length > 32);

        vm.expectRevert("GasPayingToken: string cannot be greater than 32 bytes");

        harness.exposed_sanitize(_str);
    }

    /// @notice Test that `sanitize` works as expected when the input string is empty.
    function test_sanitize_empty_succeeds() external pure {
        assertEq(GasPayingToken.sanitize(""), "");
    }
}

/// @title GasPayingToken_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `GasPayingToken`
///         library or are testing multiple functions at once.
contract GasPayingToken_Uncategorized_Test is GasPayingToken_TestInit {
    /// @notice Test that the gas paying token correctly sets values in storage when input name
    ///         and symbol are strings.
    function testFuzz_setGetWithSanitize_succeeds(
        address _token,
        uint8 _decimals,
        string calldata _name,
        string calldata _symbol
    )
        external
    {
        vm.assume(_token != address(0));
        vm.assume(_token != Constants.ETHER);
        if (bytes(_name).length > 32) {
            _name = string(bytes(_name)[0:32]);
        }
        if (bytes(_symbol).length > 32) {
            _symbol = string(bytes(_symbol)[0:32]);
        }

        GasPayingToken.set(_token, _decimals, GasPayingToken.sanitize(_name), GasPayingToken.sanitize(_symbol));

        (address token, uint8 decimals) = GasPayingToken.getToken();
        assertEq(_token, token);
        assertEq(_decimals, decimals);

        assertEq(_name, GasPayingToken.getName());
        assertEq(_symbol, GasPayingToken.getSymbol());
    }
}
