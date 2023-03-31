// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimistConstants } from "./libraries/OptimistConstants.sol";
import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import { AttestationStation } from "./AttestationStation.sol";
import { SignatureChecker } from "@openzeppelin/contracts/utils/cryptography/SignatureChecker.sol";
import {
    EIP712Upgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/draft-EIP712Upgradeable.sol";

/**
 * @custom:upgradeable
 * @title  OptimistInviter
 * @notice OptimistInviter issues "optimist.can-invite" and "optimist.can-mint-from-invite"
 *         attestations. Accounts that have invites can issue signatures that allow other
 *         accounts to claim an invite. The invitee uses a claim and reveal flow to claim the
 *         invite to an address of their choosing.
 *
 *         Parties involved:
 *           1) INVITE_GRANTER: trusted account that can allow accounts to issue invites
 *           2) issuer: account that is allowed to issue invites
 *           3) claimer: account that receives the invites
 *
 *         Flow:
 *           1) INVITE_GRANTER calls _setInviteCount to allow an issuer to issue a certain number
 *              of invites, and also creates a "optimist.can-invite" attestation for the issuer
 *           2) Off-chain, the issuer signs (EIP-712) a ClaimableInvite to produce a signature
 *           3) Off-chain, invite issuer sends the plaintext ClaimableInvite and the signature
 *              to the recipient
 *           4) claimer chooses an address they want to receive the invite on
 *           5) claimer commits the hash of the address they want to receive the invite on and the
 *              received signature keccak256(abi.encode(addressToReceiveTo, receivedSignature))
 *              using the commitInvite function
 *           6) claimer waits for the MIN_COMMITMENT_PERIOD to pass.
 *           7) claimer reveals the plaintext ClaimableInvite and the signature using the
 *              claimInvite function, receiving the "optimist.can-mint-from-invite" attestation
 */
contract OptimistInviter is Semver, EIP712Upgradeable {
    /**
     * @notice Emitted when an invite is claimed.
     *
     * @param issuer  Address that issued the signature.
     * @param claimer Address that claimed the invite.
     */
    event InviteClaimed(address indexed issuer, address indexed claimer);

    /**
     * @notice Version used for the EIP712 domain separator. This version is separated from the
     *         contract semver because the EIP712 domain separator is used to sign messages, and
     *         changing the domain separator invalidates all existing signatures. We should only
     *         bump this version if we make a major change to the signature scheme.
     */
    string public constant EIP712_VERSION = "1.0.0";

    /**
     * @notice EIP712 typehash for the ClaimableInvite type.
     */
    bytes32 public constant CLAIMABLE_INVITE_TYPEHASH =
        keccak256("ClaimableInvite(address issuer,bytes32 nonce)");

    /**
     * @notice Attestation key for that signals that an account was allowed to issue invites
     */
    bytes32 public constant CAN_INVITE_ATTESTATION_KEY = bytes32("optimist.can-invite");

    /**
     * @notice Granter who can set accounts' invite counts.
     */
    address public immutable INVITE_GRANTER;

    /**
     * @notice Address of the AttestationStation contract.
     */
    AttestationStation public immutable ATTESTATION_STATION;

    /**
     * @notice Minimum age of a commitment (in seconds) before it can be revealed using claimInvite.
     *         Currently set to 60 seconds.
     *
     *         Prevents an attacker from front-running a commitment by taking the signature in the
     *         claimInvite call and quickly committing and claiming it before the the claimer's
     *         transaction succeeds. With this, frontrunning a commitment requires that an attacker
     *         be able to prevent the honest claimer's claimInvite transaction from being included
     *         for this long.
     */
    uint256 public constant MIN_COMMITMENT_PERIOD = 60;

    /**
     * @notice Struct that represents a claimable invite that will be signed by the issuer.
     *
     * @custom:field issuer Address that issued the signature. Reason this is explicitly included,
     *                      and not implicitly assumed to be the recovered address from the
     *                      signature is that the issuer may be using a ERC-1271 compatible
     *                      contract wallet, where the recovered address is not the same as the
     *                      issuer, or the signature is not an ECDSA signature at all.
     * @custom:field nonce  Pseudorandom nonce to prevent replay attacks.
     */
    struct ClaimableInvite {
        address issuer;
        bytes32 nonce;
    }

    /**
     * @notice Maps from hashes to the timestamp when they were committed.
     */
    mapping(bytes32 => uint256) public commitmentTimestamps;

    /**
     * @notice Maps from addresses to nonces to whether or not they have been used.
     */
    mapping(address => mapping(bytes32 => bool)) public usedNonces;

    /**
     * @notice Maps from addresses to number of invites they have.
     */
    mapping(address => uint256) public inviteCounts;

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
     * @notice Initializes this contract, setting the EIP712 context.
     *
     *         Only update the EIP712_VERSION when there is a change to the signature scheme.
     *         After the EIP712 version is changed, any signatures issued off-chain but not
     *         claimed yet will no longer be accepted by the claimInvite function. Please make
     *         sure to notify the issuers that they must re-issue their invite signatures.
     *
     * @param _name Contract name.
     */
    function initialize(string memory _name) public initializer {
        __EIP712_init(_name, EIP712_VERSION);
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

        AttestationStation.AttestationData[]
            memory attestations = new AttestationStation.AttestationData[](length);

        for (uint256 i; i < length; ) {
            // Set invite count for account to _inviteCount
            inviteCounts[_accounts[i]] = _inviteCount;

            // Create an attestation for posterity that the account is allowed to create invites
            attestations[i] = AttestationStation.AttestationData({
                about: _accounts[i],
                key: CAN_INVITE_ATTESTATION_KEY,
                val: bytes("true")
            });

            unchecked {
                ++i;
            }
        }

        ATTESTATION_STATION.attest(attestations);
    }

    /**
     * @notice Allows anyone (but likely the claimer) to commit a received signature along with the
     *         address to claim to.
     *
     *         Before calling this function, the claimer should have received a signature from the
     *         issuer off-chain. The claimer then calls this function with the hash of the
     *         claimer's address and the received signature. This is necessary to prevent
     *         front-running when the invitee is claiming the invite. Without a commit and reveal
     *         scheme, anyone who is watching the mempool can take the signature being submitted
     *         and front run the transaction to claim the invite to their own address.
     *
     *         The same commitment can only be made once, and the function reverts if the
     *         commitment has already been made. This prevents griefing where a malicious party can
     *         prevent the original claimer from being able to claimInvite.
     *
     *
     * @param _commitment A hash of the claimer and signature concatenated.
     *                    keccak256(abi.encode(_claimer, _signature))
     */
    function commitInvite(bytes32 _commitment) public {
        // Check that the commitment hasn't already been made. This prevents griefing where
        // a malicious party continuously re-submits the same commitment, preventing the original
        // claimer from claiming their invite by resetting the minimum commitment period.
        require(commitmentTimestamps[_commitment] == 0, "OptimistInviter: commitment already made");

        commitmentTimestamps[_commitment] = block.timestamp;
    }

    /**
     * @notice Allows anyone to reveal a commitment and claim an invite.
     *
     *         The hash, keccak256(abi.encode(_claimer, _signature)), should have been already
     *         committed using commitInvite. Before issuing the "optimist.can-mint-from-invite"
     *         attestation, this function checks that
     *           1) the hash corresponding to the _claimer and the _signature was committed
     *           2) MIN_COMMITMENT_PERIOD has passed since the commitment was made.
     *           3) the _signature is signed correctly by the issuer
     *           4) the _signature hasn't already been used to claim an invite before
     *           5) the _signature issuer has not used up all of their invites
     *         This function doesn't require that the _claimer is calling this function.
     *
     * @param _claimer         Address that will be granted the invite.
     * @param _claimableInvite ClaimableInvite struct containing the issuer and nonce.
     * @param _signature       Signature signed over the claimable invite.
     */
    function claimInvite(
        address _claimer,
        ClaimableInvite calldata _claimableInvite,
        bytes memory _signature
    ) public {
        uint256 commitmentTimestamp = commitmentTimestamps[
            keccak256(abi.encode(_claimer, _signature))
        ];

        // Make sure the claimer and signature have been committed.
        require(
            commitmentTimestamp > 0,
            "OptimistInviter: claimer and signature have not been committed yet"
        );

        // Check that MIN_COMMITMENT_PERIOD has passed since the commitment was made.
        require(
            commitmentTimestamp + MIN_COMMITMENT_PERIOD <= block.timestamp,
            "OptimistInviter: minimum commitment period has not elapsed yet"
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
            SignatureChecker.isValidSignatureNow(_claimableInvite.issuer, digest, _signature),
            "OptimistInviter: invalid signature"
        );

        // The issuer's signature commits to a nonce to prevent replay attacks.
        // This checks that the nonce has not been used for this issuer before. The nonces are
        // scoped to the issuer address, so the same nonce can be used by different issuers without
        // clashing.
        require(
            usedNonces[_claimableInvite.issuer][_claimableInvite.nonce] == false,
            "OptimistInviter: nonce has already been used"
        );

        // Set the nonce as used for the issuer so that it cannot be replayed.
        usedNonces[_claimableInvite.issuer][_claimableInvite.nonce] = true;

        // Failing this check means that the issuer has used up all of their existing invites.
        require(
            inviteCounts[_claimableInvite.issuer] > 0,
            "OptimistInviter: issuer has no invites"
        );

        // Reduce the issuer's invite count by 1. Can be unchecked because we check above that
        // count is > 0.
        unchecked {
            --inviteCounts[_claimableInvite.issuer];
        }

        // Create the attestation that the claimer can mint from the issuer's invite.
        // The invite issuer is included in the data of the attestation.
        ATTESTATION_STATION.attest(
            _claimer,
            OptimistConstants.OPTIMIST_CAN_MINT_FROM_INVITE_ATTESTATION_KEY,
            abi.encode(_claimableInvite.issuer)
        );

        emit InviteClaimed(_claimableInvite.issuer, _claimer);
    }
}
