// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @title IMessageExpiryHub
/// @notice Interface for the MessageExpiryHub contract.
interface IMessageExpiryHub is ISemver {
    error MessageExpiryHub_InvalidChain();
    error MessageExpiryHub_InvalidSender();
    error MessageExpiryHub_UnknownFact();

    event UndeliveredMessageRecorded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );
    event UndeliveredMessageForwarded(
        bytes32 indexed cluster, bytes32 indexed messageHash, uint256 indexed source, uint256 undeliveredAt
    );

    function facts(bytes32) external view returns (bool);
    function receiveUndeliveredMessage(bytes32 _messageHash, uint256 _source, uint256 _undeliveredAt) external;
    function forwardUndeliveredMessage(
        ISystemConfig _source,
        bytes32 _messageHash,
        uint256 _undeliveredAt,
        uint32 _minGasLimit
    )
        external;
    function factId(
        bytes32 _cluster,
        bytes32 _messageHash,
        uint256 _source,
        uint256 _undeliveredAt
    )
        external
        pure
        returns (bytes32);
    function cluster(ISystemConfig _systemConfig) external view returns (bytes32 cluster_);

    function __constructor__() external;
}
