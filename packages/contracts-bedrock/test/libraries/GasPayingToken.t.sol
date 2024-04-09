// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Target contract
import { GasPayingToken } from "src/libraries/GasPayingToken.sol";
import { Test } from "forge-std/Test.sol";

/// @title GasPayingToken_Roundtrip_Test
/// @notice Tests the roundtrip of setting and getting the gas paying token.
contract GasPayingToken_Roundtrip_Test is Test {
    /// @notice Test that the gas paying token correctly sets values in storage
    function testFuzz_set_succeeds(
        address _token,
        uint8 _decimals,
        string memory _name,
        string memory _symbol
    )
        external
    {
        GasPayingToken.set(_token, _decimals, _name, _symbol);

        // Check the token address and decimals
        assertEq(
            bytes32(uint256(_decimals) << 160 | uint256(uint160(_token))),
            uint256(vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_SLOT()))
        );

        // Check the token name
        assertEq(
            bytes32(abi.encodePacked(_name)),
            uint256(vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_NAME_SLOT()))
        );

        // Check the token symbol
        assertEq(
            bytes32(abi.encodePacked(_symbol)),
            uint256(vm.load(address(this), GasPayingToken.GAS_PAYING_TOKEN_SYMBOL_SLOT()))
        );
    }

    /// @dev Test that the gas paying token correctly gets values from storage
    function testFuzz_setGet_succeeds(
        address _token,
        uint8 _decimals,
        string memory _name,
        string memory _symbol
    )
        external
    {
        GasPayingToken.set(_token, _decimals, _name, _symbol);

        assertEq((_token, _decimals), GasPayingToken.getToken());

        assertEq(_name, GasPayingToken.getName());

        assertEq(_symbol, GasPayingToken.getSymbol());
    }
}
