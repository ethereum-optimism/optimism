// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { BadgeAdmin } from "../../universal/citizen-house/BadgeAdmin.sol";
import { Badge } from "../../universal/citizen-house/Badge.sol";
import { Test, stdError } from "forge-std/Test.sol";

contract OPTests is Test {
    BadgeAdmin internal badgeAdmin;
    Badge internal badge;
    address opAdr = makeAddr("op"); //[0xa8B3478A436e8B909B5E9636090F2B15f9B311e7];
    address[] opAdrs = [opAdr];
    bytes32 testIPFSHash = 0x0170171c23281b16a3c58934162488ad6d039df686eca806f21eba0cebd03486;

    function setUp() public {
        badge = new Badge("example", "ex", "example.com");
        badgeAdmin = new BadgeAdmin(address(badge), 100, 100, 100, opAdrs);
    }

    // Expect to be able to add OPs
    function test_addOPs() public {
        vm.prank(opAdr);
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("alice");
        badgeAdmin.addOPs(testAdrs);
    }

    // Expect to be able to add OPCOs
    function test_addOPCOs() public {
        vm.prank(opAdr);
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("opco0");
        badgeAdmin.addOPCOs(testAdrs, new uint256[](1));
    }

    // Expect to be able to invalidate an OPCO
    function test_invalidateOPCO() public {
        vm.prank(opAdr);
        address opco0 = makeAddr("opco0");
        badgeAdmin.invalidateOPCO(opco0);
    }

    // Expect to be able to update OP metadata
    function test_updateOPMetadata() public {
        vm.prank(opAdr);
        bytes32 metadata = "metadata";
        badgeAdmin.updateOPMetadata(metadata);
    }

    // Expect revert when adding OPs because caller is not an OP
    function testFail_nonOPAddOPs() public {
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("opco0");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.addOPs(testAdrs);
        vm.expectRevert("BadgeAdmin: Invalid OP");
    }

    // Expect revert when adding OPCOs because caller is not an OP
    function testFail_nonOPAddOPCOs() public {
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("opco0");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.addOPCOs(testAdrs, new uint256[](1));
        vm.expectRevert("BadgeAdmin: Invalid OP");
    }

    // Expect revert when adding a duplicate OPCOs
    function testFail_noDuplicateOPCOs() public {
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("opco0");
        vm.startPrank(opAdr);
        badgeAdmin.addOPCOs(testAdrs, new uint256[](1));
        badgeAdmin.addOPCOs(testAdrs, new uint256[](1));
        vm.expectRevert("Address already OPCO");
    }

    // Expect revert when adding an OPCO that is a citizen
    function testFail_OPCOAddingAlreadyCitizen() public {
        address[] memory testAdrs = new address[](1);
        testAdrs[0] = makeAddr("citizen0");
        vm.startPrank(opAdr);
        badgeAdmin.addOPCOs(testAdrs, new uint256[](1));
        vm.expectRevert("Address already citizen");
    }

    // Expect revert when invalidating an OPCO because address is not an OP
    function testFail_nonOPInvalidateOPCO() public {
        address opco0 = makeAddr("opco0");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.invalidateOPCO(opco0);
        vm.expectRevert("BadgeAdmin: Invalid OP");
    }
}

