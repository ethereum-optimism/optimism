// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";

/// @custom:proxied
/// @custom:predeploy 0x420000000000000000000000000000000000000F
/// @title GasPriceOracle
/// @notice This contract maintains the variables responsible for computing the L1 portion of the
///         total fee charged on L2. Before Bedrock, this contract held variables in state that were
///         read during the state transition function to compute the L1 portion of the transaction
///         fee. After Bedrock, this contract now simply proxies the L1Block contract, which has
///         the values used to compute the L1 portion of the fee in its state.
///
///         The contract exposes an API that is useful for knowing how large the L1 portion of the
///         transaction fee will be. The following events were deprecated with Bedrock:
///         - event OverheadUpdated(uint256 overhead);
///         - event ScalarUpdated(uint256 scalar);
///         - event DecimalsUpdated(uint256 decimals);
contract GasPriceOracle is ISemver {
    /// @notice Number of decimals used in the scalar.
    uint256 public constant DECIMALS = 6;

    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice Indicates whether the network has gone through the Ecotone upgrade.
    bool public isEcotone;

    /// @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
    ///         transaction, the current L1 base fee, and the various dynamic parameters.
    /// @param _data Unsigned fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function getL1Fee(bytes memory _data) external view returns (uint256) {
        if (isEcotone) {
            return _getL1FeeEcotone(_data);
        }
        return _getL1FeeBedrock(_data);
    }

    /// @notice Set chain to be Ecotone chain (callable by depositor account)
    function setEcotone() external {
        require(
            msg.sender == L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).DEPOSITOR_ACCOUNT(),
            "GasPriceOracle: only the depositor account can set isEcotone flag"
        );
        require(isEcotone == false, "GasPriceOracle: Ecotone already active");
        isEcotone = true;
    }

    /// @notice Retrieves the current gas price (base fee).
    /// @return Current L2 gas price (base fee).
    function gasPrice() public view returns (uint256) {
        return block.basefee;
    }

    /// @notice Retrieves the current base fee.
    /// @return Current L2 base fee.
    function baseFee() public view returns (uint256) {
        return block.basefee;
    }

    /// @custom:legacy
    /// @notice Retrieves the current fee overhead.
    /// @return Current fee overhead.
    function overhead() public view returns (uint256) {
        require(!isEcotone, "GasPriceOracle: overhead() is deprecated");
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeOverhead();
    }

    /// @custom:legacy
    /// @notice Retrieves the current fee scalar.
    /// @return Current fee scalar.
    function scalar() public view returns (uint256) {
        require(!isEcotone, "GasPriceOracle: scalar() is deprecated");
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeScalar();
    }

    /// @notice Retrieves the latest known L1 base fee.
    /// @return Latest known L1 base fee.
    function l1BaseFee() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).basefee();
    }

    /// @notice Retrieves the current blob base fee.
    /// @return Current blob base fee.
    function blobBaseFee() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).blobBaseFee();
    }

    /// @notice Retrieves the current base fee scalar.
    /// @return Current base fee scalar.
    function baseFeeScalar() public view returns (uint32) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).baseFeeScalar();
    }

    /// @notice Retrieves the current blob base fee scalar.
    /// @return Current blob base fee scalar.
    function blobBaseFeeScalar() public view returns (uint32) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).blobBaseFeeScalar();
    }

    /// @custom:legacy
    /// @notice Retrieves the number of decimals used in the scalar.
    /// @return Number of decimals used in the scalar.
    function decimals() public pure returns (uint256) {
        return DECIMALS;
    }

    /// @notice Computes the amount of L1 gas used for a transaction. Adds 68 bytes
    ///         of padding to account for the fact that the input does not have a signature.
    /// @param _data Unsigned fully RLP-encoded transaction to get the L1 gas for.
    /// @return Amount of L1 gas used to publish the transaction.
    function getL1GasUsed(bytes memory _data) public view returns (uint256) {
        uint256 l1GasUsed = _getCalldataGas(_data);
        if (isEcotone) {
            return l1GasUsed;
        }
        return l1GasUsed + L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeOverhead();
    }

    /// @notice Computation of the L1 portion of the fee for Bedrock.
    /// @param _data Unsigned fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function _getL1FeeBedrock(bytes memory _data) internal view returns (uint256) {
        uint256 l1GasUsed = _getCalldataGas(_data);
        uint256 fee = (l1GasUsed + L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeOverhead()) * l1BaseFee()
            * L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeScalar();
        return fee / (10 ** DECIMALS);
    }

    /// @notice L1 portion of the fee after Ecotone.
    /// @param _data Unsigned fully RLP-encoded transaction to get the L1 fee for.
    /// @return L1 fee that should be paid for the tx
    function _getL1FeeEcotone(bytes memory _data) internal view returns (uint256) {
        uint256 l1GasUsed = _getCalldataGas(_data);
        uint256 scaledBaseFee = baseFeeScalar() * 16 * l1BaseFee();
        uint256 scaledBlobBaseFee = blobBaseFeeScalar() * blobBaseFee();
        uint256 fee = l1GasUsed * (scaledBaseFee + scaledBlobBaseFee);
        return fee / (16 * 10 ** DECIMALS);
    }

    /// @notice L1 gas estimation calculation.
    /// @param _data Unsigned fully RLP-encoded transaction to get the L1 gas for.
    /// @return Amount of L1 gas used to publish the transaction.
    function _getCalldataGas(bytes memory _data) internal pure returns (uint256) {
        uint256 total = 0;
        uint256 length = _data.length;
        for (uint256 i = 0; i < length; i++) {
            if (_data[i] == 0) {
                total += 4;
            } else {
                total += 16;
            }
        }
        return total + (68 * 16);
    }

    /// @notice LZ77 implementation based on FastLZ.
    ///         Equivalent to level 1 compression and decompression at the following commit:
    ///         https://github.com/ariya/FastLZ/commit/344eb4025f9ae866ebf7a2ec48850f7113a97a42
    ///         Decompression is backwards compatible.
    /// @dev Returns the compressed `data`.
    /// @custom:attribution Solady <https://github.com/Vectorized/Solady>
    function flzCompress(bytes memory data) internal pure returns (bytes memory result) {
        /// @solidity memory-safe-assembly
        assembly {
            function ms8(d_, v_) -> _d {
                mstore8(d_, v_)
                _d := add(d_, 1)
            }
            function u24(p_) -> _u {
                let w := mload(p_)
                _u := or(shl(16, byte(2, w)), or(shl(8, byte(1, w)), byte(0, w)))
            }
            function cmp(p_, q_, e_) -> _l {
                for { e_ := sub(e_, q_) } lt(_l, e_) { _l := add(_l, 1) } {
                    e_ := mul(iszero(byte(0, xor(mload(add(p_, _l)), mload(add(q_, _l))))), e_)
                }
            }
            function literals(runs_, src_, dest_) -> _o {
                for { _o := dest_ } iszero(lt(runs_, 0x20)) { runs_ := sub(runs_, 0x20) } {
                    mstore(ms8(_o, 31), mload(src_))
                    _o := add(_o, 0x21)
                    src_ := add(src_, 0x20)
                }
                if iszero(runs_) { leave }
                mstore(ms8(_o, sub(runs_, 1)), mload(src_))
                _o := add(1, add(_o, runs_))
            }
            function match(l_, d_, o_) -> _o {
                for { d_ := sub(d_, 1) } iszero(lt(l_, 263)) { l_ := sub(l_, 262) } {
                    o_ := ms8(ms8(ms8(o_, add(224, shr(8, d_))), 253), and(0xff, d_))
                }
                if iszero(lt(l_, 7)) {
                    _o := ms8(ms8(ms8(o_, add(224, shr(8, d_))), sub(l_, 7)), and(0xff, d_))
                    leave
                }
                _o := ms8(ms8(o_, add(shl(5, l_), shr(8, d_))), and(0xff, d_))
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
            let op := add(mload(0x40), 0x8000)
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
                if gt(ip, a) { op := literals(sub(ip, a), a, op) }
                let l := cmp(add(r, 3), add(ip, 3), add(ipLimit, 9))
                op := match(l, d, op)
                ip := setNextHash(setNextHash(add(ip, l), ipStart), ipStart)
                a := ip
            }
            op := literals(sub(add(ipStart, mload(data)), a), a, op)
            result := mload(0x40)
            let t := add(result, 0x8000)
            let n := sub(op, t)
            mstore(result, n) // Store the length.
            // Copy the result to compact the memory, overwriting the hashmap.
            let o := add(result, 0x20)
            for { let i } lt(i, n) { i := add(i, 0x20) } { mstore(add(o, i), mload(add(t, i))) }
            mstore(add(o, n), 0) // Zeroize the slot after the string.
            mstore(0x40, add(add(o, n), 0x20)) // Allocate the memory.
        }
    }
}
