// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { LibKeccak } from "@lib-keccak/LibKeccak.sol";
import "./libraries/CannonErrors.sol";
import "./libraries/CannonTypes.sol";

/// @title PreimageOracle
/// @notice A contract for storing permissioned pre-images.
/// @custom:attribution Solady <https://github.com/Vectorized/solady/blob/main/src/utils/MerkleProofLib.sol#L13-L43>
/// @custom:attribution Beacon Deposit Contract <https://etherscan.io/address/0x00000000219ab540356cbb839cbe05303d7705fa#code>
contract PreimageOracle is IPreimageOracle {
    ////////////////////////////////////////////////////////////////
    //                         Constants                          //
    ////////////////////////////////////////////////////////////////

    /// @notice The depth of the keccak256 merkle tree. Supports up to 32,768 blocks, or ~4.46MB preimages.
    uint256 public constant KECCAK_TREE_DEPTH = 15;
    /// @notice The maximum number of keccak blocks that can fit into the merkle tree.
    uint256 public constant MAX_LEAF_COUNT = 2**KECCAK_TREE_DEPTH - 1;

    ////////////////////////////////////////////////////////////////
    //                 Authorized Preimage Parts                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Mapping of pre-image keys to pre-image lengths.
    mapping(bytes32 => uint256) public preimageLengths;
    /// @notice Mapping of pre-image keys to pre-image parts.
    mapping(bytes32 => mapping(uint256 => bytes32)) public preimageParts;
    /// @notice Mapping of pre-image keys to pre-image part offsets.
    mapping(bytes32 => mapping(uint256 => bool)) public preimagePartOk;

    ////////////////////////////////////////////////////////////////
    //                  Large Preimage Proposals                  //
    ////////////////////////////////////////////////////////////////

    /// @notice A raw leaf of the large preimage proposal merkle tree.
    struct Leaf {
        /// @notice The input absorbed for the block, exactly 136 bytes.
        bytes input;
        /// @notice The index of the block in the absorbtion process.
        uint256 index;
        /// @notice The hash of the internal state after absorbing the input.
        bytes32 stateCommitment;
    }

    /// @notice Static padding hashes. These values are persisted in storage, but are entirely immutable
    ///         after the constructor's execution.
    bytes32[KECCAK_TREE_DEPTH] public zeroHashes;
    /// @notice Mapping of addresses to UUIDs to the current branch path of the merkleization process.
    mapping(address => mapping(uint256 => bytes32[KECCAK_TREE_DEPTH])) public proposalBranches;
    /// @notice Mapping of addresses to UUIDs to the current number of blocks absorbed (# of leaves).
    mapping(address => mapping(uint256 => uint256)) public proposalBlocksProcessed;
    /// @notice Mapping of addresses to UUIDs to the timestamp of creation of the proposal as well as the challenged
    ///         status.
    mapping(address => mapping(uint256 => LPPMetaData)) public proposalMetadata;

    ////////////////////////////////////////////////////////////////
    //                        Constructor                         //
    ////////////////////////////////////////////////////////////////

    constructor() {
        // Compute hashes in empty sparse Merkle tree
        for (uint256 height = 0; height < KECCAK_TREE_DEPTH - 1; height++) {
            zeroHashes[height + 1] = keccak256(abi.encodePacked(zeroHashes[height], zeroHashes[height]));
        }
    }

    ////////////////////////////////////////////////////////////////
    //             Standard Preimage Route (External)             //
    ////////////////////////////////////////////////////////////////

    /// @inheritdoc IPreimageOracle
    function readPreimage(bytes32 _key, uint256 _offset) external view returns (bytes32 dat_, uint256 datLen_) {
        require(preimagePartOk[_key][_offset], "pre-image must exist");

        // Calculate the length of the pre-image data
        // Add 8 for the length-prefix part
        datLen_ = 32;
        uint256 length = preimageLengths[_key];
        if (_offset + 32 >= length + 8) {
            datLen_ = length + 8 - _offset;
        }

        // Retrieve the pre-image data
        dat_ = preimageParts[_key][_offset];
    }

    /// @inheritdoc IPreimageOracle
    function loadLocalData(
        uint256 _ident,
        bytes32 _localContext,
        bytes32 _word,
        uint256 _size,
        uint256 _partOffset
    )
        external
        returns (bytes32 key_)
    {
        // Compute the localized key from the given local identifier.
        key_ = PreimageKeyLib.localizeIdent(_ident, _localContext);

        // Revert if the given part offset is not within bounds.
        if (_partOffset > _size + 8 || _size > 32) {
            revert PartOffsetOOB();
        }

        // Prepare the local data part at the given offset
        bytes32 part;
        assembly {
            // Clean the memory in [0x20, 0x40)
            mstore(0x20, 0x00)

            // Store the full local data in scratch space.
            mstore(0x00, shl(192, _size))
            mstore(0x08, _word)

            // Prepare the local data part at the requested offset.
            part := mload(_partOffset)
        }

        // Store the first part with `_partOffset`.
        preimagePartOk[key_][_partOffset] = true;
        preimageParts[key_][_partOffset] = part;
        // Assign the length of the preimage at the localized key.
        preimageLengths[key_] = _size;
    }

    /// @inheritdoc IPreimageOracle
    function loadKeccak256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external {
        uint256 size;
        bytes32 key;
        bytes32 part;
        assembly {
            // len(sig) + len(partOffset) + len(preimage offset) = 4 + 32 + 32 = 0x44
            size := calldataload(0x44)

            // revert if part offset > size+8 (i.e. parts must be within bounds)
            if gt(_partOffset, add(size, 8)) {
                // Store "PartOffsetOOB()"
                mstore(0, 0xfe254987)
                // Revert with "PartOffsetOOB()"
                revert(0x1c, 4)
            }
            // we leave solidity slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80
            // put size as big-endian uint64 at start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)
            // copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, _preimage.offset, size)
            // Note that it includes the 8-byte big-endian uint64 length prefix.
            // this will be zero-padded at the end, since memory at end is clean.
            part := mload(add(sub(ptr, 8), _partOffset))
            let h := keccak256(ptr, size) // compute preimage keccak256 hash
            // mask out prefix byte, replace with type 2 byte
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }
        preimagePartOk[key][_partOffset] = true;
        preimageParts[key][_partOffset] = part;
        preimageLengths[key] = size;
    }

    ////////////////////////////////////////////////////////////////
    //            Large Preimage Proposals (External)             //
    ////////////////////////////////////////////////////////////////

    /// @notice Challenge a keccak256 block that was committed to in the merkle tree.
    function challenge(
        address _claimant,
        uint256 _uuid,
        LibKeccak.StateMatrix memory _stateMatrix,
        Leaf calldata _prevState,
        bytes32[] calldata _prevStateProof,
        Leaf calldata _postState,
        bytes32[] calldata _postStateProof
    ) external {
        // Hash the pre and post states to get the leaf values.
        bytes32 prevStateHash = keccak256(abi.encode(_prevState));
        bytes32 postStateHash = keccak256(abi.encode(_postState));

        // Verify that both leaves are present in the merkle tree.
        bytes32 root = getTreeRoot(_claimant, _uuid);
        if (!(_verify(_prevStateProof, root, prevStateHash) && _verify(_postStateProof, root, postStateHash))) {
            revert InvalidProof();
        }

        // Verify that the prestate passed matches the intermediate state claimed in the leaf.
        if (keccak256(abi.encode(_stateMatrix)) != _prevState.stateCommitment) revert InvalidPreimage();

        // Verify that the pre/post state are contiguous.
        if (_prevState.index + 1 != _postState.index) revert StatesNotContiguous();

        // Absorb and permute the input bytes.
        LibKeccak.absorb(_stateMatrix, _postState.input);
        LibKeccak.permutation(_stateMatrix);

        // Verify that the post state hash doesn't match the expected hash.
        if (keccak256(abi.encode(_stateMatrix)) ==_postState.stateCommitment) revert PostStateMatches();

        // Mark the keccak claim as challenged.
        LPPMetaData metaData = proposalMetadata[_claimant][_uuid];
        metaData = LPPMetaData.wrap(bytes32(uint256(1)) | LPPMetaData.unwrap(metaData));
        proposalMetadata[_claimant][_uuid] = metaData;
    }

    /// @notice Challenge the first keccak256 block that was absorbed.
    function challengeFirst(
        address _claimant,
        uint256 _uuid,
        Leaf calldata _postState,
        bytes32[] calldata _postStateProof
    ) external {
        // Hash the post state to get the leaf value.
        bytes32 prevStateHash = keccak256(abi.encode(_postState));

        // Verify that the leaf is present in the merkle tree.
        bytes32 root = getTreeRoot(_claimant, _uuid);
        if (!_verify(_postStateProof, root, prevStateHash)) {
            revert InvalidProof();
        }

        // The prestate index must be 0 in order to challenge it with this function.
        if (_postState.index != 0) revert StatesNotContiguous();

        // Absorb and permute the input bytes into a fresh state matrix.
        LibKeccak.StateMatrix memory stateMatrix;
        LibKeccak.absorb(stateMatrix, _postState.input);
        LibKeccak.permutation(stateMatrix);

        // Verify that the post state hash doesn't match the expected hash.
        if (keccak256(abi.encode(stateMatrix)) == _postState.stateCommitment) revert PostStateMatches();

        // Mark the keccak claim as challenged.
        LPPMetaData metaData = proposalMetadata[_claimant][_uuid];
        metaData = LPPMetaData.wrap(bytes32(uint256(1)) | LPPMetaData.unwrap(metaData));
        proposalMetadata[_claimant][_uuid] = metaData;
    }

    /// @notice Gets the current merkle root of the large preimage proposal tree.
    function getTreeRoot(address _owner, uint256 _uuid) public view returns (bytes32 treeRoot_) {
        bytes32 node;
        uint256 blocksProcessed = proposalBlocksProcessed[_owner][_uuid];
        uint256 size = blocksProcessed;
        for (uint256 height; height < KECCAK_TREE_DEPTH; height++) {
            if ((size & 1) == 1) {
                node = keccak256(abi.encode(proposalBranches[_owner][_uuid][height], node));
            } else {
                node = keccak256(abi.encode(node, zeroHashes[height]));
            }
            size /= 2;
        }
        return keccak256(abi.encode(node, blocksProcessed));
    }

    /// @notice Adds a contiguous list of keccak state matrices to the merkle tree.
    function addLeaves(uint256 _uuid, Leaf[] calldata _leaves) external {
        // Pull storage variables onto the stack / into memory for operations.
        bytes32[KECCAK_TREE_DEPTH] memory branch_ = proposalBranches[msg.sender][_uuid];
        uint256 blocks_ = proposalBlocksProcessed[msg.sender][_uuid];

        for (uint256 i = 0; i < _leaves.length;) {
            if (_leaves[i].input.length != 136) revert InvalidInputSize();

            bytes32 node = keccak256(abi.encode(_leaves[i]));

            unchecked { ++blocks_; }

            uint256 size = blocks_;
            for (uint256 height = 0; height < KECCAK_TREE_DEPTH;) {
                if ((size & 1) == 1) {
                    branch_[height] = node;
                    break;
                }
                node = keccak256(abi.encode(branch_[height], node));
                size /= 2;

                unchecked { ++height; }
            }

            unchecked { ++i; }
        }

        // Perist the branch and number of blocks absorbed to storage.
        proposalBranches[msg.sender][_uuid] = branch_;
        proposalBlocksProcessed[msg.sender][_uuid] = blocks_;
    }

    /// @dev Returns whether `leaf` exists in the Merkle tree with `root`, given `proof`.
    function _verify(bytes32[] calldata _proof, bytes32 _root, bytes32 _leaf)
        internal
        pure
        returns (bool isValid_)
    {
        /// @solidity memory-safe-assembly
        assembly {
            if _proof.length {
                // Left shift by 5 is equivalent to multiplying by 0x20.
                let end := add(_proof.offset, shl(5, _proof.length))
                // Initialize `offset` to the offset of `_proof` in the calldata.
                let offset := _proof.offset
                // Iterate over proof elements to compute root hash.
                for {} 1 {} {
                    // Slot of `leaf` in scratch space.
                    // If the condition is true: 0x20, otherwise: 0x00.
                    let scratch := shl(5, gt(_leaf, calldataload(offset)))
                    // Store elements to hash contiguously in scratch space.
                    // Scratch space is 64 bytes (0x00 - 0x3f) and both elements are 32 bytes.
                    mstore(scratch, _leaf)
                    mstore(xor(scratch, 0x20), calldataload(offset))
                    // Reuse `leaf` to store the hash to reduce stack operations.
                    _leaf := keccak256(0x00, 0x40)
                    offset := add(offset, 0x20)
                    if iszero(lt(offset, end)) { break }
                }
            }
            isValid_ := eq(_leaf, _root)
        }
    }
}

