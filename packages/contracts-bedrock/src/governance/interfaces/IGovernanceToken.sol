// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20Metadata } from "@openzeppelin/contracts/token/ERC20/extensions/IERC20Metadata.sol";
import { IOwnable } from "src/universal/interfaces/IOwnable.sol";
import { ERC20Votes } from "@openzeppelin/contracts/token/ERC20/extensions/ERC20Votes.sol";

/// @title IGovernanceToken
/// @notice Interface for the GovernanceToken contract.
interface IGovernanceToken is IERC20Metadata, IOwnable {
    event DelegateChanged(address indexed delegator, address indexed fromDelegate, address indexed toDelegate);
    event DelegateVotesChanged(address indexed delegate, uint256 previousBalance, uint256 newBalance);

    function DOMAIN_SEPARATOR() external view returns (bytes32);
    function burn(uint256 amount) external;
    function burnFrom(address account, uint256 amount) external;
    function checkpoints(address account, uint32 pos) external view returns (ERC20Votes.Checkpoint memory);
    function decreaseAllowance(address spender, uint256 subtractedValue) external returns (bool);
    function delegate(address delegatee) external;
    function delegateBySig(address delegatee, uint256 nonce, uint256 expiry, uint8 v, bytes32 r, bytes32 s) external;
    function delegates(address account) external view returns (address);
    function getPastTotalSupply(uint256 blockNumber) external view returns (uint256);
    function getPastVotes(address account, uint256 blockNumber) external view returns (uint256);
    function getVotes(address account) external view returns (uint256);
    function increaseAllowance(address spender, uint256 addedValue) external returns (bool);
    function mint(address _account, uint256 _amount) external;
    function nonces(address owner) external view returns (uint256);
    function numCheckpoints(address account) external view returns (uint32);
    function permit(
        address owner,
        address spender,
        uint256 value,
        uint256 deadline,
        uint8 v,
        bytes32 r,
        bytes32 s
    )
        external;
}
