// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import {AccessControlUpgradeable} from "@openzeppelin/contracts-upgradeable/access/AccessControlUpgradeable.sol";
import {PausableUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/PausableUpgradeable.sol";
import {ReentrancyGuardUpgradeable} from "@openzeppelin/contracts-upgradeable/utils/ReentrancyGuardUpgradeable.sol";
import {IEAS} from "src/EAS/IEAS.sol";
import {Attestation} from "src/EAS/Common.sol";

import {AllowlistResolverUpgradeable} from "src/periphery/jomo/abstract/AllowlistResolverUpgradeable.sol";
import {SchemaResolverUpgradeable} from "src/periphery/jomo/abstract/SchemaResolverUpgradeable.sol";

/**
 * @title EAS Schema Resolver for Optimist Attestation Resolver
 * @notice Manages schemas related to Optimist attestations.
 * @dev Only allowlisted entities can attest; successful attestations are record and allow to mint.
 */
contract OptimistAllowlistAttestationResolver is
    SchemaResolverUpgradeable,
    AllowlistResolverUpgradeable,
    AccessControlUpgradeable,
    PausableUpgradeable,
    ReentrancyGuardUpgradeable
{
    event AttestationCreated(address indexed recipient);
    event AttestationRevoked(address indexed recipient);

    bytes32 public constant PAUSE_ROLE = keccak256("optimist.allowlist-attestation-issuer.pause-role");
    bytes32 public constant ADMIN_ROLE = keccak256("optimist.allowlist-attestation-issuer.admin-role");
    bytes32 public constant ALLOWLIST_ROLE = keccak256("optimist.allowlist-attestation-issuer.allowlist-role");

    /// @notice track recipient attestationUid
    mapping (address => bytes32) private attestationUidByRecipient;

    /**
    * @dev Locks the contract, preventing any future reinitialization. This implementation contract was designed to be called through proxies.
    * @custom:oz-upgrades-unsafe-allow constructor
    */
    constructor() {
        _disableInitializers();
    }

    /**
    * @dev Initializes the contract.
    * @param admin The address to be granted with the default admin Role.
    * @param eas The address of the EAS attestation contract.
    */
    function initialize(address admin, IEAS eas) initializer public {
        __SchemaResolver_init(eas);
        __AccessControl_init();
        __Pausable_init();
        __ReentrancyGuard_init();
        __AllowlistResolver_init();

        require(_grantRole(ADMIN_ROLE, admin));
        _setRoleAdmin(PAUSE_ROLE, ADMIN_ROLE);
        _setRoleAdmin(ALLOWLIST_ROLE, ADMIN_ROLE);
    }

    /// @notice check user has attestation
    function hasAttestation(address user) public view returns (bool) {
        return attestationUidByRecipient[user] != bytes32(0);
    }

    /// @notice get user attestationUid
    function getAttestationUid(address user) public view returns (bytes32) {
        return attestationUidByRecipient[user];
    }

    /// @inheritdoc SchemaResolverUpgradeable
    function onAttest(
        Attestation calldata attestationInput,
        uint256 value
    ) internal
    whenNotPaused
    override(SchemaResolverUpgradeable, AllowlistResolverUpgradeable)
    returns (bool)
    {
        require(AllowlistResolverUpgradeable.onAttest(attestationInput, value), "OptimistAttestationResolver: attester is not allowed");
        require(attestationUidByRecipient[attestationInput.recipient] == bytes32(0), "OptimistAttestationResolver: recipient already has an attestation");
        attestationUidByRecipient[attestationInput.recipient] = attestationInput.uid;
        emit AttestationCreated(attestationInput.recipient);
        return true;
    }

    /// @inheritdoc SchemaResolverUpgradeable
    function onRevoke(
        Attestation calldata attestationInput,
        uint256 value
    ) internal whenNotPaused override(SchemaResolverUpgradeable, AllowlistResolverUpgradeable) returns (bool) {
        require(AllowlistResolverUpgradeable.onRevoke(attestationInput, value), "OptimistAttestationResolver: attester is not allowed");
        require(attestationUidByRecipient[attestationInput.recipient] != bytes32(0), "OptimistAttestationResolver: recipient does not have an attestation");
        attestationUidByRecipient[attestationInput.recipient] = bytes32(0);
        emit AttestationRevoked(attestationInput.recipient);
        return true;
    }

    /**
    * @dev Pause the contract.
    */
    function pause() external onlyRole(PAUSE_ROLE) {
        _pause();
    }

    /**
    * @dev UnPause the contract.
    */
    function unpause() external onlyRole(PAUSE_ROLE) {
        _unpause();
    }

    /*
    * @dev Add a new attester to allowedAttesters
    */
    function addAttesterToAttesterAllowlist(address _newAttester)
        external
        onlyRole(ALLOWLIST_ROLE)
    {
        _allowAttester(_newAttester);
    }

    /*
    * @dev remove a attester from allowedAttesters
    */
    function removeAttesterFromAttesterAllowlist(address _attesterToRemove)
        external
        onlyRole(ALLOWLIST_ROLE)
    {
        _removeAttester(_attesterToRemove);
    }
}
