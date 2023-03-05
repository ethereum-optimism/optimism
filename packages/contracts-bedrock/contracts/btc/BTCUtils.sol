// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @author philogy <https://github.com/philogy>
library BTCUtils {
    error NotSegwitTx();
    error InsufficientTxInputs();
    error InsufficientWitnessElements();
    error UnrecognizedInscriptionFormat();

    function isValidMerkle(bytes32 _root, bytes32 _baseEl, uint256 _index, bytes32[] calldata _proofEls)
        internal
        view
        returns (bool isValid)
    {
        assembly {
            let proofSize := _proofEls.length
            let proofOffset := _proofEls.offset

            let proofEnd := add(proofOffset, shl(5, proofSize))
            for {} lt(proofOffset, proofEnd) {} {
                // Use Solady trick to efficiently order elements in memory
                let scratch := shl(5, and(_index, 1))
                mstore(scratch, _baseEl)
                mstore(xor(0x20, scratch), calldataload(proofOffset))
                pop(staticcall(gas(), 0x02, 0x00, 0x40, 0x00, 0x20))
                pop(staticcall(gas(), 0x02, 0x00, 0x20, 0x00, 0x20))
                _baseEl := mload(0x00)
                _index := shr(1, _index)
                proofOffset := add(proofOffset, 0x20)
            }
            isValid := eq(_root, _baseEl)
        }
    }

    function getInscription(bytes calldata _witnessTx, uint256 _inIndex)
        internal
        pure
        returns (bytes memory inscription)
    {
        assembly {
            function getVarint(ptr) -> newPtr, length {
                let maxData := and(calldataload(add(ptr, 0x8)), 0xffffffffffffffffff)
                let varLengthByte := shr(0x40, maxData)
                newPtr := add(ptr, 1)
                switch lt(varLengthByte, 0xfd)
                case 1 { length := varLengthByte }
                default {
                    // 0xfd -> 8 * 6
                    // 0xfe -> 8 * 4
                    // 0xff -> 8 * 0
                    // 8 * (8 - 2^(vlb - 0xfc)) = 64 - 8 * 2^(vlb - 0xfc)
                    let size := shl(sub(varLengthByte, 0xfc), 1)
                    newPtr := add(newPtr, size)
                    let data := and(maxData, 0xffffffffffffffff)
                    length := shl(sub(0x40, shl(3, size)), data)
                }
            }

            function varintSkip(ptr) -> newPtr {
                let maxData := and(calldataload(add(ptr, 0x8)), 0xffffffffffffffffff)
                let varLengthByte := shr(0x40, maxData)
                newPtr := add(ptr, 1)
                switch lt(varLengthByte, 0xfd)
                case 1 { newPtr := add(newPtr, varLengthByte) }
                default {
                    // 0xfd -> 8 * 6
                    // 0xfe -> 8 * 4
                    // 0xff -> 8 * 0
                    // 8 * (8 - 2^(vlb - 0xfc)) = 64 - 8 * 2^(vlb - 0xfc)
                    let size := shl(sub(varLengthByte, 0xfc), 1)
                    newPtr := add(newPtr, size)
                    let data := and(maxData, 0xffffffffffffffff)
                    newPtr := add(shl(sub(0x40, shl(3, size)), data), newPtr)
                }
            }

            // Start pointer 4 bytes after beginning, skipping version bytes
            let ptr := sub(_witnessTx.offset, 0x1a)

            // Check mark and flag bytes
            if iszero(eq(and(calldataload(ptr), 0xffff), 0x0001)) {
                mstore(0x00, 0xf39be9d7)
                revert(0x1c, 0x04)
            }
            ptr := add(ptr, 1)

            // Read and jump over tx inputs
            let txinCount
            ptr, txinCount := getVarint(ptr)
            if iszero(lt(_inIndex, txinCount)) {
                mstore(0x00, 0x03db5ed9)
                revert(0x1c, 0x04)
            }
            for { let i := 0 } lt(i, txinCount) { i := add(i, 1) } {
                ptr := add(ptr, 36)
                ptr := varintSkip(ptr)
                ptr := add(ptr, 4)
            }

            // Read and jump over tx outputs
            let txoutCount
            ptr, txoutCount := getVarint(ptr)
            for { let i := 0 } lt(i, txoutCount) { i := add(i, 1) } {
                ptr := add(ptr, 8)
                ptr := varintSkip(ptr)
            }

            // Jump over unneeded witnesses.
            let witnessEls
            for { let i := 0 } lt(i, _inIndex) { i := add(i, 1) } {
                ptr, witnessEls := getVarint(ptr)
                for { let j := 0 } lt(j, witnessEls) { j := add(j, 1) } {
                    // Skip actual witness element
                    ptr := varintSkip(ptr)
                }
            }

            // Get actual witness
            ptr, witnessEls := getVarint(ptr)
            // Checkt that there's at least two witnesses
            if iszero(gt(witnessEls, 1)) {
                mstore(0x00, 0x0d6fd37b)
                revert(0x1c, 0x04)
            }
            let witnessesToSkip := sub(witnessEls, 2)
            for { let i := 0 } lt(i, witnessesToSkip) { i := add(i, 1) } { ptr := varintSkip(ptr) }

            let witnessLen
            ptr, witnessLen := getVarint(ptr)

            // Load and copy witness data to memory
            inscription := mload(0x40)
            let insLength := 0

            // Check inscription header is `OP_TRUE OP_FALSE OP_IF` (0x00)
            ptr := add(ptr, 2)
            if iszero(eq(and(calldataload(ptr), 0xffffff), 0x510063)) {
                // `revert UnrecognizedInscriptionFormat()`
                mstore(0x00, 0x01aeb1d4)
                revert(0x1c, 0x04)
            }

            // Position ptr to read OP-byte + 4
            ptr := add(ptr, 5)

            let freeMem := add(inscription, 0x20)

            for {} 1 {} {
                let opLookahead := and(calldataload(ptr), 0xffffffffff)
                let op := shr(0x20, opLookahead)
                let skipForward
                switch lt(sub(op, 1), 0x4e)
                case 1 {
                    // Push opcodes (PUSH{1-75}, PUSHDATA1, PUSHDATA2, PUSHDATA4)
                    let pushLength
                    switch op
                    case 0x4c {
                        // OP_PUSHDATA1
                        pushLength := and(shr(0x18, opLookahead), 0xff)
                        ptr := add(ptr, 2)
                    }
                    case 0x4d {
                        // OP_PUSHDATA2
                        pushLength := or(shl(8, byte(29, opLookahead)), byte(28, opLookahead))
                        ptr := add(ptr, 3)
                    }
                    case 0x4e {
                        // OP_PUSHDATA4
                        pushLength := and(opLookahead, 0xffffffff)
                        pushLength := or(shl(8, and(pushLength, 0x00ff00ff)), and(shr(8, pushLength), 0x00ff00ff))
                        pushLength := and(or(shr(0x10, pushLength), shl(0x10, pushLength)), 0xffffffff)
                        ptr := add(ptr, 5)
                    }
                    default {
                        // OP_PUSH{1-75}
                        pushLength := op
                        ptr := add(ptr, 1)
                    }

                    calldatacopy(freeMem, add(ptr, 0x1b), pushLength)
                    ptr := add(ptr, pushLength)
                    freeMem := add(freeMem, pushLength)
                    insLength := add(insLength, pushLength)
                }
                default {
                    ptr := add(ptr, 1)
                    if iszero(sub(op, 0x68)) { break }
                }
            }

            mstore(inscription, insLength)
            freeMem := and(add(freeMem, add(insLength, 0x1f)), 0xffffffffffffffe0)
            mstore(0x40, freeMem)
        }
    }

    /// @dev Will use all gas if transaction does not contain witness commitment.
    /// @dev Expects tx without marker, flag or witness
    function getWitnessRootFromCoinbase(bytes calldata _coinbaseTx) internal pure returns (bytes32 witnessRoot) {
        assembly {
            function getVarint(ptr) -> newPtr, length {
                let maxData := and(calldataload(add(ptr, 0x8)), 0xffffffffffffffffff)
                let varLengthByte := shr(0x40, maxData)
                newPtr := add(ptr, 1)
                switch lt(varLengthByte, 0xfd)
                case 1 { length := varLengthByte }
                default {
                    // 0xfd -> 8 * 6
                    // 0xfe -> 8 * 4
                    // 0xff -> 8 * 0
                    // 8 * (8 - 2^(vlb - 0xfc)) = 64 - 8 * 2^(vlb - 0xfc)
                    let size := shl(sub(varLengthByte, 0xfc), 1)
                    newPtr := add(newPtr, size)
                    let data := and(maxData, 0xffffffffffffffff)
                    length := shl(sub(0x40, shl(3, size)), data)
                }
            }

            function varintSkip(ptr) -> newPtr {
                let maxData := and(calldataload(add(ptr, 0x8)), 0xffffffffffffffffff)
                let varLengthByte := shr(0x40, maxData)
                newPtr := add(ptr, 1)
                switch lt(varLengthByte, 0xfd)
                case 1 { newPtr := add(newPtr, varLengthByte) }
                default {
                    // 0xfd -> 8 * 6
                    // 0xfe -> 8 * 4
                    // 0xff -> 8 * 0
                    // 8 * (8 - 2^(vlb - 0xfc)) = 64 - 8 * 2^(vlb - 0xfc)
                    let size := shl(sub(varLengthByte, 0xfc), 1)
                    newPtr := add(newPtr, size)
                    let data := and(maxData, 0xffffffffffffffff)
                    newPtr := add(shl(sub(0x40, shl(3, size)), data), newPtr)
                }
            }

            // Start pointer 4 bytes after beginning, skipping version bytes
            let ptr := sub(_coinbaseTx.offset, 0x1b)

            // Read and jump over tx inputs
            let count
            ptr, count := getVarint(ptr)
            for { let i := 0 } lt(i, count) { i := add(i, 1) } {
                ptr := add(ptr, 36)
                ptr := varintSkip(ptr)
                ptr := add(ptr, 4)
            }
            // Find witness root commitment output
            ptr, count := getVarint(ptr)
            let scriptLen
            for {} 1 {} {
                ptr := add(ptr, 8)
                ptr, scriptLen := getVarint(ptr)
                // Check length and witness commitment header.
                if iszero(or(sub(scriptLen, 0x26), sub(and(calldataload(add(ptr, 5)), 0xffffffffffff), 0x6a24aa21a9ed)))
                {
                    // Get witness root
                    witnessRoot := calldataload(add(ptr, 37))
                    break
                }
                ptr := add(ptr, scriptLen)
            }
        }
    }

    function isValidMerkleCoinbase(bytes32 _txRoot, bytes32 _baseEl, bytes32[] calldata _proofEls)
        internal
        view
        returns (bool isValid)
    {
        assembly {
            let proofSize := _proofEls.length
            let proofOffset := _proofEls.offset

            let proofEnd := add(proofOffset, shl(5, proofSize))
            for {} lt(proofOffset, proofEnd) {} {
                mstore(0x00, _baseEl)
                mstore(0x20, calldataload(proofOffset))
                pop(staticcall(gas(), 0x02, 0x00, 0x40, 0x00, 0x20))
                pop(staticcall(gas(), 0x02, 0x00, 0x20, 0x00, 0x20))
                _baseEl := mload(0x00)
                proofOffset := add(proofOffset, 0x20)
            }
            isValid := eq(_txRoot, _baseEl)
        }
    }

    function reverseWord(bytes32 _b) internal pure returns (bytes32 r) {
        assembly {
            function swapRound(inp, mask, shift) -> res {
                res := or(shl(shift, and(inp, mask)), and(shr(shift, inp), mask))
            }
            r := swapRound(_b, 0x00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff, 0x08)
            r := swapRound(r, 0x0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff, 0x10)
            r := swapRound(r, 0x00000000ffffffff00000000ffffffff00000000ffffffff00000000ffffffff, 0x20)
            r := swapRound(r, 0x0000000000000000ffffffffffffffff0000000000000000ffffffffffffffff, 0x40)
            r := or(shr(0x80, r), shl(0x80, r))
        }
    }

    function sha256d(bytes calldata _d) internal pure returns (bytes32) {
        return sha256(abi.encode(sha256(_d)));
    }

    function sha256d_mem(bytes memory _d) internal pure returns (bytes32) {
        return sha256(abi.encode(sha256(_d)));
    }
}
