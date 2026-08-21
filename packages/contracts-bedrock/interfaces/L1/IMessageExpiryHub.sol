// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @title IMessageExpiryHub
/// @notice Interface for the MessageExpiryHub contract.
interface IMessageExpiryHub {
    error MessageExpiryHub_InvalidMessenger();
    error MessageExpiryHub_UnauthorizedPortal();
    error MessageExpiryHub_InvalidChainId();
    error MessageExpiryHub_InvalidCrossDomainSender();
    error MessageExpiryHub_InvalidSourceChain();
    error MessageExpiryHub_InvalidTimestamp();
    error MessageExpiryHub_StaleNotice();
    error MessageExpiryHub_NoticeNotFound();
    error MessageExpiryHub_ChainNotRegistered();
    error MessageExpiryHub_ClusterMismatch();
    error MessageExpiryHub_AlreadyRegistered();

    event ChainRegistered(address indexed ethLockbox, uint256 indexed chainId, address systemConfig);
    event ExpiryNoticeReceived(
        address indexed ethLockbox,
        uint256 indexed attestorChainId,
        bytes32 indexed msgHash,
        uint256 sourceChainId,
        uint256 attestedAt
    );
    event ExpiryNoticeForwarded(
        address indexed ethLockbox, uint256 indexed attestorChainId, bytes32 indexed msgHash, uint256 sourceChainId
    );

    function version() external view returns (string memory);
    function notices(
        address,
        uint256,
        uint256,
        bytes32
    )
        external
        view
        returns (address anchorStateRegistry, uint64 attestedAt);
    function registeredChains(address, uint256) external view returns (ISystemConfig);
    function registerChain(ISystemConfig _systemConfig) external;
    function receiveExpiryNotice(bytes32 _msgHash, uint256 _sourceChainId, uint256 _attestedAt) external;
    function forwardExpiryNotice(
        address _ethLockbox,
        uint256 _attestorChainId,
        uint256 _sourceChainId,
        bytes32 _msgHash,
        uint32 _minGasLimit
    )
        external;

    function __constructor__() external;
}
