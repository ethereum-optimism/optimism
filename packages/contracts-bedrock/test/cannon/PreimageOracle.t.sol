// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";
import { LibKeccak } from "@lib-keccak/LibKeccak.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import "src/cannon/libraries/CannonErrors.sol";

contract PreimageOracle_Test is Test {
    PreimageOracle oracle;

    /// @notice Sets up the testing suite.
    function setUp() public {
        oracle = new PreimageOracle();
        vm.label(address(oracle), "PreimageOracle");
    }

    /// @notice Test the pre-image key computation with a known pre-image.
    function test_keccak256PreimageKey_succeeds() public {
        bytes memory preimage = hex"deadbeef";
        bytes32 key = PreimageKeyLib.keccak256PreimageKey(preimage);
        bytes32 known = 0x02fd4e189132273036449fc9e11198c739161b4c0116a9a2dccdfa1c492006f1;
        assertEq(key, known);
    }

    /// @notice Tests that context-specific data [0, 24] bytes in length can be loaded correctly.
    function test_loadLocalData_onePart_succeeds() public {
        uint256 ident = 1;
        bytes32 word = bytes32(uint256(0xdeadbeef) << 224);
        uint8 size = 4;
        uint8 partOffset = 0;

        // Load the local data into the preimage oracle under the test contract's context.
        bytes32 contextKey = oracle.loadLocalData(ident, 0, word, size, partOffset);

        // Validate that the pre-image part is set
        bool ok = oracle.preimagePartOk(contextKey, partOffset);
        assertTrue(ok);

        // Validate the local data part
        bytes32 expectedPart = 0x0000000000000004deadbeef0000000000000000000000000000000000000000;
        assertEq(oracle.preimageParts(contextKey, partOffset), expectedPart);

        // Validate the local data length
        uint256 length = oracle.preimageLengths(contextKey);
        assertEq(length, size);
    }

    /// @notice Tests that multiple local key contexts can be used by the same address for the
    ///         same local data identifier.
    function test_loadLocalData_multipleContexts_succeeds() public {
        uint256 ident = 1;
        uint8 size = 4;
        uint8 partOffset = 0;

        // Form the words we'll be storing
        bytes32[2] memory words = [bytes32(uint256(0xdeadbeef) << 224), bytes32(uint256(0xbeefbabe) << 224)];

        for (uint256 i; i < words.length; i++) {
            // Load the local data into the preimage oracle under the test contract's context
            // and the given local context.
            bytes32 contextKey = oracle.loadLocalData(ident, bytes32(i), words[i], size, partOffset);

            // Validate that the pre-image part is set
            bool ok = oracle.preimagePartOk(contextKey, partOffset);
            assertTrue(ok);

            // Validate the local data part
            bytes32 expectedPart = bytes32(uint256(words[i] >> 64) | uint256(size) << 192);
            assertEq(oracle.preimageParts(contextKey, partOffset), expectedPart);

            // Validate the local data length
            uint256 length = oracle.preimageLengths(contextKey);
            assertEq(length, size);
        }
    }

    /// @notice Tests that context-specific data [0, 32] bytes in length can be loaded correctly.
    function testFuzz_loadLocalData_varyingLength_succeeds(
        uint256 ident,
        bytes32 localContext,
        bytes32 word,
        uint256 size,
        uint256 partOffset
    )
        public
    {
        // Bound the size to [0, 32]
        size = bound(size, 0, 32);
        // Bound the part offset to [0, size + 8]
        partOffset = bound(partOffset, 0, size + 8);

        // Load the local data into the preimage oracle under the test contract's context.
        bytes32 contextKey = oracle.loadLocalData(ident, localContext, word, uint8(size), uint8(partOffset));

        // Validate that the first local data part is set
        bool ok = oracle.preimagePartOk(contextKey, partOffset);
        assertTrue(ok);
        // Validate the first local data part
        bytes32 expectedPart;
        assembly {
            mstore(0x20, 0x00)

            mstore(0x00, shl(192, size))
            mstore(0x08, word)

            expectedPart := mload(partOffset)
        }
        assertEq(oracle.preimageParts(contextKey, partOffset), expectedPart);

        // Validate the local data length
        uint256 length = oracle.preimageLengths(contextKey);
        assertEq(length, size);
    }

    /// @notice Tests that a pre-image is correctly set.
    function test_loadKeccak256PreimagePart_succeeds() public {
        // Set the pre-image
        bytes memory preimage = hex"deadbeef";
        bytes32 key = PreimageKeyLib.keccak256PreimageKey(preimage);
        uint256 offset = 0;
        oracle.loadKeccak256PreimagePart(offset, preimage);

        // Validate the pre-image part
        bytes32 part = oracle.preimageParts(key, offset);
        bytes32 expectedPart = 0x0000000000000004deadbeef0000000000000000000000000000000000000000;
        assertEq(part, expectedPart);

        // Validate the pre-image length
        uint256 length = oracle.preimageLengths(key);
        assertEq(length, preimage.length);

        // Validate that the pre-image part is set
        bool ok = oracle.preimagePartOk(key, offset);
        assertTrue(ok);
    }

    /// @notice Tests that a pre-image cannot be set with an out-of-bounds offset.
    function test_loadLocalData_outOfBoundsOffset_reverts() public {
        bytes32 preimage = bytes32(uint256(0xdeadbeef));
        uint256 offset = preimage.length + 9;

        vm.expectRevert(PartOffsetOOB.selector);
        oracle.loadLocalData(1, 0, preimage, 32, offset);
    }

    /// @notice Tests that a pre-image cannot be set with an out-of-bounds offset.
    function test_loadKeccak256PreimagePart_outOfBoundsOffset_reverts() public {
        bytes memory preimage = hex"deadbeef";
        uint256 offset = preimage.length + 9;

        vm.expectRevert(PartOffsetOOB.selector);
        oracle.loadKeccak256PreimagePart(offset, preimage);
    }

    /// @notice Reading a pre-image part that has not been set should revert.
    function testFuzz_readPreimage_missingPreimage_reverts(bytes32 key, uint256 offset) public {
        vm.expectRevert("pre-image must exist");
        oracle.readPreimage(key, offset);
    }
}

