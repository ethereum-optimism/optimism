// SPDX-License-Identifier: GPL-3.0-only
pragma solidity 0.8.15;

/// @author philogy <https://github.com/philogy>
abstract contract BitcoinSPV {
    error InvalidLength();
    error InvalidHeaderchain();

    bytes32[] internal txRoots;
    bytes32 public lastBlockhash;

    constructor(bytes32 _startHash) {
        lastBlockhash = _startHash;
    }

    function _addHeaders(bytes calldata _headerData) internal {
        assembly {
            // Define utility functions
            function swapRound(inp, mask, shift) -> res {
                res := or(shl(shift, and(inp, mask)), and(shr(shift, inp), mask))
            }
            function reverseWord(inp) -> res {
                res := swapRound(inp, 0x00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff, 0x08)
                res := swapRound(res, 0x0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff0000ffff, 0x10)
                res := swapRound(res, 0x00000000ffffffff00000000ffffffff00000000ffffffff00000000ffffffff, 0x20)
                res := swapRound(res, 0x0000000000000000ffffffffffffffff0000000000000000ffffffffffffffff, 0x40)
                res := or(shr(0x80, res), shl(0x80, res))
            }
            function reverseSmall(inp) -> res {
                res := swapRound(inp, 0x00ff00ff, 0x08)
                res := or(shr(0x10, res), shl(0x10, res))
            }

            // Validate input data stream length.
            let headersLen := _headerData.length
            if mod(headersLen, 0x50) {
                mstore(0x00, 0x947d5a84)
                revert(0x1c, 0x04)
            }

            // Copy headers to memory.
            let freeMem := mload(0x40)
            calldatacopy(freeMem, _headerData.offset, headersLen)

            // Derive slots and load length to be able to push new roots.
            mstore(0x00, txRoots.slot)
            let txRootsValueSlot := keccak256(0x00, 0x20)
            let totalRoots := sload(txRoots.slot)
            let txRootsStartSlot := add(txRootsValueSlot, totalRoots)

            // Validate headers.
            let headersValid := 1
            let lastHash := sload(lastBlockhash.slot)
            let totalHeaders := div(_headerData.length, 0x50)
            let headerStart
            for { let i := 0 } lt(i, totalHeaders) { i := add(i, 1) } {
                headerStart := add(freeMem, mul(i, 0x50))

                // Validate Hashchain
                let prevBlockhash := mload(add(headerStart, 0x04))
                pop(staticcall(gas(), 0x02, headerStart, 0x50, 0x00, 0x20))
                pop(staticcall(gas(), 0x02, 0x00, 0x20, 0x00, 0x20))
                headersValid := and(headersValid, eq(prevBlockhash, lastHash))
                lastHash := mload(0x00)

                // Validate PoW Above Target
                let nBits := and(reverseSmall(mload(add(headerStart, 44))), 0xff7fffff)
                let target := shl(shl(3, sub(shr(24, nBits), 3)), and(nBits, 0xffffff))
                let fh := reverseWord(lastHash)
                headersValid := and(headersValid, iszero(gt(fh, target)))

                // Store merkle roots
                sstore(add(txRootsStartSlot, i), mload(add(headerStart, 0x24)))
            }
            if iszero(headersValid) {
                mstore(0x00, 0x34207b29)
                revert(0x1c, 0x04)
            }

            // Update total `txRoots`.
            sstore(txRoots.slot, add(totalRoots, totalHeaders))
            // Update `lastBlockhash`.
            sstore(lastBlockhash.slot, lastHash)
        }
    }

    function totalTxRoots() public view returns (uint256) {
        return txRoots.length;
    }

    function getTxRoot(uint256 _i) public view returns (bytes32) {
        return txRoots[_i];
    }
}
