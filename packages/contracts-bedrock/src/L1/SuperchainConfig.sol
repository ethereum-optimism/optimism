// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Types as T } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
/// @custom:audit none
contract SuperchainConfig is OwnableUpgradeable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value INITIATOR           Represents an update to the initiator.
    /// @custom:value VETOER              Represents an update to the vetoer.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    /// @custom:value DELAY               Represents an update to the delay time.
    /// @custom:value PAUSED              Represents an update to the paused status.
    /// @custom:value ADD_SEQUENCER       Represents an update to add a sequencer to the allowed list.
    /// @custom:value REMOVE_SEQUENCER    Represents an update to remove a sequencer from the allowed list.
    enum UpdateType {
        INITIATOR,
        VETOER,
        GUARDIAN,
        DELAY,
        PAUSED,
        ADD_SEQUENCER,
        REMOVE_SEQUENCER
    }

    /// @notice Event version identifier.
    uint256 public constant VERSION = 0;

    // todo(maurelian): We can change these vars to EIP 1967 style storage slots later, but
    //    during the mock up phase this is easier to work with.

    // todo(maurelian): I believe we will want to replace the initator with the owner of this contract.
    /// @notice The address of the initiator who may initiate an upgrade or change to critical config values.
    ///         This is expected to be the security council.
    address public initiator;

    /// @notice The address of the vetoer.
    ///         This is expected to the foundation.
    address public vetoer;

    /// @notice The address of the guardian, can pause the OptimismPortal.
    address public guardian;

    /// @notice The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    uint256 public delay;

    /// @notice The pause status of withdrawals from an chain in the superchain.
    bool public paused;

    /// @notice Mapping of allowed sequencers.
    ///         The initiator should be able to add to it instantly, but removing is subject to delay.
    mapping(bytes32 => bool) public allowedSequencers;

    /// @notice Emitted when the pause is triggered.
    event Paused();

    /// @notice Emitted when the pause is lifted.
    event Unpaused();

    /// @notice Emitted when configuration is updated.
    /// @param version    SystemConfig version.
    /// @param updateType Type of update.
    /// @param data       Encoded update data.
    event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Constructs the SuperchainConfig contract. Cannot set
    ///         the owner to `address(0)` due to the Ownable contract's
    ///         implementation, so set it to `address(0xdEaD)`
    constructor() {
        initialize({
            _owner: address(0xdEaD),
            _initiator: address(0),
            _vetoer: address(0),
            _guardian: address(0),
            _delay: 0,
            _sequencers: new T.SequencerKeys[](0)
        });
    }

    /// @notice Initializer.
    ///         The resource config must be set before the require check.
    /// @param _owner     Initial owner of the contract.
    /// @param _initiator Address of the initiator who may initiate an upgrade or change to critical config values.
    /// @param _vetoer    Address of the vetoer.
    /// @param _guardian  Address of the guardian, can pause the OptimismPortal.
    /// @param _delay     The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    /// @param _sequencers The initial list of allowed sequencers
    function initialize(
        address _owner,
        address _initiator,
        address _vetoer,
        address _guardian,
        uint256 _delay,
        T.SequencerKeys[] memory _sequencers
    )
        public
        reinitializer(2)
    {
        __Ownable_init();
        transferOwnership(_owner);

        initiator = _initiator;
        vetoer = _vetoer;
        guardian = _guardian;
        delay = _delay;

        for (uint256 i = 0; i < _sequencers.length; i++) {
            bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencers[i]);
            allowedSequencers[sequencerHash] = true;
        }
    }

    /// @notice Pauses withdrawals.
    function pause() external {
        require(msg.sender == guardian, "SuperchainConfig: only guardian can pause");
        paused = true;
        emit Paused();
    }

    /// @notice Unpauses withdrawals.
    function unpause() external {
        require(msg.sender == guardian, "SuperchainConfig: only guardian can unpause");
        paused = false;
        emit Unpaused();
    }

    /// @notice Checks if a sequencer is allowed.
    /// @dev This is a convenience function which accepts a SequencerKeys struct as an argument,
    ///      hashes it, and checks the mapping. It can be used as an alternative to the
    ///      `allowedSequencers()` getter.
    function isAllowedSequencer(T.SequencerKeys memory _sequencer) external view returns (bool) {
        bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencer);
        return allowedSequencers[sequencerHash];
    }

    /// @notice Adds a new sequencer to the allowed list.
    /// @param _sequencer The sequencer to be added.
    function addSequencer(T.SequencerKeys memory _sequencer) external {
        // Adding a new a sequencer is not subject to delay, so can be called by the initiator.
        require(msg.sender == initiator, "SuperchainConfig: only initiator can add sequencer");
        bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencer);
        require(!allowedSequencers[sequencerHash], "SuperchainConfig: sequencer already allowed");

        allowedSequencers[sequencerHash] = true;
        emit ConfigUpdate(VERSION, UpdateType.ADD_SEQUENCER, abi.encode(_sequencer));
    }

    /// @notice Removes a sequencer from the allowed list.
    /// @param _sequencer The sequencer to be removed.
    function removeSequencer(T.SequencerKeys memory _sequencer) external {
        // Removing a sequencer is subject to the delay, so can only be called by the systemOwner.
        require(msg.sender == systemOwner, "SuperchainConfig: only systemOwner can remove a sequencer");
        bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencer);

        delete allowedSequencers[sequencerHash];
        emit ConfigUpdate(VERSION, UpdateType.REMOVE_SEQUENCER, abi.encode(_sequencer));
    }
}
