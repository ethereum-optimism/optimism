// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, console2 as console } from "forge-std/Test.sol";

import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";
import { LibKeccak } from "@lib-keccak/LibKeccak.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import "src/cannon/libraries/CannonErrors.sol";
import "src/cannon/libraries/CannonTypes.sol";

contract PreimageOracle_Test is Test {
    PreimageOracle oracle;

    /// @notice Sets up the testing suite.
    function setUp() public {
        oracle = new PreimageOracle(0, 0);
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

    /// @notice Tests that adding a global keccak256 pre-image at the part boundary reverts.
    function test_loadKeccak256PreimagePart_partBoundary_reverts() public {
        bytes memory preimage = hex"deadbeef";
        uint256 offset = preimage.length + 8;

        vm.expectRevert(PartOffsetOOB.selector);
        oracle.loadKeccak256PreimagePart(offset, preimage);
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

contract PreimageOracle_LargePreimageProposals_Test is Test {
    uint256 internal constant MIN_SIZE_BYTES = 0;
    uint256 internal constant CHALLENGE_PERIOD = 1 days;
    uint256 internal constant TEST_UUID = 0xFACADE;

    PreimageOracle internal oracle;

    /// @notice Sets up the testing suite.
    function setUp() public {
        oracle = new PreimageOracle({ _minProposalSize: MIN_SIZE_BYTES, _challengePeriod: CHALLENGE_PERIOD });
        vm.label(address(oracle), "PreimageOracle");

        // Set `tx.origin` and `msg.sender` to `address(this)` so that it may behave like an EOA for `addLeavesLPP`.
        vm.startPrank(address(this), address(this));
    }

    /// @notice Tests that the `initLPP` function reverts when the part offset is out of bounds of the full preimage.
    function test_initLPP_partOffsetOOB_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        vm.expectRevert(PartOffsetOOB.selector);
        oracle.initLPP(TEST_UUID, 136 + 8, uint32(data.length));
    }

    /// @notice Tests that the `initLPP` function reverts when the part offset is out of bounds of the full preimage.
    function test_initLPP_sizeTooSmall_reverts() public {
        oracle = new PreimageOracle({ _minProposalSize: 1000, _challengePeriod: CHALLENGE_PERIOD });

        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        vm.expectRevert(InvalidInputSize.selector);
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));
    }

    /// @notice Gas snapshot for `addLeaves`
    function test_addLeaves_gasSnapshot() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 500);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);

        // Allocate the calldata so it isn't included in the gas measurement.
        bytes memory cd = abi.encodeCall(oracle.addLeavesLPP, (TEST_UUID, 0, data, stateCommitments, true));

        uint256 gas = gasleft();
        (bool success,) = address(oracle).call(cd);
        uint256 gasUsed = gas - gasleft();
        assertTrue(success);

        console.log("Gas used: %d", gasUsed);
        console.log("Gas per byte (%d bytes streamed): %d", data.length, gasUsed / data.length);
        console.log("Gas for 4MB: %d", (gasUsed / data.length) * 4000000);
    }

    /// @notice Tests that the `addLeavesLPP` function may never be called when `tx.origin != msg.sender`
    function test_addLeaves_notEOA_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 500);

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);

        // Replace the global prank, set `tx.origin` to `address(0)`, and set `msg.sender` to `address(this)`.
        vm.stopPrank();
        vm.prank(address(0), address(this));

        vm.expectRevert(NotEOA.selector);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);
    }

    /// @notice Tests that the `addLeavesLPP` function reverts when the starting block index is not what is expected.
    function test_addLeaves_notContiguous_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 500);

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);

        vm.expectRevert(WrongStartingBlock.selector);
        oracle.addLeavesLPP(TEST_UUID, 1, data, stateCommitments, true);
    }

    /// @notice Tests that leaves can be added the large preimage proposal mapping and proven to be contained within
    ///         the computed merkle root.
    function test_addLeaves_multipleParts_succeeds() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 3);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));
        // Ensure that the proposal keys are present in the array.
        (address claimant, uint256 uuid) = oracle.proposals(0);
        assertEq(oracle.proposalCount(), 1);
        assertEq(claimant, address(this));
        assertEq(uuid, TEST_UUID);

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);

        uint256 midPoint = stateCommitments.length / 2;
        bytes32[] memory commitmentsA = new bytes32[](midPoint);
        bytes32[] memory commitmentsB = new bytes32[](midPoint);
        for (uint256 i = 0; i < midPoint; i++) {
            commitmentsA[i] = stateCommitments[i];
            commitmentsB[i] = stateCommitments[i + midPoint];
        }

        oracle.addLeavesLPP(TEST_UUID, 0, Bytes.slice(data, 0, 136 * 2), commitmentsA, false);

        // MetaData assertions
        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertEq(metaData.timestamp(), 0);
        assertEq(metaData.partOffset(), 0);
        assertEq(metaData.claimedSize(), data.length);
        assertEq(metaData.blocksProcessed(), 2);
        assertEq(metaData.bytesProcessed(), 136 * 2);
        assertFalse(metaData.countered());

        // Move ahead one block.
        vm.roll(block.number + 1);

        oracle.addLeavesLPP(TEST_UUID, 2, Bytes.slice(data, 136 * 2, 136), commitmentsB, true);

        // MetaData assertions
        metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertEq(metaData.timestamp(), 1);
        assertEq(metaData.partOffset(), 0);
        assertEq(metaData.claimedSize(), data.length);
        assertEq(metaData.blocksProcessed(), 4);
        assertEq(metaData.bytesProcessed(), data.length);
        assertFalse(metaData.countered());

        // Preimage part assertions
        bytes32 expectedPart = bytes32((~uint256(0) & ~(uint256(type(uint64).max) << 192)) | (data.length << 192));
        assertEq(oracle.proposalParts(address(this), TEST_UUID), expectedPart);

        assertEq(oracle.proposalBlocks(address(this), TEST_UUID, 0), block.number - 1);
        assertEq(oracle.proposalBlocks(address(this), TEST_UUID, 1), block.number);

        // Should revert if we try to add new leaves.
        vm.expectRevert(AlreadyFinalized.selector);
        oracle.addLeavesLPP(TEST_UUID, 4, data, stateCommitments, true);
    }

    /// @notice Tests that leaves cannot be added until the large preimage proposal has been initialized.
    function test_addLeaves_notInitialized_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 500);

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);

        // Allocate the calldata so it isn't included in the gas measurement.
        vm.expectRevert(NotInitialized.selector);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);
    }

    /// @notice Tests that leaves can be added the large preimage proposal mapping and finalized to be added to the
    ///         authorized mappings.
    function test_squeeze_challengePeriodPassed_succeeds() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        vm.warp(block.timestamp + oracle.challengePeriod() + 1 seconds);

        // Finalize the proposal.
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 1),
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });

        bytes32 finalDigest = keccak256(data);
        bytes32 expectedPart = bytes32((~uint256(0) & ~(uint256(type(uint64).max) << 192)) | (data.length << 192));
        assertTrue(oracle.preimagePartOk(finalDigest, 0));
        assertEq(oracle.preimageLengths(finalDigest), data.length);
        assertEq(oracle.preimageParts(finalDigest, 0), expectedPart);
    }

    /// @notice Tests that a proposal cannot be finalized until it has passed the challenge period.
    function test_squeeze_proposalChallenged_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136);

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        // Should succeed since the commitment was wrong.
        oracle.challengeFirstLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _postState: leaves[0],
            _postStateProof: preProof
        });

        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertTrue(metaData.countered());

        vm.warp(block.timestamp + oracle.challengePeriod() + 1 seconds);

        // Finalize the proposal.
        vm.expectRevert(BadProposal.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 1),
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });
    }

    /// @notice Tests that a proposal cannot be finalized until it has passed the challenge period.
    function test_squeeze_challengePeriodActive_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Finalize the proposal.
        vm.expectRevert(ActiveProposal.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 1),
            _preState: leaves[0],
            _preStateProof: new bytes32[](16),
            _postState: leaves[1],
            _postStateProof: new bytes32[](16)
        });
    }

    /// @notice Tests that a proposal cannot be finalized until it has passed the challenge period.
    function test_squeeze_incompleteAbsorbtion_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Finalize the proposal.
        vm.expectRevert(ActiveProposal.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 1),
            _preState: leaves[0],
            _preStateProof: new bytes32[](16),
            _postState: leaves[1],
            _postStateProof: new bytes32[](16)
        });
    }

    /// @notice Tests that the `squeeze` function reverts when the passed states are not contiguous.
    function test_squeeze_statesNotContiguous_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        vm.warp(block.timestamp + oracle.challengePeriod() + 1 seconds);

        // Finalize the proposal.
        vm.expectRevert(StatesNotContiguous.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 2),
            _preState: leaves[1],
            _preStateProof: postProof,
            _postState: leaves[0],
            _postStateProof: preProof
        });
    }

    /// @notice Tests that the `squeeze` function reverts when the post state passed
    function test_squeeze_invalidPreimage_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        vm.warp(block.timestamp + oracle.challengePeriod() + 1 seconds);

        // Finalize the proposal.
        vm.expectRevert(InvalidPreimage.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 2),
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });
    }

    /// @notice Tests that the `squeeze` function reverts when the claimed size is not equal to the bytes processed.
    function test_squeeze_invalidClaimedSize_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length) - 1);

        // Add the leaves to the tree (2 keccak blocks.)
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        vm.warp(block.timestamp + oracle.challengePeriod() + 1 seconds);

        vm.expectRevert(InvalidInputSize.selector);
        oracle.squeezeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: _stateMatrixAtBlockIndex(data, 1),
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });
    }

    /// @notice Tests that a valid leaf cannot be countered with the `challengeFirst` function.
    function test_challengeFirst_validCommitment_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory p = new bytes32[](16);
        p[0] = _hashLeaf(leaves[1]);
        for (uint256 i = 1; i < p.length; i++) {
            p[i] = oracle.zeroHashes(i);
        }

        vm.expectRevert(PostStateMatches.selector);
        oracle.challengeFirstLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _postState: leaves[0],
            _postStateProof: p
        });

        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertFalse(metaData.countered());
    }

    /// @notice Tests that an invalid leaf cannot be countered with `challengeFirst` if it is not the first leaf.
    function test_challengeFirst_statesNotContiguous_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136);

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];

        // Create a proof array with 16 elements.
        bytes32[] memory p = new bytes32[](16);
        p[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < p.length; i++) {
            p[i] = oracle.zeroHashes(i);
        }

        // Should succeed since the commitment was wrong.
        vm.expectRevert(StatesNotContiguous.selector);
        oracle.challengeFirstLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _postState: leaves[1],
            _postStateProof: p
        });
    }

    /// @notice Tests that an invalid leaf can be countered with the `challengeFirst` function.
    function test_challengeFirst_invalidCommitment_succeeds() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136);

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];

        // Create a proof array with 16 elements.
        bytes32[] memory p = new bytes32[](16);
        p[0] = _hashLeaf(leaves[1]);
        for (uint256 i = 1; i < p.length; i++) {
            p[i] = oracle.zeroHashes(i);
        }

        // Should succeed since the commitment was wrong.
        oracle.challengeFirstLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _postState: leaves[0],
            _postStateProof: p
        });

        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertTrue(metaData.countered());
    }

    /// @notice Tests that a valid leaf cannot be countered with the `challenge` function in the middle of the tree.
    function test_challenge_validCommitment_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, data, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, data);

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        for (uint256 i = 1; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        LibKeccak.StateMatrix memory preMatrix = _stateMatrixAtBlockIndex(data, 1);

        vm.expectRevert(PostStateMatches.selector);
        oracle.challengeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: preMatrix,
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });

        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertFalse(metaData.countered());
    }

    /// @notice Tests that an invalid leaf can not be countered with non-contiguous states.
    function test_challenge_statesNotContiguous_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 2);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136 * 2);
        for (uint256 i = 0; i < phonyData.length / 2; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];
        leaves[2].stateCommitment = stateCommitments[2];

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        preProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        postProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        for (uint256 i = 2; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        LibKeccak.StateMatrix memory preMatrix = _stateMatrixAtBlockIndex(data, 2);

        vm.expectRevert(StatesNotContiguous.selector);
        oracle.challengeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: preMatrix,
            _preState: leaves[1],
            _preStateProof: postProof,
            _postState: leaves[0],
            _postStateProof: preProof
        });
    }

    /// @notice Tests that an invalid leaf can not be countered with an incorrect prestate matrix reveal.
    function test_challenge_invalidPreimage_reverts() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 2);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136 * 2);
        for (uint256 i = 0; i < phonyData.length / 2; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];
        leaves[2].stateCommitment = stateCommitments[2];

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        preProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        postProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        for (uint256 i = 2; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        LibKeccak.StateMatrix memory preMatrix = _stateMatrixAtBlockIndex(data, 2);

        vm.expectRevert(InvalidPreimage.selector);
        oracle.challengeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: preMatrix,
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });
    }

    /// @notice Tests that an invalid leaf can be countered with the `challenge` function in the middle of the tree.
    function test_challenge_invalidCommitment_succeeds() public {
        // Allocate the preimage data.
        bytes memory data = new bytes(136 * 2);
        for (uint256 i; i < data.length; i++) {
            data[i] = 0xFF;
        }
        bytes memory phonyData = new bytes(136 * 2);
        for (uint256 i = 0; i < phonyData.length / 2; i++) {
            data[i] = 0xFF;
        }

        // Initialize the proposal.
        oracle.initLPP(TEST_UUID, 0, uint32(data.length));

        // Add the leaves to the tree with mismatching state commitments.
        LibKeccak.StateMatrix memory stateMatrix;
        bytes32[] memory stateCommitments = _generateStateCommitments(stateMatrix, data);
        oracle.addLeavesLPP(TEST_UUID, 0, phonyData, stateCommitments, true);

        // Construct the leaf preimage data for the blocks added.
        LibKeccak.StateMatrix memory matrix;
        PreimageOracle.Leaf[] memory leaves = _generateLeaves(matrix, phonyData);
        leaves[0].stateCommitment = stateCommitments[0];
        leaves[1].stateCommitment = stateCommitments[1];
        leaves[2].stateCommitment = stateCommitments[2];

        // Create a proof array with 16 elements.
        bytes32[] memory preProof = new bytes32[](16);
        preProof[0] = _hashLeaf(leaves[1]);
        preProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        bytes32[] memory postProof = new bytes32[](16);
        postProof[0] = _hashLeaf(leaves[0]);
        postProof[1] = keccak256(abi.encode(_hashLeaf(leaves[2]), bytes32(0)));
        for (uint256 i = 2; i < preProof.length; i++) {
            bytes32 zeroHash = oracle.zeroHashes(i);
            preProof[i] = zeroHash;
            postProof[i] = zeroHash;
        }

        LibKeccak.StateMatrix memory preMatrix = _stateMatrixAtBlockIndex(data, 1);

        oracle.challengeLPP({
            _claimant: address(this),
            _uuid: TEST_UUID,
            _stateMatrix: preMatrix,
            _preState: leaves[0],
            _preStateProof: preProof,
            _postState: leaves[1],
            _postStateProof: postProof
        });

        LPPMetaData metaData = oracle.proposalMetadata(address(this), TEST_UUID);
        assertTrue(metaData.countered());
    }

    /// @notice Hashes leaf data for the preimage proposals tree
    function _hashLeaf(PreimageOracle.Leaf memory _leaf) internal pure returns (bytes32 leaf_) {
        leaf_ = keccak256(abi.encodePacked(_leaf.input, _leaf.index, _leaf.stateCommitment));
    }

    /// @notice Helper to construct the keccak merkle tree's leaves from a given input `_data`.
    function _generateLeaves(
        LibKeccak.StateMatrix memory _stateMatrix,
        bytes memory _data
    )
        internal
        pure
        returns (PreimageOracle.Leaf[] memory leaves_)
    {
        bytes memory data = LibKeccak.padMemory(_data);
        uint256 numLeaves = data.length / LibKeccak.BLOCK_SIZE_BYTES;

        leaves_ = new PreimageOracle.Leaf[](numLeaves);
        for (uint256 i = 0; i < numLeaves; i++) {
            bytes memory blockSlice = Bytes.slice(data, i * LibKeccak.BLOCK_SIZE_BYTES, LibKeccak.BLOCK_SIZE_BYTES);
            LibKeccak.absorb(_stateMatrix, blockSlice);
            LibKeccak.permutation(_stateMatrix);
            bytes32 stateCommitment = keccak256(abi.encode(_stateMatrix));

            leaves_[i] = PreimageOracle.Leaf({ input: blockSlice, index: i, stateCommitment: stateCommitment });
        }
    }

    /// @notice Helper to get the keccak state matrix before applying the block at `_blockIndex` within `_data`.
    function _stateMatrixAtBlockIndex(
        bytes memory _data,
        uint256 _blockIndex
    )
        internal
        pure
        returns (LibKeccak.StateMatrix memory matrix_)
    {
        bytes memory data = LibKeccak.padMemory(_data);

        for (uint256 i = 0; i < _blockIndex; i++) {
            bytes memory blockSlice = Bytes.slice(data, i * LibKeccak.BLOCK_SIZE_BYTES, LibKeccak.BLOCK_SIZE_BYTES);
            LibKeccak.absorb(matrix_, blockSlice);
            LibKeccak.permutation(matrix_);
        }
    }

    /// @notice Helper to construct the keccak state commitments for each block processed in the input `_data`.
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
