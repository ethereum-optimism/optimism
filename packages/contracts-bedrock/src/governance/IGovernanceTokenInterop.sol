// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IVotes } from "@openzeppelin/contracts/governance/utils/IVotes.sol";
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title IGovernanceTokenInterop
/// @notice Interface for the IGovernanceTokenInterop contract.
interface IGovernanceTokenInterop is IERC20, IVotes {
    /// @notice Migrates a delegator to the `GovernanceDelegation` contract.
    /// @param _delegator The account to migrate.
    function migrate(address _delegator) external;
}
