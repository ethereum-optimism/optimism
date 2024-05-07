// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ISemver } from "src/universal/ISemver.sol";

import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";
import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";

import { RLPReader } from "src/libraries/rlp/RLPReader.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";
import "src/dispute/lib/Types.sol";

/// @title AnchorStateRegistry
/// @notice The AnchorStateRegistry is a contract that stores the latest "anchor" state for each available
///         FaultDisputeGame type. The anchor state is the latest state that has been proposed on L1 and was not
///         challenged within the challenge period. By using stored anchor states, new FaultDisputeGame instances can
///         be initialized with a more recent starting state which reduces the amount of required offchain computation.
contract AnchorStateRegistry is Initializable, IAnchorStateRegistry, ISemver {
    /// @notice Describes an initial anchor state for a game type.
    struct StartingAnchorRoot {
        GameType gameType;
        OutputRoot outputRoot;
    }

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @notice The index of the block number in the RLP-encoded block header.
    /// @dev Consensus encoding reference:
    /// https://github.com/paradigmxyz/reth/blob/5f82993c23164ce8ccdc7bf3ae5085205383a5c8/crates/primitives/src/header.rs#L368
    uint256 internal constant HEADER_BLOCK_NUMBER_INDEX = 8;

    /// @notice DisputeGameFactory address.
    IDisputeGameFactory internal immutable DISPUTE_GAME_FACTORY;

    /// @notice Dispute game finalization delay.
    Duration internal immutable DISPUTE_GAME_FINALIZATION_DELAY;

    /// @inheritdoc IAnchorStateRegistry
    mapping(GameType => OutputRoot) public anchors;

    /// @inheritdoc IAnchorStateRegistry
    mapping(IDisputeGame => bool) public verifiedGames;

    /// @param _disputeGameFactory DisputeGameFactory address.
    constructor(IDisputeGameFactory _disputeGameFactory, Duration _gameFinalizationDelay) {
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
        DISPUTE_GAME_FINALIZATION_DELAY = _gameFinalizationDelay;

        // Initialize the implementation with an empty array of starting anchor roots.
        initialize(new StartingAnchorRoot[](0));
    }

    /// @notice Initializes the contract.
    /// @param _startingAnchorRoots An array of starting anchor roots.
    function initialize(StartingAnchorRoot[] memory _startingAnchorRoots) public initializer {
        for (uint256 i = 0; i < _startingAnchorRoots.length; i++) {
            StartingAnchorRoot memory startingAnchorRoot = _startingAnchorRoots[i];
            anchors[startingAnchorRoot.gameType] = startingAnchorRoot.outputRoot;
        }
    }

    /// @inheritdoc IAnchorStateRegistry
    function disputeGameFactory() external view returns (IDisputeGameFactory) {
        return DISPUTE_GAME_FACTORY;
    }

    /// @notice Verifies that the output root proposed as the root claim of a dispute game corresponds to the claimed
    ///         L2 block number.
    /// @param _disputeGame The dispute game contract.
    /// @param _outputRootProof The output root proof corresponding to the proposed output root in the dispute game.
    /// @param _headerRLP The RLP-encoded block header corresponding to the L2 block in the output root proof.
    function verifyAnchor(
        IFaultDisputeGame _disputeGame,
        Types.OutputRootProof calldata _outputRootProof,
        bytes calldata _headerRLP
    )
        external
    {
        (GameType gameType, Claim rootClaim, bytes memory extraData) = _disputeGame.gameData();
        (uint256 l2BlockNumber) = abi.decode(extraData, (uint256));

        // Grab the verified address of the game based on the game data.
        (IDisputeGame factoryRegisteredGame,) =
            DISPUTE_GAME_FACTORY.games({ _gameType: gameType, _rootClaim: rootClaim, _extraData: extraData });

        // Must be a valid game.
        require(
            address(factoryRegisteredGame) == address(_disputeGame),
            "AnchorStateRegistry: fault dispute game not registered with factory"
        );

        // No need to update anything if the anchor state is already newer.
        require(
            l2BlockNumber > anchors[gameType].l2BlockNumber,
            "AnchorStateRegistry: block number of proposal does not advance anchor"
        );

        // Must be a game that resolved in favor of the state.
        require(
            _disputeGame.status() == GameStatus.DEFENDER_WINS,
            "AnchorStateRegistry: status of proposal is not DEFENDER_WINS"
        );

        require(
            _disputeGame.resolvedAt().raw() + DISPUTE_GAME_FINALIZATION_DELAY.raw() <= block.timestamp,
            "AnchorStateRegistry: proposal not finalized"
        );

        // Verify the output root preimage.
        require(
            Hashing.hashOutputRootProof(_outputRootProof) == _disputeGame.rootClaim().raw(),
            "AnchorStateRegistry: output root proof invalid"
        );

        // Verify the block preimage.
        require(keccak256(_headerRLP) == _outputRootProof.latestBlockhash, "AnchorStateRegistry: header rlp invalid");

        // Decode the header RLP to find the number of the block. In the consensus encoding, the timestamp
        // is the 9th element in the list that represents the block header.
        RLPReader.RLPItem memory headerRLP = RLPReader.toRLPItem(_headerRLP);
        RLPReader.RLPItem[] memory headerContents = RLPReader.readList(headerRLP);
        bytes memory rawBlockNumber = RLPReader.readBytes(headerContents[HEADER_BLOCK_NUMBER_INDEX]);

        require(rawBlockNumber.length <= 32, "AnchorStateRegistry: bad block header timestamp");

        // Convert the raw, left-aligned block number to a uint256 by aligning it as a big-endian
        // number in the low-order bytes of a 32-byte word.
        //
        // SAFETY: The length of `rawBlockNumber` is checked above to ensure it is at most 32 bytes.
        uint256 blockNumber;
        assembly {
            blockNumber := shr(shl(0x03, sub(0x20, mload(rawBlockNumber))), mload(add(rawBlockNumber, 0x20)))
        }

        require(blockNumber == l2BlockNumber, "AnchorStateRegistry: block number mismatch");

        // Update the anchor state.
        anchors[gameType] = OutputRoot({ l2BlockNumber: l2BlockNumber, root: Hash.wrap(rootClaim.raw()) });

        // Flag the game as verified.
        verifiedGames[_disputeGame] = true;
    }
}
