// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2CrossDomainMessenger as LegacyL2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
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

    // CrossDomainMessenger: some copied internals & updated dispatch/message spec

    /// @notice Latest message version identifier (interop-enabled)
    uint16 public constant MESSAGE_VERSION = 2;

    /// @notice Chain identifier of the network
    bytes32 public immutable CHAIN_ID = bytes32(uint256(block.chainid));

    // TODO: Events

    /// @notice Nonce for the next message to be sent
    uint240 internal msgNonce;

    /// @notice Address of the sender of the currently executing message
    address internal xDomainMsgSender;

    /// @notice Mapping of succesfully delivered messages
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Mapping of delivered messages in a failed state
    mapping(bytes32 => bool) public failedMessages;

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
        // L2->L1 Support: Required as we are leaving the CrossDomainMessenger untouched
        bytes32 ETH_MAINNET_ID = bytes32(uint256(1));
        if (_destination == ETH_MAINNET_ID) {
            LegacyL2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage(
                _target, _message, _minGasLimit
            );
            return;
        }

        require(_destination != CHAIN_ID, "NewL2CrossDomainMessenger: message cant be sent to self");

        uint256 nonce = Encoding.encodeVersionedNonce(msgNonce, MESSAGE_VERSION);
        bytes memory data = abi.encodeWithSelector(
            this.relayMessage.selector, nonce, CHAIN_ID, msg.sender, _target, msg.value, _minGasLimit, _message
        );

        // (1) Send to the CrossL2Inbox.
        // With an updated CrossDomainMessenger message format, this contract will
        // replace existing L2CrossDomainmMessenger predeploy and switch between
        // L2ToL1MessagePasser/CrossL2Inbox based on the provided `_destination`.
        //
        // i.e: CrossL2Inbox.sendMessage(source, destination, _minGasLimit, msg.value, data)

        // (2) Emit Events

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

        bytes32 messageHash =
            hashCrossDomainMessageV2(_nonce, _source, CHAIN_ID, _sender, _target, _value, _minGasLimit, _message);
        require(successfulMessages[messageHash] == false, "NewL2CrossDomainMessenger: message already processed");

        // (1) Allow for replay
        if (_sender == address(this)) {
            assert(msg.value == _value);
            assert(failedMessages[messageHash] == false);
        } else {
            require(msg.value == 0, "cannot replay with more funds");
            require(failedMessages[messageHash], "NewL2CrossDomainMessenger: message cannot be replayed");
        }

        // **CrossDomainMessenger Checks**. min gas, unsafe target, etc.

        // (2) Relay Message
        uint64 RELAY_RESERVED_GAS = 40_000;
        xDomainMsgSender = _sender;
        bool success = SafeCall.call({
            _target: _target,
            _gas: gasleft() - RELAY_RESERVED_GAS,
            _value: _value,
            _calldata: _message
        });
        xDomainMsgSender = Constants.DEFAULT_L2_SENDER;
        if (success) {
            successfulMessages[messageHash] = true;
            // (3) Emit event
        } else {
            failedMessages[messageHash] = true;
            // (3) Emit event
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
