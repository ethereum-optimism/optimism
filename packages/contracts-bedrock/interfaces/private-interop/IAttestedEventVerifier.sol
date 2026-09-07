// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IEventProofVerifier } from "interfaces/L2/IEventProofVerifier.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

interface IAttestedEventVerifier is IEventProofVerifier, ISemver {
    error AttestedEventVerifier_InvalidConfiguration();

    event EventAccepted(bytes32 indexed eventId, bytes32 indexed payloadHash);

    function signer() external view returns (address);
    function inbox() external view returns (address);
    function eventDigest(Identifier calldata _id, bytes32 _payloadHash) external view returns (bytes32);
    function consumedEvents(bytes32) external view returns (bool);
    function __constructor__(address _signer, address _inbox) external;
}
