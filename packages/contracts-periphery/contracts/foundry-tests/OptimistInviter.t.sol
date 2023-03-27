//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { OptimistInviter } from "../universal/op-nft/OptimistInviter.sol";
import { Optimist } from "../universal/op-nft/Optimist.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { TestERC1271Wallet } from "../testing/helpers/TestERC1271Wallet.sol";

contract OptimistInviter_Initializer is Test {
    event InviteClaimed(address indexed issuer, address indexed claimer);
    event Initialized(uint8 version);
    event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    bytes32 CLAIMABLE_INVITE_TYPEHASH;
    bytes32 EIP712_DOMAIN_TYPEHASH;

    address constant alice_inviteGranter = address(128);
    address constant sally = address(512);
    address constant ted = address(1024);
    address constant eve = address(2048);

    address internal bob;
    uint256 internal bobPrivateKey;
    address internal carol;
    uint256 internal carolPrivateKey;

    uint256 currentNonce;

    TestERC1271Wallet carolERC1271Wallet;

    AttestationStation attestationStation;
    OptimistInviter optimistInviter;

    function setUp() public {
        currentNonce = 0;

        bobPrivateKey = 0xB0B0B0B0;
        bob = vm.addr(bobPrivateKey);

        carolPrivateKey = 0xC0C0C0C0;
        carol = vm.addr(carolPrivateKey);

        carolERC1271Wallet = new TestERC1271Wallet(carol);

        // Give alice and bob and sally some ETH
        vm.deal(alice_inviteGranter, 1 ether);
        vm.deal(bob, 1 ether);
        vm.deal(sally, 1 ether);
        vm.deal(eve, 1 ether);

        vm.label(alice_inviteGranter, "alice_inviteGranter");
        vm.label(bob, "bob");
        vm.label(sally, "sally");
        vm.label(carol, "carol");

        CLAIMABLE_INVITE_TYPEHASH = keccak256("ClaimableInvite(address issuer,bytes32 nonce)");
        EIP712_DOMAIN_TYPEHASH = keccak256(
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
        );

        _initializeContracts();
    }

    /**
     * @notice Instantiates an AttestationStation, and an OptimistInviter.
     */
    function _initializeContracts() internal {
        attestationStation = new AttestationStation();
        vm.expectEmit(false, false, false, false);
        emit Initialized(1);

        optimistInviter = new OptimistInviter(alice_inviteGranter, attestationStation);
        optimistInviter.initialize("OptimistInviter");
    }

    /**
     * @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
     *         pseudorandom nonces.
     */
    function _consumeNonce() internal returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /**
     * @notice Returns a user's current invite count, as stored in the AttestationStation.
     */
    function _getInviteCount(address _issuer) internal view returns (uint256) {
        bytes memory attestation = attestationStation.attestations(
            address(optimistInviter),
            _issuer,
            bytes32("optimist.can-invite")
        );
        return abi.decode(attestation, (uint256));
    }

    /**
     * @notice Returns true if claimer has the proper attestation from OptimistInviter to mint.
     */
    function _hasMintAttestation(address _claimer) internal view returns (bool) {
        bytes memory attestation = attestationStation.attestations(
            address(optimistInviter),
            _claimer,
            bytes32("optimist.can-mint-from-invite")
        );
        return attestation.length > 0;
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

    /**
     * @notice Signs a claimable invite with the given private key and returns the signature using
     *         correct EIP712 domain separator.
     */
    function _issueInviteAs(uint256 _privateKey)
        internal
        returns (OptimistInviter.ClaimableInvite memory, bytes memory)
    {
        return
            _issueInviteWithEIP712Domain(
                _privateKey,
                bytes("OptimistInviter"),
                bytes(optimistInviter.version()),
                block.chainid,
                address(optimistInviter)
            );
    }

    /**
     * @notice Signs a claimable invite with the given private key and returns the signature using
     *         the given EIP712 domain separator. This assumes that the issuer's address is the
     *         corresponding public key to _issuerPrivateKey.
     */
    function _issueInviteWithEIP712Domain(
        uint256 _issuerPrivateKey,
        bytes memory _eip712Name,
        bytes memory _eip712Version,
        uint256 _eip712Chainid,
        address _eip712VerifyingContract
    ) internal returns (OptimistInviter.ClaimableInvite memory, bytes memory) {
        bytes32 nonce = _consumeNonce();
        address issuer = vm.addr(_issuerPrivateKey);
        OptimistInviter.ClaimableInvite memory claimableInvite = OptimistInviter.ClaimableInvite(
            issuer,
            nonce
        );

        return (
            claimableInvite,
            _getSignature(
                _issuerPrivateKey,
                _getEIP712Digest(
                    claimableInvite,
                    _eip712Name,
                    _eip712Version,
                    _eip712Chainid,
                    _eip712VerifyingContract
                )
            )
        );
    }

    /**
     * @notice Commits a signature and claimer address to the OptimistInviter contract.
     */
    function _commitInviteAs(address _as, bytes memory _signature) internal {
        vm.prank(_as);
        bytes32 hashedSignature = keccak256(abi.encode(_as, _signature));
        optimistInviter.commitInvite(hashedSignature);

        // Check that the commitment was stored correctly
        assertTrue(optimistInviter.commitments(hashedSignature));
    }

    /**
     * @notice Signs a claimable invite with the given private key. The claimer commits then claims
     *         the invite. Checks that all expected events are emitted and that state is updated
     *         correctly. Returns the signature and invite for use in tests.
     */
    function _issueThenClaimShouldSucceed(uint256 _issuerPrivateKey, address _claimer)
        internal
        returns (OptimistInviter.ClaimableInvite memory, bytes memory)
    {
        address issuer = vm.addr(_issuerPrivateKey);
        uint256 prevInviteCount = _getInviteCount(issuer);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteAs(_issuerPrivateKey);

        _commitInviteAs(_claimer, signature);

        // The hash(claimer ++ signature) should be committed
        assertEq(optimistInviter.commitments(keccak256(abi.encode(_claimer, signature))), true);

        // OptimistInviter should issue a new attestation allowing claimer to mint
        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            _claimer,
            bytes32("optimist.can-mint-from-invite"),
            abi.encode(issuer)
        );

        // OptimistInviter should issue a new attestation with updated invite count
        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            issuer,
            bytes32("optimist.can-invite"),
            abi.encode(prevInviteCount - 1)
        );

        // Should emit an event indicating that the invite was claimed
        vm.expectEmit(true, false, false, false, address(optimistInviter));
        emit InviteClaimed(issuer, _claimer);

        vm.prank(_claimer);
        optimistInviter.claimInvite(_claimer, claimableInvite, signature);

        // The nonce that issuer used should be marked as used
        assertTrue(optimistInviter.usedNonces(issuer, claimableInvite.nonce));

        // Issuer should have one less invite
        assertEq(prevInviteCount - 1, _getInviteCount(issuer));

        // Claimer should have the mint attestation from the OptimistInviter contract
        assertTrue(_hasMintAttestation(_claimer));

        return (claimableInvite, signature);
    }

    /**
     * @notice Issues 3 invites to the given address. Checks that all expected events are emitted
     *         and that state is updated correctly.
     */
    function _grantInvitesTo(address _to) internal {
        address[] memory addresses = new address[](1);
        addresses[0] = _to;

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            _to,
            bytes32("optimist.can-invite"),
            abi.encode(3)
        );

        vm.prank(alice_inviteGranter);
        optimistInviter.setInviteCounts(addresses, 3);

        assertEq(_getInviteCount(_to), 3);
    }

    /**
     * @notice Gets the hashed typed object for a EIP712 signature.
     */
    function _getStructHash(OptimistInviter.ClaimableInvite memory _claimableInvite)
        internal
        view
        returns (bytes32)
    {
        return
            keccak256(
                abi.encode(
                    CLAIMABLE_INVITE_TYPEHASH,
                    _claimableInvite.issuer,
                    _claimableInvite.nonce
                )
            );
    }

    /**
     * @notice Gets the signable digest for a EIP712 signature.
     */
    function _getEIP712Digest(
        OptimistInviter.ClaimableInvite memory _claimableInvite,
        bytes memory _name,
        bytes memory _version,
        uint256 _chainid,
        address _verifyingContract
    ) internal view returns (bytes32) {
        bytes32 domainSeparator = keccak256(
            abi.encode(
                EIP712_DOMAIN_TYPEHASH,
                keccak256(_name),
                keccak256(_version),
                _chainid,
                _verifyingContract
            )
        );
        return
            keccak256(
                abi.encodePacked("\x19\x01", domainSeparator, _getStructHash(_claimableInvite))
            );
    }
}

