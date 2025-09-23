// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @notice The struct for a pointer to a message payload in a remote (or local) chain.
/// @custom:field origin The origin address of the message.
/// @custom:field blockNumber The block number of the message.
/// @custom:field logIndex The log index of the message.
/// @custom:field timestamp The timestamp of the message.
/// @custom:field chainId The origin chain ID of the message.
struct Identifier {
    address origin;
    uint256 blockNumber;
    uint256 logIndex;
    uint256 timestamp;
    uint256 chainId;
}

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000022
/// @title CrossL2Inbox
/// @notice The CrossL2Inbox is responsible for executing a cross chain message on the destination
///         chain. It is permissionless to execute a cross chain message on behalf of any user.
/// @dev Processes cross-chain messages that are pre-declared in EIP-2930 access lists. Each message
///      requires three specific access-list entries to be valid. It will verify that the storage
///      slot containing the message checksum is "warm" (pre-accessed), which fails if not included
///      in the tx's access list. Nodes pre-check message validity before execution. The checksum
///      combines the message's `Identifier` and `msgHash` with type-3 bit masking.
contract CrossL2Inbox is ISemver {
    /// @notice Thrown when trying to validate a cross chain message with a checksum
    ///         that is invalid or was not provided in the transaction's access list to set the slot
    ///         as warm.
    error NotInAccessList();

    /// @notice Thrown when trying to validate a cross chain message with a block number
    ///         that is greater than 2^64.
    error BlockNumberTooHigh();

    /// @notice Thrown when trying to validate a cross chain message with a timestamp
    ///         that is greater than 2^64.
    error TimestampTooHigh();

    /// @notice Thrown when trying to validate a cross chain message with a log index
    ///         that is greater than 2^32.
    error LogIndexTooHigh();

    /// @notice Semantic version.
    /// @custom:semver 1.0.2
    string public constant version = "1.0.2";

    /// @notice The mask for the most significant bits of the checksum.
    /// @dev    Used to set the most significant byte to zero.
    bytes32 internal constant _MSB_MASK = bytes32(~uint256(0xff << 248));

    /// @notice Mask used to set the first byte of the bare checksum to 3 (0x03).
    bytes32 internal constant _TYPE_3_MASK = bytes32(uint256(0x03 << 248));

    /// @notice The threshold to use to know whether the slot is warm or not.
    uint256 internal constant _WARM_READ_THRESHOLD = 1000;

    /// @notice Emitted when a cross chain message is being executed.
    /// @param msgHash Hash of message payload being executed.
    /// @param id Encoded Identifier of the message.
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    /// @notice Validates a cross chain message on the destination chain and emits an ExecutingMessage
    ///         event. This function is useful for applications that understand the schema of the
    ///         message payload and want to process it in a custom way.
    /// @dev    Makes sure the checksum's slot is warm to ensure the tx included it in the access list.
    /// @dev    `Identifier.blockNumber` and `Identifier.timestamp` must be less than 2^64, whereas
    ///         `Identifier.logIndex` must be less than 2^32 to properly fit into the checksum.
    /// @param _id      Identifier of the message.
    /// @param _msgHash Hash of the message payload to call target with.
    function validateMessage(Identifier calldata _id, bytes32 _msgHash) external {
        bytes32 checksum = calculateChecksum(_id, _msgHash);
        (bool isWarm,) = _isWarm(checksum);
        if (!isWarm) revert NotInAccessList();

        emit ExecutingMessage(_msgHash, _id);
    }

    /// @notice Calculates a custom checksum for a cross chain message `Identifier` and `msgHash`.
    /// @param _id The identifier of the message.
    /// @param _msgHash The hash of the message.
    /// @return checksum_ The checksum of the message.
    function calculateChecksum(Identifier memory _id, bytes32 _msgHash) public pure returns (bytes32 checksum_) {
        if (_id.blockNumber > type(uint64).max) revert BlockNumberTooHigh();
        if (_id.logIndex > type(uint32).max) revert LogIndexTooHigh();
        if (_id.timestamp > type(uint64).max) revert TimestampTooHigh();

        // Hash the origin address and message hash together
        bytes32 logHash = keccak256(abi.encodePacked(_id.origin, _msgHash));

        // Downsize the identifier fields to match the needed type for the custom checksum calculation.
        uint64 blockNumber = uint64(_id.blockNumber);
        uint64 timestamp = uint64(_id.timestamp);
        uint32 logIndex = uint32(_id.logIndex);

        // Pack identifier fields with a left zero padding (uint96(0))
        bytes32 idPacked = bytes32(abi.encodePacked(uint96(0), blockNumber, timestamp, logIndex));

        // Hash the logHash with the packed identifier data
        bytes32 idLogHash = keccak256(abi.encodePacked(logHash, idPacked));

        // Create the final hash by combining idLogHash with chainId
        bytes32 bareChecksum = keccak256(abi.encodePacked(idLogHash, _id.chainId));

        // Apply bit masking to create the final checksum
        checksum_ = (bareChecksum & _MSB_MASK) | _TYPE_3_MASK;
    }

    /// @notice Checks if a slot is warm by measuring the gas cost of loading the slot.
    /// @dev    Stores and returns the slot value so that the compiler doesn't optimize out the
    ///         `sload`, this adds cost to the read
    /// @param _slot The slot to check.
    /// @return isWarm_ Whether the slot is warm.
    /// @return value_ The slot value.
    function _isWarm(bytes32 _slot) internal view returns (bool isWarm_, uint256 value_) {
        assembly {
            // Get the gas cost of the reading the slot with `sload`.
            let startGas := gas()
            value_ := sload(_slot)
            let endGas := gas()
            // If the gas cost of the `sload` is below than the threshold, the slot is warm.
            isWarm_ := iszero(gt(sub(startGas, endGas), _WARM_READ_THRESHOLD))
        }
    }
}
