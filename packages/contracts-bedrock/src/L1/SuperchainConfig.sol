// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @custom:proxied true
/// @custom:audit none This contracts is not yet audited.
/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
contract SuperchainConfig is ProxyAdminOwnedBase, Initializable, ReinitializableBase, ISemver {
    /// @notice Thrown when a caller is not the guardian but tries to call a guardian-only function
    error SuperchainConfig_OnlyGuardian();

    /// @notice Thrown when attempting to pause an identifier that is already paused
    error SuperchainConfig_AlreadyPaused(address identifier);

    /// @notice Thrown when attempting to extend a pause that is not already paused.
    error SuperchainConfig_NotAlreadyPaused(address identifier);

    /// @notice Enum representing different types of updates.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    enum UpdateType {
        GUARDIAN
    }

    /// @notice The duration after which a pause expires. This value is set to exactly 3 months in
    ///         seconds. Any duration longer than this value is incompatible with Stage 1.
    uint256 internal constant PAUSE_EXPIRY = 7_884_000;

    /// @notice The address of the guardian, which can pause withdrawals from the System.
    ///         It can only be modified by an upgrade.
    address public guardian;

    /// @notice Mapping of pause identifiers to their pause timestamps
    mapping(address => uint256) public pauseTimestamps;

    /// @notice Emitted when the pause is triggered.
    /// @param identifier A string helping to identify provenance of the pause transaction.
    event Paused(address identifier);

    /// @notice Emitted when the pause is lifted.
    event Unpaused(address identifier);

    /// @notice Emitted when configuration is updated.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(UpdateType indexed updateType, bytes data);

    /// @notice Semantic version.
    /// @custom:semver 2.4.1
    string public constant version = "2.4.1";

    /// @notice Constructs the SuperchainConfig contract.
    constructor() ReinitializableBase(2) {
        _disableInitializers();
    }

    /// @notice Initializer.
    /// @param _guardian    Address of the guardian, can pause the OptimismPortal.
    function initialize(address _guardian) external reinitializer(initVersion()) {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        // Now perform initialization logic.
        _setGuardian(_guardian);
    }

    /// @notice Returns the duration after which a pause expires.
    /// @return The duration after which a pause expires.
    function pauseExpiry() external pure returns (uint256) {
        return PAUSE_EXPIRY;
    }

    /// @notice Pauses the system for a specific superchain cluster identifier.
    /// @param _identifier The address identifier for the pause.
    function pause(address _identifier) external {
        // Only the Guardian can pause the system.
        _assertOnlyGuardian();

        // Cannot pause if the identifier is already paused to prevent re-pausing without either
        // unpausing, extending, or resetting the pause timestamp.
        if (pauseTimestamps[_identifier] != 0) {
            revert SuperchainConfig_AlreadyPaused(_identifier);
        }

        // Set the pause timestamp.
        pauseTimestamps[_identifier] = block.timestamp;
        emit Paused(_identifier);
    }

    /// @notice Unpauses the system for a specific identifier.
    /// @param _identifier The address identifier to unpause.
    function unpause(address _identifier) external {
        // Only the Guardian can unpause the system.
        _assertOnlyGuardian();

        // Unpause the system.
        pauseTimestamps[_identifier] = 0;
        emit Unpaused(_identifier);
    }

    /// @notice Extends the pause for a specific identifier by resetting the pause timestamp.
    /// @param _identifier The address identifier to extend.
    function extend(address _identifier) external {
        // Only the Guardian can extend the pause.
        _assertOnlyGuardian();

        // Cannot extend the pause if not already paused.
        if (pauseTimestamps[_identifier] == 0) {
            revert SuperchainConfig_NotAlreadyPaused(_identifier);
        }

        // Reset the pause timestamp.
        pauseTimestamps[_identifier] = block.timestamp;
        emit Paused(_identifier);
    }

    /// @notice Checks if the system can be paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system can be paused for this identifier.
    function pausable(address _identifier) external view returns (bool) {
        return pauseTimestamps[_identifier] == 0;
    }

    /// @custom:legacy
    /// @notice Checks if the global superchain system is paused. NOTE that this is a legacy
    ///         function that provides support for systems that still rely on the older interface.
    ///         Contracts should use paused(address) instead when possible.
    /// @return True if the global superchain system is paused.
    function paused() external view returns (bool) {
        return paused(address(0));
    }

    /// @notice Checks if the system is currently paused for a specific identifier.
    /// @param _identifier The address identifier to check.
    /// @return True if the system is paused for this identifier and not expired.
    function paused(address _identifier) public view returns (bool) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return false;
        return block.timestamp < timestamp + PAUSE_EXPIRY;
    }

    /// @notice Gets the expiration timestamp for a specific pause identifier.
    /// @param _identifier The address identifier to check.
    /// @return The timestamp when the pause expires, or 0 if not paused.
    function expiration(address _identifier) external view returns (uint256) {
        uint256 timestamp = pauseTimestamps[_identifier];
        if (timestamp == 0) return 0;
        return timestamp + PAUSE_EXPIRY;
    }

    /// @notice Sets the guardian address. This is only callable during initialization, so an upgrade
    ///         will be required to change the guardian.
    /// @param _guardian The new guardian address.
    function _setGuardian(address _guardian) internal {
        guardian = _guardian;
        emit ConfigUpdate(UpdateType.GUARDIAN, abi.encode(_guardian));
    }

    /// @notice Asserts that the caller is the guardian.
    function _assertOnlyGuardian() internal view {
        if (msg.sender != guardian) {
            revert SuperchainConfig_OnlyGuardian();
        }
    }
}
