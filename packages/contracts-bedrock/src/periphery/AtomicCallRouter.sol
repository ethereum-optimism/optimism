// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { AtomicRemoteProxy } from "src/periphery/AtomicRemoteProxy.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import {
    AtomicResultWitness,
    AtomicRemoteCall,
    AtomicWitnessRequest,
    AtomicStreamCursor,
    AtomicCallback
} from "src/libraries/AtomicCallTypes.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title AtomicCallRouter
/// @notice Experimental atomic-call router using the unchanged CrossL2Inbox and access-list rules.
///         Deploy the same router at the same address on every participating chain. Remote peers
///         must have this implementation; the deploying application chooses that trust domain.
///         Result witnesses are authenticated during cross-chain verification, not by this EVM.
///         Native ETH forwarding, STATICCALL, and distributed
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

    error AtomicCallRouter_InvalidReader();
    error AtomicCallRouter_InsufficientGas();
    error AtomicCallRouter_CallLimit();
    error AtomicCallRouter_CallbackFailed();

    /// @notice Prototype version; this contract is not a protocol predeploy.
    /// @custom:semver 0.2.0
    string public constant version = "0.2.0";

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
    AtomicRemoteCall[] internal remoteCalls;
    Identifier internal completion;
    mapping(uint256 => AtomicRemoteCall[]) internal callbacks;
    bool internal callbackFailure;
    bool internal nested;
    uint256 internal callbacksUsed;
    uint256 internal callbacksTotal;

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
        return _executeRoot(_nonce, _target, _data, _witnesses, 0);
    }

    /// @notice Executes an intent with a fixed application gas budget, independent of tape preload cost.
    function executeRootWithGas(
        uint256 _nonce,
        address _target,
        bytes calldata _data,
        AtomicResultWitness[] calldata _witnesses,
        uint64 _applicationGas
    )
        external
        returns (bytes memory result_)
    {
        if (_applicationGas == 0) revert AtomicCallRouter_InsufficientGas();
        return _executeRoot(_nonce, _target, _data, _witnesses, _applicationGas);
    }

    /// @notice Executes a root with callbacks inside its pending outbound calls.
    function executeRootNested(
        uint256 _nonce,
        address _target,
        bytes calldata _data,
        AtomicResultWitness[] calldata _witnesses,
        AtomicCallback[] calldata _callbacks,
        uint64 _applicationGas
    )
        external
        returns (bytes memory result_)
    {
        if (_applicationGas == 0) revert AtomicCallRouter_InsufficientGas();
        _loadCallbacks(_callbacks);
        return _executeRoot(_nonce, _target, _data, _witnesses, _applicationGas);
    }

    /// @notice Executes a destination batch with callbacks inside its pending outbound calls.
    function executeRemoteNested(
        bytes32 _bundleId,
        AtomicRemoteCall[] calldata _calls,
        AtomicResultWitness[] calldata _witnesses,
        AtomicCallback[] calldata _callbacks,
        Identifier calldata _rootCompletion,
        uint64 _applicationGas,
        uint16 _maxCalls
    )
        external
        returns (bytes[] memory results_)
    {
        if (_applicationGas == 0) revert AtomicCallRouter_InsufficientGas();
        _loadCallbacks(_callbacks);
        return _executeRemote(_bundleId, _calls, _witnesses, _rootCompletion, _applicationGas, _maxCalls);
    }

    function _loadCallbacks(AtomicCallback[] calldata _callbacks) internal {
        if (entered) revert AtomicCallRouter_AlreadyEntered();
        nested = true;
        callbacksTotal = _callbacks.length;
        callbacksUsed = 0;
        callbackFailure = false;
        for (uint256 i; i < _callbacks.length; i++) {
            AtomicCallback calldata item = _callbacks[i];
            callbacks[item.waitingSequence].push(item.call);
            if (!item.success) callbackFailure = true;
        }
    }

    function _executeRoot(
        uint256 _nonce,
        address _target,
        bytes calldata _data,
        AtomicResultWitness[] calldata _witnesses,
        uint64 _applicationGas
    )
        internal
        returns (bytes memory result_)
    {
        if (_nonce != nonces[msg.sender]) revert AtomicCallRouter_InvalidNonce();
        bytes32 id = computeBundleId(msg.sender, _nonce);
        _begin(id, _witnesses);
        nonces[msg.sender] = _nonce + 1;
        sourceChain = block.chainid;
        sourceSender = msg.sender;
        bool success;
        (success, result_) = _tryCall(_target, _data, _applicationGas);
        if (!success) _revert(result_);
        _finish();
        emit BundleCompleted(id);
    }

    /// @notice Executes one chain's ordered remote operations in one locally atomic transaction.
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
        return _executeRemote(_bundleId, _calls, _witnesses, _rootCompletion, 0, _calls.length);
    }

    /// @notice Streams a bounded remote batch with the same gas budget for each application call.
    ///         Canonical execution reads the complete calldata tape; a builder may pause at its getters.
    function executeRemoteWithGas(
        bytes32 _bundleId,
        AtomicRemoteCall[] calldata _calls,
        AtomicResultWitness[] calldata _witnesses,
        Identifier calldata _rootCompletion,
        uint64 _applicationGas,
        uint16 _maxCalls
    )
        external
        returns (bytes[] memory results_)
    {
        if (_applicationGas == 0) revert AtomicCallRouter_InsufficientGas();
        return _executeRemote(_bundleId, _calls, _witnesses, _rootCompletion, _applicationGas, _maxCalls);
    }

    function _executeRemote(
        bytes32 _bundleId,
        AtomicRemoteCall[] calldata _calls,
        AtomicResultWitness[] calldata _witnesses,
        Identifier calldata _rootCompletion,
        uint64 _applicationGas,
        uint256 _maxCalls
    )
        internal
        returns (bytes[] memory results_)
    {
        if (_calls.length > _maxCalls) revert AtomicCallRouter_CallLimit();
        _begin(_bundleId, _witnesses);
        for (uint256 i; i < _calls.length; i++) {
            remoteCalls.push(_calls[i]);
        }
        completion = _rootCompletion;
        results_ = new bytes[](_maxCalls);
        uint256 count;
        bytes memory previous;
        while (true) {
            (bool found, AtomicRemoteCall memory item) = this.remoteCallAt(AtomicStreamCursor(count, previous));
            if (!found) break;
            if (count == _maxCalls) revert AtomicCallRouter_CallLimit();
            bytes memory result = _executeIncoming(item, _applicationGas);
            results_[count++] = result;
            previous = result;
        }
        if (count == 0) revert AtomicCallRouter_EmptyBatch();
        assembly {
            mstore(results_, count)
        }
        Identifier memory root = this.completionIdentifier();
        _checkPeer(root, root.chainId);
        _validate(root, keccak256(abi.encodePacked(BundleCompleted.selector, _bundleId)));
        _finish();
    }

    function _executeIncoming(AtomicRemoteCall memory _item, uint64 _gas) internal returns (bytes memory result_) {
        _checkPeer(_item.identifier, _item.identifier.chainId);
        bytes32 callId = keccak256(abi.encode(bundle, _item.identifier.chainId, _item.sequence));
        if (consumedCalls[callId]) revert AtomicCallRouter_ReplayedCall();
        bytes32 requestHash = keccak256(abi.encode(block.chainid, _item.target, _item.sender, keccak256(_item.data)));
        _validate(_item.identifier, keccak256(abi.encodePacked(CallRequested.selector, callId, requestHash)));
        consumedCalls[callId] = true;
        uint256 previousChain = sourceChain;
        address previousSender = sourceSender;
        sourceChain = _item.identifier.chainId;
        sourceSender = _item.sender;
        bool success;
        (success, result_) = _tryCall(_item.target, _item.data, _gas);
        sourceChain = previousChain;
        sourceSender = previousSender;
        if (success) emit CallResult(callId, keccak256(result_));
        // Discovery can observe a nested result before either local parent unwinds.
        // Canonical execution traverses this exact read-only acknowledgement too.
        if (nested) this.callFinished(_item.identifier.chainId, _item.sequence, success, result_);
        if (!success) revert AtomicCallRouter_RemoteReverted(_item.sequence, result_);
    }

    /// @notice Reads callback work associated with the currently waiting outbound call.
    function callbackAt(
        AtomicWitnessRequest calldata _request,
        uint256 _index
    )
        external
        view
        returns (bool found_, AtomicRemoteCall memory call_)
    {
        _checkReader();
        if (_index < callbacks[_request.sequence].length) return (true, callbacks[_request.sequence][_index]);
    }

    /// @notice Reports an incoming call's result to a suspending adapter without changing state.
    function callFinished(uint256, uint256, bool, bytes calldata) external view {
        _checkReader();
    }

    /// @notice Reads callback completion requirements outside application rollback scopes.
    function callbackStatus() external view returns (bool failed_, uint256 total_) {
        _checkReader();
        return (callbackFailure, callbacksTotal);
    }

    /// @notice Reads a preloaded result. Only router self-calls may inspect the witness tape.
    ///         Request fields describe the waiting application call to a suspending builder.
    function witnessAt(AtomicWitnessRequest calldata _request)
        external
        view
        returns (bool found_, AtomicResultWitness memory witness_)
    {
        _checkReader();
        if (_request.sequence < witnesses.length) return (true, witnesses[_request.sequence]);
    }

    /// @notice Reads a preloaded remote operation; the cursor also reports the preceding result.
    function remoteCallAt(AtomicStreamCursor calldata _cursor)
        external
        view
        returns (bool found_, AtomicRemoteCall memory call_)
    {
        _checkReader();
        if (_cursor.index < remoteCalls.length) return (true, remoteCalls[_cursor.index]);
    }

    /// @notice Reads the tape length during finalization.
    function witnessCount() external view returns (uint256 count_) {
        _checkReader();
        return witnesses.length;
    }

    /// @notice Reads the root-completion dependency after all remote operations finish.
    function completionIdentifier() external view returns (Identifier memory identifier_) {
        _checkReader();
        return completion;
    }

    function _checkReader() internal view {
        if (msg.sender != address(this)) revert AtomicCallRouter_InvalidReader();
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
        uint256 current = sequence++;
        bytes32 callId = keccak256(abi.encode(bundle, block.chainid, current));
        bytes32 requestHash = keccak256(abi.encode(_chainId, _target, _sender, keccak256(_data)));
        emit CallRequested(callId, requestHash);
        AtomicWitnessRequest memory request = AtomicWitnessRequest(current, _chainId, _target, _sender, _data);
        if (nested) {
            uint256 index;
            while (true) {
                (bool hasCallback, AtomicRemoteCall memory item) = this.callbackAt(request, index);
                if (!hasCallback) break;
                index++;
                callbacksUsed++;
                // A callback spends gas in this existing call tree. Reserve half for unwinding
                // and result verification; never manufacture a fresh application-sized budget.
                uint64 budget = uint64(gasleft() / 2);
                if (budget == 0) revert AtomicCallRouter_InsufficientGas();
                _executeIncoming(item, budget);
            }
        }
        (bool found, AtomicResultWitness memory witness) = this.witnessAt(request);
        if (!found) revert AtomicCallRouter_MissingWitness(_chainId, _target, _sender, _data, current);
        if (!witness.success) _revert(witness.returnData);
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
        (bool failedCallbacks, uint256 totalCallbacks) = this.callbackStatus();
        if (failedCallbacks) revert AtomicCallRouter_CallbackFailed();
        if (callbacksUsed != totalCallbacks) revert AtomicCallRouter_UnusedWitnesses();
        uint256 count = this.witnessCount();
        for (uint256 i; i < count; i++) {
            (bool found, AtomicResultWitness memory witness) =
                this.witnessAt(AtomicWitnessRequest(i, 0, address(0), address(0), ""));
            if (!found) revert AtomicCallRouter_UnusedWitnesses();
            if (!witness.success) _revert(witness.returnData);
            delete callbacks[i];
        }
        if (sequence != count) revert AtomicCallRouter_UnusedWitnesses();
        delete witnesses;
        delete remoteCalls;
        delete completion;
        delete bundle;
        delete sequence;
        delete sourceChain;
        delete sourceSender;
        delete callbackFailure;
        delete callbacksUsed;
        delete callbacksTotal;
        nested = false;
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

    function _tryCall(
        address _target,
        bytes memory _data,
        uint64 _gas
    )
        internal
        returns (bool success_, bytes memory result_)
    {
        if (_gas == 0) return _target.call(_data);
        // Check after copying input to memory. Never silently truncate the application budget.
        if (!SafeCall.hasMinGas(_gas, 0)) revert AtomicCallRouter_InsufficientGas();
        // Call the existing buffer directly: another Solidity ABI copy here can
        // consume unbounded memory-expansion gas after the EIP-150 check.
        assembly {
            success_ := call(_gas, _target, 0, add(_data, 32), mload(_data), 0, 0)
            let size := returndatasize()
            result_ := mload(0x40)
            mstore(result_, size)
            returndatacopy(add(result_, 32), 0, size)
            mstore(0x40, add(add(result_, 32), and(add(size, 31), not(31))))
        }
    }

    function _revert(bytes memory _reason) internal pure {
        assembly {
            revert(add(_reason, 32), mload(_reason))
        }
    }
}
