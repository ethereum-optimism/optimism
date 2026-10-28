// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";

interface IVisionPolicyV1 is IPolicy {
    error VisionPolicyV1_NotYetHoldable(uint256 holdableAt);
    error VisionPolicyV1_HoldDelayTooLong(uint64 max);

    event AllowlistUpdated(address indexed account, bool allowlisted);
    event DenylistUpdated(address indexed account, bool denylisted);
    event MaxClearanceAgeSet(uint64 maxClearanceAge);
    event HoldDelaySet(uint64 holdDelay);

    function screen(
        BridgeHookTypes.Item calldata _item,
        address[] calldata _parties
    )
        external
        view
        override
        returns (bool pass_);
    function screenRelease(
        BridgeHookTypes.Item calldata _item,
        address[] calldata _parties,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
        view
        override
        returns (bool pass_);
    function module() external view returns (IComplianceModule);
    function maxClearanceAge() external view returns (uint64);
    function holdDelay() external view returns (uint64);
    function allowlisted(address _account) external view returns (bool);
    function denylisted(address _account) external view returns (bool);
    function clear(bytes32 _id) external;
    function revoke(bytes32 _id) external;
    function clearAndCompleteDeposit(BridgeHookTypes.Item calldata _item) external;
    function clearAndCompleteWithdrawal(BridgeHookTypes.Item calldata _item) external;
    function addToAllowlist(address[] calldata _accounts) external;
    function removeFromAllowlist(address[] calldata _accounts) external;
    function addToDenylist(address[] calldata _accounts) external;
    function removeFromDenylist(address[] calldata _accounts) external;
    function setMaxClearanceAge(uint64 _maxClearanceAge) external;
    function setHoldDelay(uint64 _holdDelay) external;
}
