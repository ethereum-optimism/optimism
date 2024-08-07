// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { TransientContext, TransientReentrancyAware } from "src/libraries/TransientContext.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

/// @title IDependencySet
/// @notice Interface for L1Block with only `isInDependencySet(uint256)` method.
interface IDependencySet {
    /// @notice Returns true if the chain associated with input chain ID is in the interop dependency set.
    ///         Every chain is in the interop dependency set of itself.
    /// @param _chainId Input chain ID.
    /// @return True if the input chain ID corresponds to a chain in the interop dependency set, and false otherwise.
    function isInDependencySet(uint256 _chainId) external view returns (bool);
}

/// @notice Thrown when a non-written transient storage slot is attempted to be read from.
error NotEntered();

/// @notice Thrown when trying to execute a cross chain message with an invalid Identifier timestamp.
error InvalidTimestamp();

/// @notice Thrown when trying to execute a cross chain message with an invalid Identifier chain ID.
error InvalidChainId();

/// @notice Thrown when trying to execute a cross chain message and the target call fails.
error TargetCallFailed();

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000022
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
contract CrossL2Inbox is ICrossL2Inbox, ISemver, TransientReentrancyAware {
    /// @notice Transient storage slot that the origin for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.origin")) - 1)
    bytes32 internal constant ORIGIN_SLOT = 0xd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c815;

    /// @notice Transient storage slot that the blockNumber for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.blocknumber")) - 1)
    bytes32 internal constant BLOCK_NUMBER_SLOT = 0x5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc;

    /// @notice Transient storage slot that the logIndex for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.logindex")) - 1)
    bytes32 internal constant LOG_INDEX_SLOT = 0xab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d;

    /// @notice Transient storage slot that the timestamp for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.timestamp")) - 1)
    bytes32 internal constant TIMESTAMP_SLOT = 0x2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e02;

    /// @notice Transient storage slot that the chainId for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.chainid")) - 1)
    bytes32 internal constant CHAINID_SLOT = 0x6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d313;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.4
    string public constant version = "1.0.0-beta.4";

    /// @notice Emitted when a cross chain message is being executed.
    /// @param msgHash Hash of message payload being executed.
    /// @param id Encoded Identifier of the message.
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    /// @notice Returns the origin address of the Identifier. If not entered, reverts.
    /// @return Origin address of the Identifier.
    function origin() external view notEntered returns (address) {
        return address(uint160(TransientContext.get(ORIGIN_SLOT)));
    }

    /// @notice Returns the block number of the Identifier. If not entered, reverts.
    /// @return Block number of the Identifier.
    function blockNumber() external view notEntered returns (uint256) {
        return TransientContext.get(BLOCK_NUMBER_SLOT);
    }

    /// @notice Returns the log index of the Identifier. If not entered, reverts.
    /// @return Log index of the Identifier.
    function logIndex() external view notEntered returns (uint256) {
        return TransientContext.get(LOG_INDEX_SLOT);
    }

    /// @notice Returns the timestamp of the Identifier. If not entered, reverts.
    /// @return Timestamp of the Identifier.
    function timestamp() external view notEntered returns (uint256) {
        return TransientContext.get(TIMESTAMP_SLOT);
    }

    /// @notice Returns the chain ID of the Identifier. If not entered, reverts.
    /// @return _chainId The chain ID of the Identifier.
    function chainId() external view notEntered returns (uint256) {
        return TransientContext.get(CHAINID_SLOT);
    }

    /// @notice Executes a cross chain message on the destination chain.
    /// @param _id      Identifier of the message.
    /// @param _target  Target address to call.
    /// @param _message Message payload to call target with.
    function executeMessage(
        Identifier calldata _id,
        address _target,
        bytes memory _message
    )
        external
        payable
        reentrantAware
    {
        // Check the Identifier.
        _checkIdentifier(_id);

        // Store the Identifier in transient storage.
        _storeIdentifier(_id);

        // Call the target account with the message payload.
        bool success = SafeCall.call(_target, msg.value, _message);

        // Revert if the target call failed.
        if (!success) revert TargetCallFailed();

        emit ExecutingMessage(keccak256(_message), _id);
    }

    /// @notice Validates a cross chain message on the destination chain
    ///         and emits an ExecutingMessage event. This function is useful
    ///         for applications that understand the schema of the _message payload and want to
    ///         process it in a custom way.
    /// @param _id      Identifier of the message.
    /// @param _msgHash Hash of the message payload to call target with.
    function validateMessage(Identifier calldata _id, bytes32 _msgHash) external {
        // Check the Identifier.
        _checkIdentifier(_id);

        emit ExecutingMessage(_msgHash, _id);
    }

    /// @notice Validates that for a given cross chain message identifier,
    ///         it's timestamp is not in the future and the source chainId
    ///         is in the destination chain's dependency set.
    /// @param _id Identifier of the message.
    function _checkIdentifier(Identifier calldata _id) internal view {
        if (_id.timestamp > block.timestamp) revert InvalidTimestamp();
        if (!IDependencySet(Predeploys.L1_BLOCK_ATTRIBUTES).isInDependencySet(_id.chainId)) {
            revert InvalidChainId();
        }
    }

    /// @notice Stores the Identifier in transient storage.
    /// @param _id Identifier to store.
    function _storeIdentifier(Identifier calldata _id) internal {
        TransientContext.set(ORIGIN_SLOT, uint160(_id.origin));
        TransientContext.set(BLOCK_NUMBER_SLOT, _id.blockNumber);
        TransientContext.set(LOG_INDEX_SLOT, _id.logIndex);
        TransientContext.set(TIMESTAMP_SLOT, _id.timestamp);
        TransientContext.set(CHAINID_SLOT, _id.chainId);
    }
}
