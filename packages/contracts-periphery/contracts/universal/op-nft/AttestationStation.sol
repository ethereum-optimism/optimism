// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";

/**
 * @title AttestationStation
 * @author Optimism Collective
 * @author Gitcoin
 * @notice Where attestations live.
 */
contract AttestationStation is Semver {
    /**
     * @notice Struct representing data that is being attested.
     *
     * @custom:field about Address for which the attestation is about.
     * @custom:field key   A bytes32 key for the attestation.
     * @custom:field val   The attestation as arbitrary bytes.
     */
    struct AttestationData {
        address about;
        bytes32 key;
        bytes val;
    }

    /**
     * @notice Maps addresses to attestations. Creator => About => Key => Value.
     */
    mapping(address => mapping(address => mapping(bytes32 => bytes))) public attestations;

    /**
     * @notice Emitted when Attestation is created.
     *
     * @param creator Address that made the attestation.
     * @param about   Address attestation is about.
     * @param key     Key of the attestation.
     * @param val     Value of the attestation.
     */
    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    /**
     * @custom:semver 1.0.0
     */
    constructor() Semver(1, 0, 0) {}

    /**
     * @notice Allows anyone to create attestations.
     *
     * @param _attestations An array of attestation data.
     */
    function attest(AttestationData[] memory _attestations) public {
        uint256 length = _attestations.length;
        for (uint256 i = 0; i < length; ) {
            AttestationData memory attestation = _attestations[i];
            attestations[msg.sender][attestation.about][attestation.key] = attestation.val;

            emit AttestationCreated(
                msg.sender,
                attestation.about,
                attestation.key,
                attestation.val
            );

            unchecked {
                ++i;
            }
        }
    }
}
