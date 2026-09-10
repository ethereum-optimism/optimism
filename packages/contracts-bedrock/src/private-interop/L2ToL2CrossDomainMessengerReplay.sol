// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000023
/// @title L2ToL2CrossDomainMessengerReplay
/// @notice Alternative implementation of the `L2ToL2CrossDomainMessenger` predeploy, installed at
///         the standard messenger predeploy address in the genesis of a private chain's *public
///         rendering*. The public rendering is a derived-only OP Stack chain whose blocks contain
///         batcher-signed "replay" transactions: one transaction per message the private chain exported,
///         re-emitting a `SentMessage` event that is byte-identical to the one the private chain's
///         stock messenger emitted. Because this implementation lives at the standard predeploy
///         address, the emitter address, the event topics and the event data all match what every
///         stock interop consumer (message database, cross-safety judge, counterparty relayer)
///         already expects, so no consumer needs to know the chain is a rendering.
///
///         This contract deliberately implements *only* the replay-emit path. It cannot originate
///         a message, it cannot relay one, and it holds no message accounting: every stock
///         messenger entry point reverts with `L2ToL2CrossDomainMessengerReplay_Unsupported`
///         rather than silently succeeding or silently returning a zero value.
///
///         Replay faithfulness is not checked on-chain. In v1 the operator is trusted to render
///         only messages the private chain actually exported; the counterparty-checked import path
///         is what a lying operator still cannot fake. The one message class this contract refuses
///         outright is the protocol ETH path (see `replaySentMessage`).
contract L2ToL2CrossDomainMessengerReplay is ISemver {
    /// @notice Thrown when attempting to replay a message whose embedded sender is the
    ///         `SuperchainETHBridge` predeploy.
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeSender();

    /// @notice Thrown when attempting to replay a message whose embedded target is the
    ///         `SuperchainETHBridge` predeploy.
    error L2ToL2CrossDomainMessengerReplay_ETHBridgeTarget();

    /// @notice Thrown by every stock `L2ToL2CrossDomainMessenger` entry point that this
    ///         implementation does not support.
    error L2ToL2CrossDomainMessengerReplay_Unsupported();

    /// @notice Emitted whenever a message is sent to a destination. Declared with exactly the
    ///         signature and parameter layout of the stock `L2ToL2CrossDomainMessenger` event so
    ///         that a replayed log is byte-identical to a natively emitted one: `destination`,
    ///         `target` and `messageNonce` are indexed (topics 1-3) and `sender` and `message` are
    ///         ABI-encoded into the data section.
    ///
    /// @param destination  Chain ID of the destination chain.
    /// @param target       Target contract or wallet address.
    /// @param messageNonce Nonce associated with the message sent.
    /// @param sender       Address initiating this message call.
    /// @param message      Message payload to call target with.
    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    /// @notice Current message version identifier. Matches the stock messenger so that consumers
    ///         reading the version off the predeploy see the same value.
    uint16 public constant messageVersion = uint16(0);

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Re-emits a `SentMessage` event on behalf of the private chain. The emitted log is
    ///         byte-identical to the log the private chain's stock messenger produced for the same
    ///         message: same emitter (this predeploy address), same four topics, same data.
    ///         Parameter order matches the stock messenger's `resendMessage` so operator tooling
    ///         can encode either call the same way.
    ///
    ///         The public rendering shares the private chain's chain ID, so the message's identity
    ///         on the public chain — origin, log position, timestamp — is the message's identity,
    ///         and no further bookkeeping is required here.
    ///
    ///         Replaying a message whose sender OR target is the `SuperchainETHBridge` predeploy is
    ///         refused. The private chain is a custom gas token chain: its native unit is not ETH,
    ///         and its `SuperchainETHBridge.sendETH` would burn that custom unit while asking a
    ///         counterparty to mint real ETH. The protocol ETH bridge and liquidity implementation
    ///         are absent from the private genesis, and ETH-denominated value enters only through
    ///         the application-level lock-mint bridge, but this contract must not be the weak link
    ///         that renders such a message into a public, relayable log.
    ///         The two checks are deliberately symmetric: the receiving bridge's own
    ///         `InvalidCrossDomainSender` check already makes the target-side refusal redundant in
    ///         theory, but the point of a deny list is to hold without anyone having to reason
    ///         about what every receiver checks.
    ///
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the message as sent on the private chain.
    /// @param _sender      Address that sent the message on the private chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message payload to call target with.
    ///
    /// @return messageHash_ Hash of the message that was replayed.
    function replaySentMessage(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 messageHash_)
    {
        if (_sender == Predeploys.SUPERCHAIN_ETH_BRIDGE) revert L2ToL2CrossDomainMessengerReplay_ETHBridgeSender();
        if (_target == Predeploys.SUPERCHAIN_ETH_BRIDGE) revert L2ToL2CrossDomainMessengerReplay_ETHBridgeTarget();

        messageHash_ = Hashing.hashL2toL2CrossDomainMessage({
            _destination: _destination,
            _source: block.chainid,
            _nonce: _nonce,
            _sender: _sender,
            _target: _target,
            _message: _message
        });

        emit SentMessage(_destination, _target, _nonce, _sender, _message);
    }

    /// @notice Unsupported. The public rendering never originates a message; every message it
    ///         carries was exported by the private chain and arrives through `replaySentMessage`.
    function sendMessage(uint256, address, bytes calldata) external pure returns (bytes32) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. This implementation keeps no record of sent messages, so it cannot
    ///         authenticate a resend. Use `replaySentMessage` instead.
    function resendMessage(uint256, uint256, address, address, bytes calldata) external pure returns (bytes32) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. Imports on the public rendering are executed directly against the
    ///         stock `CrossL2Inbox` by the batcher's import replay transactions.
    function relayMessage(Identifier calldata, bytes calldata) external payable returns (bytes memory) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. No message is ever relayed through this implementation, so there is no
    ///         cross domain message context to report.
    function crossDomainMessageSender() external pure returns (address) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. No message is ever relayed through this implementation, so there is no
    ///         cross domain message context to report.
    function crossDomainMessageSource() external pure returns (uint256) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. No message is ever relayed through this implementation, so there is no
    ///         cross domain message context to report.
    function crossDomainMessageContext() external pure returns (address, uint256) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. Nonces are assigned by the private chain and supplied to
    ///         `replaySentMessage`; this implementation does not track its own.
    function messageNonce() external pure returns (uint256) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. Reverts rather than returning `false`, so a caller can never mistake
    ///         "this implementation has no record" for "this message was never relayed".
    function successfulMessages(bytes32) external pure returns (bool) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }

    /// @notice Unsupported. Reverts rather than returning zero, so a caller can never mistake
    ///         "this implementation has no record" for "no message was sent with this nonce".
    function sentMessages(uint256) external pure returns (bytes32) {
        revert L2ToL2CrossDomainMessengerReplay_Unsupported();
    }
}
