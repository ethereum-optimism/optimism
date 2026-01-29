// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { MIPS64Memory } from "src/cannon/libraries/MIPS64Memory.sol";
import { InvalidMemoryProof } from "src/cannon/libraries/CannonErrors.sol";

contract MIPS64MemoryWithCalldata {
    function readMem(
        bytes32 _root,
        uint64 _addr,
        uint8 _proofIndex,
        bytes calldata /* _proof */
    )
        external
        pure
        returns (uint64 out_)
    {
        uint256 proofDataOffset = 4 + 32 + 32 + 32 + 32 + 32;
        uint256 proofOffset = MIPS64Memory.memoryProofOffset(proofDataOffset, _proofIndex);
        return MIPS64Memory.readMem(_root, _addr, proofOffset);
    }

    function writeMem(
        uint64 _addr,
        uint64 _value,
        uint8 _proofIndex,
        bytes calldata /* _proof */
    )
        external
        pure
        returns (bytes32 root_)
    {
        uint256 proofDataOffset = 4 + 32 + 32 + 32 + 32 + 32;
        uint256 proofOffset = MIPS64Memory.memoryProofOffset(proofDataOffset, _proofIndex);
        return MIPS64Memory.writeMem(_addr, proofOffset, _value);
    }

    function isValidProof(
        bytes32 _root,
        uint64 _addr,
        uint8 _proofIndex,
        bytes calldata /* _proof */
    )
        external
        pure
        returns (bool valid_)
    {
        uint256 proofDataOffset = 4 + 32 + 32 + 32 + 32 + 32;
        uint256 proofOffset = MIPS64Memory.memoryProofOffset(proofDataOffset, _proofIndex);
        return MIPS64Memory.isValidProof(_root, _addr, proofOffset);
    }

    function memoryProofOffset(uint256 _proofDataOffset, uint8 _proofIndex) external pure returns (uint256 offset_) {
        return MIPS64Memory.memoryProofOffset(_proofDataOffset, _proofIndex);
    }
}

/// @title MIPS64Memory_TestInit
/// @notice Reusable test initialization for `MIPS64Memory` tests.
abstract contract MIPS64Memory_TestInit is CommonTest {
    MIPS64MemoryWithCalldata mem;

    error InvalidAddress();

    function setUp() public virtual override {
        super.setUp();
        mem = new MIPS64MemoryWithCalldata();
    }
}

/// @title MIPS64Memory_ReadMem_Test
/// @notice Tests the `readMem` function of the `MIPS64Memory` contract.
contract MIPS64Memory_ReadMem_Test is MIPS64Memory_TestInit {
    /// @notice Static unit test for basic memory access
    function test_readMem_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        uint64 readWord = mem.readMem(root, addr, 0, proof);
        assertEq(readWord, word);
    }

    /// @notice Static unit test asserting that reading from the zero address succeeds.
    function test_readMemAtZero_succeeds() external {
        uint64 addr = 0x0;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        uint64 readWord = mem.readMem(root, addr, 0, proof);
        assertEq(readWord, word);
    }

    /// @notice Static unit test asserting that reading from high memory area succeeds.
    function test_readMemHighMem_succeeds() external {
        uint64 addr = 0xFF_FF_FF_FF_00_00_00_88;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        uint64 readWord = mem.readMem(root, addr, 0, proof);
        assertEq(readWord, word);
    }

    /// @notice Static unit test asserting that reads revert when a misaligned memory address is
    ///         provided.
    function test_readMem_readInvalidAddress_reverts() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        vm.expectRevert(InvalidAddress.selector);
        mem.readMem(root, addr + 4, 0, proof);
    }

    /// @notice Static unit test asserting that reads revert when an invalid proof is provided.
    function test_readMem_readInvalidProof_reverts() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        vm.assertTrue(proof[64] != 0x0); // make sure the proof is tampered
        proof[64] = 0x00;
        vm.expectRevert(InvalidMemoryProof.selector);
        mem.readMem(root, addr, 0, proof);
    }

    /// @notice Static unit test asserting that reads from a non-zero proof index succeeds.
    function test_readMem_nonZeroProofIndex_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        uint64 addr2 = 0xFF_FF_FF_FF_00_00_00_88;
        uint64 word2 = 0xF1_F2_F3_F4_F5_F6_F7_F8;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word, addr2, word2);

        uint64 readWord = mem.readMem(root, addr, 0, proof);
        assertEq(readWord, word);

        readWord = mem.readMem(root, addr2, 1, proof);
        assertEq(readWord, word2);
    }
}

