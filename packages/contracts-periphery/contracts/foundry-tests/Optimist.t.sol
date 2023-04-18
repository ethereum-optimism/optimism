//SPDX-License-Identifier: MIT
pragma solidity >=0.6.2 <0.9.0;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { Optimist } from "../universal/op-nft/Optimist.sol";
import { OptimistAllowlist } from "../universal/op-nft/OptimistAllowlist.sol";
import { OptimistInviter } from "../universal/op-nft/OptimistInviter.sol";
import { OptimistInviterHelper } from "../testing/helpers/OptimistInviterHelper.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

interface IMulticall3 {
    struct Call3 {
        address target;
        bool allowFailure;
        bytes callData;
    }

    struct Result {
        bool success;
        bytes returnData;
    }

    function aggregate3(Call3[] calldata calls)
        external
        payable
        returns (Result[] memory returnData);
}

contract Optimist_Initializer is Test {
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event Initialized(uint8);
    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    string constant name = "Optimist name";
    string constant symbol = "OPTIMISTSYMBOL";
    string constant base_uri =
        "https://storageapi.fleek.co/6442819a1b05-bucket/optimist-nft/attributes";
    AttestationStation attestationStation;
    Optimist optimist;
    OptimistAllowlist optimistAllowlist;
    OptimistInviter optimistInviter;

    // Helps with EIP-712 signature generation
    OptimistInviterHelper optimistInviterHelper;

    // To test multicall for claiming and minting in one call
    IMulticall3 multicall3;

    address internal carol_baseURIAttestor;
    address internal alice_allowlistAttestor;
    address internal eve_inviteGranter;
    address internal ted_coinbaseAttestor;
    address internal bob;
    address internal sally;

    /**
     * @notice BaseURI attestor sets the baseURI of the Optimist NFT.
     */
    function _attestBaseURI(string memory _baseUri) internal {
        bytes32 baseURIAttestationKey = optimist.BASE_URI_ATTESTATION_KEY();
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        attestationData[0] = AttestationStation.AttestationData(
            address(optimist),
            baseURIAttestationKey,
            bytes(_baseUri)
        );

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            carol_baseURIAttestor,
            address(optimist),
            baseURIAttestationKey,
            bytes(_baseUri)
        );
        vm.prank(carol_baseURIAttestor);
        attestationStation.attest(attestationData);
    }

    /**
     * @notice Allowlist attestor creates an attestation for an address.
     */
    function _attestAllowlist(address _about) internal {
        bytes32 attestationKey = optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY();
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non empty value
        attestationData[0] = AttestationStation.AttestationData({
            about: _about,
            key: attestationKey,
            val: bytes("true")
        });

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(alice_allowlistAttestor, _about, attestationKey, bytes("true"));

        vm.prank(alice_allowlistAttestor);
        attestationStation.attest(attestationData);

        assertTrue(optimist.isOnAllowList(_about));
    }

    /**
     * @notice Coinbase Quest attestor creates an attestation for an address.
     */
    function _attestCoinbaseQuest(address _about) internal {
        bytes32 attestationKey = optimistAllowlist.COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY();
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non empty value
        attestationData[0] = AttestationStation.AttestationData({
            about: _about,
            key: attestationKey,
            val: bytes("true")
        });

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(ted_coinbaseAttestor, _about, attestationKey, bytes("true"));

        vm.prank(ted_coinbaseAttestor);
        attestationStation.attest(attestationData);

        assertTrue(optimist.isOnAllowList(_about));
    }

    /**
     * @notice Issues invite, then claims it using the claimer's address.
     */
    function _inviteAndClaim(address _about) internal {
        uint256 inviterPrivateKey = 0xbeefbeef;
        address inviter = vm.addr(inviterPrivateKey);

        address[] memory addresses = new address[](1);
        addresses[0] = inviter;

        vm.prank(eve_inviteGranter);

        // grant invites to Inviter;
        optimistInviter.setInviteCounts(addresses, 3);

        // issue a new invite
        OptimistInviter.ClaimableInvite memory claimableInvite = optimistInviterHelper
            .getClaimableInviteWithNewNonce(inviter);

        // EIP-712 sign with Inviter's private key

        (uint8 v, bytes32 r, bytes32 s) = vm.sign(
            inviterPrivateKey,
            optimistInviterHelper.getDigest(claimableInvite)
        );
        bytes memory signature = abi.encodePacked(r, s, v);

        bytes32 hashedCommit = keccak256(abi.encode(_about, signature));

        // commit the invite
        vm.prank(_about);
        optimistInviter.commitInvite(hashedCommit);

        // wait minimum commitment period
        vm.warp(optimistInviter.MIN_COMMITMENT_PERIOD() + block.timestamp);

        // reveal and claim the invite
        optimistInviter.claimInvite(_about, claimableInvite, signature);

        assertTrue(optimist.isOnAllowList(_about));
    }

    /**
     * @notice Mocks the allowlistAttestor to always return true for a given address.
     */
    function _mockAllowlistTrueFor(address _claimer) internal {
        vm.mockCall(
            address(optimistAllowlist),
            abi.encodeWithSelector(OptimistAllowlist.isAllowedToMint.selector, _claimer),
            abi.encode(true)
        );

        assertTrue(optimist.isOnAllowList(_claimer));
    }

    /**
     * @notice Returns address as uint256.
     */
    function _getTokenId(address _owner) internal pure returns (uint256) {
        return uint256(uint160(address(_owner)));
    }

    function setUp() public {
        carol_baseURIAttestor = makeAddr("carol_baseURIAttestor");
        alice_allowlistAttestor = makeAddr("alice_allowlistAttestor");
        eve_inviteGranter = makeAddr("eve_inviteGranter");
        ted_coinbaseAttestor = makeAddr("ted_coinbaseAttestor");
        bob = makeAddr("bob");
        sally = makeAddr("sally");
        _initializeContracts();
    }

    function _initializeContracts() internal {
        attestationStation = new AttestationStation();
        vm.expectEmit(true, true, false, false);
        emit Initialized(1);

        optimistInviter = new OptimistInviter({
            _inviteGranter: eve_inviteGranter,
            _attestationStation: attestationStation
        });

        optimistInviter.initialize("OptimistInviter");

        // Initialize the helper which helps sign EIP-712 signatures
        optimistInviterHelper = new OptimistInviterHelper(optimistInviter, "OptimistInviter");

        optimistAllowlist = new OptimistAllowlist({
            _attestationStation: attestationStation,
            _allowlistAttestor: alice_allowlistAttestor,
            _coinbaseQuestAttestor: ted_coinbaseAttestor,
            _optimistInviter: address(optimistInviter)
        });

        optimist = new Optimist({
            _name: name,
            _symbol: symbol,
            _baseURIAttestor: carol_baseURIAttestor,
            _attestationStation: attestationStation,
            _optimistAllowlist: optimistAllowlist
        });

        // address test = deployCode("Multicall3.sol");
        multicall3 = IMulticall3(deployCode("Multicall3.sol"));
    }
}

