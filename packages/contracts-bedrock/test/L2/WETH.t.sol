// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract
import { WETH } from "src/L2/WETH.sol";

contract WETH_Test is CommonTest {
    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();
    }

    /// @dev Tests that the name function returns the correct value.
    function testFuzz_name_succeds(string memory _gasPayingTokenName) external {
        vm.mockCall(address(l1Block), abi.encodeWithSignature("gasPayingTokenName()"), abi.encode(_gasPayingTokenName));

        assertEq(string.concat("WETH ", _gasPayingTokenName), weth.name());
    }

    /// @dev Tests that the symbol function returns the correct value.
    function testFuzz_symbol_succeds(string memory _gasPayingTokenSymbol) external {
        vm.mockCall(
            address(l1Block), abi.encodeWithSignature("gasPayingTokenSymbol()"), abi.encode(_gasPayingTokenSymbol)
        );

        assertEq(string.concat("W", _gasPayingTokenSymbol), weth.symbol());
    }

    /// @dev Tests that the name function returns the correct value.
    function test_name_ether_succeds() external {
        assertEq("Wrapped Ether", weth.name());
    }

    /// @dev Tests that the symbol function returns the correct value.
    function test_symbol_ether_succeds() external {
        assertEq("WETH", weth.symbol());
    }
}
