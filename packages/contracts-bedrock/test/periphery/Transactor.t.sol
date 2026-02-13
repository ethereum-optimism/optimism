// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { Transactor } from "src/periphery/Transactor.sol";

/// @title Transactor_TestInit
/// @notice Reusable test initialization for `Transactor` tests.
abstract contract Transactor_TestInit is Test {
    address alice = address(128);
    address bob = address(256);

    Transactor transactor;

    function setUp() public {
        // Deploy Transactor contract
        transactor = new Transactor(address(alice));
        vm.label(address(transactor), "Transactor");

        // Give alice and bob some ETH
        vm.deal(alice, 1 ether);
        vm.deal(bob, 1 ether);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
    }
}

/// @title Transactor_Constructor_Test
/// @notice Tests the constructor of the `Transactor` contract.
contract Transactor_Constructor_Test is Transactor_TestInit {
    /// @notice Tests if the owner was set correctly during deploy
    function test_constructor_succeeds() external view {
        assertEq(address(alice), transactor.owner());
    }
}

/// @title Transactor_Call_Test
/// @notice Tests the `CALL` function of the `Transactor` contract.
contract Transactor_Call_Test is Transactor_TestInit {
    /// @notice Tests CALL, should do a call to target
    function test_call_succeeds() external {
        address target = makeAddr("target");
        // Initialize call data
        bytes memory data = hex"aabbccdd";
        // Run CALL
        vm.prank(alice);
        vm.expectCall(target, 200_000 wei, data);
        transactor.CALL(target, data, 200_000 wei);
    }

    /// @notice It should revert if called by non-owner
    function test_call_unauthorized_reverts() external {
        // Initialize call data
        address target = makeAddr("target");
        bytes memory data = hex"aabbccdd";
        // Run CALL
        vm.prank(bob);
        vm.expectRevert("UNAUTHORIZED");
        transactor.CALL(target, data, 200_000 wei);
    }
}

/// @title Transactor_DelegateCall_Test
/// @notice Tests the `DELEGATECALL` function of the `Transactor` contract.
contract Transactor_DelegateCall_Test is Transactor_TestInit {
    /// @notice Deletate call succeeds.
    function test_delegateCall_succeeds() external {
        // Initialize call data and target
        address target = address(0x1234);
        bytes memory data = hex"aabbccdd";
        // Run CALL
        vm.prank(alice);
        vm.expectCall(target, data);
        transactor.DELEGATECALL(target, data);
    }

    /// @notice It should revert if called by non-owner
    function test_delegateCall_unauthorized_reverts() external {
        // Initialize call data and target
        address target = address(0x1234);
        bytes memory data = hex"aabbccdd";
        // Run CALL
        vm.prank(bob);
        vm.expectRevert("UNAUTHORIZED");
        transactor.DELEGATECALL(target, data);
    }
}
