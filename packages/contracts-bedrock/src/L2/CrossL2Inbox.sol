// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { Predeploys } from "src/libraries/Predeploys.sol";
import { TransientContext, TransientReentrancyAware } from "src/libraries/TransientContext.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { ICrossL2Inbox } from "src/L2/ICrossL2Inbox.sol";

/// @title IDependencySet
/// @notice Interface for L1Block with only `isInDependencySet(uint256)` method.
interface IDependencySet {
    /// @notice Returns true iff the chain associated with input chain ID is in the interop dependency set.
    ///         Every chain is in the interop dependency set of itself.
    /// @param _chainId The input chain ID.
    /// @return True if the input chain ID corresponds to a chain in the interop dependency set. False otherwise.
    function isInDependencySet(uint256 _chainId) external view returns (bool);
}

/// @notice Thrown when a non-written tstore slot is attempted to be read from.
error NotEntered();

/// @notice Thrown when trying to execute a cross chain message with an invalid Identifier timestamp.
/// @param timestamp The timestamp of the Identifier.
/// @param blockTimestamp The current block timestamp.
error InvalidIdTimestamp(uint256 timestamp, uint256 blockTimestamp);

/// @notice Thrown when trying to execute a cross chain message with a chain ID that is not in the dependency set.
/// @param chainId The chain ID of the Identifier.
error ChainNotInDependencySet(uint256 chainId);

/// @notice Thrown when trying to execute a cross chain message and the target call fails.
/// @param target The target account that was called.
/// @param message The message payload that was sent to the target account.
error TargetCallFailed(address target, bytes message);

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000022
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
contract CrossL2Inbox is ICrossL2Inbox, ISemver, TransientReentrancyAware {
    /// @notice Transient storage slot that `entered` is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.entered")) - 1)
    bytes32 internal constant ENTERED_SLOT = 0x6705f1f7a14e02595ec471f99cf251f123c2b0258ceb26554fcae9056c389a51;

    /// @notice Transient storage slot that the origin for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.origin")) - 1)
    bytes32 internal constant ORIGIN_SLOT = 0xd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c815;

    /// @notice Transient storage slot that the blocknumber for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.blocknumber")) - 1)
    bytes32 internal constant BLOCKNUMBER_SLOT = 0x5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc;

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
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Enforces that cross domain message sender and source are set. Reverts if not.
    ///         This is leveraged to differentiate between 0 and nil at tstorage slots.
    modifier notEntered() {
        if (TransientContext.get(ENTERED_SLOT) == 0) revert NotEntered();
        _;
    }

    /// @notice Returns the origin address of the Identifier. If not entered, reverts.
    /// @return Origin address of the Identifier.
    function origin() external view notEntered returns (address) {
        return address(uint160(TransientContext.get(ORIGIN_SLOT)));
    }

    /// @notice Returns the block number of the Identifier. If not entered, reverts.
    /// @return Block number of the Identifier.
    function blockNumber() external view notEntered returns (uint256) {
        return TransientContext.get(BLOCKNUMBER_SLOT);
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
    /// @param _id An Identifier pointing to the initiating message.
    /// @param _target Account that is called with _message.
    /// @param _message The message payload, matching the initiating message.
    function executeMessage(
        Identifier calldata _id,
        address _target,
        bytes memory _message
    )
        external
        payable
        reentrantAware
    {
        if (_id.timestamp > block.timestamp) revert InvalidIdTimestamp(_id.timestamp, block.timestamp);
        if (!IDependencySet(Predeploys.L1_BLOCK_ATTRIBUTES).isInDependencySet(_id.chainId)) {
            revert ChainNotInDependencySet(_id.chainId);
        }

        // Store the Identifier in transient storage.
        _storeIdentifier(_id);

        // Call the target account with the message payload.
        bool success = _callWithAllGas(_target, _message);

        // Revert if the target call failed.
        if (!success) revert TargetCallFailed(_target, _message);
    }

    /// @notice Stores the Identifier in transient storage.
    /// @param _id Identifier to store.
    function _storeIdentifier(Identifier calldata _id) internal {
        // Update `entered` to non-zero
        TransientContext.set(ENTERED_SLOT, 1);

        TransientContext.set(ORIGIN_SLOT, uint160(_id.origin));
        TransientContext.set(BLOCKNUMBER_SLOT, _id.blockNumber);
        TransientContext.set(LOG_INDEX_SLOT, _id.logIndex);
        TransientContext.set(TIMESTAMP_SLOT, _id.timestamp);
        TransientContext.set(CHAINID_SLOT, _id.chainId);
    }

    /// @notice Calls the target account with the message payload and all available gas.
    function _callWithAllGas(address _target, bytes memory _message) internal returns (bool _success) {
        assembly {
            _success :=
                call(
                    gas(), // gas
                    _target, // recipient
                    callvalue(), // ether value
                    add(_message, 32), // inloc
                    mload(_message), // inlen
                    0, // outloc
                    0 // outlen
                )
        }
    }
}
