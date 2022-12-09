// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

contract AttestationStation is Initializable, Semver {
    struct AttestationData {
        address about;
        bytes32 key;
        bytes val;
    }

    mapping(address => mapping(address => mapping(bytes32 => bytes))) public attestations;

    event AttestationCreated(
        address indexed creator,
        address indexed about,
        bytes32 indexed key,
        bytes val
    );

    constructor() Semver(0, 0, 1) {}

    /**
     * @notice  Attest to the given data.
     * @dev     Attests to the given data from the sender.
     * @param   _about  The address of the attestation subject.
     * @param   _key  The key of the attestation.
     * @param   _val  The value of the attestation.
     */
    function attest(
        address _about,
        bytes32 _key,
        bytes memory _val
    ) public {
        attestations[msg.sender][_about][_key] = _val;
        emit AttestationCreated(msg.sender, _about, _key, _val);
    }

    /**
     * @notice  Attest to the given data.
     * @dev     Attests to the given data from the sender.
     * @param   _attestations  The array of attestation data.
     */
    function attestBulk(AttestationData[] memory _attestations) public {
        for (uint256 i = 0; i < _attestations.length; ++i) {
            attest(_attestations[i].about, _attestations[i].key, _attestations[i].val);
        }
    }
}
