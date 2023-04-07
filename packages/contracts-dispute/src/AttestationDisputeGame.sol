// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "src/types/Types.sol";
import { Owner } from "src/util/Owner.sol";
import { Clone } from "src/util/Clone.sol";
import { Initializable } from "src/util/Initializable.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IOutputOracle } from "src/interfaces/IOutputOracle.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";

/// @title AttestationDisputeGame
/// @author refcell <https://github.com/refcell>
/// @author clabby <https://github.com/clabby>
/// @notice The attestation dispute game allows a permissioned set of challengers to dispute an output.
/// @notice The contract owner should be the `L2OutputOracle`.
/// @notice Whereas the provided challengerSet is intended to be a multisig responsible for resolving the dispute.
contract AttestationDisputeGame is IDisputeGame, Owner, Initializable {
    /// @notice The starting timestamp of the game
    Timestamp public gameStart;

    /// @notice The l2 block number for which the output to dispute
    uint256 public l2BlockNumber;

    /// @notice The set of challengers that can challenge the output.
    /// @dev This should be a multisig that can resolve the dispute.
    /// @dev The multisig must reach a quorum before calling `resolve`.
    address public challengeSet;

    /// @notice The game status.
    GameStatus internal gameStatus;

    /// @notice Instantiates a new AttestationDisputeGame contract.
    /// @param _owner The owner of the contract.
    /// @param _blockNum The l2 block number for which the output to dispute.
    /// @param _challengeSet The set of challengers that can challenge the output.
    constructor(address _owner, uint256 _blockNum, address _challengeSet) Owner(_owner) {
        l2BlockNumber = _blockNum;
        challengeSet = _challengeSet;
    }

    /// @notice Initializes the challenge contract.
    function initialize() external initializer {
        gameStatus = GameStatus.IN_PROGRESS;
        gameStart = Timestamp.wrap(uint64(block.timestamp));
    }

    /// @notice Returns the semantic version.
    function version() external pure override returns (string memory) {
        assembly {
            // Store the pointer to the string
            mstore(returndatasize(), 0x20)
            // Store the version ("0.0.1")
            // len |   "0.0.1"
            // 0x05|302E302E31
            mstore(0x25, 0x05302E302E31)
            // Return the semantic version of the contract
            return(returndatasize(), 0x60)
        }
    }

    /// @notice Returns the current status of the game.
    function status() external view override returns (GameStatus _status) {
        _status = gameStatus;
    }

    /// @notice Returns the dispute game type.
    /// @return _gameType The type of proof system being used.
    function gameType() external pure override returns (GameType _gameType) {
        _gameType = GameType.ATTESTATION;
    }

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return _extraData Any extra data supplied to the dispute game contract by the creator.
    function extraData() external pure override returns (bytes memory _extraData) {
        return bytes("");
    }

    /// @notice Attestation games do not have bond managers.
    /// @notice This will return an invalid IBondManager at address 0x0.
    function bondManager() external pure returns (IBondManager _bondManager) {
        return IBondManager(address(0));
    }

    /// @notice Returns the output that is being disputed.
    /// @return _rootClaim The root claim of the DisputeGame.
    function rootClaim() external view override returns (Claim _rootClaim) {
        IOutputOracle.OutputProposal memory out = IOutputOracle(_owner).getL2Output(l2BlockNumber);
        _rootClaim = Claim.wrap(out.outputRoot);
    }

    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    function resolve() external override returns (GameStatus _status) {
        require(msg.sender == challengeSet, "AttestationDisputeGame: Only the challenge set can resolve the game.");
        require(gameStatus == GameStatus.IN_PROGRESS, "AttestationDisputeGame: Game must be in progress to resolve.");
        IOutputOracle(_owner).deleteL2Outputs(l2BlockNumber);
        gameStatus = GameStatus.CHALLENGER_WINS;
        return GameStatus.CHALLENGER_WINS;
    }
}
