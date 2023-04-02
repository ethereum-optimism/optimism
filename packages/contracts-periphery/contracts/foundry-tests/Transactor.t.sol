//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { CallRecorder } from "../testing/helpers/CallRecorder.sol";
import { Reverter } from "../testing/helpers/Reverter.sol";
import { Transactor } from "../universal/Transactor.sol";

contract Transactor_Initializer is Test {
    address alice = address(128);
    address bob = address(256);

    Transactor transactor;
    Reverter reverter;
    CallRecorder callRecorded;

    function setUp() public {
        // Deploy Reverter and CallRecorder helper contracts
        reverter = new Reverter();
        callRecorded = new CallRecorder();

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

contract TransactorTest is Transactor_Initializer {
    // Tests if the owner was set correctly during deploy
    function test_constructor() external {
        assertEq(address(alice), transactor.owner());
    }

    // Tests CALL, should do a call to target
    function test_CALL() external {
        // Initialize call data
        bytes memory data = abi.encodeWithSelector(callRecorded.record.selector);
        // Run CALL
        vm.prank(alice);
        vm.expectCall(address(callRecorded), 200_000 wei, data);
        transactor.CALL(address(callRecorded), data, 200_000 wei);
    }

    // It should revert if called by non-owner
    function testFail_CALL() external {
        // Initialize call data
        bytes memory data = abi.encodeWithSelector(callRecorded.record.selector);
        // Run CALL
        vm.prank(bob);
        transactor.CALL(address(callRecorded), data, 200_000 wei);
        vm.expectRevert("UNAUTHORIZED");
    }

    function test_DELEGATECALL() external {
        // Initialize call data
        bytes memory data = abi.encodeWithSelector(reverter.doRevert.selector);
        // Run CALL
        vm.prank(alice);
        vm.expectCall(address(reverter), data);
        transactor.DELEGATECALL(address(reverter), data);
    }

    // It should revert if called by non-owner
    function testFail_DELEGATECALLL() external {
        // Initialize call data
        bytes memory data = abi.encodeWithSelector(reverter.doRevert.selector);
        // Run CALL
        vm.prank(bob);
        transactor.DELEGATECALL(address(reverter), data);
        vm.expectRevert("UNAUTHORIZED");
    }
}
