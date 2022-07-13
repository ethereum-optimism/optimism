// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Badge } from "../../universal/citizen-house/Badge.sol";
import { Test } from "forge-std/Test.sol";

contract TBadge is Badge {
    constructor() Badge("example", "ex", "example.com") {}
}

contract BadgeTest is Test {
    TBadge internal badge;

    address deployer = 0xb4c79daB8f259C7Aee6E5b2Aa729821864227e84;
    address testAdr1 = makeAddr("admin");
    address citizen1 = makeAddr("citizen1");
    address citizen2 = makeAddr("citizen2");
    string baseURI = "ipfs://test";

    function setUp() public {
        badge = new TBadge();
    }

    // Expect revert when minting if caller address is not Admin
    function test_invalidMint() public {
        vm.prank(testAdr1);
        vm.expectRevert("Badge: Sender is not Admin");
        badge.mint(testAdr1);
    }

    // Expect revert when burning if caller address is not Admin
    function test_invalidBurn() public {
        vm.prank(testAdr1);
        vm.expectRevert("Badge: Sender is not Admin");
        badge.burn(0);
    }

    // Expect revert when updating admin contract address if caller address is not owner
    function test_invalidUpdateAdminContract() public {
        vm.prank(citizen1);
        vm.expectRevert("Ownable: caller is not the owner");
        badge.updateAdminContract(citizen1);
    }

    // Expect owner to be able to update admin contract address
    function test_updateAdminContract() public {
        vm.prank(deployer);
        badge.updateAdminContract(testAdr1);
    }

    // Expect admin contract to be able to mint
    function test_validMint() public {
        vm.prank(deployer);
        badge.updateAdminContract(testAdr1);
        vm.prank(testAdr1);
        badge.mint(citizen1);
    }

    // Expect admin contract to be able to burn
    function test_validBurn() public {
        vm.prank(deployer);
        badge.updateAdminContract(testAdr1);
        vm.prank(testAdr1);
        badge.mint(citizen1);
        vm.prank(testAdr1);
        badge.burn(0);
    }

    // Expect revert when updating baseURI if caller address is not owner
    function test_invalidUpdateBaseURI() public {
        vm.prank(testAdr1);
        vm.expectRevert("Ownable: caller is not the owner");
        badge.updateBaseURI(baseURI);
    }

    // Expect owner to be able to update baseURI
    function test_updateBaseURI() public {
        vm.prank(deployer);
        badge.updateBaseURI(baseURI);
    }

    // Expect revert a citizen calls transerFrom function
    function test_transferFromRevert() public {
        vm.prank(citizen1);
        vm.expectRevert("Badge: SOULBOUND");
        badge.transferFrom(citizen1, citizen2, 0);
    }

    // Expect revert a citizen calls approve function
    function test_approveRevert() public {
        vm.prank(citizen1);
        vm.expectRevert("Badge: SOULBOUND");
        badge.approve(citizen2, 0);
    }

    // Expect revert a citizen calls setApprovalForAll function
    function test_setApprovalForAllRevert() public {
        vm.prank(citizen1);
        vm.expectRevert("Badge: SOULBOUND");
        badge.setApprovalForAll(citizen2, true);
    }
}
