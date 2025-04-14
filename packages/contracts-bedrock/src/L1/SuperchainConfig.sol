// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:audit none This contracts is not yet audited.
/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
/// @dev WARNING: When upgrading this contract, any active pause states will be lost as the pause state
///      is stored in storage variables that are not preserved during upgrades. Therefore, this contract
///      should not be upgraded while the system is paused.
contract SuperchainConfig is Initializable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    enum UpdateType {
        GUARDIAN
    }

    /// @notice The address of the guardian, which can pause withdrawals from the System.
    ///         It can only be modified by an upgrade.
    address public guardian;

    /// @notice The duration after which a pause expires
    uint256 public pauseExpiry;

    /// @notice Mapping of pause identifiers to their pause timestamps
    mapping(address => uint256) public pauseTimestamps;

    /// @notice Emitted when the pause is triggered.
    /// @param identifier A string helping to identify provenance of the pause transaction.
    event Paused(string identifier);

    /// @notice Emitted when the pause is lifted.
    event Unpaused(string identifier);

    /// @notice Emitted when configuration is updated.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(UpdateType indexed updateType, bytes data);

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Thrown when a caller is not the guardian but tries to call a guardian-only function
    error SuperchainConfig_OnlyGuardian();

    /// @notice Thrown when attempting to pause an identifier that is already paused
    error SuperchainConfig_AlreadyPaused(string identifier);

    /// @notice Constructs the SuperchainConfig contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _guardian    Address of the guardian, can pause the OptimismPortal.
    /// @param _pauseExpiry Duration in seconds after which a pause expires.
    function initialize(address _guardian, uint256 _pauseExpiry) external initializer {
        _setGuardian(_guardian);
        pauseExpiry = _pauseExpiry;
    }

    /// @notice Pauses the system for a specific identifier.
    /// @param _identifier The address identifier for the pause.
    function pause(address _identifier) external {
        if (msg.sender != guardian) {
            revert SuperchainConfig_OnlyGuardian();
        }
        if (pauseTimestamps[_identifier] != 0) {
            revert SuperchainConfig_AlreadyPaused(string(abi.encodePacked(_identifier)));
        }

        pauseTimestamps[_identifier] = block.timestamp;
        emit Paused(string(abi.encodePacked(_identifier)));
    }

    /// @notice Unpauses the system for a specific identifier.
    /// @param _identifier The address identifier to unpause.
    function unpause(address _identifier) external {
        if (msg.sender != guardian) {
            revert SuperchainConfig_OnlyGuardian();
        }

        pauseTimestamps[_identifier] = 0;
        emit Unpaused(string(abi.encodePacked(_identifier)));
    }

    /// @notice Checks if the system can be paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system can be paused for this identifier.
    function pausable(address _identifier) external view returns (bool) {
        return pauseTimestamps[_identifier] == 0;
    }

    /// @notice Checks if the system is currently paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system is paused for this identifier and not expired.
    function paused(address _identifier) public view returns (bool) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return false;

        return block.timestamp < timestamp + pauseExpiry;
    }

    /// @notice Gets the expiration timestamp for a specific pause identifier.
    /// @param _identifier The address identifier to check.
    /// @return The timestamp when the pause expires, or 0 if not paused.
    function expiration(address _identifier) external view returns (uint256) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return 0;

        return timestamp + pauseExpiry;
    }

    /// @notice Extends the pause for a specific identifier by resetting the pause timestamp.
    /// @param _identifier The address identifier to extend.
    function extend(address _identifier) external {
        if (msg.sender != guardian) {
            revert SuperchainConfig_OnlyGuardian();
        }
        pauseTimestamps[_identifier] = block.timestamp;
        emit Paused(string(abi.encodePacked(_identifier)));
    }

    /// @notice Sets the guardian address. This is only callable during initialization, so an upgrade
    ///         will be required to change the guardian.
    /// @param _guardian The new guardian address.
    function _setGuardian(address _guardian) internal {
        guardian = _guardian;
        emit ConfigUpdate(UpdateType.GUARDIAN, abi.encode(_guardian));
    }
}
