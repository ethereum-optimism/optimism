// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract SocialContract {
    mapping(address => mapping(address => mapping(bytes32 => bytes))) public attestations;

    struct AttestationData {
        address about;
        bytes32 key;
        bytes val;
    }

    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    function attest(AttestationData[] memory _attestations) public {
        for (uint256 i = 0; i < _attestations.length; ++i) {
            AttestationData memory attestation = _attestations[i];
            attestations[msg.sender][attestation.about][attestation.key] = attestation.val;
            emit AttestationCreated(
                msg.sender,
                attestation.about,
                attestation.key,
                attestation.val
            );
        }
    }
}
