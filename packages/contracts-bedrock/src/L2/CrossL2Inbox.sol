// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { Predeploys } from "src/libraries/Predeploys.sol";
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

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000022
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
contract CrossL2Inbox is ICrossL2Inbox, ISemver {
    /// @notice Transient storage slot that `entered` is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.entered")) - 1)
    bytes32 public constant ENTERED_SLOT = 0x6705f1f7a14e02595ec471f99cf251f123c2b0258ceb26554fcae9056c389a51;

    /// @notice Transient storage slot that the origin for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.origin")) - 1)
    bytes32 public constant ORIGIN_SLOT = 0xd2b7c5071ec59eb3ff0017d703a8ea513a7d0da4779b0dbefe845808c300c815;

    /// @notice Transient storage slot that the blocknumber for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.blocknumber")) - 1)
    bytes32 public constant BLOCKNUMBER_SLOT = 0x5a1da0738b7fdc60047c07bb519beb02aa32a8619de57e6258da1f1c2e020ccc;

    /// @notice Transient storage slot that the logIndex for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.logindex")) - 1)
    bytes32 public constant LOG_INDEX_SLOT = 0xab8acc221aecea88a685fabca5b88bf3823b05f335b7b9f721ca7fe3ffb2c30d;

    /// @notice Transient storage slot that the timestamp for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.timestamp")) - 1)
    bytes32 public constant TIMESTAMP_SLOT = 0x2e148a404a50bb94820b576997fd6450117132387be615e460fa8c5e11777e02;

    /// @notice Transient storage slot that the chainId for an Identifier is stored at.
    ///         Equal to bytes32(uint256(keccak256("crossl2inbox.identifier.chainid")) - 1)
    bytes32 public constant CHAINID_SLOT = 0x6e0446e8b5098b8c8193f964f1b567ec3a2bdaeba33d36acb85c1f1d3f92d313;

    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Enforces that cross domain message sender and source are set. Reverts if not.
    ///         This is leveraged to differentiate between 0 and nil at tstorage slots.
    modifier notEntered() {
        assembly {
            if eq(tload(ENTERED_SLOT), 0) {
                mstore(0x00, 0xbca35af6) // 0xbca35af6 is the 4-byte selector of "NotEntered()"
                revert(0x1C, 0x04)
            }
        }
        _;
    }

    /// @notice Returns the origin address of the Identifier. If not entered, reverts.
    /// @return _origin The origin address of the Identifier.
    function origin() external view notEntered returns (address _origin) {
        assembly {
            _origin := tload(ORIGIN_SLOT)
        }
    }

    /// @notice Returns the block number of the Identifier. If not entered, reverts.
    /// @return _blocknumber The block number of the Identifier.
    function blocknumber() external view notEntered returns (uint256 _blocknumber) {
        assembly {
            _blocknumber := tload(BLOCKNUMBER_SLOT)
        }
    }

    /// @notice Returns the log index of the Identifier. If not entered, reverts.
    /// @return _logIndex The log index of the Identifier.
    function logIndex() external view notEntered returns (uint256 _logIndex) {
        assembly {
            _logIndex := tload(LOG_INDEX_SLOT)
        }
    }

    /// @notice Returns the timestamp of the Identifier. If not entered, reverts.
    /// @return _timestamp The timestamp of the Identifier.
    function timestamp() external view notEntered returns (uint256 _timestamp) {
        assembly {
            _timestamp := tload(TIMESTAMP_SLOT)
        }
    }

    /// @notice Returns the chain ID of the Identifier. If not entered, reverts.
    /// @return _chainId The chain ID of the Identifier.
    function chainId() external view notEntered returns (uint256 _chainId) {
        assembly {
            _chainId := tload(CHAINID_SLOT)
        }
    }

    /// @notice Executes a cross chain message on the destination chain.
    /// @param _id An Identifier pointing to the initiating message.
    /// @param _target Account that is called with _msg.
    /// @param _msg The message payload, matching the initiating message.
    function executeMessage(Identifier calldata _id, address _target, bytes memory _msg) external payable {
        require(msg.sender == tx.origin, "CrossL2Inbox: not EOA sender");
        require(_id.timestamp <= block.timestamp, "CrossL2Inbox: invalid id timestamp");
        require(
            IDependencySet(Predeploys.L1_BLOCK_ATTRIBUTES).isInDependencySet(_id.chainId),
            "CrossL2Inbox: id chain not in dependency set"
        );

        // Store the Identifier in transient storage.
        _storeIdentifier();

        // Call the target account with the message payload.
        bool success = _callWithAllGas(_target, _msg);

        // Revert if the target call failed.
        require(success, "CrossL2Inbox: target call failed");
    }

    /// @notice Stores the Identifier in transient storage.
    function _storeIdentifier() internal {
        assembly {
            // update `entered` to non-zero
            tstore(ENTERED_SLOT, 1)

            tstore(ORIGIN_SLOT, calldataload(4))
            tstore(BLOCKNUMBER_SLOT, calldataload(36))
            tstore(LOG_INDEX_SLOT, calldataload(68))
            tstore(TIMESTAMP_SLOT, calldataload(100))
            tstore(CHAINID_SLOT, calldataload(132))
        }
    }

    /// @notice Calls the target account with the message payload and all available gas.
    function _callWithAllGas(address _target, bytes memory _msg) internal returns (bool _success) {
        assembly {
            _success :=
                call(
                    gas(), // gas
                    _target, // recipient
                    callvalue(), // ether value
                    add(_msg, 32), // inloc
                    mload(_msg), // inlen
                    0, // outloc
                    0 // outlen
                )
        }
    }
}
