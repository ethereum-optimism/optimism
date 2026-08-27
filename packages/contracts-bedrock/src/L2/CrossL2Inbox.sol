// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IL1EventRegistry } from "interfaces/L1/IL1EventRegistry.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { ILocalLogOracle } from "interfaces/L2/ILocalLogOracle.sol";

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
contract CrossL2Inbox is ProxyAdminOwnedBase, ISemver {
    /// @notice Thrown when trying to validate a cross chain message in a deposit transaction.
    error CrossL2Inbox_NoExecutingDeposits();

    /// @notice Thrown when the configured L1 event registry is the zero address.
    error CrossL2Inbox_InvalidEventRegistry();

    /// @notice Thrown when an event certificate is not imported by the configured L1 event registry.
    error CrossL2Inbox_NotEventRegistry();

    /// @notice Thrown when the local log oracle cannot find the claimed event.
    error CrossL2Inbox_EventNotFound();

    /// @notice Thrown when an event is older than the local lookup window.
    error CrossL2Inbox_EventTooOld();

    /// @notice Thrown when attempting to export an event from another chain.
    error CrossL2Inbox_EventFromAnotherChain();

    /// @notice Thrown when attempting to export an event that is not from a previous block.
    error CrossL2Inbox_EventNotInPreviousBlock();

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
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice Maximum age of an event accepted by the local log oracle.
    uint256 public constant EVENT_LOOKUP_WINDOW = 7 days;

    /// @notice Gas limit used to register an exported event on L1.
    uint256 public constant REGISTER_EVENT_GAS_LIMIT = 200_000;

    /// @notice The mask for the most significant bits of the checksum.
    /// @dev    Used to set the most significant byte to zero.
    bytes32 internal constant _MSB_MASK = bytes32(~uint256(0xff << 248));

    /// @notice Mask used to set the first byte of the bare checksum to 3 (0x03).
    bytes32 internal constant _TYPE_3_MASK = bytes32(uint256(0x03 << 248));

    /// @notice The threshold to use to know whether the slot is warm or not.
    uint256 internal constant _WARM_READ_THRESHOLD = 1000;

    /// @notice L1 registry trusted to import finalized event certificates.
    address public l1EventRegistry;

    /// @notice Event checksums certified through L1.
    mapping(bytes32 => bool) public certifiedMessages;

    /// @notice Emitted when a cross chain message is being executed.
    /// @param msgHash Hash of message payload being executed.
    /// @param id Encoded Identifier of the message.
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);

    /// @notice Emitted when an L1-certified event is being executed.
    /// @dev This is intentionally distinct from ExecutingMessage. Interop clients must not apply
    ///      the ordinary access-list dependency rules to this event.
    event ExecutingCertifiedMessage(bytes32 indexed msgHash, Identifier id);

    /// @notice Emitted when a locally proven event is exported through the L2ToL1MessagePasser.
    event EventExported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);

    /// @notice Emitted when an L1-certified event is imported on this chain.
    event EventImported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);

    /// @notice Emitted when the trusted L1 event registry changes.
    event L1EventRegistryUpdated(address indexed oldRegistry, address indexed newRegistry);

    /// @notice Configures the L1 event registry used by the censorship-resistant relay path.
    /// @dev The ProxyAdmin or its owner may update this value as part of an upgrade or migration.
    function setL1EventRegistry(address _l1EventRegistry) external {
        _assertOnlyProxyAdminOrProxyAdminOwner();
        if (_l1EventRegistry == address(0)) revert CrossL2Inbox_InvalidEventRegistry();

        address oldRegistry = l1EventRegistry;
        l1EventRegistry = _l1EventRegistry;
        emit L1EventRegistryUpdated(oldRegistry, _l1EventRegistry);
    }

    /// @notice Proves a recent event on this L2 and exports a certificate to L1.
    /// @dev The local log oracle is a consensus-critical execution-client precompile. The event
    ///      must be exported while its receipt remains inside the lookup window.
    function exportEvent(Identifier calldata _id, bytes32 _payloadHash) external {
        address registry = l1EventRegistry;
        if (registry == address(0)) revert CrossL2Inbox_InvalidEventRegistry();
        if (_id.chainId != block.chainid) revert CrossL2Inbox_EventFromAnotherChain();
        if (_id.blockNumber >= block.number || _id.timestamp > block.timestamp) {
            revert CrossL2Inbox_EventNotInPreviousBlock();
        }
        if (block.timestamp - _id.timestamp > EVENT_LOOKUP_WINDOW) revert CrossL2Inbox_EventTooOld();

        bool containsLog;
        try ILocalLogOracle(Predeploys.LOCAL_LOG_ORACLE).containsLog(_id, _payloadHash) returns (bool exists_) {
            containsLog = exists_;
        } catch {
            revert CrossL2Inbox_EventNotFound();
        }
        if (!containsLog) revert CrossL2Inbox_EventNotFound();

        bytes32 checksum = calculateChecksum(_id, _payloadHash);
        bytes memory data = abi.encodeCall(IL1EventRegistry.registerEvent, (_id, _payloadHash));
        IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).initiateWithdrawal(
            registry, REGISTER_EVENT_GAS_LIMIT, data
        );

        emit EventExported(checksum, _payloadHash, _id);
    }

    /// @notice Imports an event certificate delivered by the configured L1 event registry.
    function importEvent(Identifier calldata _id, bytes32 _payloadHash) external {
        _importEvent(_id, _payloadHash);
    }

    /// @notice Imports a certificate and relays a SentMessage event in the same deposit transaction.
    function importAndExecute(Identifier calldata _id, bytes calldata _sentMessage) external {
        _importEvent(_id, keccak256(_sentMessage));
        IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).relayMessage(_id, _sentMessage);
    }

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
        if (certifiedMessages[checksum]) {
            emit ExecutingCertifiedMessage(_msgHash, _id);
            return;
        }

        // Deposits have a zero gas price. When the base fee is positive, user transactions cannot.
        if (block.basefee > 0 && tx.gasprice == 0) revert CrossL2Inbox_NoExecutingDeposits();

        (bool isWarm,) = _isWarm(checksum);
        if (!isWarm) revert NotInAccessList();

        emit ExecutingMessage(_msgHash, _id);
    }

    /// @notice Imports an event certificate after authenticating its L1 sender.
    function _importEvent(Identifier calldata _id, bytes32 _payloadHash) internal {
        address registry = l1EventRegistry;
        if (registry == address(0) || AddressAliasHelper.undoL1ToL2Alias(msg.sender) != registry) {
            revert CrossL2Inbox_NotEventRegistry();
        }

        bytes32 checksum = calculateChecksum(_id, _payloadHash);
        certifiedMessages[checksum] = true;
        emit EventImported(checksum, _payloadHash, _id);
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
