// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ICompliance } from "interfaces/universal/ICompliance.sol";
import { IRule } from "interfaces/universal/IRule.sol";

/// @title MockRuleApprove
/// @notice Mock rule that always returns Approved.
contract MockRuleApprove is IRule {
    function check(
        address,
        address,
        uint256,
        uint64,
        bool,
        bytes calldata,
        uint256
    )
        external
        pure
        returns (ICompliance.Status)
    {
        return ICompliance.Status.Approved;
    }
}

/// @title MockRulePending
/// @notice Mock rule that always returns Pending.
contract MockRulePending is IRule {
    function check(
        address,
        address,
        uint256,
        uint64,
        bool,
        bytes calldata,
        uint256
    )
        external
        pure
        returns (ICompliance.Status)
    {
        return ICompliance.Status.Pending;
    }
}

/// @title MockRuleReject
/// @notice Mock rule that always returns Rejected.
contract MockRuleReject is IRule {
    function check(
        address,
        address,
        uint256,
        uint64,
        bool,
        bytes calldata,
        uint256
    )
        external
        pure
        returns (ICompliance.Status)
    {
        return ICompliance.Status.Rejected;
    }
}

/// @title MockRuleRefunded
/// @notice Mock rule that always gives a Refunded result (invalid for rules).
contract MockRuleRefunded is IRule {
    function check(
        address,
        address,
        uint256,
        uint64,
        bool,
        bytes calldata,
        uint256
    )
        external
        pure
        returns (ICompliance.Status)
    {
        return ICompliance.Status.Refunded;
    }
}

/// @title MockRuleConfigurable
/// @notice Mock rule with a configurable status return value.
contract MockRuleConfigurable is IRule {
    ICompliance.Status private _result;

    constructor(ICompliance.Status initialStatus) {
        _result = initialStatus;
    }

    function setStatus(ICompliance.Status s) external {
        _result = s;
    }

    function check(
        address,
        address,
        uint256,
        uint64,
        bool,
        bytes calldata,
        uint256
    )
        external
        view
        returns (ICompliance.Status)
    {
        return _result;
    }
}

/// @title ReentrantSettler
/// @notice Contract that attempts to re-enter settle() when receiving ETH refunds.
contract ReentrantSettler {
    ICompliance private _compliance;
    address private _to;
    uint256 private _value;
    uint256 private _mint;
    uint64 private _gasLimit;
    bool private _isCreation;
    bytes private _data;
    uint256 private _nonce;

    function setup(
        address compliance_,
        address to_,
        uint256 value_,
        uint256 mint_,
        uint64 gasLimit_,
        bool isCreation_,
        bytes calldata data_,
        uint256 nonce_
    )
        external
    {
        _compliance = ICompliance(compliance_);
        _to = to_;
        _value = value_;
        _mint = mint_;
        _gasLimit = gasLimit_;
        _isCreation = isCreation_;
        _data = data_;
        _nonce = nonce_;
    }

    receive() external payable {
        // Attempt reentrancy
        _compliance.settle(address(this), _to, _value, _mint, _gasLimit, _isCreation, _data, _nonce);
    }
}
