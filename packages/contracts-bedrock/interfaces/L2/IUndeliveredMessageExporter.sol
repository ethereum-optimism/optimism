// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IUndeliveredMessageExporter
/// @notice Interface for the UndeliveredMessageExporter contract.
interface IUndeliveredMessageExporter {
    /// @notice Thrown when exporting a message that was relayed on this chain.
    error UndeliveredMessageExporter_MessageRelayed();

    /// @notice Emitted when a message to this chain is exported as not relayed.
    /// @param messageHash     Hash of the message.
    /// @param source          Chain ID of the message's source chain.
    /// @param sourceMessenger The source chain's L1CrossDomainMessenger the word is sent to.
    /// @param undeliveredAt   Timestamp at which the message had not been relayed.
    event UndeliveredMessageExported(
        bytes32 indexed messageHash, uint256 indexed source, address sourceMessenger, uint256 undeliveredAt
    );

    function version() external view returns (string memory);

    /// @notice Tells the source chain, through its L1CrossDomainMessenger, that a message to this
    ///         chain has not been relayed by now.
    /// @param _sourceMessenger The source chain's L1CrossDomainMessenger.
    /// @param _source          Chain ID of the source chain.
    /// @param _nonce           Nonce of the message.
    /// @param _sender          Address that sent the message.
    /// @param _target          Target contract or wallet address.
    /// @param _message         Message payload.
    /// @param _minGasLimit     Minimum gas limit for the call on L1.
    /// @return messageHash_ Hash of the message.
    function exportUndeliveredMessage(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
        returns (bytes32 messageHash_);

    function __constructor__() external;
}