contract KeccakDispute_LargePreimageProposals_Test is Test {
    uint256 internal constant TEST_UUID = 0xFACADE;

    PreimageOracle internal oracle;

    /// @notice Sets up the testing suite.
    function setUp() public {
        oracle = new PreimageOracle();
        vm.label(address(oracle), "PreimageOracle");
    }

    /// @notice Tests that leaves can be added the large preimage proposal mapping.
    function testLoadZ() public {
        bytes memory data = new bytes(136);

        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeaves(TEST_UUID, data, stateCommitments, true);

        // for (uint i = 0; i < stateCommitments.length; i++) {
        //     console.logBytes32(stateCommitments[i]);
        // }

        // We should have processed 1 block + the padding block.
        assertEq(oracle.proposalBlocksProcessed(address(this), TEST_UUID), 2);

        bytes32[15] memory proof = oracle.getProposalBranch(address(this), TEST_UUID);
        bytes32[] memory p;
        assembly {
            p := mload(0x40)
            let destPtr := add(p, 0x20)
            for { let i := 0 } lt(i, 15) { i := add(i, 1) } {
                mstore(add(destPtr, shl(0x05, i)), mload(add(proof, shl(0x05, i))))
            }
            mstore(p, 15)
            mstore(0x40, add(p, 0x200))
        }
        assertEq(keccak256(abi.encodePacked(p)), keccak256(abi.encodePacked(proof)));

        PreimageOracle.Leaf memory leaf = PreimageOracle.Leaf({
            input: Bytes.slice(LibKeccak.padMemory(data), LibKeccak.BLOCK_SIZE_BYTES, LibKeccak.BLOCK_SIZE_BYTES),
            index: 1,
            stateCommitment: stateCommitments[1]
        });

        oracle._verify(p, oracle.getTreeRoot(address(this), TEST_UUID), 1, _hashLeaf(leaf));
    }

    /// @notice Hashes leaf data for the preimage proposals tree
    function _hashLeaf(PreimageOracle.Leaf memory _leaf) internal pure returns (bytes32 leaf_) {
        leaf_ = keccak256(abi.encodePacked(_leaf.input, _leaf.index, _leaf.stateCommitment));
    }

    /// @notice Helper to construct
    function _generateStateCommitments(
        LibKeccak.StateMatrix memory _stateMatrix,
        bytes memory _data
    )
        internal
        pure
        returns (bytes32[] memory stateCommitments_)
    {
        bytes memory data = LibKeccak.padMemory(_data);
        uint256 numCommitments = data.length / LibKeccak.BLOCK_SIZE_BYTES;

        stateCommitments_ = new bytes32[](numCommitments);
        for (uint256 i = 0; i < numCommitments; i++) {
            bytes memory blockSlice = Bytes.slice(data, i * LibKeccak.BLOCK_SIZE_BYTES, LibKeccak.BLOCK_SIZE_BYTES);
            LibKeccak.absorb(_stateMatrix, blockSlice);
            LibKeccak.permutation(_stateMatrix);

            stateCommitments_[i] = keccak256(abi.encode(_stateMatrix));
        }
    }
}
