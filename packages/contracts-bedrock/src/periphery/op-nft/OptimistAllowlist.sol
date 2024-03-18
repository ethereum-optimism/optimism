// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { AttestationStation } from "src/periphery/op-nft/AttestationStation.sol";
import { OptimistConstants } from "src/periphery/op-nft/libraries/OptimistConstants.sol";

/// @title  OptimistAllowlist
/// @notice Source of truth for whether an address is able to mint an Optimist NFT.
///         isAllowedToMint function checks various signals to return boolean value
///         for whether an address is eligible or not.
contract OptimistAllowlist is ISemver {
    /// @notice Attestation key used by the AllowlistAttestor to manually add addresses to the
    ///         allowlist.
    bytes32 public constant OPTIMIST_CAN_MINT_ATTESTATION_KEY = bytes32("optimist.can-mint");

    /// @notice Attestation key used by Coinbase to issue attestations for Quest participants.
    bytes32 public constant COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY = bytes32("coinbase.quest-eligible");

    /// @notice Address of the AttestationStation contract.
    AttestationStation public immutable ATTESTATION_STATION;

    /// @notice Attestor that issues 'optimist.can-mint' attestations.
    address public immutable ALLOWLIST_ATTESTOR;

    /// @notice Attestor that issues 'coinbase.quest-eligible' attestations.
    address public immutable COINBASE_QUEST_ATTESTOR;

    /// @notice Address of OptimistInviter contract that issues 'optimist.can-mint-from-invite'
    ///         attestations.
    address public immutable OPTIMIST_INVITER;

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @param _attestationStation    Address of the AttestationStation contract.
    /// @param _allowlistAttestor     Address of the allowlist attestor.
    /// @param _coinbaseQuestAttestor Address of the Coinbase Quest attestor.
    /// @param _optimistInviter       Address of the OptimistInviter contract.
    constructor(
        AttestationStation _attestationStation,
        address _allowlistAttestor,
        address _coinbaseQuestAttestor,
        address _optimistInviter
    ) {
        ATTESTATION_STATION = _attestationStation;
        ALLOWLIST_ATTESTOR = _allowlistAttestor;
        COINBASE_QUEST_ATTESTOR = _coinbaseQuestAttestor;
        OPTIMIST_INVITER = _optimistInviter;
    }

    /// @notice Checks whether a given address is allowed to mint the Optimist NFT yet. Since the
    ///         Optimist NFT will also be used as part of the Citizens House, mints are currently
    ///         restricted. Eventually anyone will be able to mint.
    ///         Currently, address is allowed to mint if it satisfies any of the following:
    ///          1) Has a valid 'optimist.can-mint' attestation from the allowlist attestor.
    ///          2) Has a valid 'coinbase.quest-eligible' attestation from Coinbase Quest attestor
    ///          3) Has a valid 'optimist.can-mint-from-invite' attestation from the OptimistInviter
    ///             contract.
    /// @param _claimer Address to check.
    /// @return allowed_ Whether or not the address is allowed to mint yet.
    function isAllowedToMint(address _claimer) public view returns (bool allowed_) {
        allowed_ = _hasAttestationFromAllowlistAttestor(_claimer) || _hasAttestationFromCoinbaseQuestAttestor(_claimer)
            || _hasAttestationFromOptimistInviter(_claimer);
    }

    /// @notice Checks whether an address has a valid 'optimist.can-mint' attestation from the
    ///         allowlist attestor.
    /// @param _claimer Address to check.
    /// @return valid_ Whether or not the address has a valid attestation.
    function _hasAttestationFromAllowlistAttestor(address _claimer) internal view returns (bool valid_) {
        // Expected attestation value is bytes32("true")
        valid_ = _hasValidAttestation(ALLOWLIST_ATTESTOR, _claimer, OPTIMIST_CAN_MINT_ATTESTATION_KEY);
    }

    /// @notice Checks whether an address has a valid attestation from the Coinbase attestor.
    /// @param _claimer Address to check.
    /// @return valid_ Whether or not the address has a valid attestation.
    function _hasAttestationFromCoinbaseQuestAttestor(address _claimer) internal view returns (bool valid_) {
        // Expected attestation value is bytes32("true")
        valid_ = _hasValidAttestation(COINBASE_QUEST_ATTESTOR, _claimer, COINBASE_QUEST_ELIGIBLE_ATTESTATION_KEY);
    }

    /// @notice Checks whether an address has a valid attestation from the OptimistInviter contract.
    /// @param _claimer Address to check.
    /// @return valid_ Whether or not the address has a valid attestation.
    function _hasAttestationFromOptimistInviter(address _claimer) internal view returns (bool valid_) {
        // Expected attestation value is the inviter's address
        valid_ = _hasValidAttestation(
            OPTIMIST_INVITER, _claimer, OptimistConstants.OPTIMIST_CAN_MINT_FROM_INVITE_ATTESTATION_KEY
        );
    }

    /// @notice Checks whether an address has a valid truthy attestation.
    ///         Any attestation val other than bytes32("") is considered truthy.
    /// @param _creator Address that made the attestation.
    /// @param _about   Address attestation is about.
    /// @param _key     Key of the attestation.
    /// @return valid_ Whether or not the address has a valid truthy attestation.
    function _hasValidAttestation(address _creator, address _about, bytes32 _key) internal view returns (bool valid_) {
        valid_ = ATTESTATION_STATION.attestations(_creator, _about, _key).length > 0;
    }
}
