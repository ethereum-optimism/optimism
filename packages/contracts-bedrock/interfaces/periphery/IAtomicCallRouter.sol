// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

import { AtomicResultWitness, AtomicRemoteCall } from "src/libraries/AtomicCallTypes.sol";

interface IAtomicCallRouter {
    error AtomicCallRouter_AlreadyEntered();
    error AtomicCallRouter_InvalidNonce();
    error AtomicCallRouter_InvalidPeer();
    error AtomicCallRouter_InvalidProxy();
    error AtomicCallRouter_MissingWitness(uint256 chainId, address target, address sender, bytes data, uint256 sequence);
    error AtomicCallRouter_UnusedWitnesses();
    error AtomicCallRouter_ReplayedCall();
    error AtomicCallRouter_EmptyBatch();
    error AtomicCallRouter_RemoteReverted(uint256 sequence, bytes reason);

    event CallRequested(bytes32 indexed callId, bytes32 requestHash);
    event CallResult(bytes32 indexed callId, bytes32 resultHash);
    event BundleCompleted(bytes32 indexed bundleId);
    event ProxyCreated(uint256 indexed chainId, address indexed target, address proxy);

    function __constructor__() external;
    function version() external view returns (string memory);
    function nonces(address) external view returns (uint256);
    function consumedCalls(bytes32) external view returns (bool);
    function proxies(address) external view returns (bool);
    function proxyFor(uint256 _chainId, address _target) external returns (address proxy_);
    function computeBundleId(address _sender, uint256 _nonce) external view returns (bytes32);
    function crossChainContext() external view returns (uint256 sourceChainId_, address sender_);
    function executeRoot(
        uint256 _nonce,
        address _target,
        bytes calldata _data,
        AtomicResultWitness[] calldata _witnesses
    )
        external
        returns (bytes memory result_);
    function executeRemote(
        bytes32 _bundleId,
        AtomicRemoteCall[] calldata _calls,
        AtomicResultWitness[] calldata _witnesses,
        Identifier calldata _rootCompletion
    )
        external
        returns (bytes[] memory results_);
    function remoteCall(
        uint256 _chainId,
        address _target,
        address _sender,
        bytes calldata _data
    )
        external
        returns (bytes memory result_);
}
