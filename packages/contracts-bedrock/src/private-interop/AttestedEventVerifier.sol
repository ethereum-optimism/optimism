// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @notice Minimal event verifier trusting one configured signer to attest canonical event facts.
///         This is an attestation scheme, not a private execution proof or a TEE attestation checker.
///         The signer is responsible for execution validity, projected log identity and accounting.
contract AttestedEventVerifier is ISemver {
    error AttestedEventVerifier_InvalidConfiguration();

    address internal immutable SIGNER;
    address internal immutable INBOX;
    bytes32 internal constant DOMAIN = keccak256("optimism.private-interop.event-attestation.v1");

    /// @notice Event identifiers already accepted, including acceptance of a zero payload hash.
    mapping(bytes32 => bool) public consumedEvents;

    event EventAccepted(bytes32 indexed eventId, bytes32 indexed payloadHash);

    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    constructor(address _signer, address _inbox) {
        if (_signer == address(0) || _inbox == address(0)) revert AttestedEventVerifier_InvalidConfiguration();
        SIGNER = _signer;
        INBOX = _inbox;
    }

    function signer() external view returns (address) {
        return SIGNER;
    }

    function inbox() external view returns (address) {
        return INBOX;
    }

    /// @notice Digest signed directly as a 65-byte ECDSA signature, including chain and verifier domain.
    function eventDigest(Identifier calldata _id, bytes32 _payloadHash) public view returns (bytes32) {
        return keccak256(abi.encode(DOMAIN, block.chainid, address(this), INBOX, _id, _payloadHash));
    }

    /// @notice Accepts each attested event identifier once, only when called by the configured inbox.
    function verifyAndConsumeEvent(
        Identifier calldata _id,
        bytes32 _payloadHash,
        bytes calldata _proof
    )
        external
        returns (bool)
    {
        if (msg.sender != INBOX || _id.chainId != block.chainid) return false;
        bytes32 eventId = keccak256(abi.encode(_id));
        if (consumedEvents[eventId]) return false;
        (address recovered, ECDSA.RecoverError err) = ECDSA.tryRecover(eventDigest(_id, _payloadHash), _proof);
        if (err != ECDSA.RecoverError.NoError || recovered != SIGNER) return false;
        consumedEvents[eventId] = true;
        emit EventAccepted(eventId, _payloadHash);
        return true;
    }
}
