// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { IAtomicCounter } from "interfaces/integration/IAtomicCounter.sol";

/// @notice An application uses ordinary ABI calls without handling result witnesses itself.
contract AtomicCallExample {
    error AtomicCallExample_LimitExceeded();

    /// @custom:semver 0.1.0
    string public constant version = "0.1.0";
    uint256 public result;

    function runAcross(address _first, address _second, uint256 _amount) external returns (uint256 result_) {
        uint256 first = IAtomicCounter(_first).add(_amount);
        result = first;
        result_ = IAtomicCounter(_second).add(first);
        result = result_;
    }

    /// @notice The second remote call depends on the actual result of the first call.
    function run(address _proxy, uint256 _amount, uint256 _limit) external returns (uint256 result_) {
        uint256 first = IAtomicCounter(_proxy).add(_amount);
        result = first;
        result_ = IAtomicCounter(_proxy).add(first);
        if (result_ > _limit) revert AtomicCallExample_LimitExceeded();
        result = result_;
    }

    /// @notice Adversarial fixture: catching a remote revert must not permit a partial commit.
    function catchRemoteFailure(address _proxy) external {
        result = 123;
        // eip150-safe: this adversarial fixture deliberately catches any failure;
        // the enclosing router must still roll back the entire transaction.
        try IAtomicCounter(_proxy).add(6) returns (uint256 value_) {
            result = value_;
        } catch {
            result = 456;
        }
    }
}