/// @title MIPS64Memory_WriteMem_Test
/// @notice Tests the `writeMem` function of the `MIPS64Memory` contract.
contract MIPS64Memory_WriteMem_Test is MIPS64Memory_TestInit {
    /// @notice Static unit test asserting basic memory write functionality.
    function test_writeMem_succeeds() external {
        uint64 addr = 0x100;
        bytes memory zeroProof;
        (, zeroProof) = ffi.getCannonMemory64Proof(addr, 0);

        uint64 word = 0x11_22_33_44_55_66_77_88;
        (bytes32 expectedRoot,) = ffi.getCannonMemory64Proof(addr, word);

        bytes32 newRoot = mem.writeMem(addr, word, 0, zeroProof);
        assertEq(newRoot, expectedRoot);
    }

    /// @notice Static unit test asserting that writes to high memory succeeds.
    function test_writeMem_highMem_succeeds() external {
        uint64 addr = 0xFF_FF_FF_FF_00_00_00_88;
        bytes memory zeroProof;
        (, zeroProof) = ffi.getCannonMemory64Proof(addr, 0);

        uint64 word = 0x11_22_33_44_55_66_77_88;
        (bytes32 expectedRoot,) = ffi.getCannonMemory64Proof(addr, word);

        bytes32 newRoot = mem.writeMem(addr, word, 0, zeroProof);
        assertEq(newRoot, expectedRoot);
    }

    /// @notice Static unit test asserting that non-zero memory word is overwritten.
    function test_writeMem_nonZeroProofOffset_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        uint64 addr2 = 0x108;
        uint64 word2 = 0x55_55_55_55_77_77_77_77;
        bytes memory initProof;
        (, initProof) = ffi.getCannonMemory64Proof(addr, word, addr2, word2);

        uint64 word3 = 0x44_44_44_44_44_44_44_44;
        (bytes32 expectedRoot,) = ffi.getCannonMemory64Proof(addr, word, addr2, word2, addr2, word3);

        bytes32 newRoot = mem.writeMem(addr2, word3, 1, initProof);
        assertEq(newRoot, expectedRoot);
    }

    /// @notice Static unit test asserting that a zerod memory word is set for a non-zero memory
    ///         proof.
    function test_writeMem_uniqueAccess_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        uint64 addr2 = 0x108;
        uint64 word2 = 0x55_55_55_55_77_77_77_77;
        bytes memory initProof;
        (, initProof) = ffi.getCannonMemory64Proof(addr, word, addr2, word2);

        uint64 addr3 = 0xAA_AA_AA_AA_00;
        uint64 word3 = 0x44_44_44_44_44_44_44_44;
        (, bytes memory addr3Proof) = ffi.getCannonMemory64Proof2(addr, word, addr2, word2, addr3);
        (bytes32 expectedRoot,) = ffi.getCannonMemory64Proof(addr, word, addr2, word2, addr3, word3);

        bytes32 newRoot = mem.writeMem(addr3, word3, 0, addr3Proof);
        assertEq(newRoot, expectedRoot);

        newRoot = mem.writeMem(addr3 + 8, word3, 0, addr3Proof);
        assertNotEq(newRoot, expectedRoot);

        newRoot = mem.writeMem(addr3, word3 + 1, 0, addr3Proof);
        assertNotEq(newRoot, expectedRoot);
    }

    /// @notice Static unit test asserting that writes succeeds in overwriting a non-zero memory
    ///         word.
    function test_writeMem_nonZeroMem_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes memory initProof;
        (, initProof) = ffi.getCannonMemory64Proof(addr, word);

        uint64 word2 = 0x55_55_55_55_77_77_77_77;
        (bytes32 expectedRoot,) = ffi.getCannonMemory64Proof(addr, word, addr + 8, word2);

        bytes32 newRoot = mem.writeMem(addr + 8, word2, 0, initProof);
        assertEq(newRoot, expectedRoot);
    }

    /// @notice Static unit test asserting that writes revert when a misaligned memory address is
    ///         provided.
    function test_writeMem_writeMemInvalidAddress_reverts() external {
        bytes memory zeroProof;
        (, zeroProof) = ffi.getCannonMemory64Proof(0x100, 0);
        vm.expectRevert(InvalidAddress.selector);
        mem.writeMem(0x104, 0x0, 0, zeroProof);
    }
}

