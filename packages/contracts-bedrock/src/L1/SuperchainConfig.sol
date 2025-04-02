// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

// Libraries
import { Storage } from "src/libraries/Storage.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:audit none This contracts is not yet audited.
/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
contract SuperchainConfig is Initializable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    enum UpdateType {
        GUARDIAN
    }

    /// @notice The address of the guardian, which can pause withdrawals from the System.
    ///         It can only be modified by an upgrade.
    bytes32 public constant GUARDIAN_SLOT = bytes32(uint256(keccak256("superchainConfig.guardian")) - 1);

    /// @notice The duration after which a pause expires
    bytes32 public constant PAUSE_EXPIRY_SLOT = bytes32(uint256(keccak256("superchainConfig.pauseExpiry")) - 1);

    /// @notice Mapping of pause identifiers to their pause timestamps
    mapping(address => uint256) public pauseTimestamps;

    /// @notice Mapping of pause identifiers to their pausable flags (false = ready to pause, true = used/unavailable)
    mapping(address => bool) public pausableFlags;

    /// @notice Emitted when the pause is triggered.
    /// @param identifier A string helping to identify provenance of the pause transaction.
    event Paused(string identifier);

    /// @notice Emitted when the pause is lifted.
    event Unpaused();

    /// @notice Emitted when configuration is updated.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(UpdateType indexed updateType, bytes data);

    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice Thrown when a caller is not the guardian but tries to call a guardian-only function
    error OnlyGuardian();

    /// @notice Thrown when attempting to pause an identifier that has already been used and not reset
    error PauseAlreadyUsed();

    /// @notice Constructs the SuperchainConfig contract.
    constructor() {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _guardian    Address of the guardian, can pause the OptimismPortal.
    /// @param _pauseExpiry Duration in seconds after which a pause expires.
    function initialize(address _guardian, uint256 _pauseExpiry) external initializer {
        _setGuardian(_guardian);
        Storage.setUint(PAUSE_EXPIRY_SLOT, _pauseExpiry);
    }

    /// @notice Getter for the guardian address.
    function guardian() public view returns (address guardian_) {
        guardian_ = Storage.getAddress(GUARDIAN_SLOT);
    }

    /// @notice Pauses the system for a specific identifier.
    /// @param _identifier The address identifier for the pause.
    function pause(address _identifier) external {
        if (msg.sender != guardian()) {
            revert OnlyGuardian();
        }
        if (pausableFlags[_identifier]) {
            revert PauseAlreadyUsed();
        }

        pauseTimestamps[_identifier] = block.timestamp;
        pausableFlags[_identifier] = true;
        emit Paused(string(abi.encodePacked(_identifier)));
    }

    /// @notice Unpauses the system for a specific identifier.
    /// @param _identifier The address identifier to unpause.
    function unpause(address _identifier) external {
        if (msg.sender != guardian()) {
            revert OnlyGuardian();
        }

        pauseTimestamps[_identifier] = 0;
        pausableFlags[_identifier] = false;
        emit Unpaused();
    }

    /// @notice Checks if the system can be paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system can be paused for this identifier.
    function pausable(address _identifier) external view returns (bool) {
        return !pausableFlags[_identifier];
    }

    /// @notice Checks if the system is currently paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system is paused for this identifier and not expired.
    function paused(address _identifier) public view returns (bool) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return false;

        uint256 expiry = Storage.getUint(PAUSE_EXPIRY_SLOT);
        return block.timestamp < timestamp + expiry;
    }

    /// @notice Gets the expiration timestamp for a specific pause identifier.
    /// @param _identifier The address identifier to check.
    /// @return The timestamp when the pause expires, or 0 if not paused.
    function expiration(address _identifier) external view returns (uint256) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return 0;

        uint256 expiry = Storage.getUint(PAUSE_EXPIRY_SLOT);
        return timestamp + expiry;
    }

    /// @notice Resets the pause mechanism for a specific identifier.
    /// @param _identifier The address identifier to reset.
    function reset(address _identifier) external {
        if (msg.sender != guardian()) {
            revert OnlyGuardian();
        }
        pausableFlags[_identifier] = false;
    }

    /// @notice Sets the guardian address. This is only callable during initialization, so an upgrade
    ///         will be required to change the guardian.
    /// @param _guardian The new guardian address.
    function _setGuardian(address _guardian) internal {
        Storage.setAddress(GUARDIAN_SLOT, _guardian);
        emit ConfigUpdate(UpdateType.GUARDIAN, abi.encode(_guardian));
    }
}
