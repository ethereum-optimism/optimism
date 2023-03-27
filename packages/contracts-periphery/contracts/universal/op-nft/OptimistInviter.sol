// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import { AttestationStation } from "./AttestationStation.sol";
import {
    SignatureCheckerUpgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/SignatureCheckerUpgradeable.sol";
import {
    EIP712Upgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/draft-EIP712Upgradeable.sol";

/**
 * @custom:upgradeable
 * @title OptimistInviter
 * @notice OptimistInviter is a contract that issues "optimist.can-invite" and
 *         "optimist.can-mint-from-invite" attestations. Accounts that have a "optimist.can-invite"
 *         attestation can issue signatures that allow other accounts to claim an invite. The
 *         invitee uses a claim and reveal flow to claim the invite to an address of their choosing.
 */
contract OptimistInviter is Semver, EIP712Upgradeable {
    /**
     * @notice Emitted when an invite is claimed.
     *
     * @param issuer Address that issued the signature.
     * @param claimer Address that claimed the invite.
     */
    event InviteClaimed(address indexed issuer, address indexed claimer);

    /**
     * @notice Version identifier, used for upgrades.
     */
    uint8 public constant VERSION = 1;

    /**
     * @notice EIP712 typehash for the ClaimableInvite type.
     *         keccak256("ClaimableInvite(address issuer,bytes32 nonce)")
     */
    bytes32 public immutable CLAIMABLE_INVITE_TYPEHASH =
        0x6529fd129351e725d7bcbc468b0b0b4675477e56b58514e69ab7e66ddfd20fce;

    /**
     * @notice Granter who can set accounts' invite counts.
     */
    address public immutable INVITE_GRANTER;

    /**
     * @notice Address of the AttestationStation contract.
     */
    AttestationStation public immutable ATTESTATION_STATION;

    /**
     * @notice Struct that represents a claimable invite that will be signed by the issuer.
     *
     * @custom:field issuer   Address that issued the signature. Reason this is explicitly included,
     *                        and not implicitly assumed to be the recovered address from the
     *                        signature is that the issuer may be using a ERC-1271 compatible
     *                        contract wallet, where the recovered address is not the same as the
     *                        issuer, or the signature is not an ECDSA signature at all.
     * @custom:field nonce    Pseudorandom nonce to prevent replay attacks.
     */
    struct ClaimableInvite {
        address issuer;
        bytes32 nonce;
    }

    /**
     * @notice Maps from hashes to whether or not they have been committed.
     */
    mapping(bytes32 => bool) public commitments;

    /**
     * @notice Maps from addresses to nonces to whether or not they have been used.
     */
    mapping(address => mapping(bytes32 => bool)) public usedNonces;

    /**
     * @custom:semver 1.0.0
     *
     * @param _inviteGranter      Address of the invite granter.
     * @param _attestationStation Address of the AttestationStation contract.
     */
    constructor(address _inviteGranter, AttestationStation _attestationStation) Semver(1, 0, 0) {
        INVITE_GRANTER = _inviteGranter;
        ATTESTATION_STATION = _attestationStation;
    }

    /**
     * @notice Initializes the OptimistInviter contract, setting the EIP712 context.
     *
     * @param _name Contract name
     */
    function initialize(string memory _name) public reinitializer(VERSION) {
        __EIP712_init(_name, version());
    }

    /**
     * @notice Allows invite granter to set the number of invites an address has.
     *
     * @param _accounts    An array of accounts to update the invite counts of.
     * @param _inviteCount Number of invites to set to.
     */
    function setInviteCounts(address[] calldata _accounts, uint256 _inviteCount) public {
        // Only invite granter can grant invites
        require(
            msg.sender == INVITE_GRANTER,
            "OptimistInviter: only invite granter can grant invites"
        );

        uint256 length = _accounts.length;

        for (uint256 i; i < length; ) {
            // The granted invites are stored as an attestation from this contract on the
            // AttestationStation contract. Number of invites is stored as a encoded uint256 in the
            // data field of the attetation.
            ATTESTATION_STATION.attest(
                _accounts[i],
                bytes32("optimist.can-invite"),
                abi.encode(_inviteCount)
            );

            unchecked {
                i++;
            }
        }
    }

    /**
     * @notice Allows anyone to commit a received signature along with the address to claim to.
     *         This is necessary to prevent front-running when the invitee is claiming the invite.
     *
     * @param _commitment A hash of the claimer and signature concatenated.
                          keccak256(abi.encode(_claimer, _signature))
     */
    function commitInvite(bytes32 _commitment) public {
        commitments[_commitment] = true;
    }

    /**
     * @notice Allows anyone to reveal a commitment and claim an invite.
     *         The claimer ++ signature pair should have been previously committed using commitInvite.
     *         Doesn't require that the claimer is calling this function.
     *
     * @param _claimer Address that will be granted the invite. This should should be committed.
     * @param _claimableInvite ClaimableInvite struct containing the issuer and nonce.
     * @param _signature Signature signed over the claimable invite. This should have been committed.
     */
    function claimInvite(
        address _claimer,
        ClaimableInvite calldata _claimableInvite,
        bytes memory _signature
    ) public {
        // Make sure the claimer and signature have been committed.
        require(
            commitments[keccak256(abi.encode(_claimer, _signature))],
            "OptimistInviter: claimer and signature have not been committed yet"
        );

        // Generate a EIP712 typed data hash to compare against the signature.
        bytes32 digest = _hashTypedDataV4(
            keccak256(
                abi.encode(
                    CLAIMABLE_INVITE_TYPEHASH,
                    _claimableInvite.issuer,
                    _claimableInvite.nonce
                )
            )
        );

        // Uses SignatureChecker, which supports both regular ECDSA signatures from EOAs as well as
        // ERC-1271 signatures from contract wallets or multi-sigs. This means that if the issuer
        // wants to revoke a signature, they can use a smart contract wallet to issue the signature,
        // then invalidate the signature after issuing it.
        require(
            SignatureCheckerUpgradeable.isValidSignatureNow(
                _claimableInvite.issuer,
                digest,
                _signature
            ),
            "OptimistInviter: invalid signature"
        );

        // The issuer includes a pseudorandom nonce in the signature to prevent replay attacks.
        // This checks that the nonce has not been used for this issuer before. The nonces are
        // scoped to the issuer address, so the same nonce can be used by different issuers without
        // clashing.
        require(
            !usedNonces[_claimableInvite.issuer][_claimableInvite.nonce],
            "OptimistInviter: nonce has already been used"
        );

        // Set the nonce as used for the issuer so that it cannot be replayed.
        usedNonces[_claimableInvite.issuer][_claimableInvite.nonce] = true;

        // Check the AttestationStation contract to see how many invites the issuer has left.
        bytes memory attestation = ATTESTATION_STATION.attestations(
            address(this),
            _claimableInvite.issuer,
            bytes32("optimist.can-invite")
        );
        // Failing this check means that the issuer was never granted any invites to begin with.
        require(attestation.length > 0, "OptimistInviter: issuer has no invites");

        uint256 count = abi.decode(attestation, (uint256));

        // Failing this check means that the issuer has used up all of their existing invites.
        require(count > 0, "OptimistInviter: issuer has no invites");

        // Create the attestation that the claimer can mint from the issuer's invite.
        // The invite issuer is included in the data of the attestation.
        ATTESTATION_STATION.attest(
            _claimer,
            bytes32("optimist.can-mint-from-invite"),
            abi.encode(_claimableInvite.issuer)
        );

        // Reduce the issuer's invite count by 1 by re-attesting the optimist.can-invite attestation
        // with the new count.
        count--;
        ATTESTATION_STATION.attest(
            _claimableInvite.issuer,
            bytes32("optimist.can-invite"),
            abi.encode(count)
        );

        emit InviteClaimed(_claimableInvite.issuer, _claimer);
    }
}
