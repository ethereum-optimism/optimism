// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL2ToL2CrossDomainMessenger
/// @notice Interface for the L2ToL2CrossDomainMessenger contract.
interface IL2ToL2CrossDomainMessenger {
    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only
    ///         be present in this mapping if it has successfully been relayed on this chain, and
    ///         can therefore not be relayed again.
    /// @param _msgHash message hash to check.
    /// @return Returns true if the message corresponding to the `_msgHash` was successfully relayed.
    function successfulMessages(bytes32 _msgHash) external view returns (bool);

    /// @notice Retrieves the next message nonce. Message version will be added to the upper two
    ///         bytes of the message nonce. Message version allows us to treat messages as having
    ///         different structures.
    /// @return Nonce of the next message to be sent, with added message version.
    function messageNonce() external view returns (uint256);

    /// @notice Retrieves the sender of the current cross domain message.
    /// @return _sender Address of the sender of the current cross domain message.
    function crossDomainMessageSender() external view returns (address _sender);

    /// @notice Retrieves the source of the current cross domain message.
    /// @return _source Chain ID of the source of the current cross domain message.
    function crossDomainMessageSource() external view returns (uint256 _source);

    /// @notice Sends a message to some target address on a destination chain. Note that if the call
    ///         always reverts, then the message will be unrelayable, and any ETH sent will be
    ///         permanently locked. The same will occur if the target on the other chain is
    ///         considered unsafe (see the _isUnsafeTarget() function).
    /// @param _destination Chain ID of the destination chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message to trigger the target address with.
    function sendMessage(uint256 _destination, address _target, bytes calldata _message) external;

    /// @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only
    ///         be executed via cross-chain call from the other messenger OR if the message was
    ///         already received once and is currently being replayed.
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the message being relayed.
    /// @param _sender      Address of the user who sent the message.
    /// @param _source      Chain ID of the source chain.
    /// @param _target      Address that the message is targeted at.
    /// @param _message     Message to send to the target.
    function relayMessage(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message
    )
        external
        payable;
}
