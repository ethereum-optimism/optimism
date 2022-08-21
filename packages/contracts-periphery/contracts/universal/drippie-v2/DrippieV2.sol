// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { VM } from "./weiroll/VM.sol";
import { AssetReceiver } from "../AssetReceiver.sol";

/**
 * @title DrippieV2
 */
contract DrippieV2 is AssetReceiver, VM {
    /**
     * @notice Enum representing different status options for a given drip.
     *
     * @custom:value NONE     Drip does not exist.
     * @custom:value ACTIVE   Drip is active and can be executed.
     * @custom:value PAUSED   Drip is paused and cannot be executed until reactivated.
     * @custom:value ARCHIVED Drip is archived and can no longer be executed or reactivated.
     */
    enum DripStatus {
        NONE,
        ACTIVE,
        PAUSED,
        ARCHIVED
    }

    /**
     * @notice Represents a drip action.
     */
    struct DripAction {
        address payable target;
        bytes data;
        uint256 value;
    }

    /**
     * @notice Represents the configuration for a given drip.
     */
    struct DripConfig {
        bytes32[] init;
        bytes32[] checks;
        bytes32[] actions;
        bytes[] stateI;
        bytes[] stateC;
        bytes[] stateA;
    }

    /**
     * @notice Represents the state of an active drip.
     */
    struct DripState {
        DripStatus status;
        DripConfig config;
        uint256 last;
        uint256 count;
    }

    /**
     * @notice Emitted when a new drip is created.
     *
     * @param nameref Indexed name parameter (hashed).
     * @param name    Unindexed name parameter (unhashed).
     * @param config  Config for the created drip.
     */
    event DripCreated(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        DripConfig config
    );

    /**
     * @notice Emitted when a drip status is updated.
     *
     * @param nameref Indexed name parameter (hashed).
     * @param name    Unindexed name parameter (unhashed).
     * @param status  New drip status.
     */
    event DripStatusUpdated(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        DripStatus status
    );

    /**
     * @notice Emitted when a drip is executed.
     *
     * @param nameref   Indexed name parameter (hashed).
     * @param name      Unindexed name parameter (unhashed).
     * @param executor  Address that executed the drip.
     * @param timestamp Time when the drip was executed.
     */
    event DripExecuted(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        address executor,
        uint256 timestamp
    );

    /**
     * @notice Maps from drip names to drip states.
     */
    mapping(string => DripState) public drips;

    /**
     * @param _owner Initial contract owner.
     */
    constructor(address _owner) AssetReceiver(_owner) {}

    /**
     * @notice Creates a new drip with the given name and configuration. Once created, drips cannot
     *         be modified in any way (this is a security measure). If you want to update a drip,
     *         simply pause (and potentially archive) the existing drip and create a new one.
     *
     * @param _name   Name of the drip.
     * @param _config Configuration for the drip.
     */
    function create(string memory _name, DripConfig memory _config) external onlyOwner {
        // Make sure this drip doesn't already exist. We *must* guarantee that no other function
        // will ever set the status of a drip back to NONE after it's been created. This is why
        // archival is a separate status.
        require(
            drips[_name].status == DripStatus.NONE,
            "Drippie: drip with that name already exists"
        );

        // We initialize this way because Solidity won't let us copy arrays into storage yet.
        DripState storage state = drips[_name];
        state.status = DripStatus.PAUSED;
        state.config.checks = _config.checks;
        state.config.actions = _config.actions;
        state.config.stateC = _config.stateC;
        state.config.stateA = _config.stateA;

        // Execute weiroll.
        bytes[] memory stateI = _config.stateI;
        _execute(_config.init, stateI);

        // Tell the world!
        emit DripCreated(_name, _name, _config);
    }

    /**
     * @notice Sets the status for a given drip. The behavior of this function depends on the
     *         status that the user is trying to set. A drip can always move between ACTIVE and
     *         PAUSED, but it can never move back to NONE and once ARCHIVED, it can never move back
     *         to ACTIVE or PAUSED.
     *
     * @param _name   Name of the drip to update.
     * @param _status New drip status.
     */
    function status(string memory _name, DripStatus _status) external onlyOwner {
        // Make sure we can never set drip status back to NONE. A simple security measure to
        // prevent accidental overwrites if this code is ever updated down the line.
        require(
            _status != DripStatus.NONE,
            "Drippie: drip status can never be set back to NONE after creation"
        );

        // Make sure the drip in question actually exists. Not strictly necessary but there doesn't
        // seem to be any clear reason why you would want to do this, and it may save some gas in
        // the case of a front-end bug.
        require(
            drips[_name].status != DripStatus.NONE,
            "Drippie: drip with that name does not exist"
        );

        // Once a drip has been archived, it cannot be un-archived. This is, after all, the entire
        // point of archiving a drip.
        require(
            drips[_name].status != DripStatus.ARCHIVED,
            "Drippie: drip with that name has been archived"
        );

        // Although not strictly necessary, we make sure that the status here is actually changing.
        // This may save the client some gas if there's a front-end bug and the user accidentally
        // tries to "change" the status to the same value as before.
        require(
            drips[_name].status != _status,
            "Drippie: cannot set drip status to same status as before"
        );

        // If the user is trying to archive this drip, make sure the drip has been paused. We do
        // not allow users to archive active drips so that the effects of this action are more
        // abundantly clear.
        if (_status == DripStatus.ARCHIVED) {
            require(
                drips[_name].status == DripStatus.PAUSED,
                "Drippie: drip must be paused to be archived"
            );
        }

        // If we made it here then we can safely update the status.
        drips[_name].status = _status;
        emit DripStatusUpdated(_name, _name, drips[_name].status);
    }

    /**
     * @notice Checks if a given drip is executable.
     *
     * @param _name Drip to check.
     *
     * @return True if the drip is executable, false otherwise.
     */
    function executable(string memory _name) public returns (bool) {
        DripState storage state = drips[_name];

        // Only allow active drips to be executed, an obvious security measure.
        require(
            state.status == DripStatus.ACTIVE,
            "Drippie: selected drip does not exist or is not currently active"
        );

        // Execute weiroll.
        bytes[] memory stateC = drips[_name].config.stateC;
        _execute(state.config.checks, stateC);

        // Prevent state from being updated unless it's because this contract is updating it.
        if (msg.sender == address(this)) {
            return true;
        } else {
            revert("Drippie: drip is executable but we are reverting for safety");
        }
    }

    /**
     * @notice Triggers a drip. This function is deliberately left as a public function because the
     *         assumption being made here is that setting the drip to ACTIVE is an affirmative
     *         signal that the drip should be executable according to the drip parameters, drip
     *         check, and drip interval. Note that drip parameters are read entirely from the state
     *         and are not supplied as user input, so there should not be any way for a
     *         non-authorized user to influence the behavior of the drip.
     *
     * @param _name Name of the drip to trigger.
     */
    function drip(string memory _name) external {
        DripState storage state = drips[_name];

        // Make sure the drip can be executed.
        require(
            this.executable(_name) == true,
            "Drippie: drip cannot be executed at this time, try again later"
        );

        // Execute weiroll.
        bytes[] memory stateA = drips[_name].config.stateA;
        _execute(state.config.actions, stateA);

        // Update count and emit event.
        state.count++;
        emit DripExecuted(_name, _name, msg.sender, block.timestamp);
    }
}
