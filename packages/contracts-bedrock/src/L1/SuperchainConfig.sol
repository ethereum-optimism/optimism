// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { ISemver } from "../universal/ISemver.sol";

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

    /// @title SequencerKeys
    /// @notice The SequencerKeys struct is used to store the batcherHash and unsafeBlockSigner keys for sequencers.
    struct SequencerKeys {
        /// @notice The batcherHash key for a sequencer.
        bytes32 batcherHash;
        /// @notice The unsafeBlockSigner key for a sequencer.
        bytes32 unsafeBlockSigner;
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
            initiator: address(0),
            vetoer: address(0),
            guardian: address(0),
            delay: 0,
            paused: false,
            sequencers: new SequencerKeys[](0)
        });
    }

    /// @notice Initializer.
    ///         The resource config must be set before the require check.
    /// @param _owner     Initial owner of the contract.
    /// @param _initiator Address of the initiator who may initiate an upgrade or change to critical config values.
    /// @param _vetoer    Address of the vetoer.
    /// @param _guardian  Address of the guardian, can pause the OptimismPortal.
    /// @param _delay     The delay time in seconds between when an upgrade is initiated and when it can be finalized.
    /// @param _paused    The pause status of withdrawals from an chain in the superchain.
    /// @param _sequencers The initial list of allowed sequencers
    function initialize(
        address _owner,
        address _initiator,
        address _vetoer,
        address _guardian,
        uint256 _delay,
        bool _paused,
        SequencerKeys[] calldata _sequencers
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
        paused = false;

        for (uint256 i = 0; i < _sequencers.length; i++) {
            bytes32 fingerprint = keccak256(abi.encode(_sequencers[i]));
            allowedSequencers[fingerprint] = true;
        }
    }

    // todo(maurelian): Add a bunch of getters and setters in the style of the SystemConfig contract.
    //   This is straightforward work, so can defer until the arch is fully sketched out.
}
