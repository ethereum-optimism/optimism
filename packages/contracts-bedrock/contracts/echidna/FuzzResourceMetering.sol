pragma solidity 0.8.15;

import { ResourceMetering } from "../L1/ResourceMetering.sol";
import { Arithmetic } from "../libraries/Arithmetic.sol";
import { StdUtils } from "forge-std/Test.sol";

contract EchidnaFuzzResourceMetering is ResourceMetering, StdUtils {
    bool failedMaxGasPerBlock;
    bool failedRaiseBasefee;
    bool failedLowerBasefee;
    bool failedNeverBelowMinBasefee;
    bool failedMaxRaiseBasefeePerBlock;
    bool failedMaxLowerBasefeePerBlock;

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
            failedNeverBelowMinBasefee = true;
        }
        // check that the last block didn't consume more than the max amount of gas
        if (cachedPrevBoughtGas > uint256(MAX_RESOURCE_LIMIT)) {
            failedMaxGasPerBlock = true;
        }

        // Part2: we perform the gas burn

        // force the gasToBurn into the correct range based on whether we intend to
        // raise or lower the basefee after this block, respectively
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
        uint256 maxBasefeeChange = cachedPrevBaseFee / uint256(BASE_FEE_MAX_CHANGE_DENOMINATOR);

        // If the last block used more than the target amount of gas (and there were no
        // empty blocks in between), ensure this block's basefee increased, but not by
        // more than the max amount per block
        if (
            (cachedPrevBoughtGas > uint256(TARGET_RESOURCE_LIMIT)) &&
            (uint256(params.prevBlockNum) - cachedPrevBlockNum == 1)
        ) {
            failedRaiseBasefee = failedRaiseBasefee || (params.prevBaseFee <= cachedPrevBaseFee);
            failedMaxRaiseBasefeePerBlock =
                failedMaxRaiseBasefeePerBlock ||
                ((uint256(params.prevBaseFee) - cachedPrevBaseFee) < maxBasefeeChange);
        }

        // If the last blocked used less than the target amount of gas ensure this block's basefee
        // decreased, but not by more than the max amount
        if (
            (cachedPrevBoughtGas < uint256(TARGET_RESOURCE_LIMIT)) ||
            (uint256(params.prevBlockNum) - cachedPrevBlockNum > 1)
        ) {
            failedLowerBasefee =
                failedLowerBasefee ||
                (uint256(params.prevBaseFee) > cachedPrevBaseFee);
            if (params.prevBlockNum - cachedPrevBlockNum == 1) {
                failedMaxLowerBasefeePerBlock =
                    failedMaxLowerBasefeePerBlock ||
                    ((cachedPrevBaseFee - uint256(params.prevBaseFee)) < maxBasefeeChange);
            }

            // Update the maxBasefeeChange to account for multiple blocks having passed
            maxBasefeeChange = uint256(
                Arithmetic.cdexp(
                    int256(cachedPrevBaseFee),
                    BASE_FEE_MAX_CHANGE_DENOMINATOR,
                    int256(uint256(params.prevBlockNum) - cachedPrevBlockNum)
                )
            );
            if (params.prevBlockNum - cachedPrevBlockNum > 1) {
                failedMaxLowerBasefeePerBlock =
                    failedMaxLowerBasefeePerBlock ||
                    ((cachedPrevBaseFee - uint256(params.prevBaseFee)) < maxBasefeeChange);
            }
        }
    }

    function _burnInternal(uint64 _gasToBurn) private metered(_gasToBurn) {}

    function echidna_high_usage_raise_basefee() public view returns (bool) {
        return !failedRaiseBasefee;
    }

    function echidna_low_usage_lower_basefee() public view returns (bool) {
        return !failedLowerBasefee;
    }

    function echidna_never_below_min_basefee() public view returns (bool) {
        return !failedNeverBelowMinBasefee;
    }

    function echidna_never_above_max_gas_limit() public view returns (bool) {
        return !failedMaxGasPerBlock;
    }

    function echidna_never_exceed_max_increase() public view returns (bool) {
        return !failedMaxRaiseBasefeePerBlock;
    }

    function echidna_never_exceed_max_decrease() public view returns (bool) {
        return !failedMaxLowerBasefeePerBlock;
    }
}
