// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Types as T } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

/// @title SuperchainConfig
/// @notice The SuperchainConfig contract is used to manage configuration of global superchain values.
/// @custom:audit none
contract SuperchainConfig is Initializable, ISemver {
    /// @notice Enum representing different types of updates.
    /// @custom:value SYSTEM_OWNER        Represents an update to the systemOwner.
    /// @custom:value INITIATOR           Represents an update to the initiator.
    /// @custom:value VETOER              Represents an update to the vetoer.
    /// @custom:value GUARDIAN            Represents an update to the guardian.
    /// @custom:value DELAY               Represents an update to the delay time.
    /// @custom:value ADD_SEQUENCER       Represents an update to add a sequencer to the allowed list.
    /// @custom:value REMOVE_SEQUENCER    Represents an update to remove a sequencer from the allowed list.
    enum UpdateType {
        SYSTEM_OWNER,
        INITIATOR,
        VETOER,
        GUARDIAN,
        DELAY,
        ADD_SEQUENCER,
        REMOVE_SEQUENCER
    }

    /// @notice The address of the systemOwner who may trigger an upgrade or change to critical config values.
    ///         This will be a DelayedVetoable contract.
    ///         It can only be modified by an upgrade.
    bytes32 public constant SYSTEM_OWNER_SLOT = bytes32(uint256(keccak256("superchainConfig.systemowner")) - 1);

    /// @notice The address of the initiator who may initiate an upgrade or change to critical config values, via the
    ///         systemOwner contract.
    ///         It can only be modified by an upgrade.
    bytes32 public constant INITIATOR_SLOT = bytes32(uint256(keccak256("superchainConfig.initiator")) - 1);

    /// @notice The address of the vetoer, who may veto an upgrade or change to critical config values.
    ///         This is expected to be the foundation.
    ///         It can only be modified by an upgrade.
    bytes32 public constant VETOER_SLOT = bytes32(uint256(keccak256("superchainConfig.vetoer")) - 1);

    /// @notice The address of the guardian, can pause the OptimismPortal.
    ///         It can only be modified by an upgrade.
    bytes32 public constant GUARDIAN_SLOT = bytes32(uint256(keccak256("superchainConfig.guardian")) - 1);

    /// @notice The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    ///         It can only be modified by an upgrade.
    bytes32 public constant DELAY_SLOT = bytes32(uint256(keccak256("superchainConfig.delay")) - 1);

    /// @notice The pause status of withdrawals from an chain in the superchain.
    ///         Set by the guardian role.
    bytes32 public constant PAUSED_SLOT = bytes32(uint256(keccak256("superchainConfig.paused")) - 1);

    /// @notice Mapping of allowed sequencers.
    ///         The initiator should be able to add to it instantly, but removing is subject to delay.
    mapping(bytes32 => bool) public allowedSequencers;

    /// @notice Emitted when the pause is triggered.
    event Paused();

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
            _systemOwner: address(0),
            _initiator: address(0),
            _vetoer: address(0),
            _guardian: address(0),
            _delay: 0,
            _sequencers: new T.SequencerKeys[](0)
        });
    }

    /// @notice Initializer.
    ///         The resource config must be set before the require check.
    /// @param _systemOwner     Initial owner of the contract.
    /// @param _initiator Address of the initiator who may initiate an upgrade or change to critical config values.
    /// @param _vetoer    Address of the vetoer.
    /// @param _guardian  Address of the guardian, can pause the OptimismPortal.
    /// @param _delay     The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    /// @param _sequencers The initial list of allowed sequencers
    function initialize(
        address _systemOwner,
        address _initiator,
        address _vetoer,
        address _guardian,
        uint256 _delay,
        T.SequencerKeys[] memory _sequencers
    )
        public
        reinitializer(2)
    {
        _setAddress(SYSTEM_OWNER_SLOT, _systemOwner);
        emit ConfigUpdate(UpdateType.SYSTEM_OWNER, abi.encode(_systemOwner));

        _setAddress(INITIATOR_SLOT, _initiator);
        emit ConfigUpdate(UpdateType.INITIATOR, abi.encode(_initiator));

        _setAddress(VETOER_SLOT, _vetoer);
        emit ConfigUpdate(UpdateType.VETOER, abi.encode(_vetoer));

        _setAddress(GUARDIAN_SLOT, _guardian);
        emit ConfigUpdate(UpdateType.GUARDIAN, abi.encode(_guardian));

        _setValue(DELAY_SLOT, _delay);
        emit ConfigUpdate(UpdateType.DELAY, abi.encode(_delay));

        for (uint256 i = 0; i < _sequencers.length; i++) {
            bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencers[i]);
            allowedSequencers[sequencerHash] = true;
            emit ConfigUpdate(UpdateType.ADD_SEQUENCER, abi.encode(_sequencers[i]));
        }
    }

    /// @notice Returns an address stored in an arbitrary storage slot.
    ///         These storage slots decouple the storage layout from
    ///         solc's automation.
    /// @param _slot The storage slot to retrieve the address from.
    function _getAddress(bytes32 _slot) internal view returns (address addr_) {
        assembly {
            addr_ := sload(_slot)
        }
    }

    /// @notice Stores an address in an arbitrary storage slot, `_slot`.
    /// @param _slot The storage slot to store the address in.
    /// @param _address The protocol version to store
    /// @dev WARNING! This function must be used cautiously, as it allows for overwriting addresses
    ///      in arbitrary storage slots.
    function _setAddress(bytes32 _slot, address _address) internal {
        assembly {
            sstore(_slot, _address)
        }
    }

    /// @notice Returns a uint256 stored in an arbitrary storage slot.
    ///         These storage slots decouple the storage layout from
    ///         solc's automation.
    /// @param _slot The storage slot to retrieve the address from.
    function _getValue(bytes32 _slot) internal view returns (uint256 value_) {
        assembly {
            value_ := sload(_slot)
        }
    }

    /// @notice Stores a value in an arbitrary storage slot, `_slot`.
    /// @param _slot The storage slot to store the address in.
    /// @param _value The protocol version to store
    /// @dev WARNING! This function must be used cautiously, as it allows for overwriting values
    ///      in arbitrary storage slots.
    function _setValue(bytes32 _slot, uint256 _value) internal {
        assembly {
            sstore(_slot, _value)
        }
    }

    /// @notice Getter for the systemOwner address.
    function systemOwner() public view returns (address systemOwner_) {
        systemOwner_ = _getAddress(SYSTEM_OWNER_SLOT);
    }

    /// @notice Getter for the initiator address.
    function initiator() public view returns (address initiator_) {
        initiator_ = _getAddress(INITIATOR_SLOT);
    }

    /// @notice Getter for the vetoer address.
    function vetoer() public view returns (address vetoer_) {
        vetoer_ = _getAddress(VETOER_SLOT);
    }

    /// @notice Getter for the guardian address.
    function guardian() public view returns (address guardian_) {
        guardian_ = _getAddress(GUARDIAN_SLOT);
    }

    /// @notice Getter for the delay address.
    function delay() public view returns (uint256 delay_) {
        // We do some casting rather than define a new getter.
        delay_ = uint256(uint160(_getAddress(DELAY_SLOT)));
    }

    /// @notice Getter for the paused address.
    function paused() public view returns (bool paused_) {
        paused_ = _getAddress(PAUSED_SLOT) != address(0);
    }

    /// @notice Pauses withdrawals.
    function pause() external {
        require(msg.sender == guardian(), "SuperchainConfig: only guardian can pause");
        _setValue(PAUSED_SLOT, uint256(1));
        emit Paused();
    }

    /// @notice Unpauses withdrawals.
    function unpause() external {
        require(msg.sender == guardian(), "SuperchainConfig: only guardian can unpause");
        _setValue(PAUSED_SLOT, uint256((0)));
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
        require(msg.sender == initiator(), "SuperchainConfig: only initiator can add sequencer");
        bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencer);

        allowedSequencers[sequencerHash] = true;
        emit ConfigUpdate(UpdateType.ADD_SEQUENCER, abi.encode(_sequencer));
    }

    /// @notice Removes a sequencer from the allowed list.
    /// @param _sequencer The sequencer to be removed.
    function removeSequencer(T.SequencerKeys memory _sequencer) external {
        // Removing a sequencer is subject to the delay, so can only be called by the systemOwner.
        require(msg.sender == systemOwner(), "SuperchainConfig: only systemOwner can remove a sequencer");
        bytes32 sequencerHash = Hashing.hashSequencerKeys(_sequencer);

        delete allowedSequencers[sequencerHash];
        emit ConfigUpdate(UpdateType.REMOVE_SEQUENCER, abi.encode(_sequencer));
    }
}
