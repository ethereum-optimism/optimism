// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IPaymaster } from "@account-abstraction/interfaces/IPaymaster.sol";
import { IEntryPoint } from "@account-abstraction/interfaces/IEntryPoint.sol";

interface IAtomicPaymaster is IPaymaster {
    error AtomicPaymaster_InvalidConfiguration();
    error AtomicPaymaster_NotSponsored();
    error OwnableInvalidOwner(address owner);
    error OwnableUnauthorizedAccount(address account);

    event AccountAllowed(address indexed account, bool allowed);
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    function __constructor__(address _owner, address _router, uint256 _maxSponsoredCost) external;
    function version() external view returns (string memory);
    function allowedAccounts(address _account) external view returns (bool);
    function router() external view returns (address);
    function maxSponsoredCost() external view returns (uint256);
    function setAccountAllowed(address _account, bool _allowed) external;
    function entryPoint() external view returns (IEntryPoint);
    function owner() external view returns (address);
    function transferOwnership(address _newOwner) external;
    function renounceOwnership() external;
    function deposit() external payable;
    function getDeposit() external view returns (uint256);
    function withdrawTo(address payable _withdrawAddress, uint256 _amount) external;
    function addStake(uint32 _unstakeDelaySec) external payable;
    function unlockStake() external;
    function withdrawStake(address payable _withdrawAddress) external;
}
