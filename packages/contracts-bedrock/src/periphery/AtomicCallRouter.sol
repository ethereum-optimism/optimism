// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { AtomicRemoteProxy } from "src/periphery/AtomicRemoteProxy.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { AtomicResultWitness, AtomicRemoteCall } from "src/libraries/AtomicCallTypes.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title AtomicCallRouter
/// @notice Experimental atomic-call router using the unchanged CrossL2Inbox and access-list rules.
///         Deploy the same router at the same address on every participating chain. Remote peers
///         must have this implementation; the deploying application chooses that trust domain.
///         Result witnesses are authenticated during cross-chain verification, not by this EVM.
///         Native ETH forwarding, STATICCALL, nested callbacks into an active chain, and distributed
///         caught-revert semantics are not supported by this initial facade.
contract AtomicCallRouter {
    error AtomicCallRouter_AlreadyEntered();
    error AtomicCallRouter_InvalidNonce();
    error AtomicCallRouter_InvalidPeer();
    error AtomicCallRouter_InvalidProxy();
    error AtomicCallRouter_MissingWitness(uint256 chainId, address target, address sender, bytes data, uint256 sequence);
    error AtomicCallRouter_UnusedWitnesses();
    error AtomicCallRouter_ReplayedCall();
    error AtomicCallRouter_EmptyBatch();
    error AtomicCallRouter_RemoteReverted(uint256 sequence, bytes reason);

    /// @notice Prototype version; this contract is not a protocol predeploy.
    /// @custom:semver 0.1.0
    string public constant version = "0.1.0";

    event CallRequested(bytes32 indexed callId, bytes32 requestHash);
    event CallResult(bytes32 indexed callId, bytes32 resultHash);
    event BundleCompleted(bytes32 indexed bundleId);
    event ProxyCreated(uint256 indexed chainId, address indexed target, address proxy);

    mapping(address => uint256) public nonces;
    mapping(bytes32 => bool) public consumedCalls;
    mapping(address => bool) public proxies;

    bool internal entered;
    bytes32 internal bundle;
    uint256 internal sequence;
    uint256 internal sourceChain;
    address internal sourceSender;
    AtomicResultWitness[] internal witnesses;

    /// @notice Creates or returns a deterministic local proxy for a remote contract.
    function proxyFor(uint256 _chainId, address _target) external returns (address proxy_) {
        proxy_ = predictProxy(_chainId, _target);
        if (proxy_.code.length == 0) {
            bytes32 salt = keccak256(abi.encode(_chainId, _target));
            proxy_ = address(new AtomicRemoteProxy{ salt: salt }(address(this), _chainId, _target));
            proxies[proxy_] = true;
            emit ProxyCreated(_chainId, _target, proxy_);
        }
    }

    /// @notice Predicts the ordinary local address accepting the remote contract's ABI.
    ///         Deploy it with proxyFor before submitting an operation that calls it.
    function predictProxy(uint256 _chainId, address _target) public view returns (address proxy_) {
        if (_chainId == block.chainid || _target == address(0)) revert AtomicCallRouter_InvalidPeer();
        bytes32 salt = keccak256(abi.encode(_chainId, _target));
        bytes memory init =
            abi.encodePacked(type(AtomicRemoteProxy).creationCode, abi.encode(address(this), _chainId, _target));
        proxy_ =
            address(uint160(uint256(keccak256(abi.encodePacked(bytes1(0xff), address(this), salt, keccak256(init))))));
    }

    /// @notice Computes the root intent's identity without committing to discovered result witnesses.
    function computeBundleId(address _sender, uint256 _nonce) public view returns (bytes32) {
        return keccak256(abi.encode(block.chainid, address(this), _sender, _nonce));
    }

    /// @notice Returns the authenticated caller context of the currently executing local operation.
    function crossChainContext() external view returns (uint256 sourceChainId_, address sender_) {
        return (sourceChain, sourceSender);
    }

    /// @notice Executes the root operation and commits only if it consumes the complete witness tape.
    function executeRoot(
        uint256 _nonce,
        address _target,
        bytes calldata _data,
        AtomicResultWitness[] calldata _witnesses
    )
        external
        returns (bytes memory result_)
    {
        if (_nonce != nonces[msg.sender]) revert AtomicCallRouter_InvalidNonce();
        bytes32 id = computeBundleId(msg.sender, _nonce);
        _begin(id, _witnesses);
        nonces[msg.sender] = _nonce + 1;
        sourceChain = block.chainid;
        sourceSender = msg.sender;
        result_ = _call(_target, _data);
        _finish();
        emit BundleCompleted(id);
    }

    /// @notice Executes one chain's ordered remote operations in a single locally atomic transaction.
    ///         The final dependency on root completion prevents orphaned remote work from becoming safe.
    function executeRemote(
        bytes32 _bundleId,
        AtomicRemoteCall[] calldata _calls,
        AtomicResultWitness[] calldata _witnesses,
        Identifier calldata _rootCompletion
    )
        external
        returns (bytes[] memory results_)
    {
        if (_calls.length == 0) revert AtomicCallRouter_EmptyBatch();
        _begin(_bundleId, _witnesses);
        results_ = new bytes[](_calls.length);
        for (uint256 i; i < _calls.length; i++) {
            AtomicRemoteCall calldata item = _calls[i];
            _checkPeer(item.identifier, item.identifier.chainId);
            bytes32 callId = keccak256(abi.encode(_bundleId, item.identifier.chainId, item.sequence));
            if (consumedCalls[callId]) revert AtomicCallRouter_ReplayedCall();
            bytes32 requestHash = keccak256(abi.encode(block.chainid, item.target, item.sender, keccak256(item.data)));
            _validate(item.identifier, keccak256(abi.encodePacked(CallRequested.selector, callId, requestHash)));
            consumedCalls[callId] = true;
            sourceChain = item.identifier.chainId;
            sourceSender = item.sender;
            (bool success, bytes memory result) = item.target.call(item.data);
            if (!success) revert AtomicCallRouter_RemoteReverted(item.sequence, result);
            results_[i] = result;
            emit CallResult(callId, keccak256(result));
        }
        _checkPeer(_rootCompletion, _rootCompletion.chainId);
        _validate(_rootCompletion, keccak256(abi.encodePacked(BundleCompleted.selector, _bundleId)));
        _finish();
    }

    /// @notice Called by factory-created proxies to consume and authenticate the next remote result.
    function remoteCall(
        uint256 _chainId,
        address _target,
        address _sender,
        bytes calldata _data
    )
        external
        returns (bytes memory result_)
    {
        if (!entered || !proxies[msg.sender]) revert AtomicCallRouter_InvalidProxy();
        if (sequence >= witnesses.length) {
            revert AtomicCallRouter_MissingWitness(_chainId, _target, _sender, _data, sequence);
        }
        AtomicResultWitness storage witness = witnesses[sequence];
        if (!witness.success) _revert(witness.returnData);
        bytes32 callId = keccak256(abi.encode(bundle, block.chainid, sequence));
        sequence++;
        bytes32 requestHash = keccak256(abi.encode(_chainId, _target, _sender, keccak256(_data)));
        emit CallRequested(callId, requestHash);
        Identifier memory id = witness.identifier;
        _checkPeer(id, _chainId);
        result_ = witness.returnData;
        _validate(id, keccak256(abi.encodePacked(CallResult.selector, callId, keccak256(result_))));
    }

    function _begin(bytes32 _bundleId, AtomicResultWitness[] calldata _witnesses) internal {
        if (entered) revert AtomicCallRouter_AlreadyEntered();
        entered = true;
        bundle = _bundleId;
        for (uint256 i; i < _witnesses.length; i++) {
            witnesses.push(_witnesses[i]);
        }
    }

    function _finish() internal {
        // Scan in the root frame: a failure flag set inside remoteCall would be
        // rolled back when the application catches that call's revert.
        for (uint256 i; i < witnesses.length; i++) {
            if (!witnesses[i].success) _revert(witnesses[i].returnData);
        }
        if (sequence != witnesses.length) revert AtomicCallRouter_UnusedWitnesses();
        delete witnesses;
        delete bundle;
        delete sequence;
        delete sourceChain;
        delete sourceSender;
        entered = false;
    }

    function _checkPeer(Identifier memory _id, uint256 _chainId) internal view {
        if (
            _id.origin != address(this) || _id.chainId != _chainId || _chainId == block.chainid
                || _id.timestamp != block.timestamp
        ) revert AtomicCallRouter_InvalidPeer();
    }

    function _validate(Identifier memory _id, bytes32 _payloadHash) internal {
        ICrossL2Inbox(Predeploys.CROSS_L2_INBOX).validateMessage(_id, _payloadHash);
    }

    function _call(address _target, bytes memory _data) internal returns (bytes memory result_) {
        bool success;
        (success, result_) = _target.call(_data);
        if (!success) _revert(result_);
    }

    function _revert(bytes memory _reason) internal pure {
        assembly {
            revert(add(_reason, 32), mload(_reason))
        }
    }
}
