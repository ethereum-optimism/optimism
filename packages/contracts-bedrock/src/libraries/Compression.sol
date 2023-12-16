// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title Compression
/// @notice Compression is a library containing compression utilities.
library Compression {
    /// @notice Version of https://github.com/Vectorized/solady/blob/main/src/utils/LibZip.sol
    ///         that only returns the length of the data if it were to be compressed. This saves
    ///         gas over actually compressing the data, given we only need the length.
    /// @dev Returns the length of the compressed data.
    /// @custom:attribution Solady <https://github.com/Vectorized/solady>
    function flzCompressLen(bytes memory data) internal pure returns (uint256 n) {
        /// @solidity memory-safe-assembly
        assembly {
            function u24(p_) -> _u {
                let w := mload(p_)
                _u := or(shl(16, byte(2, w)), or(shl(8, byte(1, w)), byte(0, w)))
            }
            function cmp(p_, q_, e_) -> _l {
                for { e_ := sub(e_, q_) } lt(_l, e_) { _l := add(_l, 1) } {
                    e_ := mul(iszero(byte(0, xor(mload(add(p_, _l)), mload(add(q_, _l))))), e_)
                }
            }
            function literals(runs_, n_) -> _n {
                let d := div(runs_, 0x20)
                runs_ := mod(runs_, 0x20)
                _n := add(n_, mul(0x21, d))
                if iszero(runs_) { leave }
                _n := add(1, add(_n, runs_))
            }
            function match(l_, n_) -> _n {
                l_ := sub(l_, 1)
                n_ := add(n_, mul(3, div(l_, 262)))
                if iszero(lt(mod(l_, 262), 6)) {
                    _n := add(n_, 3)
                    leave
                }
                _n := add(n_, 2)
            }
            function setHash(i_, v_) {
                let p := add(mload(0x40), shl(2, i_))
                mstore(p, xor(mload(p), shl(224, xor(shr(224, mload(p)), v_))))
            }
            function getHash(i_) -> _h {
                _h := shr(224, mload(add(mload(0x40), shl(2, i_))))
            }
            function hash(v_) -> _r {
                _r := and(shr(19, mul(2654435769, v_)), 0x1fff)
            }
            function setNextHash(ip_, ipStart_) -> _ip {
                setHash(hash(u24(ip_)), sub(ip_, ipStart_))
                _ip := add(ip_, 1)
            }
            codecopy(mload(0x40), codesize(), 0x8000) // Zeroize the hashmap.
            n := 0
            let a := add(data, 0x20)
            let ipStart := a
            let ipLimit := sub(add(ipStart, mload(data)), 13)
            for { let ip := add(2, a) } lt(ip, ipLimit) {} {
                let r := 0
                let d := 0
                for {} 1 {} {
                    let s := u24(ip)
                    let h := hash(s)
                    r := add(ipStart, getHash(h))
                    setHash(h, sub(ip, ipStart))
                    d := sub(ip, r)
                    if iszero(lt(ip, ipLimit)) { break }
                    ip := add(ip, 1)
                    if iszero(gt(d, 0x1fff)) { if eq(s, u24(r)) { break } }
                }
                if iszero(lt(ip, ipLimit)) { break }
                ip := sub(ip, 1)
                if gt(ip, a) { n := literals(sub(ip, a), n) }
                let l := cmp(add(r, 3), add(ip, 3), add(ipLimit, 9))
                n := match(l, n)
                ip := setNextHash(setNextHash(add(ip, l), ipStart), ipStart)
                a := ip
            }
            n := literals(sub(add(ipStart, mload(data)), a), n)
        }
    }
}
