//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { SimpleStorage } from "../testing/helpers/SimpleStorage.sol";
import { MockTeleportr } from "../testing/helpers/MockTeleportr.sol";
import { TeleportrWithdrawer } from "../universal/TeleportrWithdrawer.sol";

contract TeleportrWithdrawer_Initializer is Test {
    address alice = address(128);
    address bob = address(256);

    TeleportrWithdrawer teleportrWithdrawer;
    MockTeleportr mockTeleportr;
    SimpleStorage simpleStorage;

    function _setUp() public {
        // Deploy MockTeleportr and SimpleStorage helper contracts
        mockTeleportr = new MockTeleportr();
        simpleStorage = new SimpleStorage();

        // Deploy Transactor contract
        teleportrWithdrawer = new TeleportrWithdrawer(address(alice));
        vm.label(address(teleportrWithdrawer), "TeleportrWithdrawer");

        // Give alice and bob some ETH
        vm.deal(alice, 1 ether);
        vm.deal(bob, 1 ether);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
    }
}

contract TeleportrWithdrawerTest is TeleportrWithdrawer_Initializer {
    function setUp() public {
        super._setUp();
    }

    // Tests if the owner was set correctly during deploy
    function test_constructor() external {
        assertEq(address(alice), teleportrWithdrawer.owner());
    }

    // Tests setRecipient function when called by authorized address
    function test_setRecipient() external {
        // Call setRecipient from alice
        vm.prank(alice);
        teleportrWithdrawer.setRecipient(address(alice));
        assertEq(teleportrWithdrawer.recipient(), address(alice));
    }

    // setRecipient should fail if called by unauthorized address
    function testFail_setRecipient() external {
        teleportrWithdrawer.setRecipient(address(alice));
        vm.expectRevert("UNAUTHORIZED");
    }

    // Tests setTeleportr function when called by authorized address
    function test_setTeleportr() external {
        // Call setRecipient from alice
        vm.prank(alice);
        teleportrWithdrawer.setTeleportr(address(mockTeleportr));
        assertEq(teleportrWithdrawer.teleportr(), address(mockTeleportr));
    }

    // setTeleportr should fail if called by unauthorized address
    function testFail_setTeleportr() external {
        teleportrWithdrawer.setTeleportr(address(bob));
        vm.expectRevert("UNAUTHORIZED");
    }

    // Tests setData function when called by authorized address
    function test_setData() external {
        bytes memory data = "0x1234567890";
        // Call setData from alice
        vm.prank(alice);
        teleportrWithdrawer.setData(data);
        assertEq(teleportrWithdrawer.data(), data);
    }

    // setData should fail if called by unauthorized address
    function testFail_setData() external {
        bytes memory data = "0x1234567890";
        teleportrWithdrawer.setData(data);
        vm.expectRevert("UNAUTHORIZED");
    }

    // Tests withdrawFromTeleportr, when called expected to withdraw the balance
    // to the recipient address when the target is an EOA
    function test_withdrawFromTeleportrToEOA() external {
        // Fund the Teleportr contract with 1 ETH
        vm.deal(address(teleportrWithdrawer), 1 ether);
        // Set target address and Teleportr
        vm.startPrank(alice);
        teleportrWithdrawer.setRecipient(address(bob));
        teleportrWithdrawer.setTeleportr(address(mockTeleportr));
        vm.stopPrank();
        // Run withdrawFromTeleportr
        assertEq(address(bob).balance, 1 ether);
        teleportrWithdrawer.withdrawFromTeleportr();
        assertEq(address(bob).balance, 2 ether);
    }

    // When called from a contract account it should withdraw the balance and trigger the code
    function test_withdrawFromTeleportrToContract() external {
        bytes32 key = 0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa;
        bytes32 value = 0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb;
        bytes memory data = abi.encodeWithSelector(simpleStorage.set.selector, key, value);
        // Fund the Teleportr contract with 1 ETH
        vm.deal(address(teleportrWithdrawer), 1 ether);
        // Set target address and Teleportr
        vm.startPrank(alice);
        teleportrWithdrawer.setRecipient(address(simpleStorage));
        teleportrWithdrawer.setTeleportr(address(mockTeleportr));
        teleportrWithdrawer.setData(data);
        vm.stopPrank();
        // Run withdrawFromTeleportr
        assertEq(address(simpleStorage).balance, 0);
        teleportrWithdrawer.withdrawFromTeleportr();
        assertEq(address(simpleStorage).balance, 1 ether);
        assertEq(simpleStorage.get(key), value);
    }
}
