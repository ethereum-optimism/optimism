// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title WETH_Version_Test
/// @notice Tests the `version` function of the `WETH` contract.
contract WETH_Version_Test is CommonTest {
    /// @notice Tests that the version returns a valid semver string.
    function test_version_succeeds() external view {
        SemverComp.parse(ISemver(address(weth)).version());
    }
}

/// @title WETH_Name_Test
/// @notice Tests the `name` function of the `WETH` contract.
contract WETH_Name_Test is CommonTest {
    /// @notice Tests that the `name` function returns the correct value.
    function testFuzz_name_succeeds(string memory _gasPayingTokenName) external {
        vm.mockCall(address(l1Block), abi.encodeCall(l1Block.gasPayingTokenName, ()), abi.encode(_gasPayingTokenName));

        assertEq(string.concat("Wrapped ", _gasPayingTokenName), weth.name());
    }

    /// @notice Tests that the `name` function returns 'Wrapped Ether' by default.
    function test_name_ether_succeeds() external view {
        assertEq("Wrapped Ether", weth.name());
    }
}

/// @title WETH_Symbol_Test
/// @notice Tests the `symbol` function of the `WETH` contract.
contract WETH_Symbol_Test is CommonTest {
    /// @notice Tests that the `symbol` function returns the correct value.
    function testFuzz_symbol_succeeds(string memory _gasPayingTokenSymbol) external {
        vm.mockCall(
            address(l1Block), abi.encodeCall(l1Block.gasPayingTokenSymbol, ()), abi.encode(_gasPayingTokenSymbol)
        );

        assertEq(string.concat("W", _gasPayingTokenSymbol), weth.symbol());
    }

    /// @notice Tests that the `symbol` function returns 'WETH' by default.
    function test_symbol_ether_succeeds() external view {
        assertEq("WETH", weth.symbol());
    }
}
