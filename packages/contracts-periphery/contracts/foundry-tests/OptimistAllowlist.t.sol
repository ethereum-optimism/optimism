//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { OptimistAllowlist } from "../universal/op-nft/OptimistAllowlist.sol";
import { OptimistInviter } from "../universal/op-nft/OptimistInviter.sol";
import { OptimistInviterHelper } from "../testing/helpers/OptimistInviterHelper.sol";
import { OptimistConstants } from "../universal/op-nft/libraries/OptimistConstants.sol";

contract OptimistAllowlist_Initializer is Test {
    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );
    address internal alice_allowlistAttestor;
    address internal sally_coinbaseQuestAttestor;
    address internal ted;

    uint256 internal bobPrivateKey;
    address internal bob;

    AttestationStation attestationStation;
    OptimistAllowlist optimistAllowlist;
    OptimistInviter optimistInviter;

    // Helps with EIP-712 signature generation
    OptimistInviterHelper optimistInviterHelper;

    function setUp() public {
        alice_allowlistAttestor = makeAddr("alice_allowlistAttestor");
        sally_coinbaseQuestAttestor = makeAddr("sally_coinbaseQuestAttestor");
        ted = makeAddr("ted");

        bobPrivateKey = 0xB0B0B0B0;
        bob = vm.addr(bobPrivateKey);
        vm.label(bob, "bob");

        // Give alice and bob and sally some ETH
        vm.deal(alice_allowlistAttestor, 1 ether);
        vm.deal(sally_coinbaseQuestAttestor, 1 ether);
        vm.deal(bob, 1 ether);
        vm.deal(ted, 1 ether);

        _initializeContracts();
    }

    function attestAllowlist(address _about) internal {
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non empty value
        attestationData[0] = AttestationStation.AttestationData({
            about: _about,
            key: optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY(),
            val: bytes("true")
        });
        vm.prank(alice_allowlistAttestor);
        attestationStation.attest(attestationData);
    }

    function attestCoinbaseQuest(address _about) internal {
        AttestationStation.AttestationData[]
            memory attestationData = new AttestationStation.AttestationData[](1);
        // we are using true but it can be any non empty value
        attestationData[0] = AttestationStation.AttestationData({
            about: _about,
            key: optimistAllowlist.COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY(),
            val: bytes("true")
        });
        vm.prank(sally_coinbaseQuestAttestor);
        attestationStation.attest(attestationData);
    }

    function inviteAndClaim(address claimer) internal {
        address[] memory addresses = new address[](1);
        addresses[0] = bob;

        vm.prank(alice_allowlistAttestor);

        // grant invites to Bob;
        optimistInviter.setInviteCounts(addresses, 3);

        // issue a new invite
        OptimistInviter.ClaimableInvite memory claimableInvite = optimistInviterHelper
            .getClaimableInviteWithNewNonce(bob);

        // EIP-712 sign with Bob's private key
        bytes memory signature = _getSignature(
            bobPrivateKey,
            optimistInviterHelper.getDigest(claimableInvite)
        );

        bytes32 hashedCommit = keccak256(abi.encode(claimer, signature));

        // commit the invite
        vm.prank(claimer);
        optimistInviter.commitInvite(hashedCommit);

        // wait minimum commitment period
        vm.warp(optimistInviter.MIN_COMMITMENT_PERIOD() + block.timestamp);

        // reveal and claim the invite
        optimistInviter.claimInvite(claimer, claimableInvite, signature);
    }

    /**
     * @notice Get signature as a bytes blob, since SignatureChecker takes arbitrary signature blobs.
     *
     */
    function _getSignature(uint256 _signingPrivateKey, bytes32 _digest)
        internal
        pure
        returns (bytes memory)
    {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(_signingPrivateKey, _digest);

        bytes memory signature = abi.encodePacked(r, s, v);
        return signature;
    }

    function _initializeContracts() internal {
        attestationStation = new AttestationStation();

        optimistInviter = new OptimistInviter(alice_allowlistAttestor, attestationStation);
        optimistInviter.initialize("OptimistInviter");

        optimistAllowlist = new OptimistAllowlist(
            attestationStation,
            alice_allowlistAttestor,
            sally_coinbaseQuestAttestor,
            address(optimistInviter)
        );

        optimistInviterHelper = new OptimistInviterHelper(optimistInviter, "OptimistInviter");
    }
}

