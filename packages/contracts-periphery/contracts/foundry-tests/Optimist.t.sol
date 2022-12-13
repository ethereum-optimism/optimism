//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { Optimist } from "../universal/op-nft/Optimist.sol";
import "@openzeppelin/contracts/utils/Strings.sol";

contract Optimist_Initializer is Test {
    address alice_admin = address(128);
    address bob = address(256);
    address sally = address(512);
    string name = "Optimist name";
    string symbol = "OPTIMISTSYMBOL";
    string base_uri = "https://storageapi.fleek.co/6442819a1b05-bucket/optimist-nft/attributes";
    AttestationStation attestationStation;
    Optimist optimist;

    function _setUp() public {
        // Give alice and bob and sally some ETH
        vm.deal(alice_admin, 1 ether);
        vm.deal(bob, 1 ether);
        vm.deal(sally, 1 ether);

        vm.label(alice_admin, "alice_admin");
        vm.label(bob, "bob");
        vm.label(sally, "sally");
        _initializeContracts();
    }

    function _initializeContracts() internal {
        attestationStation = new AttestationStation();
        optimist = new Optimist(name, symbol, alice_admin, attestationStation);
    }
}

contract OptimistTest is Optimist_Initializer {
    function setUp() public {
        super._setUp();
        _initializeContracts();
    }

    function test_optimist_initialize() external {
        // expect name to be set
        assertEq(optimist.name(), name);
        // expect symbol to be set
        assertEq(optimist.symbol(), symbol);
        // expect attestationStation to be set
        assertEq(address(optimist.ATTESTATION_STATION()), address(attestationStation));
    }

    /**
     * @dev Bob should be able to mint an NFT if he is whitelisted
     * by the attestation station and has a balance of 0
     */
    function test_optimist_mint_happy_path() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint an NFT
        vm.prank(bob);
        optimist.mint(bob);
        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(256), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @dev Sally should be able to mint a token on behalf of bob
     */
    function test_optimist_mint_secondary_minter() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as sally instead of bob
        vm.prank(sally);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(256), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @dev Bob should not be able to mint an NFT if he is not whitelisted
     */
    function test_optimist_mint_no_attestation() external {
        vm.prank(bob);
        vm.expectRevert("Optimist: address is not whitelisted");
        optimist.mint(bob);
    }

    /**
     * @dev Bob's tx should revert if he already minted
     */
    function test_optimist_mint_already_minted() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint initial nft with bob
        vm.prank(bob);
        optimist.mint(bob);
        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(256), bob);
        assertEq(optimist.balanceOf(bob), 1);

        // attempt to mint again
        vm.expectRevert("ERC721: token already minted");
        optimist.mint(bob);
    }

    /**
     * @dev The baseURI should be set by attestation station
     * by the owner of contract alice_admin
     */
    function test_optimist_baseURI() external {
        // set baseURI
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        attestationData[0] = AttestationStation.AttestationData({
            about: address(optimist),
            key: bytes32("optimist.base-uri"),
            val: bytes(base_uri)
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // assert baseURI is set
        assertEq(optimist.baseURI(), base_uri);
    }

    /**
     * @dev The tokenURI should return the token uri
     * for a minted token
     */
    function test_optimist_token_uri() external {
        // whitelist bob
        // attest baseURI
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](2);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        // we are using true but it can be any non null value
        attestationData[1] = AttestationStation.AttestationData({
            about: address(optimist),
            key: bytes32("optimist.base-uri"),
            val: bytes(base_uri)
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint an NFT
        vm.prank(bob);
        optimist.mint(bob);

        // assert tokenURI is set
        assertEq(optimist.baseURI(), base_uri);
        assertEq(
            optimist.tokenURI(256),
            // solhint-disable-next-line max-line-length
            "https://storageapi.fleek.co/6442819a1b05-bucket/optimist-nft/attributes/0x0000000000000000000000000000000000000100.json"
        );
    }

    /**
     * @dev Should return a boolean of if the address is whitelisted
     */
    function test_optimist_is_whitelisted() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // assert bob is whitelisted
        assertEq(optimist.isWhitelisted(bob), true);
        // assert sally is not whitelisted
        assertEq(optimist.isWhitelisted(sally), false);
    }

    /**
     * @dev Should return the token id of the owner
     */
    function test_optimist_token_id_of_owner() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // assert tokenid is correct
        uint256 expectedId = 256;
        assertEq(optimist.tokenIdOfAddress(address(bob)), expectedId);
    }

    /**
     * @dev It should revert if anybody attemps token transfer
     */
    function test_optimist_sbt_transfer() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.transferFrom(bob, sally, 256);

        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.safeTransferFrom(bob, sally, 256);
    }

    /**
     * @dev It should revert if anybody attemps approve
     */
    function test_optimist_sbt_approve() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // attempt to approve sally
        vm.prank(bob);
        vm.expectRevert("Optimist: soul bound token");
        optimist.approve(address(attestationStation), 256);

        assertEq(optimist.getApproved(256), address(0));
    }

    /**
     * @dev It should be able to burn token
     */
    function test_optimist_burn() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // burn as bob
        vm.prank(bob);
        optimist.burn(256);

        // expect bob to have no balance now
        assertEq(optimist.balanceOf(bob), 0);
    }

    /**
     * @dev setApprovalForAll should revert as sbt
     */
    function test_optimist_set_approval_for_all() external {
        // whitelist bob
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non null value
        attestationData[0] = AttestationStation.AttestationData({
            about: bob,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);
        vm.prank(alice_admin);
        vm.expectRevert(bytes("Optimist: soul bound token"));
        optimist.setApprovalForAll(alice_admin, true);

        // expect approval amount to stil be 0
        assertEq(optimist.getApproved(256), address(0));
        // isApprovedForAll should return false
        assertEq(optimist.isApprovedForAll(alice_admin, alice_admin), false);
    }

    /**
     * @dev should support erc721 interface
     */
    function test_optimist_supports_interface() external {
        bytes4 interface721 = 0x80ac58cd;
        // check that it supports erc721 interface
        assertEq(optimist.supportsInterface(interface721), true);
    }
}
