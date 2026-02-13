// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { IInitializable } from "interfaces/dispute/IInitializable.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IKailuaTreasury } from "interfaces/dispute/zk/IKailuaTreasury.sol";
import { KailuaLib } from "src/dispute/lib/KailuaLib.sol";
import { KailuaTournament } from "src/dispute/zk/KailuaTournament.sol";
import { KailuaTreasury } from "src/dispute/zk/KailuaTreasury.sol";
import { Duration, GameStatus, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import {
    GameNotInProgress,
    BlockNumberMismatch,
    BadExtraData,
    BlockNumberMismatch,
    NotProven,
    ClockNotExpired,
    ProvenFaulty,
    OutOfOrderResolution,
    ProposalGapRemaining,
    InvalidParent,
    BlobHashMissing,
    InvalidDuplicationCounter
} from "src/dispute/lib/Errors.sol";

contract KailuaGame is KailuaTournament {
    // ------------------------------
    // Immutable configuration
    // ------------------------------

    /// @notice The duration after which the proposal is accepted
    Duration public immutable MAX_CLOCK_DURATION;

    /// @notice The timestamp of the genesis l2 block
    uint64 public immutable GENESIS_TIME_STAMP;

    /// @notice The time between l2 blocks
    uint64 public immutable L2_BLOCK_TIME;

    constructor(
        KailuaTreasury _kailuaTreasury,
        uint64 _genesisTimeStamp,
        uint64 _l2BlockTime,
        Duration _maxClockDuration
    )
        KailuaTournament(
            IKailuaTreasury(address(_kailuaTreasury)),
            _kailuaTreasury.KAILUA_VERIFIER(),
            _kailuaTreasury.PROPOSAL_OUTPUT_COUNT(),
            _kailuaTreasury.OUTPUT_BLOCK_SPAN(),
            _kailuaTreasury.GAME_TYPE(),
            _kailuaTreasury.OPTIMISM_PORTAL()
        )
    {
        GENESIS_TIME_STAMP = _genesisTimeStamp;
        L2_BLOCK_TIME = _l2BlockTime;
        MAX_CLOCK_DURATION = _maxClockDuration;
    }

    // ------------------------------
    // IInitializable implementation
    // ------------------------------

    /// @inheritdoc IInitializable
    function initialize() external payable override {
        super.initializeInternal();

        // Revert if the calldata size is not the expected length.
        //
        // This is to prevent adding extra or omitting bytes from to `extraData` that result in a different game UUID
        // in the factory, but are not used by the game, which would allow for multiple dispute games for the same
        // output proposal to be created.
        //
        // Expected length: 0x72
        // - 0x04 selector                      0x00 0x04
        // - 0x14 creator address               0x04 0x18
        // - 0x20 root claim                    0x18 0x38
        // - 0x20 l1 head                       0x38 0x58
        // - 0x18 extraData:                    0x58 0x70
        //      + 0x08 l2SequenceNumber            0x58 0x60
        //      + 0x08 parentGameIndex          0x60 0x68
        //      + 0x08 duplicationCounter       0x68 0x70
        // - 0x02 CWIA bytes                    0x70 0x72
        if (msg.data.length != 0x72) {
            revert BadExtraData();
        }

        // Only allow monotonic duplication counter
        uint64 duplicationCounter_ = duplicationCounter();
        if (duplicationCounter_ > 0) {
            bytes memory extra = abi.encodePacked(msg.data[0x58:0x68], duplicationCounter_ - 1);
            (IDisputeGame previousDuplicate,) = DISPUTE_GAME_FACTORY.games(GAME_TYPE, rootClaim(), extra);
            if (address(previousDuplicate) == address(0x0)) {
                revert InvalidDuplicationCounter();
            }
        }

        // Do not initialize a game that does not cover the required number of l2 blocks
        KailuaTournament parentGame_ = parentGame();
        if (l2SequenceNumber() != parentGame_.l2SequenceNumber() + PROPOSAL_OUTPUT_COUNT * OUTPUT_BLOCK_SPAN) {
            revert BlockNumberMismatch(parentGame_.l2SequenceNumber(), l2SequenceNumber());
        }

        // Store the intermediate output blob hashes
        for (uint256 i = 0; i < PROPOSAL_BLOBS; i++) {
            bytes32 hash = blobhash(i);
            if (hash == 0x0) {
                revert BlobHashMissing(i, PROPOSAL_BLOBS);
            }
            proposalBlobHashes.push(Hash.wrap(hash));
        }

        // Verify that parent game is known by the treasury
        if (KAILUA_TREASURY.proposerOf(address(parentGame_)) == address(0x0)) {
            revert InvalidParent();
        }

        // If a proof was submitted, do not allow bad proposals to be created
        if (!parentGame_.isViableSignature(signature())) {
            revert ProvenFaulty();
        }

        // Register this new game in the parent game's contract
        parentGame_.appendChild();

        // Do not permit proposals if l2 block time is ahead of the l1 block time
        if (block.timestamp < minCreationTime().raw()) {
            revert ProposalGapRemaining(block.timestamp, minCreationTime().raw());
        }
    }

    // ------------------------------
    // IDisputeGame implementation
    // ------------------------------

    /// @inheritdoc IDisputeGame
    function extraData() external pure returns (bytes memory extraData_) {
        // The extra data starts at the second word within the cwia calldata and
        // is 24 bytes long.
        extraData_ = _getArgBytes(0x54, 0x18);
    }

    /// @inheritdoc IDisputeGame
    function resolve() external returns (GameStatus status_) {
        // INVARIANT: Resolution cannot occur unless the game is currently in progress.
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // INVARIANT: Optimistic resolution cannot occur unless parent game is resolved.
        KailuaTournament parentGame_ = parentGame();
        if (parentGame_.status() != GameStatus.DEFENDER_WINS) {
            revert OutOfOrderResolution();
        }

        // INVARIANT: Cannot resolve unless proven valid or the clock has expired
        if (parentGame_.validChildSignature() != 0) {
            if (signature() != parentGame_.validChildSignature()) {
                revert ProvenFaulty();
            }
        } else if (getChallengerDuration(uint64(block.timestamp)).raw() > 0) {
            revert ClockNotExpired();
        }

        // INVARIANT: Can only resolve the last remaining child
        if (parentGame_.pruneChildren(parentGame_.childCount() * 2) != this) {
            revert NotProven();
        }

        // Mark resolution timestamp
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));

        // Update the status and emit the resolved event, note that we're performing a storage update here.
        emit Resolved(status = status_ = GameStatus.DEFENDER_WINS);

        // Update lastResolved
        KAILUA_TREASURY.updateLastResolved();
    }

    // ------------------------------
    // Immutable instance data
    // ------------------------------

    /// @notice The index of the parent game in the `DisputeGameFactory`.
    function parentGameIndex() public pure returns (uint64 parentGameIndex_) {
        parentGameIndex_ = _getArgUint64(0x5C);
    }

    /// @notice The number of duplicate proposals preceeding this one.
    function duplicationCounter() public pure returns (uint64 duplicationCounter_) {
        duplicationCounter_ = _getArgUint64(0x64);
    }

    /// @inheritdoc KailuaTournament
    function parentGame() public view override returns (KailuaTournament parentGame_) {
        (,, IDisputeGame parentDisputeGame) = DISPUTE_GAME_FACTORY.gameAtIndex(parentGameIndex());

        // Interpret parent game as another instance of this game type
        parentGame_ = KailuaTournament(address(parentDisputeGame));
    }

    // ------------------------------
    // Fault proving
    // ------------------------------

    /// @inheritdoc KailuaTournament
    function verifyIntermediateOutput(
        uint64 outputNumber,
        uint256 outputFe,
        bytes calldata blobCommitment,
        bytes calldata kzgProof
    )
        external
        override
        returns (bool success)
    {
        uint256 blobIndex = KailuaLib.blobIndex(outputNumber);
        uint32 blobPosition = KailuaLib.fieldElementIndex(outputNumber);
        bytes32 proposalBlobHash = KailuaLib.versionedKZGHash(blobCommitment);
        // Note: Only known blobs can be used to validate an intermediate output
        if (proposalBlobHash != proposalBlobHashes[blobIndex].raw()) {
            success = false;
        } else {
            success = KailuaLib.verifyKZGBlobProof(proposalBlobHash, blobPosition, outputFe, blobCommitment, kzgProof);
        }
    }

    /// @inheritdoc KailuaTournament
    function getChallengerDuration(uint64 asOfTimestamp) public view override returns (Duration duration_) {
        // INVARIANT: The game must be in progress to query the remaining time to respond to a given claim.
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // Compute the duration elapsed of the potential challenger's clock.
        uint64 elapsed = asOfTimestamp - createdAt.raw();
        uint64 maximum = MAX_CLOCK_DURATION.raw();
        duration_ = elapsed >= maximum ? Duration.wrap(0) : Duration.wrap(maximum - elapsed);
    }

    /// @inheritdoc KailuaTournament
    function minCreationTime() public view override returns (Timestamp minCreationTime_) {
        minCreationTime_ = Timestamp.wrap(uint64(GENESIS_TIME_STAMP + l2SequenceNumber() * L2_BLOCK_TIME));
    }
}
