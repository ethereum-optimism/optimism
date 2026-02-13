// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import { IKailuaTournament } from "interfaces/dispute/zk/IKailuaTournament.sol";
import { Claim, Duration, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";

interface IKailuaTreasury is IKailuaTournament {
    /// @notice Emitted when the participation bond is updated
    /// @param amount The new required bond amount
    event BondUpdated(uint256 amount);

    function ELIMINATION_SPLIT_DENOM() external view returns (uint256);
    function ELIMINATION_SPLIT_PROVER_NUM() external view returns (uint256);
    function ELIMINATION_SPLIT_WINNER_NUM() external view returns (uint256);
    function L2_BLOCK_NUMBER() external view returns (uint64);
    function ROOT_CLAIM() external view returns (Claim);
    function assignVanguard(address _vanguard, Duration _vanguardAdvantage) external;
    function claimEliminationRewards() external;
    function claimProposerBond() external;
    function eliminate(address _child, address prover) external;
    function eliminationRewards(address) external view returns (uint256);
    function eliminationRound(address) external view returns (uint256);
    function isProposing() external view returns (bool);
    function lastProposal(address) external view returns (IKailuaTournament);
    function lastResolved() external view returns (address);
    function paidBonds(address) external view returns (uint256);
    function participationBond() external view returns (uint256);
    function propose(Claim _rootClaim, bytes memory _extraData) external payable returns (IKailuaTournament tournament);
    function proposerOf(address) external view returns (address);
    function setParticipationBond(uint256 amount) external;
    function treasuryAddress() external pure returns (IKailuaTreasury treasuryAddress_);
    function updateLastResolved() external;
    function vanguard() external view returns (address);
    function vanguardAdvantage() external view returns (Duration);
}
