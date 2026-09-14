// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { AtomicCallRouter } from "src/periphery/AtomicCallRouter.sol";

// Shared acceptance-test fixtures. Kept outside test/ so the normal contracts artifact
// build includes them without compiling the Forge unit-test suite.

/// @notice Exercises callbacks within the original transactions on both chains.
contract AtomicCallRouter_Nested_Harness {
    error NestedFailure();
    error UnexpectedState();

    uint256 public value;
    address internal originalOrigin;

    function run(address _peer, address _callback, uint256 _failure) external returns (uint256 result_) {
        value = 7;
        originalOrigin = tx.origin;
        assembly {
            tstore(0, 11)
        }
        (uint256 chain, address sender) = AtomicCallRouter(msg.sender).crossChainContext();
        try AtomicCallRouter_Nested_Harness(_peer).visit(_callback, _peer, _failure) returns (uint256 remoteResult_) {
            if (_failure == 5 || _failure == 6) {
                if (value != 17 || _transient() != 37 || remoteResult_ != 34) revert UnexpectedState();
            } else if (value != 12 || _transient() != 24 || remoteResult_ != 22) {
                revert UnexpectedState();
            }
            value += remoteResult_;
            result_ = value;
        } catch (bytes memory reason) {
            if (_failure != 4) {
                assembly {
                    revert(add(reason, 32), mload(reason))
                }
            }
            value = 999;
        }
        (uint256 restoredChain, address restoredSender) = AtomicCallRouter(msg.sender).crossChainContext();
        if (chain != restoredChain || sender != restoredSender) revert UnexpectedState();
        if (_failure == 3) revert NestedFailure();
    }

    function visit(address _callback, address _peer, uint256 _failure) external returns (uint256) {
        value = 3;
        assembly {
            tstore(0, 5)
        }
        (uint256 chain, address sender) = AtomicCallRouter(msg.sender).crossChainContext();
        uint256 result = AtomicCallRouter_Nested_Harness(_callback).reenter(_peer, _failure);
        if (_failure == 5 || _failure == 6) {
            result = AtomicCallRouter_Nested_Harness(_callback).reenter(_peer, _failure);
            if (value != 17 || _transient() != 21) revert UnexpectedState();
        } else if (value != 10 || _transient() != 13) {
            revert UnexpectedState();
        }
        (uint256 restoredChain, address restoredSender) = AtomicCallRouter(msg.sender).crossChainContext();
        if (chain != restoredChain || sender != restoredSender) revert UnexpectedState();
        return value + result;
    }

    function reenter(address _peer, uint256 _failure) external returns (uint256) {
        _recordCallback();
        AtomicCallRouter_Nested_Harness(_peer).touch(_failure);
        if (_failure == 2) revert NestedFailure();
        return value;
    }

    function recordCallback() external returns (uint256) {
        _recordCallback();
        return value;
    }

    function _recordCallback() internal {
        if (originalOrigin != tx.origin) revert UnexpectedState();
        if (value == 7) {
            if (_transient() != 11) revert UnexpectedState();
        } else if (value != 12 || _transient() != 24) {
            revert UnexpectedState();
        }
        value += 5;
        assembly {
            tstore(0, add(tload(0), 13))
        }
    }

    function touch(uint256 _failure) external {
        if (value == 3) {
            if (_transient() != 5) revert UnexpectedState();
        } else if (value != 10 || _transient() != 13) {
            revert UnexpectedState();
        }
        value += 7;
        assembly {
            tstore(0, add(tload(0), 8))
        }
        if (_failure == 1 || _failure == 4 || (_failure == 6 && value == 17)) revert NestedFailure();
    }

    function _transient() internal view returns (uint256 result_) {
        assembly {
            result_ := tload(0)
        }
    }
}

/// @notice A distinct contract C on A's chain; its local calls observe A's pending writes.
contract AtomicCallRouter_NestedCallback_Harness {
    AtomicCallRouter_Nested_Harness internal root;
    uint256 public callbacks;

    function configure(address _root) external {
        root = AtomicCallRouter_Nested_Harness(_root);
    }

    function reenter(address _peer, uint256 _failure) external returns (uint256 result_) {
        uint256 beforeValue = root.value();
        if (beforeValue != 7 && beforeValue != 12) revert AtomicCallRouter_Nested_Harness.UnexpectedState();
        callbacks++;
        result_ = root.recordCallback();
        AtomicCallRouter_Nested_Harness(_peer).touch(_failure);
        if (_failure == 2) revert AtomicCallRouter_Nested_Harness.NestedFailure();
    }
}
