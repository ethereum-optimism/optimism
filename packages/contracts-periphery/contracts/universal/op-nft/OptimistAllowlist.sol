// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import { AttestationStation } from "./AttestationStation.sol";

/**
 * @title OptimistAllowlist
 * @notice Allowlist logic for OptimistNFT
 */
contract OptimistAllowlist is Semver {
    /**
     * @notice Address of the AttestationStation contract.
     */
    AttestationStation public immutable ATTESTATION_STATION;

    /**
     * @notice Attestor who can manually add addresses to the allowlist through by issuing 'optimist.can-mint' attestations.
     */
    address public immutable ALLOWLIST_ATTESTOR;

    /**
     * @custom:semver 1.0.0
     * @param _allowlistAttestor           Address of the attestor.
     * @param _attestationStation Address of the AttestationStation contract.
     */
    constructor(address _allowlistAttestor, AttestationStation _attestationStation)
        Semver(1, 0, 0)
    {
        ALLOWLIST_ATTESTOR = _allowlistAttestor;
        ATTESTATION_STATION = _attestationStation;
    }

    /**
     * @notice Checks whether a given address has an optimist.can-mint attestation from the allowlist attestor.
     *
     * @return Whether or not the address has a optimist.can-mint attestation from the allowlist .
     */
    function hasAttestationFromAllowlistAttestor(address _recipient) public view returns (bool) {
        return
            ATTESTATION_STATION
                .attestations(ALLOWLIST_ATTESTOR, _recipient, bytes32("optimist.can-mint"))
                .length > 0;
    }

    /**
     * @notice Checks whether a given address is allowed to mint the Optimist NFT yet. Since the
     *         Optimist NFT will also be used as part of the Citizens House, mints are currently
     *         restricted. Eventually anyone will be able to mint.
     *
     * @return Whether or not the address is allowed to mint yet.
     */
    function isAllowedToMint(address _recipient) public view returns (bool) {
        return hasAttestationFromAllowlistAttestor(_recipient);
    }
}
