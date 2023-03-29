//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Test } from "forge-std/Test.sol";
import { AttestationStation } from "../universal/op-nft/AttestationStation.sol";
import { OptimistAllowlist } from "../universal/op-nft/OptimistAllowlist.sol";
import { OptimistAllowlist } from "../universal/op-nft/OptimistInviter.sol";
import { Optimist } from "../universal/op-nft/Optimist.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

contract OptimistAllowlist_Initializer is Test {
    address internal alice_allowlistAttestor;
    address internal sally_coinbaseQuestAttestor;
    address internal ted;

    uint256 internal bobPrivateKey;
    address internal bob;

    AttestationStation attestationStation;
    OptimistAllowlist optimistAllowlist;
    OptimistInviter optimistInviter;

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
        vm.prank(alice_allowlistAttestor);
        attestationStation.attest(attestationData);
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
                bytes(optimistInviter.EIP712_VERSION()),
                block.chainid,
                address(optimistInviter)
            );
    }

    function inviteAndClaim(address _about) internal {
        vm.prank(bobPrivateKey);
        (OptimistInviter.ClaimableInvite, bytes memory signature) = _issueInviteAs(bobPrivateKey);

    }

    function _initializeContracts() internal {
        attestationStation = new AttestationStation();

        optimistAllowlist = new OptimistAllowlist(
            attestationStation,
            alice_allowlistAttestor,
            sally_coinbaseQuestAttestor,
            optimistInviter
        );
    }
}

contract OptimistTest is OptimistAllowlist_Initializer {
    function test_constructor_success() external {
        // expect attestationStation to be set
        assertEq(address(optimistAllowlist.ATTESTATION_STATION()), address(attestationStation));
        assertEq(optimistAllowlist.ALLOWLIST_ATTESTOR(), alice_allowlistAttestor);
        assertEq(optimistAllowlist.COINBASE_QUEST_ATTESTOR(), sally_coinbaseQuestAttestor);
        assertEq(optimistAllowlist.OPTIMIST_INVITER(), address(optimistInviter));

        assertEq(optimistAllowlist.version(), "1.0.0");
    }

    // function test_hasAttestationFromAllowlistAttestor_happyPath_success() external {
    //     attestAllowlist(bob);
    //     assertTrue(optimistAllowlist.hasAttestationFromAllowlistAttestor(bob));
    // }

    // function test_hasAttestationFromAllowlistAttestor_withoutRecevingAttestation_fails() external {
    //     assertFalse(optimistAllowlist.hasAttestationFromAllowlistAttestor(bob));
    // }

    // function test_hasAttestationFromAllowlistAttestor_withWrongAttestor_fails() external {
    //     // Ted is not the allowlist attestor
    //     vm.prank(ted);
    //     attestationStation.attest(
    //         bob,
    //         optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY(),
    //         bytes("true")
    //     );
    //     assertFalse(optimistAllowlist.hasAttestationFromAllowlistAttestor(bob));
    // }

    // function test_hasAttestationFromAllowlistAttestor_withFalsyAttestationValue_fails() external {
    //     vm.prank(alice_allowlistAttestor);
    //     attestationStation.attest(
    //         bob,
    //         optimistAllowlist.OPTIMIST_CAN_MINT_ATTESTATION_KEY(),
    //         bytes("")
    //     );
    //     assertFalse(optimistAllowlist.hasAttestationFromAllowlistAttestor(bob));
    //     assertFalse(optimistAllowlist.hasAttestationFromCoinbaseQuestAttestor(bob));
    // }

    /*
- check falsy attestations from all attestors
- check no attestations from all attestors
- check wrong user making attestors
- check correct attestors making attestor
- multiple attestations should allow to mint
- (In optimist contract) - having multiple attestations shouldn't allow you to mint multiple times

    */

    function test_isAllowedToMint_fromAllowlistAttestor_success() external {
        attestAllowlist(bob);
        assertTrue(optimistAllowlist.isAllowedToMint(bob));
    }

    function test_isAllowedToMint_withoutAttestation_fails() external {
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }

    function test_isAllowedToMint_withoutAttestation_fails() external {
        assertFalse(optimistAllowlist.isAllowedToMint(bob));
    }
}