contract OPCOTest is Test {
    BadgeAdmin internal badgeAdmin;
    Badge internal badge;
    address opAdr = makeAddr("op"); //[0xa8B3478A436e8B909B5E9636090F2B15f9B311e7];
    address[] opAdrs = [opAdr];
    bytes32 testIPFSHash = 0x0170171c23281b16a3c58934162488ad6d039df686eca806f21eba0cebd03486;

    function setUp() public {
        badge = new Badge("example", "ex", "example.com");
        badgeAdmin = new BadgeAdmin(address(badge), 100, 100, 100, opAdrs);

        address[] memory opcoAdrs = new address[](2);
        opcoAdrs[0] = makeAddr("opco0");
        opcoAdrs[1] = makeAddr("opco1");

        uint256[] memory opcoSupply = new uint256[](2);
        opcoSupply[0] = 15;
        opcoSupply[1] = 20;

        address[] memory citizenAdrs = new address[](4);
        citizenAdrs[0] = makeAddr("citizen0");
        citizenAdrs[1] = makeAddr("citizen1");
        citizenAdrs[2] = makeAddr("citizen2");
        citizenAdrs[3] = makeAddr("citizen3");

        vm.prank(opAdr);
        badgeAdmin.addOPCOs(opcoAdrs, opcoSupply);
        vm.prank(opcoAdrs[0]);
        badgeAdmin.addCitizens(citizenAdrs);

        vm.prank(0xb4c79daB8f259C7Aee6E5b2Aa729821864227e84); // deployer address
        badgeAdmin.updateBadgeContract(address(badge));
        badge.updateAdminContract(address(badgeAdmin));
    }

    // Expect OPCO caller to be able to add citizens
    function test_addCitizens() public {
        address[] memory testAdrs = new address[](1);
        address opco0 = makeAddr("opco0");
        testAdrs[0] = makeAddr("citizen42");
        vm.prank(opco0);
        badgeAdmin.addCitizens(testAdrs);
    }

    // Expect OPCO caller to be able to invalidate citizens
    function test_invalidateCitizen() public {
        address[] memory testAdrs = new address[](1);
        address opco0 = makeAddr("opco0");
        address citizen2 = makeAddr("citizen2");
        vm.prank(opco0);
        badgeAdmin.invalidateCitizen(citizen2);
    }

    // Expect OPCO caller to be able to update OPCO metadata
    function test_updateOPCOMetadata() public {
        bytes32 metadata = "metadata";
        address opco0 = makeAddr("opco0");
        vm.prank(opco0);
        badgeAdmin.updateOPCOMetadata(metadata);
    }

    // Expect OPCO caller to be able to remove a citizen
    function test_removeCitizen() public {
        address[] memory testAdrs = new address[](1);
        address opco0 = makeAddr("opco0");
        address citizen2 = makeAddr("citizen2");
        vm.prank(opco0);
        badgeAdmin.removeCitizen(citizen2);
    }

    // Expect revert when non-OPCO caller tries to add citizens
    function test_nonOPCOAddCitizensReverts() public {
        address[] memory testAdrs = new address[](1);
        address opco0 = makeAddr("opco0");
        testAdrs[0] = makeAddr("citizen42");
        vm.expectRevert("BadgeAdmin: Invalid OPCO");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.addCitizens(testAdrs);
    }

    // Expect revert when an OPCO caller tries to remove a citizen that was not added by the OPCO
    function test_notOPCOofCitzenRemovalReverts() public {
        address opco1 = makeAddr("opco1");
        address citizen2 = makeAddr("citizen2");
        vm.expectRevert("Not OPCO of Citizen");
        vm.prank(opco1);
        badgeAdmin.removeCitizen(citizen2);
    }

    // Expect revert when an OPCO caller attempts a tx with citizens.length > maxCitizenLimit
    function test_addCitizensExceedsMaxCitizenLimit() public {
        address[] memory testAdrs = new address[](1000);
        address opco0 = makeAddr("opco0");
        vm.expectRevert("Max Citizen limit exceeded");
        vm.prank(opco0);
        badgeAdmin.addCitizens(testAdrs);
    }

    // Expect revert when an OPCO caller attempts to exceed its supply
    function test_addCitizensExceedsSupplyReverts() public {
        address[] memory testAdrs = new address[](1000);
        address opco0 = makeAddr("opco0");
        vm.expectRevert("Max Citizen limit exceeded");
        vm.prank(opco0);
        badgeAdmin.addCitizens(testAdrs);
    }

    // Expect revert when an OPCO caller adds a citizen that is already a citizen
    function test_addCitizenAlreadyCitizenReverts() public {
        address[] memory testAdrs = new address[](1);
        address opco0 = makeAddr("opco0");
        address citizen2 = makeAddr("citizen2");
        testAdrs[0] = citizen2;
        vm.expectRevert("Address already Citizen");
        vm.prank(opco0);
        badgeAdmin.addCitizens(testAdrs);
    }

    // Expect revert when an OPCO caller attempts to do anything while invalidated
    function test_invalidatedOPCOReverts() public {
        address opco0 = makeAddr("opco0");
        bytes32 metadata = "metadata";

        vm.prank(opAdr);
        badgeAdmin.invalidateOPCO(opco0);

        vm.startPrank(opco0);

        vm.expectRevert("BadgeAdmin: Invalid OPCO");
        address[] memory adrs = new address[](1);
        adrs[0] = makeAddr("alice");
        badgeAdmin.addCitizens(adrs);

        vm.expectRevert("BadgeAdmin: Invalid OPCO");
        badgeAdmin.invalidateCitizen(makeAddr("citizen0"));

        vm.expectRevert("BadgeAdmin: Invalid OPCO");
        badgeAdmin.updateOPCOMetadata(metadata);
    }
}

