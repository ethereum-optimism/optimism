// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { RLPReader } from "../libraries/rlp/RLPReader.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { IDisputeGameFactory } from "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame, GameStatus } from "../../interfaces/dispute/IDisputeGame.sol";
import { IAnchorStateRegistry } from "../../interfaces/dispute/IAnchorStateRegistry.sol";
import { Claim, LibClaim } from "../dispute/lib/LibUDT.sol";
import { Types } from "../libraries/Types.sol";

contract SuperRootMigrator {
    using LibClaim for Claim;
    using RLPReader for RLPReader.RLPItem;
    using RLPReader for bytes;

    /*//////////////////////////////////////////////////////////////
                                 ERRORS
    //////////////////////////////////////////////////////////////*/

    error InvalidOutput();
    error InvalidHeaderRLP();
    error InvalidGameStatus();
    error TimestampMismatch();
    error InvalidGameType();
    error InvalidGameProxy();
    error LengthMismatch();
    error ChainIDsNotAscending();
    error BlacklistedGame();
    error MissingAnchorStateRegistry();

    /*//////////////////////////////////////////////////////////////
                               CONSTANTS
    //////////////////////////////////////////////////////////////*/

    /// @notice The index of the block number in the RLP-encoded block header.
    uint256 internal constant HEADER_TIMESTAMP_INDEX = 11;
    uint8 internal constant SUPER_VERSION = uint8(1);

    /*//////////////////////////////////////////////////////////////
                                 STATE
    //////////////////////////////////////////////////////////////*/

    IDisputeGameFactory[] public gameFactories;
    uint256[] public chainIDs;
    mapping(uint256 => IAnchorStateRegistry) public anchorStateRegistries;

    constructor(
        IDisputeGameFactory[] memory _gameFactories,
        IAnchorStateRegistry[] memory _anchorStateRegistries,
        uint256[] memory _chainIDs
    ) {
        if (_gameFactories.length != _chainIDs.length || _gameFactories.length != _anchorStateRegistries.length) {
            revert LengthMismatch();
        }

        // Verify that chainIDs are in ascending order per doc
        for (uint256 i = 1; i < _chainIDs.length; i++) {
            if (_chainIDs[i] <= _chainIDs[i - 1]) {
                revert ChainIDsNotAscending();
            }
        }

        gameFactories = _gameFactories;
        chainIDs = _chainIDs;
        for (uint256 i = 0; i < _chainIDs.length; i++) {
            anchorStateRegistries[_chainIDs[i]] = _anchorStateRegistries[i];
        }
    }

    /*//////////////////////////////////////////////////////////////
                           EXTERNAL FUNCTIONS
    //////////////////////////////////////////////////////////////*/

    function chainsLen() external view returns (uint256) {
        return gameFactories.length;
    }

    function migrate(
        uint256[] calldata _gameIdxs,
        Types.OutputRootProof[] calldata _outputs,
        bytes[] calldata _headerRLP
    )
        external
        view
        returns (bytes memory super_, bytes32 superRoot_)
    {
        uint256 chainCount = gameFactories.length;
        if (_gameIdxs.length != chainCount) {
            revert LengthMismatch();
        }
        /// TODO: should this be in the loop based on the game? implies that this contract can only migrate factories
        /// with games at the same timestamp?
        uint256 expectedTimestamp = 0;
        bytes memory chainData;
        for (uint256 i = 0; i < chainCount; i++) {
            (,, IDisputeGame game) = gameFactories[i].gameAtIndex(_gameIdxs[i]);
            if (address(game) == address(0)) {
                revert InvalidGameProxy();
            }
            // Fetch the ASR for the specific chain
            IAnchorStateRegistry asr = anchorStateRegistries[chainIDs[i]];
            if (address(asr) == address(0)) {
                revert MissingAnchorStateRegistry();
            }
            if (asr.isGameBlacklisted(game)) {
                revert BlacklistedGame();
            }
            if (!game.wasRespectedGameTypeWhenCreated()) {
                revert InvalidGameType();
            }
            if (game.status() != GameStatus.DEFENDER_WINS) {
                revert InvalidGameStatus();
            }

            bytes32 outputRoot = game.rootClaim().raw();
            if (Hashing.hashOutputRootProof(_outputs[i]) != outputRoot) {
                revert InvalidOutput();
            }
            if (keccak256(_headerRLP[i]) != _outputs[i].latestBlockhash) {
                revert InvalidHeaderRLP();
            }

            // Decode the header RLP to find the number of the block. In the consensus encoding, the timestamp
            // is the 12th element in the list that represents the block header.
            RLPReader.RLPItem[] memory headerContents = _headerRLP[i].toRLPItem().readList();
            bytes memory rawTimestamp = headerContents[HEADER_TIMESTAMP_INDEX].readBytes();

            // Sanity check the block number string length.
            if (rawTimestamp.length > 32) {
                revert InvalidHeaderRLP();
            }

            // Convert the raw, left-aligned timestamp to a uint256 by aligning it as a big-endian
            // number in the low-order bytes of a 32-byte word.
            //
            // SAFETY: The length of `rawTimestamp` is checked above to ensure it is at most 32 bytes.
            /// TODO: simplify this without assembly with casting OOO
            uint256 blockTimestamp;
            assembly {
                blockTimestamp := shr(shl(0x03, sub(0x20, mload(rawTimestamp))), mload(add(rawTimestamp, 0x20)))
            }
            if (i != 0 && blockTimestamp != expectedTimestamp) {
                revert TimestampMismatch();
            }
            chainData = abi.encodePacked(chainData, chainIDs[i], outputRoot);
        }
        bytes memory superBytes = abi.encodePacked(SUPER_VERSION, uint64(expectedTimestamp), chainData);

        bytes32 superRoot = keccak256(superBytes);
        // TODO: call AnchorStateRegistry.updateAnchorState(expectedTimestamp, superRoot);
        // Or just put this method on AnchorStateRegistry and it can update itself.

        return (superBytes, superRoot);
    }
}
