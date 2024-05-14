// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { L1BlockInterop, DependencySetSizeMismatch, NotDepositor, ConfigType } from "src/L2/L1BlockInterop.sol";

contract L1BlockInteropTest is Test {
    L1BlockInterop l1Block;
    address depositor;

    function setUp() public {
        l1Block = new L1BlockInterop();
        depositor = l1Block.DEPOSITOR_ACCOUNT();
    }

    /// @dev Tests that an arbitrary dependency set can be set and that ìsInDependencySet returns
    ///      the expected results.
    function testFuzz_isInDependencySet_succeeds(uint256[] calldata _dependencySet) external {
        vm.startPrank(depositor);

        for (uint256 i = 0; i < _dependencySet.length; i++) {
            // 0xfbb67fda52d4bfb8bf is Solady's EnumerableSetLib _ZERO_SENTINEL
            vm.assume(_dependencySet[i] != 0xfbb67fda52d4bfb8bf);
            l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(_dependencySet[i]));
            assertTrue(l1Block.isInDependencySet(_dependencySet[i]));
        }
    }

    /// @dev Tests that `isInDependencySet` returns true when the current chain ID is passed as the input
    function test_isInDependencySet_isChainId_succeeds() external view {
        assertTrue(l1Block.isInDependencySet(block.chainid));
    }

    /// @dev Tests that `isInDependencySet` reverts when the input chain ID is not in the dependency set
    function testFuzz_isInDependencySet_reverts(uint256 _chainId) external {
        vm.assume(_chainId != 1);

        // Add a chain to the dependency set that is not the chain ID
        vm.prank(depositor);
        l1Block.setConfig(ConfigType.ADD_DEPENDENCY, abi.encode(uint256(1)));

        // Check that the chain ID is not in the dependency set
        assertFalse(l1Block.isInDependencySet(_chainId));
    }

    /// @dev Tests that `isInDependencySet` returns false when the dependency set is empty
    function testFuzz_isInDependencySet_dependencySetEmpty_succeeds(uint256 _chainId) external view {
        assertTrue(l1Block.dependencySetSize() == 0);
        assertFalse(l1Block.isInDependencySet(_chainId));
    }
}