contract OptimistTest is Optimist_Initializer {
    /**
     * @notice Check that constructor and initializer parameters are correctly set.
     */
    function test_initialize_success() external {
        // expect name to be set
        assertEq(optimist.name(), name);
        // expect symbol to be set
        assertEq(optimist.symbol(), symbol);
        // expect attestationStation to be set
        assertEq(address(optimist.ATTESTATION_STATION()), address(attestationStation));
        assertEq(optimist.BASE_URI_ATTESTOR(), carol_baseURIAttestor);
        assertEq(optimist.version(), "2.0.0");
    }

    /**
     * @notice Bob should be able to mint an NFT if he is allowlisted
     *         by the allowlistAttestor and has a balance of 0.
     */
    function test_mint_afterAllowlistAttestation_succeeds() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // allowlist bob
        _attestAllowlist(bob);

        assertTrue(optimistAllowlist.isAllowedToMint(bob));

        // Check that the OptimistAllowlist is checked
        bytes memory data = abi.encodeWithSelector(optimistAllowlist.isAllowedToMint.selector, bob);
        vm.expectCall(address(optimistAllowlist), data);

        // mint an NFT and expect mint transfer event to be emitted
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, _getTokenId(bob));
        vm.prank(bob);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Bob should be able to mint an NFT if he claimed an invite through OptimistInviter
     *          and has a balance of 0.
     */
    function test_mint_afterInviteClaimed_succeeds() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // bob claims an invite
        _inviteAndClaim(bob);

        assertTrue(optimistAllowlist.isAllowedToMint(bob));

        // Check that the OptimistAllowlist is checked
        bytes memory data = abi.encodeWithSelector(optimistAllowlist.isAllowedToMint.selector, bob);
        vm.expectCall(address(optimistAllowlist), data);

        // mint an NFT and expect mint transfer event to be emitted
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, _getTokenId(bob));
        vm.prank(bob);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Bob should be able to mint an NFT if he has an attestation from Coinbase Quest
     *         attestor and has a balance of 0.
     */
    function test_mint_afterCoinbaseQuestAttestation_succeeds() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // bob receives attestation from Coinbase Quest attestor
        _attestCoinbaseQuest(bob);

        assertTrue(optimistAllowlist.isAllowedToMint(bob));

        // Check that the OptimistAllowlist is checked
        bytes memory data = abi.encodeWithSelector(optimistAllowlist.isAllowedToMint.selector, bob);
        vm.expectCall(address(optimistAllowlist), data);

        // mint an NFT and expect mint transfer event to be emitted
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, _getTokenId(bob));
        vm.prank(bob);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Multiple valid attestations should allow Bob to mint.
     */
    function test_mint_afterMultipleAttestations_succeeds() external {
        // bob should start with 0 balance
        assertEq(optimist.balanceOf(bob), 0);

        // bob receives attestation from Coinbase Quest attestor
        _attestCoinbaseQuest(bob);

        // allowlist bob
        _attestAllowlist(bob);

        // bob claims an invite
        _inviteAndClaim(bob);

        assertTrue(optimistAllowlist.isAllowedToMint(bob));

        // Check that the OptimistAllowlist is checked
        bytes memory data = abi.encodeWithSelector(optimistAllowlist.isAllowedToMint.selector, bob);
        vm.expectCall(address(optimistAllowlist), data);

        // mint an NFT and expect mint transfer event to be emitted
        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, _getTokenId(bob));
        vm.prank(bob);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Sally should be able to mint a token on behalf of bob.
     */
    function test_mint_secondaryMinter_succeeds() external {
        _mockAllowlistTrueFor(bob);

        vm.expectEmit(true, true, true, true);
        emit Transfer(address(0), bob, _getTokenId(bob));

        // mint as sally instead of bob
        vm.prank(sally);
        optimist.mint(bob);

        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Bob should not be able to mint an NFT if he is not allowlisted.
     */
    function test_mint_forNonAllowlistedClaimer_reverts() external {
        vm.prank(bob);
        vm.expectRevert("Optimist: address is not on allowList");
        optimist.mint(bob);
    }

    /**
     * @notice Bob's tx should revert if he already minted.
     */
    function test_mint_forAlreadyMintedClaimer_reverts() external {
        _attestAllowlist(bob);

        // mint initial nft with bob
        vm.prank(bob);
        optimist.mint(bob);
        // expect the NFT to be owned by bob
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);

        // attempt to mint again
        vm.expectRevert("ERC721: token already minted");
        optimist.mint(bob);
    }

    /**
     * @notice The baseURI should be set by attestation station by the baseURIAttestor.
     */
    function test_baseURI_returnsCorrectBaseURI_succeeds() external {
        _attestBaseURI(base_uri);

        bytes memory data = abi.encodeWithSelector(
            attestationStation.attestations.selector,
            carol_baseURIAttestor,
            address(optimist),
            optimist.BASE_URI_ATTESTATION_KEY()
        );
        vm.expectCall(address(attestationStation), data);
        vm.prank(carol_baseURIAttestor);

        // assert baseURI is set
        assertEq(optimist.baseURI(), base_uri);
    }

    /**
     * @notice tokenURI should return the token uri for a minted token.
     */
    function test_tokenURI_returnsCorrectTokenURI_succeeds() external {
        // we are using true but it can be any non empty value
        _attestBaseURI(base_uri);

        // mint an NFT
        _mockAllowlistTrueFor(bob);
        vm.prank(bob);
        optimist.mint(bob);

        // assert tokenURI is set
        assertEq(optimist.baseURI(), base_uri);
        assertEq(
            optimist.tokenURI(_getTokenId(bob)),
            "https://storageapi.fleek.co/6442819a1b05-bucket/optimist-nft/attributes/0x1d96f2f6bef1202e4ce1ff6dad0c2cb002861d3e.json"
        );
    }

    /**
     * @notice Should return the token id of the owner.
     */
    function test_tokenIdOfAddress_returnsOwnerID_succeeds() external {
        uint256 willTokenId = 1024;
        address will = address(1024);

        _mockAllowlistTrueFor(will);

        optimist.mint(will);

        assertEq(optimist.tokenIdOfAddress(will), willTokenId);
    }

    /**
     * @notice transferFrom should revert since Optimist is a SBT.
     */
    function test_transferFrom_reverts() external {
        _mockAllowlistTrueFor(bob);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.transferFrom(bob, sally, _getTokenId(bob));

        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.safeTransferFrom(bob, sally, _getTokenId(bob));
        // attempt to transfer to sally
        vm.expectRevert(bytes("Optimist: soul bound token"));
        vm.prank(bob);
        optimist.safeTransferFrom(bob, sally, _getTokenId(bob), bytes("0x"));
    }

    /**
     * @notice approve should revert since Optimist is a SBT.
     */
    function test_approve_reverts() external {
        _mockAllowlistTrueFor(bob);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // attempt to approve sally
        vm.prank(bob);
        vm.expectRevert("Optimist: soul bound token");
        optimist.approve(address(attestationStation), _getTokenId(bob));

        assertEq(optimist.getApproved(_getTokenId(bob)), address(0));
    }

    /**
     * @notice setApprovalForAll should revert since Optimist is a SBT.
     */
    function test_setApprovalForAll_reverts() external {
        _mockAllowlistTrueFor(bob);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);
        vm.prank(alice_allowlistAttestor);
        vm.expectRevert(bytes("Optimist: soul bound token"));
        optimist.setApprovalForAll(alice_allowlistAttestor, true);

        // expect approval amount to stil be 0
        assertEq(optimist.getApproved(_getTokenId(bob)), address(0));
        // isApprovedForAll should return false
        assertEq(
            optimist.isApprovedForAll(alice_allowlistAttestor, alice_allowlistAttestor),
            false
        );
    }

    /**
     * @notice Only owner should be able to burn token.
     */
    function test_burn_byOwner_succeeds() external {
        _mockAllowlistTrueFor(bob);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        // burn as bob
        vm.prank(bob);
        optimist.burn(_getTokenId(bob));

        // expect bob to have no balance now
        assertEq(optimist.balanceOf(bob), 0);
    }

    /**
     * @notice Non-owner attempting to burn token should revert.
     */
    function test_burn_byNonOwner_reverts() external {
        _mockAllowlistTrueFor(bob);

        // mint as bob
        vm.prank(bob);
        optimist.mint(bob);

        vm.expectRevert("ERC721: caller is not token owner nor approved");
        // burn as Sally
        vm.prank(sally);
        optimist.burn(_getTokenId(bob));

        // expect bob to have still have the token
        assertEq(optimist.balanceOf(bob), 1);
    }

    /**
     * @notice Should support ERC-721 interface.
     */
    function test_supportsInterface_returnsCorrectInterfaceForERC721_succeeds() external {
        bytes4 iface721 = type(IERC721).interfaceId;
        // check that it supports ERC-721 interface
        assertEq(optimist.supportsInterface(iface721), true);
    }

    /**
     * @notice Checking that multi-call using the invite & claim flow works correctly, since the
     *         frontend will be making multicalls to improve UX. The OptimistInviter.claimInvite
     *         and Optimist.mint will be batched
     */
    function test_multicall_batchingClaimAndMint_succeeds() external {
        uint256 inviterPrivateKey = 0xbeefbeef;
        address inviter = vm.addr(inviterPrivateKey);

        address[] memory addresses = new address[](1);
        addresses[0] = inviter;

        vm.prank(eve_inviteGranter);

        // grant invites to Inviter;
        optimistInviter.setInviteCounts(addresses, 3);

        // issue a new invite
        OptimistInviter.ClaimableInvite memory claimableInvite = optimistInviterHelper
            .getClaimableInviteWithNewNonce(inviter);

        // EIP-712 sign with Inviter's private key

        (uint8 v, bytes32 r, bytes32 s) = vm.sign(
            inviterPrivateKey,
            optimistInviterHelper.getDigest(claimableInvite)
        );
        bytes memory signature = abi.encodePacked(r, s, v);

        bytes32 hashedCommit = keccak256(abi.encode(bob, signature));

        // commit the invite
        vm.prank(bob);
        optimistInviter.commitInvite(hashedCommit);

        // wait minimum commitment period
        vm.warp(optimistInviter.MIN_COMMITMENT_PERIOD() + block.timestamp);

        IMulticall3.Call3[] memory calls = new IMulticall3.Call3[](2);

        // First call is to claim the invite, receiving the attestation
        calls[0] = IMulticall3.Call3({
            target: address(optimistInviter),
            callData: abi.encodeWithSelector(
                optimistInviter.claimInvite.selector,
                bob,
                claimableInvite,
                signature
            ),
            allowFailure: false
        });

        // Second call is to mint the Optimist NFT
        calls[1] = IMulticall3.Call3({
            target: address(optimist),
            callData: abi.encodeWithSelector(optimist.mint.selector, bob),
            allowFailure: false
        });

        multicall3.aggregate3(calls);

        assertTrue(optimist.isOnAllowList(bob));
        assertEq(optimist.ownerOf(_getTokenId(bob)), bob);
        assertEq(optimist.balanceOf(bob), 1);
    }
}
