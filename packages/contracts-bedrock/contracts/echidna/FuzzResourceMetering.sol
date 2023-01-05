pragma solidity 0.8.15;

import { ResourceMetering } from "../L1/ResourceMetering.sol";
import { Arithmetic } from "../libraries/Arithmetic.sol";
import { StdUtils } from "forge-std/Test.sol";

contract EchidnaFuzzResourceMetering is ResourceMetering, StdUtils {
    bool internal failedMaxGasPerBlock;
    bool internal failedRaiseBaseFee;
    bool internal failedLowerBaseFee;
    bool internal failedNeverBelowMinBaseFee;
    bool internal failedMaxRaiseBaseFeePerBlock;
    bool internal failedMaxLowerBaseFeePerBlock;

    // Used as a special flag for the purpose of identifying unchecked math errors specifically
    // in the test contracts, not the target contracts themselves.
    bool internal underflow;

    constructor() {
        initialize();
    }

    function initialize() internal initializer {
        __ResourceMetering_init();
    }

    /**
     * @notice Takes the necessary parameters to allow us to burn arbitrary amounts of gas to test
     *         the underlying resource metering/gas market logic
     */
    function testBurn(uint256 _gasToBurn, bool _raiseBaseFee) public {
        // Part 1: we cache the current param values and do some basic checks on them.
        uint256 cachedPrevBaseFee = uint256(params.prevBaseFee);
        uint256 cachedPrevBoughtGas = uint256(params.prevBoughtGas);
        uint256 cachedPrevBlockNum = uint256(params.prevBlockNum);

        // check that the last block's base fee hasn't dropped below the minimum
        if (cachedPrevBaseFee < uint256(MINIMUM_BASE_FEE)) {
            failedNeverBelowMinBaseFee = true;
        }
        // check that the last block didn't consume more than the max amount of gas
        if (cachedPrevBoughtGas > uint256(MAX_RESOURCE_LIMIT)) {
            failedMaxGasPerBlock = true;
        }

        // Part2: we perform the gas burn

        // force the gasToBurn into the correct range based on whether we intend to
        // raise or lower the baseFee after this block, respectively
        uint256 gasToBurn;
        if (_raiseBaseFee) {
            gasToBurn = bound(
                _gasToBurn,
                uint256(TARGET_RESOURCE_LIMIT),
                uint256(MAX_RESOURCE_LIMIT)
            );
        } else {
            gasToBurn = bound(_gasToBurn, 0, uint256(TARGET_RESOURCE_LIMIT));
        }

        _burnInternal(uint64(gasToBurn));

        // Part 3: we run checks and modify our invariant flags based on the updated params values

        // Calculate the maximum allowed baseFee change (per block)
        uint256 maxBaseFeeChange = cachedPrevBaseFee / uint256(BASE_FEE_MAX_CHANGE_DENOMINATOR);

        // If the last block used more than the target amount of gas (and there were no
        // empty blocks in between), ensure this block's baseFee increased, but not by
        // more than the max amount per block
        if (
            (cachedPrevBoughtGas > uint256(TARGET_RESOURCE_LIMIT)) &&
            (uint256(params.prevBlockNum) - cachedPrevBlockNum == 1)
        ) {
            failedRaiseBaseFee = failedRaiseBaseFee || (params.prevBaseFee <= cachedPrevBaseFee);
            failedMaxRaiseBaseFeePerBlock =
                failedMaxRaiseBaseFeePerBlock ||
                ((uint256(params.prevBaseFee) - cachedPrevBaseFee) < maxBaseFeeChange);
        }

        // If the last block used less than the target amount of gas, (or was empty),
        // ensure that: this block's baseFee was decreased, but not by more than the max amount
        if (
            (cachedPrevBoughtGas < uint256(TARGET_RESOURCE_LIMIT)) ||
            (uint256(params.prevBlockNum) - cachedPrevBlockNum > 1)
        ) {
            // Invariant: baseFee should decrease
            failedLowerBaseFee =
                failedLowerBaseFee ||
                (uint256(params.prevBaseFee) > cachedPrevBaseFee);

            if (params.prevBlockNum - cachedPrevBlockNum == 1) {
                // No empty blocks
                // Invariant: baseFee should not have decreased by more than the maximum amount
                failedMaxLowerBaseFeePerBlock =
                    failedMaxLowerBaseFeePerBlock ||
                    ((cachedPrevBaseFee - uint256(params.prevBaseFee)) <= maxBaseFeeChange);
            } else if (params.prevBlockNum - cachedPrevBlockNum > 1) {
                // We have at least one empty block
                // Update the maxBaseFeeChange to account for multiple blocks having passed
                unchecked {
                    maxBaseFeeChange = uint256(
                        int256(cachedPrevBaseFee) -
                            Arithmetic.clamp(
                                Arithmetic.cdexp(
                                    int256(cachedPrevBaseFee),
                                    BASE_FEE_MAX_CHANGE_DENOMINATOR,
                                    int256(uint256(params.prevBlockNum) - cachedPrevBlockNum)
                                ),
                                MINIMUM_BASE_FEE,
                                MAXIMUM_BASE_FEE
                            )
                    );
                }

                // Detect an underflow in the previous calculation.
                // Without using unchecked above, and detecting the underflow here, echidna would
                // otherwise ignore the revert.
                underflow = underflow || maxBaseFeeChange > cachedPrevBaseFee;

                // Invariant: baseFee should not have decreased by more than the maximum amount
                failedMaxLowerBaseFeePerBlock =
                    failedMaxLowerBaseFeePerBlock ||
                    ((cachedPrevBaseFee - uint256(params.prevBaseFee)) <= maxBaseFeeChange);
            }
        }
    }

    function _burnInternal(uint64 _gasToBurn) private metered(_gasToBurn) {}

    /**
     * @custom:invariant The base fee should increase if the last block used more
     * than the target amount of gas
     *
     * If the last block used more than the target amount of gas (and there were no
     * empty blocks in between), ensure this block's baseFee increased, but not by
     * more than the max amount per block.
     */
    function echidna_high_usage_raise_baseFee() public view returns (bool) {
        return !failedRaiseBaseFee;
    }

    /**
     * @custom:invariant The base fee should decrease if the last block used less
     * than the target amount of gas
     *
     * If the previous block used less than the target amount of gas, the base fee should decrease,
     * but not more than the max amount.
     */
    function echidna_low_usage_lower_baseFee() public view returns (bool) {
        return !failedLowerBaseFee;
    }

    /**
     * @custom:invariant A block's base fee should never be below `MINIMUM_BASE_FEE`
     *
     * This test asserts that a block's base fee can never drop below the
     * `MINIMUM_BASE_FEE` threshold.
     */
    function echidna_never_below_min_baseFee() public view returns (bool) {
        return !failedNeverBelowMinBaseFee;
    }

    /**
     * @custom:invariant A block can never consume more than `MAX_RESOURCE_LIMIT` gas.
     *
     * This test asserts that a block can never consume more than the `MAX_RESOURCE_LIMIT`
     * gas threshold.
     */
    function echidna_never_above_max_gas_limit() public view returns (bool) {
        return !failedMaxGasPerBlock;
    }

    /**
     * @custom:invariant The base fee can never be raised more than the max base fee change.
     *
     * After a block consumes more gas than the target gas, the base fee cannot be raised
     * more than the maximum amount allowed. The max base fee change (per-block) is derived
     * as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR`
     */
    function echidna_never_exceed_max_increase() public view returns (bool) {
        return !failedMaxRaiseBaseFeePerBlock;
    }

    /**
     * @custom:invariant The base fee can never be lowered more than the max base fee change.
     *
     * After a block consumes less than the target gas, the base fee cannot be lowered more
     * than the maximum amount allowed. The max base fee change (per-block) is derived as
     *follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR`
     */
    function echidna_never_exceed_max_decrease() public view returns (bool) {
        return !failedMaxLowerBaseFeePerBlock;
    }

    /**
     * @custom:invariant The `maxBaseFeeChange` calculation over multiple blocks can never
     * underflow.
     *
     * When calculating the `maxBaseFeeChange` after multiple empty blocks, the calculation
     * should never be allowed to underflow.
     */
    function echidna_underflow() public view returns (bool) {
        return !underflow;
    }
}
