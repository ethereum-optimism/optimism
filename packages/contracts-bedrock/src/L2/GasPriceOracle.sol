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
                let d := div(sub(l_, 1), 262)
                n_ := add(n_, mul(3, d))
                if iszero(lt(l_, 7)) {
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
