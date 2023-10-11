// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Storage } from "src/libraries/Storage.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @custom:audit none This contracts is not yet audited.
/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
contract SuperchainConfig is Initializable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value SYSTEM_OWNER        Represents an update to the systemOwner.
    /// @custom:value INITIATOR           Represents an update to the initiator.
    /// @custom:value VETOER              Represents an update to the vetoer.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    /// @custom:value DELAY               Represents an update to the delay time.
    /// @custom:value MAX_PAUSE           Represents an update to the maximum pause time.
    /// @custom:value ADD_SEQUENCER       Represents an update to add a sequencer to the allowed list.
    /// @custom:value REMOVE_SEQUENCER    Represents an update to remove a sequencer from the allowed list.
    enum UpdateType {
        SYSTEM_OWNER,
        INITIATOR,
        VETOER,
        GUARDIAN,
        DELAY,
        MAX_PAUSE,
        ADD_SEQUENCER,
        REMOVE_SEQUENCER
    }

    /// @notice The address of the systemOwner who may trigger an upgrade or change to critical config values.
    ///         This will be a DelayedVetoable contract.
    ///         It can only be modified by an upgrade.
    bytes32 public constant SYSTEM_OWNER_SLOT = bytes32(uint256(keccak256("superchainConfig.systemowner")) - 1);

    /// @notice The address of the initiator who may initiate an upgrade or change to critical config values, via the
    ///         DelayedVetoable contract.
    ///         It can only be modified by an upgrade.
    bytes32 public constant INITIATOR_SLOT = bytes32(uint256(keccak256("superchainConfig.initiator")) - 1);

    /// @notice The address of the vetoer, who may veto an upgrade or change to critical config values.
    ///         This is expected to be the Foundation.
    ///         It can only be modified by an upgrade.
    bytes32 public constant VETOER_SLOT = bytes32(uint256(keccak256("superchainConfig.vetoer")) - 1);

    /// @notice The address of the guardian, which can pause withdrawals from the System.
    ///         It can only be modified by an upgrade.
    bytes32 public constant GUARDIAN_SLOT = bytes32(uint256(keccak256("superchainConfig.guardian")) - 1);

    /// @notice The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    ///         It can only be modified by an upgrade.
    bytes32 public constant DELAY_SLOT = bytes32(uint256(keccak256("superchainConfig.delay")) - 1);

    /// @notice The time until which the system is paused. If set to a timestamp in the future, withdrawals
    ///         are disabled until that time.
    bytes32 public constant PAUSED_TIME_SLOT = bytes32(uint256(keccak256("superchainConfig.paused")) - 1);

    /// @notice The maximum time in seconds that the system can be paused for.
    ///         It can only be modified by an upgrade.
    bytes32 public constant MAX_PAUSE_SLOT = bytes32(uint256(keccak256("superchainConfig.maxPause")) - 1);

    /// @notice Mapping of allowed sequencers.
    ///         The initiator should be able to add to it instantly, but removing is subject to delay.
    mapping(bytes32 => bool) public allowedSequencers;

    /// @notice Emitted when the pause is triggered.
    /// @param duration The duration of the pause.
    /// @param identifier A string helping to identify provenance of the pause transaction.
    event Paused(uint256 duration, string identifier);

    /// @notice Emitted when an active pause is extended.
    /// @param duration The duration of the pause.
    /// @param identifier A string helping to identify provenance of the pause transaction.
    event PauseExtended(uint256 duration, string identifier);

    /// @notice Emitted when the pause is lifted.
    event Unpaused();

    /// @notice Emitted when configuration is updated.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(UpdateType indexed updateType, bytes data);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Constructs the SuperchainConfig contract.
    constructor() {
        initialize({
            _initiator: address(0),
            _vetoer: address(0),
            _guardian: address(0),
            _delay: 0,
            _maxPause: 0,
            _sequencers: new Types.SequencerKeyPair[](0)
        });
    }

    /// @notice Initializer.
    ///         The resource config must be set before the require check.
    /// @param _initiator   Address of the initiator who may initiate an upgrade or change to critical config values.
    /// @param _vetoer      Address of the vetoer.
    /// @param _guardian    Address of the guardian, can pause the OptimismPortal.
    /// @param _delay       The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    /// @param _maxPause    The maximum time in seconds that the system can be paused for.
    /// @param _sequencers  The initial list of allowed sequencers
    function initialize(
        address _initiator,
        address _vetoer,
        address _guardian,
        uint256 _delay,
        uint256 _maxPause,
        Types.SequencerKeyPair[] memory _sequencers
    )
        public
        reinitializer(2)
    {
        _setInitiator(_initiator);
        _setVetoer(_vetoer);
        _setGuardian(_guardian);
        _setDelay(_delay);
        _setMaxPause(_maxPause);

        for (uint256 i = 0; i < _sequencers.length; i++) {
            _addSequencer(_sequencers[i]);
        }
    }

    /// @notice Getter for the systemOwner address.
    function systemOwner() public view returns (address systemOwner_) {
        systemOwner_ = Storage.getAddress(Constants.PROXY_OWNER_ADDRESS);
    }

    /// @notice Getter for the initiator address.
    function initiator() public view returns (address initiator_) {
        initiator_ = Storage.getAddress(INITIATOR_SLOT);
    }

    /// @notice Getter for the vetoer address.
    function vetoer() public view returns (address vetoer_) {
        vetoer_ = Storage.getAddress(VETOER_SLOT);
    }

    /// @notice Getter for the guardian address.
    function guardian() public view returns (address guardian_) {
        guardian_ = Storage.getAddress(GUARDIAN_SLOT);
    }

    /// @notice Getter for the delay time.
    function delay() public view returns (uint256 delay_) {
        delay_ = Storage.getUint(DELAY_SLOT);
    }

    /// @notice Getter for the maxPause duration.
    function maxPause() public view returns (uint256 maxPause_) {
        maxPause_ = Storage.getUint(MAX_PAUSE_SLOT);
    }

    /// @notice Getter for the current paused status.
    function paused() public view returns (bool paused_) {
        paused_ = Storage.getUint(PAUSED_TIME_SLOT) > block.timestamp;
    }

    /// @notice Getter for the paused until time.
    function pausedUntil() public view returns (uint256 paused_) {
        paused_ = Storage.getUint(PAUSED_TIME_SLOT);
    }

    /// @notice Pauses withdrawals by the specified duration.
    ///         If already paused, the end of the pause period will be delayed by the specified duration.
    /// @param duration The duration of the pause.
    /// @param identifier (Optional) A string to identify provenance of the pause transaction.
    function pause(uint256 duration, string memory identifier) external {
        require(msg.sender == guardian(), "SuperchainConfig: only guardian can pause");
        require(duration <= Storage.getUint(MAX_PAUSE_SLOT), "SuperchainConfig: duration exceeds maxPause");

        if (paused() == false) {
            // Pause is not active, so set the pause end time to the current time plus the duration.
            Storage.setUint(PAUSED_TIME_SLOT, uint256(block.timestamp) + duration);
            emit Paused(duration, identifier);
        } else {
            // Pause is already active, so extend the current pause end time by the duration.
            Storage.setUint(PAUSED_TIME_SLOT, Storage.getUint(PAUSED_TIME_SLOT) + duration);
            emit PauseExtended(duration, identifier);
        }
    }

    /// @notice Unpauses withdrawals.
    function unpause() external {
        require(msg.sender == guardian(), "SuperchainConfig: only guardian can unpause");
        Storage.setUint(PAUSED_TIME_SLOT, uint256((0)));
        emit Unpaused();
    }

    /// @notice Checks if a sequencer is allowed.
    /// @dev This is a convenience function which accepts a SequencerKeyPair struct as an argument,
    ///      hashes it, and checks the mapping. It can be used as an alternative to the
    ///      `allowedSequencers()` getter.
    function isAllowedSequencer(Types.SequencerKeyPair memory _sequencer) external view returns (bool) {
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(_sequencer);
        return allowedSequencers[sequencerHash];
    }

    /// @notice Adds a new sequencer to the allowed list.
    /// @param _sequencer The sequencer to be added.
    function addSequencer(Types.SequencerKeyPair memory _sequencer) external {
        // Adding a new a sequencer is not subject to delay, so can be called by the initiator.
        require(msg.sender == initiator(), "SuperchainConfig: only initiator can add sequencer");
        _addSequencer(_sequencer);
    }

    /// @notice Adds a new sequencer to the allowed list.
    /// @param _sequencer The sequencer to be added.
    function _addSequencer(Types.SequencerKeyPair memory _sequencer) internal {
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(_sequencer);

        allowedSequencers[sequencerHash] = true;
        emit ConfigUpdate(UpdateType.ADD_SEQUENCER, abi.encode(_sequencer));
    }

    /// @notice Removes a sequencer from the allowed list.
    /// @param _sequencer The sequencer to be removed.
    function removeSequencer(Types.SequencerKeyPair memory _sequencer) external {
        // Removing a sequencer is subject to the delay, so can only be called by the systemOwner.
        require(msg.sender == systemOwner(), "SuperchainConfig: only systemOwner can remove a sequencer");
        bytes32 sequencerHash = Hashing.hashSequencerKeyPair(_sequencer);

        delete allowedSequencers[sequencerHash];
        emit ConfigUpdate(UpdateType.REMOVE_SEQUENCER, abi.encode(_sequencer));
    }

    /// @notice Sets the system owner address.
    /// @param _systemOwner The new system owner address.
    function _setSystemOwner(address _systemOwner) internal {
        Storage.setAddress(SYSTEM_OWNER_SLOT, _systemOwner);
        emit ConfigUpdate(UpdateType.SYSTEM_OWNER, abi.encode(_systemOwner));
    }

    /// @notice Sets the initiator address.
    /// @param _initiator The new initiator address.
    function _setInitiator(address _initiator) internal {
        Storage.setAddress(INITIATOR_SLOT, _initiator);
        emit ConfigUpdate(UpdateType.INITIATOR, abi.encode(_initiator));
    }

    /// @notice Sets the vetoer address.
    /// @param _vetoer The new vetoer address.
    function _setVetoer(address _vetoer) internal {
        Storage.setAddress(VETOER_SLOT, _vetoer);
        emit ConfigUpdate(UpdateType.VETOER, abi.encode(_vetoer));
    }

    /// @notice Sets the guardian address.
    /// @param _guardian The new guardian address.
    function _setGuardian(address _guardian) internal {
        Storage.setAddress(GUARDIAN_SLOT, _guardian);
        emit ConfigUpdate(UpdateType.GUARDIAN, abi.encode(_guardian));
    }

    /// @notice Sets the delay.
    /// @param _delay The new delay.
    function _setDelay(uint256 _delay) internal {
        Storage.setUint(DELAY_SLOT, _delay);
        emit ConfigUpdate(UpdateType.DELAY, abi.encode(_delay));
    }

    /// @notice Sets the maxPause.
    /// @param _maxPause The new maxPause.
    function _setMaxPause(uint256 _maxPause) internal {
        Storage.setUint(MAX_PAUSE_SLOT, _maxPause);
        emit ConfigUpdate(UpdateType.MAX_PAUSE, abi.encode(_maxPause));
    }
}
