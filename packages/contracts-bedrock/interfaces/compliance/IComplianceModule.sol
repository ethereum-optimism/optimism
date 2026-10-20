// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";

interface IComplianceModule is IBridgeHook {
    error ComplianceModule_CallerNotPortal();
    error ComplianceModule_CallerNotPolicy();
    error ComplianceModule_ItemNotHeld();
    error ComplianceModule_ItemNotCleared();
    error ComplianceModule_PolicyRefused();
    error ComplianceModule_ETHAmountMismatch();
    error ComplianceModule_ZeroAddress();
    error ComplianceModule_Paused();

    event Held(bytes32 indexed id, BridgeHookTypes.Item item);
    event VerdictRecorded(bytes32 indexed id, uint64 clearedAt);
    event VerdictRevoked(bytes32 indexed id);
    event Completed(bytes32 indexed id);
    event PolicySet(IPolicy indexed policy);
    event Disabled(uint256 cutoffL2Nonce);
    event Enabled();

    function portal() external view returns (IOptimismPortal2);
    function bridge() external view returns (IL1StandardBridge);
    function l1CrossDomainMessenger() external view returns (IL1CrossDomainMessenger);
    function policy() external view returns (IPolicy);
    function disabled() external view returns (bool);
    function cutoffL2Nonce() external view returns (uint256);
    function paused() external view returns (bool);
    function items(bytes32 _id) external view returns (uint64 heldAt_, uint64 clearedAt_);
    function heldTokens(address _l1Token, address _l2Token) external view returns (uint256);
    function parties(BridgeHookTypes.Item calldata _item) external view returns (address[] memory parties_);
    function setPolicy(IPolicy _policy) external;
    function disable(uint256 _cutoffL2Nonce) external;
    function enable() external;
    function recordVerdict(bytes32 _id) external;
    function revokeVerdict(bytes32 _id) external;
    function completeDeposit(BridgeHookTypes.Item calldata _item) external;
    function completeWithdrawal(BridgeHookTypes.Item calldata _item) external;
}
