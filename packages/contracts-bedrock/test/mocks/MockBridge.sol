// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ICompliance } from "interfaces/universal/ICompliance.sol";
import { IDonatable } from "interfaces/universal/IDonatable.sol";

/// @title MockBridge
/// @notice A mock bridge that implements IDonatable and can call ICompliance.check().
///         Used in unit tests for the abstract Compliance contract via L1Compliance.
contract MockBridge is IDonatable {
    /// @notice Records the last call to approved().
    bool public approvedCalled;
    address public lastFrom;
    address public lastTo;
    uint256 public lastValue;
    uint256 public lastMint;
    uint64 public lastGasLimit;
    bool public lastIsCreation;
    bytes public lastData;

    /// @notice The compliance contract to call check() on.
    address public compliance;

    function setCompliance(address _compliance) external {
        compliance = _compliance;
    }

    /// @notice Calls ICompliance.check() as the bridge.
    function callCheck(
        address _from,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data,
        uint256 _nonce
    )
        external
        payable
        returns (bool allowed_, bytes32 id_)
    {
        allowed_ =
            ICompliance(compliance).check{ value: msg.value }(_from, _to, _value, _gasLimit, _isCreation, _data, _nonce);
        id_ = keccak256(abi.encode(_from, _to, _value, msg.value, _gasLimit, _isCreation, _data, _nonce));
    }

    /// @notice Called by L1Compliance._executeApproved via IOptimismPortal2.approved().
    function approved(
        address _from,
        address _to,
        uint256 _value,
        uint256 _mint,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data
    )
        external
        payable
    {
        approvedCalled = true;
        lastFrom = _from;
        lastTo = _to;
        lastValue = _value;
        lastMint = _mint;
        lastGasLimit = _gasLimit;
        lastIsCreation = _isCreation;
        lastData = _data;
    }

    /// @inheritdoc IDonatable
    function donateETH() external payable override {
        // Accept ETH without side effects.
    }

    receive() external payable { }
}
