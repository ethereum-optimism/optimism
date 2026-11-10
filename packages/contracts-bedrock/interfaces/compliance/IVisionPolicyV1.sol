// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";

interface IVisionPolicyV1 is IPolicy {
    error VisionPolicyV1_NotYetHoldable(uint256 holdableAt);
    error VisionPolicyV1_HoldDelayTooLong(uint64 max);
    error VisionPolicyV1_InvalidExpiry();
    error VisionPolicyV1_OutOfBounds();
    error VisionPolicyV1_Unauthorized();
    error VisionPolicyV1_ZeroAddress();
    error VisionPolicyV1_OwnershipNotRenounceable();
    error VisionPolicyV1_InvalidPartiesLength();
    error OwnableUnauthorizedAccount(address account);
    error OwnableInvalidOwner(address owner);

    event AllowlistUpdated(address indexed account, bool allowlisted);
    event DenylistUpdated(address indexed account, bool denylisted);
    event VerdictWriterSet(address indexed previousVerdictWriter, address indexed newVerdictWriter);
    event MaxVerdictValiditySet(uint64 maxVerdictValidity);
    event HoldDelaySet(uint64 holdDelay);
    event OwnershipTransferStarted(address indexed previousOwner, address indexed newOwner);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    function module() external view returns (IComplianceModule module_);
    function verdictWriter() external view returns (address);
    function maxVerdictValidity() external view returns (uint64);
    function holdDelay() external view returns (uint64);
    function allowlisted(address _account) external view returns (bool);
    function denylisted(address _account) external view returns (bool);

    function screenDeposit(
        BridgeHookTypes.Item calldata _item,
        Parties calldata _parties
    )
        external
        view
        override
        returns (bool pass_);
    function screenWithdrawal(
        BridgeHookTypes.Item calldata _item,
        Parties calldata _parties,
        uint64 _expiresAt,
        uint256 _finalizableAt
    )
        external
        view
        override
        returns (bool pass_);
    function screenRelease(
        BridgeHookTypes.Item calldata _item,
        Parties calldata _parties,
        uint64 _expiresAt
    )
        external
        view
        override
        returns (bool pass_);

    function clear(bytes32 _id, uint64 _expiresAt) external;
    function revoke(bytes32 _id) external;
    function clearAndCompleteDeposit(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external;
    function clearAndCompleteWithdrawal(BridgeHookTypes.Item calldata _item, uint64 _expiresAt) external;

    function multicall(bytes[] calldata _data) external returns (bytes[] memory results_);

    function addToAllowlist(address[] calldata _accounts) external;
    function removeFromAllowlist(address[] calldata _accounts) external;
    function addToDenylist(address[] calldata _accounts) external;
    function removeFromDenylist(address[] calldata _accounts) external;
    function setVerdictWriter(address _verdictWriter) external;
    function setMaxVerdictValidity(uint64 _maxVerdictValidity) external;
    function setHoldDelay(uint64 _holdDelay) external;

    function owner() external view returns (address);
    function pendingOwner() external view returns (address);
    function transferOwnership(address _newOwner) external; // nosemgrep
    function acceptOwnership() external;
    function renounceOwnership() external pure;

    function __constructor__(
        IComplianceModule _module,
        address _owner,
        address _verdictWriter,
        uint64 _maxVerdictValidity,
        uint64 _holdDelay,
        address[] memory _allowlist,
        address[] memory _denylist
    )
        external;
}
