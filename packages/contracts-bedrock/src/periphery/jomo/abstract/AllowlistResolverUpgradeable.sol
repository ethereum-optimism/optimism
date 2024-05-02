// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Initializable} from "openzeppelin-contracts-upgradeable/contracts/proxy/utils/Initializable.sol";
import {SchemaResolverUpgradeable} from "./SchemaResolverUpgradeable.sol";
import {Attestation} from "src/EAS/Common.sol";

/**
 * @title Allowlist Schema Resolver for EAS
 * @dev A base contract for creating an EAS Schema Resolver that can guard
 * a schema's usage based on the attester. Only attester(s) on the allowlist
 * can create attestations when this base contract is used for a schema's resolver.
 */
abstract contract AllowlistResolverUpgradeable is Initializable, SchemaResolverUpgradeable {
    /// @notice Emitted when an attester is allowed.
    event AttesterAllowed(address indexed attester);
    /// @notice Emitted when an attester is removed.
    event AttesterRemoved(address indexed attester);

    /// @notice Attester already in allowlist.
    error AttesterAlreadyAllowed(address attester);
    /// @notice Attester not in allowlist.
    error AttesterAlreadyNotAllowed(address attester);

    /// @notice Addresses that are allowed to attest using the schema this resolver is associated with.
    mapping(address => bool) public allowedAttesters;

    /// @dev Internal initialization function, only meant to be called once.
    function __AllowlistResolver_init() internal onlyInitializing {
        __AllowlistResolver_init_unchained();
    }

    function __AllowlistResolver_init_unchained() internal onlyInitializing {}

    /**
     * @dev Processes a new attestation, and checks if the attester is in the allowlist.
     * See {SchemaResolverUpgradeable-onAttest}.
     *
     * @param attestation The new attestation.
     * @return bool True if the attestation is allowed based on the attester.
     */
    function onAttest(Attestation calldata attestation, uint256) internal virtual override returns (bool) {
        return allowedAttesters[attestation.attester];
    }

    /**
     * @dev Not implemented as EAS already ensures that only the attester
     * who created an attestation can revoke it. We do not need an additional allowlist.
     * See {SchemaResolverUpgradeable-onRevoke}.
     *
     * @return bool Always true as EAS already ensures that only the attester can revoke.
     */
    function onRevoke(Attestation calldata, uint256) internal virtual override returns (bool) {
        return true;
    }

    /**
     * @dev Adds a new allowed attester.
     *
     * If this function were to be made public or external,
     * it should be protected to only allow authorized callers.
     *
     * @param attester The address of the attester to be added to allowlist.
     */
    function _allowAttester(address attester) internal {
        if (allowedAttesters[attester]) {
            revert AttesterAlreadyAllowed(attester);
        }
        allowedAttesters[attester] = true;
        emit AttesterAllowed(attester);
    }

    /**
     * @dev Removes an existing allowed attester.
     *
     * If this function were to be made public or external,
     * it should be protected to only allow authorized callers.
     *
     * @param attester The address of the attester to be removed from allowlist.
     */
    function _removeAttester(address attester) internal {
        if (!allowedAttesters[attester]) {
            revert AttesterAlreadyNotAllowed(attester);
        }
        allowedAttesters[attester] = false;
        emit AttesterRemoved(attester);
    }

    /**
     * @dev This empty reserved space is put in place to allow future versions to add new
     * variables without shifting down storage in the inheritance chain.
     * See https://docs.openzeppelin.com/contracts/4.x/upgradeable#storage_gaps
     */
    uint256[49] private __gap;
}
