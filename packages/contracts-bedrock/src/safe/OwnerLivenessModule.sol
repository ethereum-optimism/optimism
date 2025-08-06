// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";

// OpenZeppelin
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title OwnerLivenessModule
/// @notice OwnerLivenessModule is a simple module for maintaining liveness of a Safe account.
///         This module allows any user to challenge the liveness of any owner on a Safe by
///         submitting a bonded challenge against that address. The owner then has a certain fixed
///         amount of time to respond to the challenge. If the owner does not respond, the
///         owner is considered inactive and will be removed from the Safe. If the owner does
///         respond, the challenge is considered unsuccessful and the Safe receives the bond.
contract OwnerLivenessModule is ISemver {
    /// @notice Error thrown when the provided config timestamp is not zero.
    error OwnerLivenessModule_InvalidConfigTimestamp();

    /// @notice Error thrown when min owners is zero.
    error OwnerLivenessModule_TooLowMinOwners();

    /// @notice Error thrown when min owners is greater than existing owners.
    error OwnerLivenessModule_TooHighMinOwners();

    /// @notice Error thrown when the provided fallback owner is the zero address.
    error OwnerLivenessModule_InvalidFallback();

    /// @notice Error thrown when the provided challenge period is zero.
    error OwnerLivenessModule_InvalidChallengePeriod();

    /// @notice Error thrown when the provided challenge bond is zero.
    error OwnerLivenessModule_InvalidChallengeBond();

    /// @notice Error thrown when threshold percentage is zero.
    error OwnerLivenessModule_TooLowThreshold();

    /// @notice Error thrown when threshold percentage is greater than 100.
    error OwnerLivenessModule_TooHighThreshold();

    /// @notice Error thrown when the Safe is not configured.
    error OwnerLivenessModule_NotConfigured();

    /// @notice Error thrown when the provided challenge bond is not correct.
    error OwnerLivenessModule_IncorrectBond();

    /// @notice Error thrown when the provided owner is not a valid owner on the Safe.
    error OwnerLivenessModule_InvalidOwner();

    /// @notice Error thrown when the owner already has an active challenge.
    error OwnerLivenessModule_DuplicateChallenge();

    /// @notice Error thrown when there is no active challenge for the owner.
    error OwnerLivenessModule_NoActiveChallenge();

    /// @notice Error thrown when the challenge is still pending.
    error OwnerLivenessModule_ChallengeStillPending();

    /// @notice Error thrown when the challenge period has expired.
    error OwnerLivenessModule_ChallengeExpired();

    /// @notice Error thrown when the signature timestamp is outside of the challenge period.
    error OwnerLivenessModule_InvalidSignatureTimestamp();

    /// @notice Error thrown when the signature provided by the defender is invalid.
    error OwnerLivenessModule_InvalidSignature();

    /// @notice Error thrown when the owner removal fails.
    error OwnerLivenessModule_RemovalFailed(bytes ret);

    /// @notice Error thrown when the owner swap fails.
    error OwnerLivenessModule_SwapFailed(bytes ret);

    /// @notice Error thrown when the claim fails.
    error OwnerLivenessModule_ClaimFailed();

    /// @notice Safe-specific configuration for the module.
    /// @custom:field timestamp Timestamp at which the configuration was set.
    /// @custom:field challengePeriod Amount of time in seconds that an owner has to respond.
    /// @custom:field challengeBond Amount of ETH (in wei) that must be bonded to a challenge.
    /// @custom:field minOwners Minimum number of owners allowed before fallback occurs.
    /// @custom:field thresholdPercentage Threshold percentage that must be maintained.
    /// @custom:field fallbackOwner Address of the account to fallback if minimum is reached.
    struct ModuleConfig {
        uint256 timestamp;
        uint256 challengePeriod;
        uint256 challengeBond;
        uint256 minOwners;
        uint256 thresholdPercentage;
        address fallbackOwner;
    }

    /// @notice Struct representing a challenge for an owner.
    /// @custom:field timestamp Timestamp at which the challenge was created.
    /// @custom:field challenger Address of the challenger.
    /// @custom:field bond Amount of ETH (in wei) that was bonded to the challenge.
    struct Challenge {
        uint256 timestamp;
        address challenger;
        uint256 bond;
    }

    /// @notice Enum representing the result of a challenge.
    /// @custom:value FAILURE The challenge was unsuccessful.
    /// @custom:value SUCCESS The challenge was successful.
    /// @custom:value INVALIDATED The challenge was invalidated.
    enum ChallengeResult {
        FAILURE,
        SUCCESS,
        INVALIDATED
    }

    /// @notice Emitted when a challenge is created.
    /// @param safe The Safe that was challenged.
    /// @param owner The owner that was challenged.
    /// @param challenger The address that created the challenge.
    /// @param timeout The timestamp at which the challenge will timeout.
    event ChallengeCreated(Safe indexed safe, address indexed owner, address indexed challenger, uint256 timeout);

    /// @notice Emitted when a challenge is closed.
    /// @param safe The Safe that was challenged.
    /// @param owner The owner that was challenged.
    /// @param challenger The address that created the challenge.
    /// @param result The result of the challenge.
    event ChallengeClosed(Safe indexed safe, address indexed owner, address indexed challenger, ChallengeResult result);

    /// @notice Emitted when an owner is removed from a Safe.
    /// @param safe The Safe that was removed from.
    /// @param owner The owner that was removed.
    /// @param threshold The threshold that was updated.
    event OwnerRemoved(Safe indexed safe, address indexed owner, uint256 threshold);

    /// @notice Emitted when an owner is swapped with the fallback owner.
    /// @param safe The Safe that was swapped.
    /// @param oldOwner The owner that was swapped.
    /// @param newOwner The new owner that was swapped in.
    event OwnerSwapped(Safe indexed safe, address indexed oldOwner, address indexed newOwner);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Mapping of Safe addresses to their configurations.
    mapping(Safe => ModuleConfig) public configs;

    /// @notice Mapping of Safe addresses to the challenge period for each owner.
    mapping(Safe => mapping(address => Challenge)) public challenges;

    /// @notice Mapping of addresses to their rewards.
    mapping(address => uint256) public rewards;

    /// @notice Allows a Safe to set the configuration for this module. A Safe can reconfigure this
    ///         module at any time, but reconfiguration invalidates any active challenges as to
    ///         prevent updated configuration values from applying to older challenges.
    /// @param _config Configuration for the module.
    function configure(ModuleConfig memory _config) external {
        // Check that the minimum number of owners is less than or equal to the current count.
        Safe safe = Safe(payable(msg.sender));
        address[] memory owners = safe.getOwners();
        if (_config.minOwners > owners.length) {
            revert OwnerLivenessModule_TooHighMinOwners();
        }

        // Check that the minimum number of owners is greater than 0.
        if (_config.minOwners == 0) {
            revert OwnerLivenessModule_TooLowMinOwners();
        }

        // Check that the threshold percentage is greater than 0.
        if (_config.thresholdPercentage == 0) {
            revert OwnerLivenessModule_TooLowThreshold();
        }

        // Check that the threshold percentage is less than or equal to 100.
        if (_config.thresholdPercentage > 100) {
            revert OwnerLivenessModule_TooHighThreshold();
        }

        // Check that the challenge period is greater than 0.
        if (_config.challengePeriod == 0) {
            revert OwnerLivenessModule_InvalidChallengePeriod();
        }

        // Check that the challenge bond is greater than 0.
        if (_config.challengeBond == 0) {
            revert OwnerLivenessModule_InvalidChallengeBond();
        }

        // Check that the fallback owner is not the zero address.
        if (_config.fallbackOwner == address(0)) {
            revert OwnerLivenessModule_InvalidFallback();
        }

        // Check that the config timestamp is set to zero by the end user.
        if (_config.timestamp != 0) {
            revert OwnerLivenessModule_InvalidConfigTimestamp();
        }

        // Set the config timestamp to the current block timestamp so we can keep track of when the
        // configuration was last updated, challenges prior to this timestamp are invalid. Update
        // the configuration with the provided values.
        _config.timestamp = block.timestamp;
        configs[Safe(payable(msg.sender))] = _config;
    }

    /// @notice Allows any address to challenge an owner for a given Safe.
    /// @param _safe Safe to challenge.
    /// @param _owner Owner to challenge.
    function attack(Safe _safe, address _owner) external payable {
        // Check that the Safe is configured for this module, this prevents someone from
        // challenging an owner for a Safe before it has been configured for use with the module
        // and then removing the owner after the Safe has been configured.
        ModuleConfig memory config = configs[_safe];
        if (config.timestamp == 0) {
            revert OwnerLivenessModule_NotConfigured();
        }

        // Check that the provided bond is correct.
        if (msg.value != config.challengeBond) {
            revert OwnerLivenessModule_IncorrectBond();
        }

        // Check that the owner actually exists on the Safe.
        if (!_safe.isOwner(_owner)) {
            revert OwnerLivenessModule_InvalidOwner();
        }

        // Check that the owner does not already have an active challenge. We don't enforce any
        // cooldown on challenges, an owner can be challenged immediately after a previous
        // challenge was closed so each challenge should be expensive to the attacker.
        if (challenges[_safe][_owner].timestamp > 0) {
            revert OwnerLivenessModule_DuplicateChallenge();
        }

        // Create the challenge.
        challenges[_safe][_owner] = Challenge({ timestamp: block.timestamp, challenger: msg.sender, bond: msg.value });

        // Emit the challenge event.
        emit ChallengeCreated(_safe, _owner, msg.sender, block.timestamp + config.challengePeriod);
    }

    /// @notice Allows an owner to defend against a challenge. Any user can submit a signature on
    ///         behalf of the owner to defend the challenge. Owner signature must be over a message
    ///         that contains a timestamp that is inside of the challenge period (inclusive). This
    ///         allows the owner to defend across many chains/safes at the same time and
    ///         deliberately excludes the chain ID from the signed message.
    /// @param _safe Safe that was challenged.
    /// @param _owner Owner that was challenged.
    /// @param _timestamp Timestamp included in the owner's signed message.
    /// @param _signature Signature of the owner defending the challenge.
    function defend(Safe _safe, address _owner, uint256 _timestamp, bytes memory _signature) external {
        ModuleConfig memory config = configs[_safe];
        if (config.timestamp == 0) {
            revert OwnerLivenessModule_NotConfigured();
        }

        // Check that the owner has an active challenge.
        Challenge memory challenge = challenges[_safe][_owner];
        if (challenge.timestamp == 0) {
            revert OwnerLivenessModule_NoActiveChallenge();
        }

        // Check that the challenge period has not expired.
        if (challenge.timestamp + config.challengePeriod <= block.timestamp) {
            revert OwnerLivenessModule_ChallengeExpired();
        }

        // Verify that the supplied timestamp is within the challenge period (inclusive).
        if (_timestamp < challenge.timestamp || _timestamp > challenge.timestamp + config.challengePeriod) {
            revert OwnerLivenessModule_InvalidSignatureTimestamp();
        }

        // Re-create the signed message that should have been produced by the owner.
        // The exact message format is: "OwnerLivenessModule Reply <timestamp>" (no quotes).
        bytes32 digest =
            ECDSA.toEthSignedMessageHash(abi.encodePacked("OwnerLivenessModule Reply ", Strings.toString(_timestamp)));

        // Ensure the owner is the message sender, or the signature was produced by the owner.
        if (msg.sender != _owner && ECDSA.recover(digest, _signature) != _owner) {
            revert OwnerLivenessModule_InvalidSignature();
        }

        // Delete the challenge, reward the Safe.
        delete challenges[_safe][_owner];
        rewards[address(_safe)] += challenge.bond;
        emit ChallengeClosed(_safe, _owner, challenge.challenger, ChallengeResult.FAILURE);
    }

    /// @notice Allows any address to finalize a challenge.
    /// @param _safe Safe that was challenged.
    /// @param _owner Owner that was challenged.
    function finalize(Safe _safe, address _owner) external {
        // Check that the Safe is configured for this module, this prevents someone from
        // finalizing a challenge for a Safe before it has been configured for use with the module
        // and then removing the owner after the Safe has been configured.
        ModuleConfig memory config = configs[_safe];
        if (config.timestamp == 0) {
            revert OwnerLivenessModule_NotConfigured();
        }

        // Check that the owner has an active challenge. Note that if the owner has defended
        // against the challenge, the challenge will already have been deleted and this will
        // revert.
        Challenge memory challenge = challenges[_safe][_owner];
        if (challenge.timestamp == 0) {
            revert OwnerLivenessModule_NoActiveChallenge();
        }

        // Challenge is invalidated if the challenge was created before the Safe's module
        // configuration was modified or if the target is no longer an owner of the Safe.
        // Challenger gets their bond back but the challenge is deleted.
        if (challenge.timestamp < config.timestamp || !_safe.isOwner(_owner)) {
            delete challenges[_safe][_owner];
            rewards[challenge.challenger] += challenge.bond;
            emit ChallengeClosed(_safe, _owner, challenge.challenger, ChallengeResult.INVALIDATED);
            return;
        }

        // If the challenge period has not expired, the challenge is still pending. We execute this
        // check after the above invalidity checks so we can short-circuit and delete invalid
        // challenges without waiting for them to fully expire.
        if (challenge.timestamp + config.challengePeriod > block.timestamp) {
            revert OwnerLivenessModule_ChallengeStillPending();
        }

        // Delete the challenge to prevent re-entrancy.
        delete challenges[_safe][_owner];

        // At this point, the challenge is successful, remove the owner and complete.
        // Find the previous owner in the linked list, required to remove an owner. In theory a
        // malicious Safe could add an astronomical number of owners to this Safe to grief the
        // challenger so this runs out of gas, but this isn't going to happen for the accounts
        // where this module is planned to be used.
        address[] memory owners = _safe.getOwners();
        address prevOwner = address(1);
        for (uint256 i = 0; i < owners.length; i++) {
            if (owners[i] == _owner && i > 0) {
                prevOwner = owners[i - 1];
                break;
            }
        }

        // Calculate the new owner count.
        if (owners.length - 1 >= config.minOwners) {
            // We're still within the minimum owner count, so just remove the owner.
            uint256 threshold = ((owners.length - 1) * config.thresholdPercentage + 99) / 100;
            _removeOwner(_safe, _owner, prevOwner, threshold);
        } else {
            // We're below the minimum owner count, iterate over all owners and remove them one
            // by one until only a single owner remains, then replace that owner with the
            // fallback owner. Note that it's possible we hit this branch if the Safe removes an
            // owner independently (e.g., via regular Safe tx), Safe accounts need to be careful
            // not to reach the minimum owner count unexpectedly.
            for (uint256 i = owners.length - 1; i > 0; i--) {
                // We use a threshold of i here, any threshold should really be fine, using i means
                // the threshold is the same as the owner count. Anything else would only really
                // be dangerous if the Safe being called is somehow malicious and would execute a
                // nested transaction to itself (by overriding the removeOwner function), but this
                // only impacts the malicious Safe and nothing else.
                _removeOwner(_safe, owners[i], owners[i - 1], i);
            }

            // Swap the last owner with the fallback owner.
            _swapOwner(_safe, address(1), owners[0], config.fallbackOwner);
        }

        // Reward the challenger, emit event.
        rewards[challenge.challenger] += challenge.bond;
        emit ChallengeClosed(_safe, _owner, challenge.challenger, ChallengeResult.SUCCESS);
    }

    /// @notice Allows any address to claim their current reward balance.
    function claim() external {
        // Grab the reward, set the balance to 0.
        uint256 reward = rewards[msg.sender];
        rewards[msg.sender] = 0;

        // Send out the reward to the caller.
        (bool success,) = msg.sender.call{ value: reward }("");

        // If the transfer fails, revert.
        if (!success) {
            revert OwnerLivenessModule_ClaimFailed();
        }
    }

    /// @notice Removes an owner from a Safe and updates the threshold.
    /// @param _safe Safe that the owner is being removed from.
    /// @param _owner Owner that is being removed.
    /// @param _prevOwner Previous owner in the linked list.
    /// @param _threshold Threshold that is being updated.
    function _removeOwner(Safe _safe, address _owner, address _prevOwner, uint256 _threshold) internal {
        // Remove the owner from the Safe and update the threshold.
        (bool success, bytes memory ret) = _safe.execTransactionFromModuleReturnData({
            to: address(_safe),
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(OwnerManager.removeOwner, (_prevOwner, _owner, _threshold))
        });

        // If the removal fails, revert with the return data.
        if (!success) {
            revert OwnerLivenessModule_RemovalFailed(ret);
        }

        // Emit the owner removed event.
        emit OwnerRemoved(_safe, _owner, _threshold);
    }

    /// @notice Swaps an owner with the fallback owner.
    /// @param _safe Safe that the owner is being swapped.
    /// @param _prevOwner Previous owner in the linked list.
    /// @param _oldOwner Owner that is being swapped.
    /// @param _newOwner New owner that is being swapped in.
    function _swapOwner(Safe _safe, address _prevOwner, address _oldOwner, address _newOwner) internal {
        // Swap the owner with the fallback owner.
        (bool success, bytes memory ret) = _safe.execTransactionFromModuleReturnData({
            to: address(_safe),
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(OwnerManager.swapOwner, (_prevOwner, _oldOwner, _newOwner))
        });

        // If the swap fails, revert with the return data.
        if (!success) {
            revert OwnerLivenessModule_SwapFailed(ret);
        }

        // Emit the owner swapped event.
        emit OwnerSwapped(_safe, _oldOwner, _newOwner);
    }
}