contract OptimistInviterTest is OptimistInviter_Initializer {
    function test_initialize() external {
        // expect attestationStation to be set
        assertEq(address(optimistInviter.ATTESTATION_STATION()), address(attestationStation));
        assertEq(optimistInviter.INVITE_GRANTER(), alice_inviteGranter);
        assertEq(optimistInviter.version(), "1.0.0");
    }

    /**
     * @notice Alice the admin should be able to give Bob, Sally, and Carol 3 invites, and the
     *      OptimistInviter contract should create invites as 'optimist.can-invite' attestations.
     */
    function test_grantInvites_adminAddingInvites_succeeds() external {
        address[] memory addresses = new address[](3);
        addresses[0] = bob;
        addresses[1] = sally;
        addresses[2] = address(carolERC1271Wallet);

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            bob,
            bytes32("optimist.can-invite"),
            abi.encode(3)
        );

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            sally,
            bytes32("optimist.can-invite"),
            abi.encode(3)
        );

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            address(carolERC1271Wallet),
            bytes32("optimist.can-invite"),
            abi.encode(3)
        );

        vm.prank(alice_inviteGranter);
        optimistInviter.setInviteCounts(addresses, 3);

        assertEq(_getInviteCount(bob), 3);
        assertEq(_getInviteCount(sally), 3);
        assertEq(_getInviteCount(address(carolERC1271Wallet)), 3);
    }

    /**
     * @notice Bob, who is not the invite granter, should not be able to issue invites.
     */
    function test_grantInvites_nonAdminAddingInvites_reverts() external {
        address[] memory addresses = new address[](2);
        addresses[0] = bob;
        addresses[1] = sally;

        vm.expectRevert("OptimistInviter: only invite granter can grant invites");
        vm.prank(bob);
        optimistInviter.setInviteCounts(addresses, 3);
    }

    /**
     * @notice Sally should be able to commit an invite given by by Bob.
     */
    function test_commitInvite_committingForYourself_succeeds() external {
        _grantInvitesTo(bob);
        (, bytes memory signature) = _issueInviteAs(bobPrivateKey);

        vm.prank(sally);
        bytes32 hashedSignature = keccak256(abi.encode(sally, signature));
        optimistInviter.commitInvite(hashedSignature);

        assertTrue(optimistInviter.commitments(hashedSignature));
    }

    /**
     * @notice Sally should be able to Bob's for a different claimer, Eve.
     */
    function test_commitInvite_committingForSomeoneElse_succeeds() external {
        _grantInvitesTo(bob);
        (, bytes memory signature) = _issueInviteAs(bobPrivateKey);

        vm.prank(sally);
        bytes32 hashedSignature = keccak256(abi.encode(eve, signature));
        optimistInviter.commitInvite(hashedSignature);

        assertTrue(optimistInviter.commitments(hashedSignature));
    }

    /**
     * @notice Bob issues signature, and Sally claims the invite. Bob's invite count should be
     *         decremented, and Sally should be able to mint.
     */
    function test_claimInvite_succeeds() external {
        _grantInvitesTo(bob);
        _issueThenClaimShouldSucceed(bobPrivateKey, sally);
    }

    /**
     * @notice Bob issues signature, and Ted commits the invite for Sally. Eve claims for Sally.
     */
    function test_claimInvite_claimForSomeoneElse_succeeds() external {
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteAs(bobPrivateKey);

        _commitInviteAs(sally, signature);

        vm.prank(ted);
        optimistInviter.commitInvite(keccak256(abi.encode(sally, signature)));

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            sally,
            bytes32("optimist.can-mint-from-invite"),
            abi.encode(bob)
        );

        // OptimistInviter should issue a new attestation with updated invite count
        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            bob,
            bytes32("optimist.can-invite"),
            abi.encode(2)
        );

        // Should emit an event indicating that the invite was claimed
        vm.expectEmit(true, false, false, false, address(optimistInviter));
        emit InviteClaimed(bob, sally);

        vm.prank(eve);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
        assertTrue(_hasMintAttestation(sally));
        assertFalse(_hasMintAttestation(eve));
    }

    /**
     * @notice Signature issued for previous versions of the contract should fail.
     */
    function test_claimInvite_usingSignatureIssuedForDifferentVersion_reverts() external {
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteWithEIP712Domain(
                bobPrivateKey,
                "OptimismInviter",
                "0.9.1",
                block.chainid,
                address(optimistInviter)
            );

        _commitInviteAs(sally, signature);

        vm.expectRevert("OptimistInviter: invalid signature");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Replay attack for signature issued for contract on different chain (ie. mainnet)
     *         should fail.
     */
    function test_claimInvite_usingSignatureIssuedForDifferentChain_reverts() external {
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteWithEIP712Domain(
                bobPrivateKey,
                "OptimismInviter",
                bytes(optimistInviter.version()),
                1,
                address(optimistInviter)
            );

        _commitInviteAs(sally, signature);

        vm.expectRevert("OptimistInviter: invalid signature");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Replay attack for signature issued for instantiation of the OptimistInviter contract
     *         on a different address should fail.
     */
    function test_claimInvite_usingSignatureIssuedForDifferentContract_reverts() external {
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteWithEIP712Domain(
                bobPrivateKey,
                "OptimismInviter",
                bytes(optimistInviter.version()),
                block.chainid,
                address(0xBEEF)
            );

        _commitInviteAs(sally, signature);

        vm.expectRevert("OptimistInviter: invalid signature");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Attempting to claim again using the same signature again should fail.
     */
    function test_claimInvite_replayingUsedNonce_reverts() external {
        _grantInvitesTo(bob);

        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueThenClaimShouldSucceed(bobPrivateKey, sally);

        // Sally tries to claim the invite using the same signature
        _commitInviteAs(sally, signature);
        vm.expectRevert("OptimistInviter: nonce has already been used");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);

        // Carol tries to claim the invite using the same signature
        _commitInviteAs(carol, signature);
        vm.expectRevert("OptimistInviter: nonce has already been used");
        vm.prank(carol);
        optimistInviter.claimInvite(carol, claimableInvite, signature);
    }

    /**
     * @notice Issuing signatures through a contract that implements ERC1271 should succeed (ie.
     *         Gnosis Safe or other smart contract wallets). Carol is using a ERC1271 contract
     *         wallet that is simply backed by her private key.
     */
    function test_claimInvite_usingERC1271Wallet_succeeds() external {
        _grantInvitesTo(address(carolERC1271Wallet));

        bytes32 nonce = _consumeNonce();
        OptimistInviter.ClaimableInvite memory claimableInvite = OptimistInviter.ClaimableInvite(
            address(carolERC1271Wallet),
            nonce
        );

        bytes memory signature = _getSignature(
            carolPrivateKey,
            _getEIP712Digest(
                claimableInvite,
                bytes("OptimistInviter"),
                bytes(optimistInviter.version()),
                block.chainid,
                address(optimistInviter)
            )
        );

        // Sally tries to claim the invite
        _commitInviteAs(sally, signature);

        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            sally,
            bytes32("optimist.can-mint-from-invite"),
            abi.encode(address(carolERC1271Wallet))
        );

        // OptimistInviter should issue a new attestation with updated invite count
        vm.expectEmit(true, true, true, true, address(attestationStation));
        emit AttestationCreated(
            address(optimistInviter),
            address(carolERC1271Wallet),
            bytes32("optimist.can-invite"),
            abi.encode(2)
        );

        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Claimer must commit the signature before claiming the invite. Sally attempts to
     *         claim the Bob's invite without committing the signature first.
     */
    function test_claimInvite_withoutCommittingHash_reverts() external {
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteAs(bobPrivateKey);

        vm.expectRevert("OptimistInviter: claimer and signature have not been committed yet");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Using a signature that doesn't correspond to the claimable invite should fail.
     */
    function test_claimInvite_withIncorrectSignature_reverts() external {
        _grantInvitesTo(carol);
        _grantInvitesTo(bob);
        (
            OptimistInviter.ClaimableInvite memory bobClaimableInvite,
            bytes memory bobSignature
        ) = _issueInviteAs(bobPrivateKey);
        (, bytes memory carolSignature) = _issueInviteAs(carolPrivateKey);

        _commitInviteAs(sally, bobSignature);
        _commitInviteAs(sally, carolSignature);

        vm.expectRevert("OptimistInviter: invalid signature");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, bobClaimableInvite, carolSignature);
    }

    /**
     * @notice Attempting to use a signature from a issuer who never was granted invites should
     *         fail.
     */
    function test_claimInvite_whenIssuerNeverReceivedInvites_reverts() external {
        // Bob was never granted any invites, but issues an invite for Eve
        (
            OptimistInviter.ClaimableInvite memory claimableInvite,
            bytes memory signature
        ) = _issueInviteAs(bobPrivateKey);

        _commitInviteAs(sally, signature);

        vm.expectRevert("OptimistInviter: issuer has no invites");
        vm.prank(sally);
        optimistInviter.claimInvite(sally, claimableInvite, signature);
    }

    /**
     * @notice Attempting to use a signature from a issuer who has no more invites should fail.
     *         Bob has 3 invites, but issues 4 invites for Sally, Carol, Ted, and Eve. Only the
     *         first 3 invites should be claimable. The last claimer, Eve, should not be able to
     *         claim the invite.
     *
     */
    function test_claimInvite_whenIssuerHasNoInvitesLeft_reverts() external {
        _grantInvitesTo(bob);

        _issueThenClaimShouldSucceed(bobPrivateKey, sally);
        _issueThenClaimShouldSucceed(bobPrivateKey, carol);
        _issueThenClaimShouldSucceed(bobPrivateKey, ted);

        assertEq(_getInviteCount(bob), 0);

        (
            OptimistInviter.ClaimableInvite memory claimableInvite4,
            bytes memory signature4
        ) = _issueInviteAs(bobPrivateKey);

        _commitInviteAs(eve, signature4);

        vm.expectRevert("OptimistInviter: issuer has no invites");
        vm.prank(eve);
        optimistInviter.claimInvite(eve, claimableInvite4, signature4);

        assertEq(_getInviteCount(bob), 0);
    }
}
