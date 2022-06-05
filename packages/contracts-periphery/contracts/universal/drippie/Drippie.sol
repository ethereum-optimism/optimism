// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { AssetReceiver } from "../AssetReceiver.sol";
import { IDripCheck } from "./IDripCheck.sol";

/**
 * @title Drippie
 * @notice Drippie is a system for managing automated contract interactions. A specific interaction
 * is called a "drip" and can be executed according to some condition (called a dripcheck) and an
 * execution interval. Drips cannot be executed faster than the execution interval. Drips can
 * trigger arbitrary contract calls where the calling contract is this contract address. Drips can
 * also send ETH value, which makes them ideal for keeping addresses sufficiently funded with ETH.
 * Drippie is designed to be connected with smart contract automation services so that drips can be
 * executed automatically. However, Drippie is specifically designed to be separated from these
 * services so that trust assumptions are better compartmentalized.
 */
contract Drippie is AssetReceiver {
    /**
     * Enum representing different status options for a given drip.
     */
    enum DripStatus {
        NONE,
        ACTIVE,
        PAUSED,
        ARCHIVED
    }

    /**
     * Represents a drip action.
     */
    struct DripAction {
        address payable target;
        bytes data;
        uint256 value;
    }

    /**
     * Represents the configuration for a given drip.
     */
    struct DripConfig {
        uint256 interval;
        IDripCheck dripcheck;
        bytes checkparams;
        DripAction[] actions;
    }

    /**
     * Represents the state of an active drip.
     */
    struct DripState {
        DripStatus status;
        DripConfig config;
        uint256 last;
        uint256 count;
    }

    /**
     * Emitted when a new drip is created.
     */
    event DripCreated(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        DripConfig config
    );

    /**
     * Emitted when a drip status is updated.
     */
    event DripStatusUpdated(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        DripStatus status
    );

    /**
     * Emitted when a drip is executed.
     */
    event DripExecuted(
        // Emit name twice because indexed version is hashed.
        string indexed nameref,
        string name,
        address executor,
        uint256 timestamp
    );

    /**
     * Maps from drip names to drip states.
     */
    mapping(string => DripState) public drips;

    /**
     * @param _owner Initial contract owner.
     */
    constructor(address _owner) AssetReceiver(_owner) {}

    /**
     * Creates a new drip with the given name and configuration. Once created, drips cannot be
     * modified in any way (this is a security measure). If you want to update a drip, simply pause
     * (and potentially archive) the existing drip and create a new one.
     *
     * @param _name Name of the drip.
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
        state.config.interval = _config.interval;
        state.config.dripcheck = _config.dripcheck;
        state.config.checkparams = _config.checkparams;

        // Solidity doesn't let us copy arrays into storage, so we push each array one by one.
        for (uint256 i = 0; i < _config.actions.length; i++) {
            state.config.actions.push(_config.actions[i]);
        }

        // Tell the world!
        emit DripCreated(_name, _name, _config);
    }

    /**
     * Sets the status for a given drip. The behavior of this function depends on the status that
     * the user is trying to set. A drip can always move between ACTIVE and PAUSED, but it can
     * never move back to NONE and once ARCHIVED, it can never move back to ACTIVE or PAUSED.
     *
     * @param _name Name of the drip to update.
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
     * Checks if a given drip is executable.
     *
     * @param _name Drip to check.
     * @return True if the drip is executable, false otherwise.
     */
    function executable(string memory _name) public view returns (bool) {
        DripState storage state = drips[_name];

        // Only allow active drips to be executed, an obvious security measure.
        require(
            state.status == DripStatus.ACTIVE,
            "Drippie: selected drip does not exist or is not currently active"
        );

        // Don't drip if the drip interval has not yet elapsed since the last time we dripped. This
        // is a safety measure that prevents a malicious recipient from, e.g., spending all of
        // their funds and repeatedly requesting new drips. Limits the potential impact of a
        // compromised recipient to just a single drip interval, after which the drip can be paused
        // by the owner address.
        require(
            state.last + state.config.interval <= block.timestamp,
            "Drippie: drip interval has not elapsed since last drip"
        );

        // Make sure we're allowed to execute this drip.
        require(
            state.config.dripcheck.check(state.config.checkparams),
            "Drippie: dripcheck failed so drip is not yet ready to be triggered"
        );

        // Alright, we're good to execute.
        return true;
    }

    /**
     * Triggers a drip. This function is deliberately left as a public function because the
     * assumption being made here is that setting the drip to ACTIVE is an affirmative signal that
     * the drip should be executable according to the drip parameters, drip check, and drip
     * interval. Note that drip parameters are read entirely from the state and are not supplied as
     * user input, so there should not be any way for a non-authorized user to influence the
     * behavior of the drip.
     *
     * @param _name Name of the drip to trigger.
     */
    function drip(string memory _name) external {
        DripState storage state = drips[_name];

        // Make sure the drip can be executed.
        require(
            executable(_name) == true,
            "Drippie: drip cannot be executed at this time, try again later"
        );

        // Update the last execution time for this drip before the call. Note that it's entirely
        // possible for a drip to be executed multiple times per block or even multiple times
        // within the same transaction (via re-entrancy) if the drip interval is set to zero. Users
        // should set a drip interval of 1 if they'd like the drip to be executed only once per
        // block (since this will then prevent re-entrancy).
        state.last = block.timestamp;

        // Execute each action in the drip. We allow drips to have multiple actions because there
        // are scenarios in which a contract must do multiple things atomically. For example, the
        // contract may need to withdraw ETH from one account and then deposit that ETH into
        // another account within the same transaction.
        uint256 len = state.config.actions.length;
        for (uint256 i = 0; i < len; i++) {
            // Must be marked as "storage" because copying structs into memory is not yet supported
            // by Solidity. Won't significantly reduce gas costs but at least makes it easier to
            // read what the rest of this section is doing.
            DripAction storage action = state.config.actions[i];

            // Actually execute the action. We could use ExcessivelySafeCall here but not strictly
            // necessary (worst case, a drip gets bricked IFF the target is malicious, doubt this
            // will ever happen in practice). Could save a marginal amount of gas to ignore the
            // returndata.
            // slither-disable-next-line calls-loop
            (bool success, ) = action.target.call{ value: action.value }(action.data);

            // Generally should not happen, but could if there's a misconfiguration (e.g., passing
            // the wrong data to the target contract), the recipient is not payable, or
            // insufficient gas was supplied to this transaction. We revert so the drip can be
            // fixed and triggered again later. Means we cannot emit an event to alert of the
            // failure, but can reasonably be detected by off-chain services even without an event.
            // Note that this forces the drip executor to supply sufficient gas to the call
            // (assuming there is some sufficient gas limit that exists, otherwise the drip will
            // not execute).
            require(
                success,
                "Drippie: drip was unsuccessful, please check your configuration for mistakes"
            );
        }

        state.count++;
        emit DripExecuted(_name, _name, msg.sender, block.timestamp);
    }
}
