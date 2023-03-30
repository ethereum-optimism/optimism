//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { Optimist } from "../universal/op-nft/Optimist.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

contract Optimist_Initializer is Test {
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event Initialized(uint8);

    string constant name = "Optimist name";
    string constant symbol = "OPTIMISTSYMBOL";
    string constant base_uri =
        "https://storageapi.fleek.co/6442819a1b05-bucket/optimist-nft/attributes";
    AttestationStation attestationStation;
    Optimist optimist;

    address internal carol_baseURIAttestor;
    address internal alice_allowlistAttestor;
    address internal ted_coinbaseAttestor;
    address internal bob;
    address internal sally;

    function attestBaseuri(string memory _baseUri) internal {
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        attestationData[0] = AttestationStation.AttestationData(
            address(optimist),
            bytes32("optimist.base-uri"),
            bytes(_baseUri)
        );
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);
    }

    function attestAllowlist(address _about) internal {
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non empty value
        attestationData[0] = AttestationStation.AttestationData({
            about: _about,
            key: bytes32("optimist.can-mint"),
            val: bytes("true")
        });
        vm.prank(alice_admin);
        attestationStation.attest(attestationData);
    }

    function setUp() public {
        carol_baseURIAttestor = makeAddr("carol_baseURIAttestor");
        alice_allowlistAttestor = makeAddr("alice_allowlistAttestor");
        ted_coinbaseAttestor = makeAddr("ted_coinbaseAttestor");
        bob = makeAddr("bob");
        sally = makeAddr("sally");

        // Give alice and bob and sally some ETH
        vm.deal(alice_admin, 1 ether);
        vm.deal(bob, 1 ether);
        vm.deal(sally, 1 ether);

        _initializeContracts();
    }

    function _initializeContracts() internal {
        attestationStation = new AttestationStation();
        vm.expectEmit(true, true, false, false);
        emit Initialized(1);

        OptimistAllowlist optimistAllowlist = new OptimistAllowlist(attestationStation, alice_admin, );
        optimist = new Optimist(name, symbol, alice_admin, attestationStation, optimistAllowlist);
    }
}

contract OptimistTest is Optimist_Initializer {
    function test_initialize_success() external {
        // expect name to be set
        assertEq(optimist.name(), name);
        // expect symbol to be set
        assertEq(optimist.symbol(), symbol);
        // expect attestationStation to be set
        assertEq(address(optimist.ATTESTATION_STATION()), address(attestationStation));
        assertEq(optimist.BASE_URI_ATTESTOR(), alice_admin);
        assertEq(optimist.version(), "1.0.0");
    }

    /**
     * @dev Bob should be able to mint an NFT if he is allowlisted
     * by the attestation station and has a balance of 0
     */
    function test_mint_happyPath_success() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // whitelist bob
        attestAllowlist(bob);

        uint256 tokenId = uint256(uint160(bob));
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, tokenId);

        bytes memory data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            alice_admin,
            bob,
            bytes32("optimist.can-mint")
        );
        vm.expectCall(address(attestationStation), data);
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
    function test_mint_secondaryMinter_succeeds() external {
        attestAllowlist(bob);

        bytes memory data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            alice_admin,
            bob,
            bytes32("optimist.can-mint")
        );
        vm.expectCall(address(attestationStation), data);

        uint256 tokenId = uint256(uint160(bob));
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, tokenId);

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
    function test_mint_forNonAllowlistedClaimer_reverts() external {
        vm.prank(bob);
        vm.expectRevert("Optimist: address is not on allowList");
        optimist.mint(bob);
    }

    /**
     * @dev Bob's tx should revert if he already minted
     */
    function test_mint_forAlreadyMintedClaimer_reverts() external {
        attestAllowlist(bob);

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
    function test_baseURI_returnsCorrectBaseURI_succeeds() external {
        attestBaseuri(base_uri);

        bytes memory data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            alice_admin,
            address(optimist),
            bytes32("optimist.base-uri")
        );
        vm.expectCall(address(attestationStation), data);
        vm.prank(alice_admin);

        // assert baseURI is set
        assertEq(optimist.baseURI(), base_uri);
    }

    /**
     * @dev The tokenURI should return the token uri
     * for a minted token
     */
    function test_tokenURI_returnsCorrectTokenURI_succeeds() external {
        attestAllowlist(bob);
        // we are using true but it can be any non empty value
        attestBaseuri(base_uri);

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
     * @dev Should return a boolean of if the address is allowlisted
     */
    function test_isOnAllowlist_returnsTrueForAllowlistedAddresses_succeeds() external {
        attestAllowlist(bob);

        bytes memory data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            alice_admin,
            bob,
            bytes32("optimist.can-mint")
        );
        vm.expectCall(address(attestationStation), data);
        // assert bob is whitelisted
        assertEq(optimist.isOnAllowList(bob), true);
        data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            alice_admin,
            sally,
            bytes32("optimist.can-mint")
        );
        vm.expectCall(address(attestationStation), data);
        // assert sally is not whitelisted
        assertEq(optimist.isOnAllowList(sally), false);
    }

    /**
     * @dev Should return the token id of the owner
     */
    function test_tokenIdOfAddress_returnsOwnerID_succeeds() external {
        // whitelist bob
        uint256 willTokenId = 1024;
        address will = address(1024);

        attestAllowlist(will);

        optimist.mint(will);

        assertEq(optimist.tokenIdOfAddress(will), willTokenId);
    }

    /**
     * @dev It should revert if anybody attemps token transfer
     */
    function test_transferFrom_reverts() external {
        attestAllowlist(bob);

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
        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.safeTransferFrom(bob, sally, 256, bytes("0x"));
    }

    /**
     * @dev It should revert if anybody attemps approve
     */
    function test_approve_reverts() external {
        attestAllowlist(bob);

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
    function test_burn_byOwner_succeeds() external {
        attestAllowlist(bob);

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
    function test_setApprovalForAll_reverts() external {
        attestAllowlist(bob);

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
    function test_supportsInterface_returnsCorrectInterfaceForERC721_succeeds() external {
        bytes4 iface721 = type(IERC721).interfaceId;
        // check that it supports erc721 interface
        assertEq(optimist.supportsInterface(iface721), true);
    }
}