contract CitizenTest is Test {
    BadgeAdmin internal badgeAdmin;
    Badge internal badge;
    address opAdr = makeAddr("op"); //[0xa8B3478A436e8B909B5E9636090F2B15f9B311e7];
    address[] opAdrs = [opAdr];
    bytes32 testIPFSHash = 0x0170171c23281b16a3c58934162488ad6d039df686eca806f21eba0cebd03486;

    function setUp() public {
        badge = new Badge("example", "ex", "example.com");
        badgeAdmin = new BadgeAdmin(address(badge), 100, 100, 100, opAdrs);

        address[] memory opcoAdrs = new address[](2);
        opcoAdrs[0] = makeAddr("opco0");
        opcoAdrs[1] = makeAddr("opco1");

        uint256[] memory opcoSupply = new uint256[](2);
        opcoSupply[0] = 15;
        opcoSupply[1] = 20;

        address[] memory citizenAdrs = new address[](4);
        citizenAdrs[0] = makeAddr("citizen0");
        citizenAdrs[1] = makeAddr("citizen1");
        citizenAdrs[2] = makeAddr("citizen2");
        citizenAdrs[3] = makeAddr("citizen3");

        vm.prank(opAdr);
        badgeAdmin.addOPCOs(opcoAdrs, opcoSupply);
        vm.prank(opcoAdrs[0]);
        badgeAdmin.addCitizens(citizenAdrs);

        vm.prank(0xb4c79daB8f259C7Aee6E5b2Aa729821864227e84); // deployer address
        badgeAdmin.updateBadgeContract(address(badge));
        badge.updateAdminContract(address(badgeAdmin));
    }

    // Expect citizen caller to be able to update citizen metadata
    function test_updateCitizenMetadata() public {
        bytes32 metadata = "metadata";
        address citizen0 = makeAddr("citizen0");
        vm.prank(citizen0);
        badgeAdmin.updateCitizenMetadata(metadata);
    }

    // Expect citizen caller to be able to mint a badge
    function test_mint() public {
        address citizen0 = makeAddr("citizen0");
        vm.prank(citizen0);
        badgeAdmin.mint();
    }

    // Expect citizen caller to be able to burn its badge
    function test_burn() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.burn(0);
    }

    // Expect citizen caller to be able to delegate to another citizen
    function test_delegate() public {
        address citizen0 = makeAddr("citizen0");
        address citizen1 = makeAddr("citizen1");
        vm.prank(citizen1);
        badgeAdmin.mint();
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.delegate(citizen1);
        assertTrue(badgeAdmin.getCitizen(citizen0).delegate == citizen1);
        assertTrue(badgeAdmin.getCitizen(citizen1).power == 2);
    }

    // Expect citizen caller to be able to undelegate its deleagted citizen to another citizen
    function test_undelegate() public {
        address citizen0 = makeAddr("citizen0");
        address citizen1 = makeAddr("citizen1");
        address citizen2 = makeAddr("citizen2");
        vm.prank(citizen1);
        badgeAdmin.mint();
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.delegate(citizen1);
        badgeAdmin.undelegate(citizen1);
        assertTrue(badgeAdmin.getCitizen(citizen0).delegate == address(0));
        assertTrue(badgeAdmin.getCitizen(citizen1).power == 1);
    }

    // Expect revert when non-citizen caller tries to update citizen metadata
    function test_nonCitizenUpdateCitizenMetadataReverts() public {
        bytes32 metadata = "metadata";
        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.updateCitizenMetadata(metadata);
    }

    // Expect revert when non-citizen caller tries to mint a badge
    function test_nonCitizenMintReverts() public {
        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.mint();
    }

    // Expect revert when non-citizen caller tries to burn a badge
    function test_nonCitizenBurnReverts() public {
        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.burn(0);
    }

    // Expect citizen caller to be able to vote
    function test_vote() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.vote(new bytes(64));
    }

    // Expect citizen caller to be able to override its vote
    function test_overrideVote() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.vote(new bytes(64));
        badgeAdmin.vote(new bytes(64));
    }

    // Expect revert when non-citizen caller tries to vote
    function test_nonCitizenVoteReverts() public {
        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        vm.prank(makeAddr("baddy"));
        badgeAdmin.vote(new bytes(64));
    }

    // Expect revert when citizen caller tries to vote without minting a badge
    function test_citizenVoteWithoutMintReverts() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        vm.expectRevert("Citizen has not minted");
        badgeAdmin.vote(new bytes(64));
    }

    // Expect revert when citizen has deleagted but tries to vote
    function test_citizenVoteWithDelegatedReverts() public {
        address citizen0 = makeAddr("citizen0");
        address citizen1 = makeAddr("citizen1");
        vm.prank(citizen1);
        badgeAdmin.mint();
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        badgeAdmin.delegate(citizen1);
        vm.expectRevert("Delegated to another citizen");
        badgeAdmin.vote(new bytes(64));
    }

    // Expect revert when citizen caller tries to do anything while invalidated
    function test_InvalidCitizenStatusReverts() public {
        address citizen0 = makeAddr("citizen0");
        address opco0 = makeAddr("opco0");
        bytes32 metadata = "metadata";
        vm.prank(citizen0);
        badgeAdmin.mint();
        vm.prank(opco0);
        badgeAdmin.invalidateCitizen(citizen0);

        vm.startPrank(citizen0);

        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.vote(new bytes(64));

        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.updateCitizenMetadata(metadata);

        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.mint();

        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.burn(0);

        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.delegate(makeAddr("nope"));
    }

    // Expect revert when citizen caller tries to transfer a badge
    function test_badgeTransferFromReverts() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        vm.expectRevert("Badge: SOULBOUND");
        badge.transferFrom(citizen0, makeAddr("nope"), 0);

        vm.expectRevert("Badge: SOULBOUND");
        badge.safeTransferFrom(citizen0, makeAddr("nope"), 0);

        vm.expectRevert("Badge: SOULBOUND");
        badge.safeTransferFrom(citizen0, makeAddr("nope"), 0, new bytes(64));
    }

    // Expect revert when citizen caller tries approve a transfer
    function test_badgeApproveReverts() public {
        address citizen0 = makeAddr("citizen0");
        vm.startPrank(citizen0);
        badgeAdmin.mint();
        vm.expectRevert("Badge: SOULBOUND");
        badge.approve(makeAddr("nope"), 0);

        vm.expectRevert("Badge: SOULBOUND");
        badge.setApprovalForAll(makeAddr("nope"), true);
    }

    // Expect revert when caller tries burning a badge that is not theirs
    function test_noncitizenBadgeBurnReverts() public {
        address citizen0 = makeAddr("citizen0");
        vm.prank(citizen0);
        badgeAdmin.mint();
        vm.prank(makeAddr("baddy"));
        vm.expectRevert("BadgeAdmin: Invalid Citizen");
        badgeAdmin.burn(0);
    }
}
