// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

interface IComplianceModule is IBridgeHook, IProxyAdminOwnedBase {
    struct Record {
        uint64 heldAt;
        uint64 expiresAt;
    }

    error ComplianceModule_CallerNotPortal();
    error ComplianceModule_NotActiveHook();
    error ComplianceModule_CallerNotPolicy();
    error ComplianceModule_ItemNotHeld();
    error ComplianceModule_ItemNotCleared();
    error ComplianceModule_PolicyRefused();
    error ComplianceModule_ETHAmountMismatch();
    error ComplianceModule_ZeroAddress();
    error ComplianceModule_InsufficientBalance();
    error ComplianceModule_ZeroExpiry();
    error ComplianceModule_Paused();
    error ComplianceModule_OwnershipNotRenounceable();
    error InvalidInitialization();
    error NotInitializing();
    error OwnableUnauthorizedAccount(address account);
    error OwnableInvalidOwner(address owner);
    error SafeERC20FailedOperation(address token);
    error AddressInsufficientBalance(address account);
    error AddressEmptyCode(address target);
    error FailedInnerCall();

    event Initialized(uint64 version);
    event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
    event Held(bytes32 indexed id, BridgeHookTypes.Item item);
    event VerdictRecorded(bytes32 indexed id, uint64 expiresAt);
    event VerdictRevoked(bytes32 indexed id);
    event Completed(bytes32 indexed id);
    event PolicySet(IPolicy indexed previousPolicy, IPolicy indexed newPolicy);
    event Disabled(uint256 cutoffL2Nonce);
    event Enabled();

    function initialize(
        address _owner,
        IPolicy _policy,
        IOptimismPortal2 _portal,
        IL1CrossDomainMessenger _l1CrossDomainMessenger,
        IL1StandardBridge _l1StandardBridge
    )
        external;
    function version() external view returns (string memory);
    function owner() external view returns (address);
    function pendingOwner() external view returns (address);
    function transferOwnership(address newOwner) external; // nosemgrep
    function acceptOwnership() external;
    function renounceOwnership() external pure;
    function portal() external view returns (IOptimismPortal2);
    function bridge() external view returns (IL1StandardBridge);
    function l1CrossDomainMessenger() external view returns (IL1CrossDomainMessenger);
    function policy() external view returns (IPolicy);
    function disabled() external view returns (bool);
    function cutoffL2Nonce() external view returns (uint256);
    function paused() external view returns (bool);
    function items(bytes32 _id) external view returns (Record memory);
    function heldTokens(address l1Token, address l2Token) external view returns (uint256 amount);
    function parties(BridgeHookTypes.Item calldata _item) external view returns (IPolicy.Parties memory parties_);
    function setPolicy(IPolicy _policy) external;
    function disable(uint256 _cutoffL2Nonce) external;
    function enable() external;
    function recordVerdict(bytes32 _id, uint64 _expiresAt) external;
    function revokeVerdict(bytes32 _id) external;
    function completeDeposit(BridgeHookTypes.Item calldata _item) external;
    function completeWithdrawal(BridgeHookTypes.Item calldata _item) external;

    function __constructor__() external;
}
