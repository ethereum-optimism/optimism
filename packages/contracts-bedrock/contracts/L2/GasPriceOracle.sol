// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { L1Block } from "../L2/L1Block.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x420000000000000000000000000000000000000F
 * @title GasPriceOracle
 * @notice This contract maintains the variables responsible for computing the L1 portion of the
 *         total fee charged on L2. The values stored in the contract are looked up as part of the
 *         L2 state transition function and used to compute the total fee paid by the user. The
 *         contract exposes an API that is useful for knowing how large the L1 portion of their
 *         transaction fee will be.
 */
contract GasPriceOracle is Semver {
    /**
     * @custom:legacy
     * @custom:spacer _owner
     * @notice Spacer for backwards compatibility.
     */
    address private spacer_0_0_20;

    /**
     * @custom:legacy
     * @custom:spacer gasPrice
     * @notice Spacer for backwards compatibility.
     */
    uint256 private spacer_1_0_32;

    /**
     * @custom:legacy
     * @custom:spacer l1BaseFee
     * @notice Spacer for backwards compatibility.
     */
    uint256 private spacer_2_0_32;

    /**
     * @custom:legacy
     * @custom:spacer overhead
     * @notice Spacer for backwards compatibility.
     */
    uint256 private spacer_3_0_32;

    /**
     * @custom:legacy
     * @custom:spacer scalar
     * @notice Spacer for backwards compatibility.
     */
    uint256 private spacer_4_0_32;

    /**
     * @notice Number of decimals used in the scalar.
     */
    uint256 public constant decimals = 6;

    /**
     * @notice Emitted when the overhead value is updated.
     */
    event OverheadUpdated(uint256 overhead);

    /**
     * @notice Emitted when the scalar value is updated.
     */
    event ScalarUpdated(uint256 scalar);

    /**
     * @notice Emitted when the decimals value is updated.
     */
    event DecimalsUpdated(uint256 decimals);

    /**
     * @custom:semver 0.0.1
     */
    constructor() Semver(0, 0, 1) {}

    /**
     * @notice Computes the L1 portion of the fee based on the size of the rlp encoded input
     *         transaction, the current L1 base fee, and the various dynamic parameters.
     *
     * @param _data Unsigned fully RLP-encoded transaction to get the L1 fee for.
     *
     * @return L1 fee that should be paid for the tx
     */
    function getL1Fee(bytes memory _data) external view returns (uint256) {
        uint256 l1GasUsed = getL1GasUsed(_data);
        uint256 l1Fee = l1GasUsed * l1BaseFee();
        uint256 divisor = 10**decimals;
        uint256 unscaled = l1Fee * scalar();
        uint256 scaled = unscaled / divisor;
        return scaled;
    }

    /**
     * @notice Retrieves the current gas price (base fee).
     *
     * @return Current L2 gas price (base fee).
     */
    function gasPrice() public view returns (uint256) {
        return block.basefee;
    }

    /**
     * @notice Retrieves the current base fee.
     *
     * @return Current L2 base fee.
     */
    function baseFee() public view returns (uint256) {
        return block.basefee;
    }

    /**
     * @notice Retrieves the current fee overhead.
     *
     * @return Current fee overhead.
     */
    function overhead() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeOverhead();
    }

    /**
     * @notice Retrieves the current fee scalar.
     *
     * @return Current fee scalar.
     */
    function scalar() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l1FeeScalar();
    }

    /**
     * @notice Retrieves the latest known L1 base fee.
     *
     * @return Latest known L1 base fee.
     */
    function l1BaseFee() public view returns (uint256) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).basefee();
    }

    /**
     * @notice Computes the amount of L1 gas used for a transaction. Adds the overhead which
     *         represents the per-transaction gas overhead of posting the transaction and state
     *         roots to L1. Adds 68 bytes of padding to account for the fact that the input does
     *         not have a signature.
     *
     * @param _data Unsigned fully RLP-encoded transaction to get the L1 gas for.
     *
     * @return Amount of L1 gas used to publish the transaction.
     */
    function getL1GasUsed(bytes memory _data) public view returns (uint256) {
        uint256 total = 0;
        uint256 length = _data.length;
        for (uint256 i = 0; i < length; i++) {
            if (_data[i] == 0) {
                total += 4;
            } else {
                total += 16;
            }
        }
        uint256 unsigned = total + overhead();
        return unsigned + (68 * 16);
    }
}
