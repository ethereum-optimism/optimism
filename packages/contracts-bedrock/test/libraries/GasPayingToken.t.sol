// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Target contract
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Test } from "forge-std/Test.sol";
import { Bytes } from "src/libraries/Bytes.sol";

/// @title GasPayingToken_Roundtrip_Test
/// @notice Tests the roundtrip of setting and getting the gas paying token.
contract GasPayingToken_Roundtrip_Test is Test {
    /// @notice Test that the gas paying token correctly sets values in storage
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

    /// @dev Test that the gas paying token returns values associated with Ether when unset
    function testFuzz_get_empty_succeeds() external {
        (address token, uint8 decimals) = GasPayingToken.getToken();
        assertEq(Constants.ETHER, token);
        assertEq(18, decimals);

        assertEq("Ether", GasPayingToken.getName());

        assertEq("ETH", GasPayingToken.getSymbol());
    }

    /// @dev Test that the gas paying token correctly gets values from storage when set
    function testFuzz_get_nonEmpty_succeeds(address _token, uint8 _decimals, bytes32 _name, bytes32 _symbol) external {
        vm.assume(_token != address(0));

        GasPayingToken.set(_token, _decimals, _name, _symbol);

        (address token, uint8 decimals) = GasPayingToken.getToken();
        assertEq(_token, token);
        assertEq(_decimals, decimals);

        assertEq(string(abi.encodePacked(_name)), GasPayingToken.getName());

        assertEq(string(abi.encodePacked(_symbol)), GasPayingToken.getSymbol());
    }
}
