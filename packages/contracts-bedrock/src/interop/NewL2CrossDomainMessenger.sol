// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2CrossDomainMessenger as LegacyL2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { CrossL2Outbox } from "src/interop/CrossL2Outbox.sol";

import { Constants } from "src/libraries/Constants.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @title NewL2CrossDomainMessenger
/// @notice NewL2CrossDomainMessenger is an extended version of the existing predeploy
///         that supports interopability between many chains (L2-L2). Since we are
///         keeping the existing messaging contracts the same while iterating on the
////        L2-L2 implementation, changes to the interface and messaging format that
///         would reside in the CrossDomainMessenger are scoped internally in this
///         contract. The L2-L1 flow remains unchanged.
contract NewL2CrossDomainMessenger is ISemver {
    /// @custom:semver 0.0.1
    string public constant version = "0.0.1";

    // CrossDomainMessenger: copied internals & updated dispatch/message spec

    uint64 public constant RELAY_CONSTANT_OVERHEAD = 200_000;
    uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR = 64;
    uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR = 63;
    uint64 public constant MIN_GAS_CALLDATA_OVERHEAD = 16;
    uint64 public constant RELAY_CALL_OVERHEAD = 40_000;
    uint64 public constant RELAY_RESERVED_GAS = 40_000;
    uint64 public constant RELAY_GAS_CHECK_BUFFER = 5_000;

    /// @notice Chain identifier of the network
    bytes32 public immutable CHAIN_ID = bytes32(uint256(block.chainid));

    /// @notice Latest message version identifier (interop-enabled)
    uint16 public constant MESSAGE_VERSION = 2;

    /// @notice Nonce for the next message to be sent
    uint240 internal msgNonce;

    /// @notice Address of the sender of the currently executing message
    address internal xDomainMsgSender;

    /// @notice Mapping of succesfully delivered messages
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Mapping of delivered messages in a failed state
    mapping(bytes32 => bool) public failedMessages;

    /// @notice Emitted whenever a message is sent to the other chain.
    event SentMessage(
        uint256 indexed messageNonce,
        bytes32 indexed destination,
        address indexed target,
        address sender,
        bytes message,
        uint256 gasLimit,
        uint256 value
    );

    /// @notice Emitted whenever a message is successfully relayed on this chain.
    event RelayedMessage(uint256 indexed messageNonce, bytes32 indexed source_chain, bytes32 indexed msgHash);

    /// @notice Emitted whenever a message fails to be relayed on this chain.
    event FailedRelayedMessage(uint256 indexed messageNonce, bytes32 indexed source_chain, bytes32 indexed msgHash);

    /// @notice Checks if the call target is a blocked system address
    function _isUnsafeTarget(address _target) internal view returns (bool) {
        return _target == address(this) || _target == address(Predeploys.CROSS_L2_OUTBOX);
    }

    /// @notice See CrossDomainMessenger#baseGas as reference
    function baseGas(bytes calldata _message, uint32 _minGasLimit) public pure returns (uint64) {
        return RELAY_CONSTANT_OVERHEAD + (uint64(_message.length) * MIN_GAS_CALLDATA_OVERHEAD)
            + ((_minGasLimit * MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR) / MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR)
            + RELAY_CALL_OVERHEAD + RELAY_RESERVED_GAS + RELAY_GAS_CHECK_BUFFER;
    }

    /// @notice Sends a message to the specified destination chain. If the destination
    ///         chain is ETH Mainnet (0x1), then this message follows the legacy
    ///         message flow by forwarding to the existing L2CrossDomainMessenger
    ///         predeploy. When this implementation is ready as a replacement, L2ToL2
    ///         and L2ToL1 is natively handled by the internal message passers.
    function sendMessage(
        bytes32 _destination,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
        payable
    {
        // L2->L1 Support: Utilize the old pathway for now
        bytes32 ETH_MAINNET_ID = bytes32(uint256(1));
        if (_destination == ETH_MAINNET_ID) {
            LegacyL2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage(
                _target, _message, _minGasLimit
            );
            return;
        }

        require(_destination != CHAIN_ID, "NewL2CrossDomainMessenger: message cant be sent to self");

        uint256 nonce = Encoding.encodeVersionedNonce(msgNonce, MESSAGE_VERSION);
        bytes memory message = abi.encodeWithSelector(
            this.relayMessage.selector, nonce, CHAIN_ID, msg.sender, _target, msg.value, _minGasLimit, _message
        );

        // (1) Initiate the withdrawal
        // Once v2 is the official CrossDomainMessenger format, this contract will
        // replace existing L2CrossDomainmMessenger predeploy and switch between
        // L2ToL1MessagePasser/CrossL2Outbox based on the provided `_destination`.
        CrossL2Outbox(payable(Predeploys.CROSS_L2_OUTBOX)).initiateMessage(
            _destination, address(this), baseGas(_message, _minGasLimit), message
        );

        // (2) Emit Events
        emit SentMessage(nonce, _destination, _target, msg.sender, _message, _minGasLimit, msg.value);

        unchecked {
            ++msgNonce;
        }
    }

    /// @notice Relays a message that was sent from a different chain. Since L1-L2
    ///         messages utilize the V1 message format, this entrypoint only
    ///         supports relaying L2-L2 messages.
    function relayMessage(
        uint256 _nonce,
        bytes32 _source,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    )
        external
        payable
    {
        (, uint16 msgVersion) = Encoding.decodeVersionedNonce(_nonce);
        require(msgVersion == MESSAGE_VERSION, "NewL2CrossDomainMessenger: incorrect message version");

        bytes32 msgHash =
            hashCrossDomainMessageV2(_nonce, _source, CHAIN_ID, _sender, _target, _value, _minGasLimit, _message);
        require(successfulMessages[msgHash] == false, "NewL2CrossDomainMessenger: message already processed");

        // (1) Allow for replay. Initial replayer can only be the CrossL2Inbox as L1 messages
        // are relayed via old v1 pathway until v2 becomes the canonical message format.
        if (msg.sender == Predeploys.CROSS_L2_INBOX) {
            assert(msg.value == _value);
            assert(failedMessages[msgHash] == false);
        } else {
            require(msg.value == 0, "NewL2CrossDomainMessenger: cannot replay with additonal funds");
            require(failedMessages[msgHash], "NewL2CrossDomainMessenger: message cannot be replayed");
        }

        require(
            _isUnsafeTarget(_target) == false,
            "NewL2CrossDomainMessenger: cannot send message to blocked system address"
        );

        if (
            !SafeCall.hasMinGas(_minGasLimit, RELAY_RESERVED_GAS + RELAY_GAS_CHECK_BUFFER)
                || xDomainMsgSender != Constants.DEFAULT_L2_SENDER
        ) {
            failedMessages[msgHash] = true;
            emit FailedRelayedMessage(_nonce, _source, msgHash);

            // Revert in this case if the transaction was triggered by the estimation address
            if (tx.origin == Constants.ESTIMATION_ADDRESS) {
                revert("NewL2CrossDomainMessenger: failed to relay message");
            }

            return;
        }

        // (2) Relay Message
        xDomainMsgSender = _sender;
        bool success = SafeCall.call({
            _target: _target,
            _gas: gasleft() - RELAY_RESERVED_GAS,
            _value: _value,
            _calldata: _message
        });

        xDomainMsgSender = Constants.DEFAULT_L2_SENDER;
        if (success) {
            successfulMessages[msgHash] = true;
            emit RelayedMessage(_nonce, _source, msgHash);
        } else {
            failedMessages[msgHash] = true;
            emit FailedRelayedMessage(_nonce, _source, msgHash);

            // Revert in this case if the transaction was triggered by the estimation address
            if (tx.origin == Constants.ESTIMATION_ADDRESS) {
                revert("NewL2CrossDomainMessenger: failed to relay message");
            }
        }
    }

    // -----------------------------------------------------------------------------

    // Encoding.sol: v2 support

    function encodeCrossDomainMessageV2(
        uint256 _nonce,
        bytes32 _source,
        bytes32 _destination,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodeWithSignature(
            "relayMessage(uint256,bytes32,bytes32,address,address,uint256,uint256,bytes)",
            _nonce,
            _source,
            _destination,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
    }

    // Hashing.sol: v2 support

    function hashCrossDomainMessageV2(
        uint256 _nonce,
        bytes32 _source,
        bytes32 _destination,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    )
        internal
        pure
        returns (bytes32)
    {
        return keccak256(
            encodeCrossDomainMessageV2(_nonce, _source, _destination, _sender, _target, _value, _gasLimit, _data)
        );
    }
}