contract OptimistAllowlistTest is OptimistAllowlist_Initializer {
    function test_constructor_succeeds() external {
        // expect attestationStation to be set
        assertEq(address(optimistAllowlist.ATTESTATION_STATION()), address(attestationStation));
        assertEq(optimistAllowlist.ALLOWLIST_ATTESTOR(), alice_allowlistAttestor);
        assertEq(optimistAllowlist.COINBASE_QUEST_ATTESTOR(), sally_coinbaseQuestAttestor);
        assertEq(address(optimistAllowlist.OPTIMIST_INVITER()), address(optimistInviter));

        assertEq(optimistAllowlist.version(), "1.0.0");
    }

    /**
     * @notice Base case, a account without any relevant attestations should not be able to mint.
     */
    function test_isAllowedToMint_withoutAnyAttestations_fails() external {
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice After receiving a valid allowlist attestation, the account should be able to mint.
     */
    function test_isAllowedToMint_fromAllowlistAttestor_succeeds() external {
        attestAllowlist(bob);
        assertTrue(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice After receiving a valid attestation from the Coinbase Quest attestor,
     *         the account should be able to mint.
     */
    function test_isAllowedToMint_fromCoinbaseQuestAttestor_succeeds() external {
        attestCoinbaseQuest(bob);
        assertTrue(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Account that received an attestation from the OptimistInviter contract by going
     *         through the claim invite flow should be able to mint.
     */
    function test_isAllowedToMint_fromInvite_succeeds() external {
        inviteAndClaim(bob);
        assertTrue(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Attestation from the wrong allowlist attestor should not allow minting.
     */
    function test_isAllowedToMint_fromWrongAllowlistAttestor_fails() external {
        // Ted is not the allowlist attestor
        vm.prank(ted);
        attestationStation.attest(
            bob,
            optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY(),
            bytes("true")
        );
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Coinbase quest attestation from wrong attestor should not allow minting.
     */
    function test_isAllowedToMint_fromWrongCoinbaseQuestAttestor_fails() external {
        // Ted is not the coinbase quest attestor
        vm.prank(ted);
        attestationStation.attest(
            bob,
            optimistAllowlist.COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY(),
            bytes("true")
        );
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Claiming an invite on the non-official OptimistInviter contract should not allow
     *         minting.
     */
    function test_isAllowedToMint_fromWrongOptimistInviter_fails() external {
        vm.prank(ted);
        attestationStation.attest(
            bob,
            OptimistConstants.OPTIMIST_CAN_MINT_FROM_INVITE_ATTESTATION_KEY,
            bytes("true")
        );
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Having multiple signals, even if one is invalid, should still allow minting.
     */
    function test_isAllowedToMint_withMultipleAttestations_succeeds() external {
        attestAllowlist(bob);
        attestCoinbaseQuest(bob);
        inviteAndClaim(bob);

        assertTrue(optimistAllowlist.isAllowedToMint(bob));

        // A invalid attestation, as Ted is not allowlist attestor
        vm.prank(ted);
        attestationStation.attest(
            bob,
            optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY(),
            bytes("true")
        );

        // Since Bob has at least one valid attestation, he should be allowed to mint
        assertTrue(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Having falsy attestation value should not allow minting.
     */
    function test_isAllowedToMint_fromAllowlistAttestorWithFalsyValue_fails() external {
        // First sends correct attestation
        attestAllowlist(bob);

        bytes32 key = optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY();
        vm.expectEmit(true, true, true, false);
        emit AttestationCreated(alice_allowlistAttestor, bob, key, bytes("dsafsds"));

        // Invalidates existing attestation
        vm.prank(alice_allowlistAttestor);
        attestationStation.attest(bob, key, bytes(""));

        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    /**
     * @notice Having falsy attestation value from Coinbase attestor should not allow minting.
     */
    function test_isAllowedToMint_fromCoinbaseQuestAttestorWithFalsyValue_fails() external {
        // First sends correct attestation
        attestAllowlist(bob);

        bytes32 key = optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY();
        vm.expectEmit(true, true, true, true);
        emit AttestationCreated(alice_allowlistAttestor, bob, key, bytes(""));

        // Invalidates existing attestation
        vm.prank(alice_allowlistAttestor);
        attestationStation.attest(bob, key, bytes(""));

        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }
}
