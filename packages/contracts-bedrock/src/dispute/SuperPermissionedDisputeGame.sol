// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Clone } from "@solady/utils/Clone.sol";
import { Claim, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { AlreadyInitialized, BadAuth, BadExtraData, UnknownChainId } from "src/dispute/lib/Errors.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title SuperPermissionedDisputeGame

/// @custom:proxied
/// @notice A simplified permissioned super-root dispute game. The proposer creates a super-root
///         proposal, and the proposal resolves in favor of the defender. Invalid proposals are
///         invalidated through the AnchorStateRegistry blacklist before finalization.
contract SuperPermissionedDisputeGame is Clone, ISemver, IDisputeGame {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The timestamp at which the game was created.
    Timestamp public createdAt;

    /// @notice The timestamp at which the game resolved.
    Timestamp public resolvedAt;

    /// @notice The current status of the game.
    GameStatus public status;

    /// @notice Whether the game type was respected when the game was created.
    bool public wasRespectedGameTypeWhenCreated;

    /// @notice Prevents re-initialization.
    bool internal initialized;

    constructor() { }

    /// @notice Initializes the contract.
    function initialize() external payable {
        if (initialized) revert AlreadyInitialized();
        if (!_verifyInitCallDataLength()) revert BadExtraData();
        if (Hashing.hashSuperRootProof(Encoding.decodeSuperRootProof(extraData())) != rootClaim().raw()) {
            revert BadExtraData();
        }
        if (tx.origin != proposer()) revert BadAuth();

        IAnchorStateRegistry registry = anchorStateRegistry();
        (, uint256 rootL2SequenceNumber) = registry.getAnchorRoot();
        if (l2SequenceNumber() <= rootL2SequenceNumber) revert BadExtraData();

        createdAt = Timestamp.wrap(uint64(block.timestamp));
        resolvedAt = createdAt;
        status = GameStatus.DEFENDER_WINS;
        wasRespectedGameTypeWhenCreated = GameType.unwrap(registry.respectedGameType()) == GameType.unwrap(gameType());
        initialized = true;

        emit Resolved(GameStatus.DEFENDER_WINS);
    }

    /// @notice No-op because the game resolves in favor of the defender during initialization.
    function resolve() external pure returns (GameStatus status_) {
        status_ = GameStatus.DEFENDER_WINS;
    }

    /// @notice Returns the output root in the root claim for the specified L2 chain ID.
    function rootClaimByChainId(uint256 _chainId) public pure returns (Claim outputRootClaim_) {
        Types.SuperRootProof memory superRootProof = Encoding.decodeSuperRootProof(extraData());
        Types.OutputRootWithChainId[] memory outputRoots = superRootProof.outputRoots;

        for (uint256 i; i < outputRoots.length; i++) {
            if (outputRoots[i].chainId == _chainId) return Claim.wrap(outputRoots[i].root);
        }
        revert UnknownChainId();
    }

    /// @notice Returns the type, root claim, and extra data for factory registration checks.
    function gameData() external pure returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = gameType();
        rootClaim_ = rootClaim();
        extraData_ = extraData();
    }

    /// @notice Getter for the creator of the dispute game.
    function gameCreator() public pure returns (address creator_) {
        creator_ = _getArgAddress(0);
    }

    /// @notice Getter for the root claim.
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgBytes32(20));
    }

    /// @notice Getter for the parent hash of the L1 block when the dispute game was created.
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgBytes32(52));
    }

    /// @notice Getter for the game type.
    function gameType() public pure returns (GameType gameType_) {
        gameType_ = GameType.wrap(_getArgUint32(84));
    }

    /// @notice Getter for the extra data.
    function extraData() public pure returns (bytes memory extraData_) {
        extraData_ = _getArgBytes(_preExtraDataByteCount(), _extraDataByteCount());
    }

    /// @notice Getter for the super-root timestamp / L2 sequence number.
    function l2SequenceNumber() public pure returns (uint256 l2SequenceNumber_) {
        l2SequenceNumber_ = _getArgUint64(_preExtraDataByteCount() + 1);
    }

    /// @notice Getter for the AnchorStateRegistry.
    function anchorStateRegistry() public pure returns (IAnchorStateRegistry registry_) {
        registry_ = IAnchorStateRegistry(_getArgAddress(_preExtraDataByteCount() + _extraDataByteCount()));
    }

    /// @notice Getter for the proposer role.
    function proposer() public pure returns (address proposer_) {
        proposer_ = _getArgAddress(_preExtraDataByteCount() + _extraDataByteCount() + 20);
    }

    /// @notice Returns the length of the super extra data in the initialize call.
    function _extraDataByteCount() internal pure returns (uint256) {
        uint256 immutableArgsLength = msg.data.length - _getImmutableArgsOffset() - 2;
        return immutableArgsLength - _preExtraDataByteCount() - gameImplArgsByteCount();
    }

    /// @notice Returns the byte count of the data before the extra data in the CWIA payload.
    function _preExtraDataByteCount() internal pure returns (uint256) {
        return 88;
    }

    /// @notice Returns the byte count of the simplified game implementation args.
    function gameImplArgsByteCount() internal pure returns (uint256) {
        return 40;
    }

    /// @notice Validates the expected length of initialize calldata.
    function _verifyInitCallDataLength() internal pure returns (bool) {
        uint256 preExtraDataLen = 4 + 2 + _preExtraDataByteCount();
        if (msg.data.length < preExtraDataLen) return false;

        uint256 extraDataAndGameArgsLength = msg.data.length - preExtraDataLen;
        if (extraDataAndGameArgsLength < gameImplArgsByteCount()) return false;

        uint256 superLen = extraDataAndGameArgsLength - gameImplArgsByteCount();
        if (superLen < 9) return false;

        uint256 rem = superLen - 9;
        return rem != 0 && rem % 64 == 0;
    }
}