/// @title MIPS64Memory_IsValidProof_Test
/// @notice Tests the `isValidProof` function of the `MIPS64Memory` contract.
contract MIPS64Memory_IsValidProof_Test is MIPS64Memory_TestInit {
    /// @notice Static unit test asserting that a valid proof returns true.
    function test_isValidProof_validProof_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        bool valid = mem.isValidProof(root, addr, 0, proof);
        assertTrue(valid);
    }

    /// @notice Static unit test asserting that an invalid proof returns false.
    function test_isValidProof_invalidProof_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        vm.assertTrue(proof[64] != 0x0);
        proof[64] = 0x00;
        bool valid = mem.isValidProof(root, addr, 0, proof);
        assertFalse(valid);
    }

    /// @notice Static unit test asserting that a mismatched root returns false.
    function test_isValidProof_mismatchedRoot_succeeds() external {
        uint64 addr = 0x100;
        uint64 word = 0x11_22_33_44_55_66_77_88;
        bytes32 root;
        bytes memory proof;
        (root, proof) = ffi.getCannonMemory64Proof(addr, word);
        bytes32 wrongRoot = bytes32(uint256(root) ^ 1);
        bool valid = mem.isValidProof(wrongRoot, addr, 0, proof);
        assertFalse(valid);
    }
}

/// @title MIPS64Memory_MemoryProofOffset_Test
/// @notice Tests the `memoryProofOffset` function of the `MIPS64Memory` contract.
contract MIPS64Memory_MemoryProofOffset_Test is MIPS64Memory_TestInit {
    uint64 internal constant MEM_PROOF_LEAF_COUNT = 60;

    /// @notice Fuzz test asserting that memoryProofOffset computes the correct offset.
    function testFuzz_memoryProofOffset_succeeds(uint256 _proofDataOffset, uint8 _proofIndex) external view {
        _proofDataOffset = bound(_proofDataOffset, 0, type(uint128).max);
        uint256 expectedOffset = _proofDataOffset + (uint256(_proofIndex) * (MEM_PROOF_LEAF_COUNT * 32));
        uint256 actualOffset = mem.memoryProofOffset(_proofDataOffset, _proofIndex);
        assertEq(actualOffset, expectedOffset);
    }

    /// @notice Static unit test asserting that memoryProofOffset returns the base offset for index 0.
    function test_memoryProofOffset_zeroIndex_succeeds() external view {
        uint256 proofDataOffset = 164;
        uint256 offset = mem.memoryProofOffset(proofDataOffset, 0);
        assertEq(offset, proofDataOffset);
    }

    /// @notice Static unit test asserting that memoryProofOffset correctly increments for index 1.
    function test_memoryProofOffset_indexOne_succeeds() external view {
        uint256 proofDataOffset = 164;
        uint256 offset = mem.memoryProofOffset(proofDataOffset, 1);
        assertEq(offset, proofDataOffset + (MEM_PROOF_LEAF_COUNT * 32));
    }
}
