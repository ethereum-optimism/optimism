// Copyright 2024, 2025 RISC Zero, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

/// @notice Emitted when the participation bond is updated
/// @param amount The new required bond amount
event BondUpdated(uint256 amount);

interface IKailuaTreasury {
    /// @notice Returns the game index at which proposer was proven faulty
    function eliminationRound(address proposer) external view returns (uint256);

    /// @notice Returns the proposer of a game
    function proposerOf(address game) external view returns (address);

    /// @notice Eliminates a child's proposer and allocates their bond to the prover
    function eliminate(address child, address prover) external;

    /// @notice Returns true iff a proposal is currently being submitted
    function isProposing() external view returns (bool);

    /// @notice Returns the last resolved proposal contract address
    function lastResolved() external view returns (address);

    /// @notice Updates the last resolved contract address to that of the caller
    function updateLastResolved() external;

    /// @notice Returns the collateral required to submit proposals
    function participationBond() external view returns (uint256);

    /// @notice Returns the prover's number of shares in elimination rewards
    function ELIMINATION_SPLIT_PROVER_NUM() external view returns (uint256);

    /// @notice Returns the total number of shares for elimination rewards
    function ELIMINATION_SPLIT_DENOM() external view returns (uint256);
}
