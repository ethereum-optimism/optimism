// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Burn } from "../libraries/Burn.sol";
import { Arithmetic } from "../libraries/Arithmetic.sol";

/**
 * @custom:upgradeable
 * @title ResourceMetering
 * @notice ResourceMetering implements an EIP-1559 style resource metering system where pricing
 *         updates automatically based on current demand.
 */
abstract contract ResourceMetering is Initializable {
    /**
     * @notice Represents the various parameters that control the way in which resources are
     *         metered. Corresponds to the EIP-1559 resource metering system.
     *
     * @custom:field prevBaseFee   Base fee from the previous block(s).
     * @custom:field prevBoughtGas Amount of gas bought so far in the current block.
     * @custom:field prevBlockNum  Last block number that the base fee was updated.
     */
    struct ResourceParams {
        uint128 prevBaseFee;
        uint64 prevBoughtGas;
        uint64 prevBlockNum;
    }

    /**
     * @notice Maximum amount of the resource that can be used within this block.
     */
    int256 public constant MAX_RESOURCE_LIMIT = 8_000_000;

    /**
     * @notice Along with the resource limit, determines the target resource limit.
     */
    int256 public constant ELASTICITY_MULTIPLIER = 4;

    /**
     * @notice Target amount of the resource that should be used within this block.
     */
    int256 public constant TARGET_RESOURCE_LIMIT = MAX_RESOURCE_LIMIT / ELASTICITY_MULTIPLIER;

    /**
     * @notice Denominator that determines max change on fee per block.
     */
    int256 public constant BASE_FEE_MAX_CHANGE_DENOMINATOR = 8;

    /**
     * @notice Minimum base fee value, cannot go lower than this.
     */
    int256 public constant MINIMUM_BASE_FEE = 10_000;

    /**
     * @notice Maximum base fee value, cannot go higher than this.
     */
    int256 public constant MAXIMUM_BASE_FEE = int256(uint256(type(uint128).max));

    /**
     * @notice Initial base fee value.
     */
    uint128 public constant INITIAL_BASE_FEE = 1_000_000_000;

    /**
     * @notice EIP-1559 style gas parameters.
     */
    ResourceParams public params;

    /**
     * @notice Reserve extra slots (to a total of 50) in the storage layout for future upgrades.
     */
    uint256[48] private __gap;

    /**
     * @notice Meters access to a function based an amount of a requested resource.
     *
     * @param _amount Amount of the resource requested.
     */
    modifier metered(uint64 _amount) {
        // Record initial gas amount so we can refund for it later.
        uint256 initialGas = gasleft();

        // Run the underlying function.
        _;

        // Update block number and base fee if necessary.
        uint256 blockDiff = block.number - params.prevBlockNum;
        if (blockDiff > 0) {
            // Handle updating EIP-1559 style gas parameters. We use EIP-1559 to restrict the rate
            // at which deposits can be created and therefore limit the potential for deposits to
            // spam the L2 system. Fee scheme is very similar to EIP-1559 with minor changes.
            int256 gasUsedDelta = int256(uint256(params.prevBoughtGas)) - TARGET_RESOURCE_LIMIT;
            int256 baseFeeDelta = (int256(uint256(params.prevBaseFee)) * gasUsedDelta) /
                (TARGET_RESOURCE_LIMIT * BASE_FEE_MAX_CHANGE_DENOMINATOR);

            // Update base fee by adding the base fee delta and clamp the resulting value between
            // min and max.
            int256 newBaseFee = Arithmetic.clamp(
                int256(uint256(params.prevBaseFee)) + baseFeeDelta,
                MINIMUM_BASE_FEE,
                MAXIMUM_BASE_FEE
            );

            // If we skipped more than one block, we also need to account for every empty block.
            // Empty block means there was no demand for deposits in that block, so we should
            // reflect this lack of demand in the fee.
            if (blockDiff > 1) {
                // Update the base fee by repeatedly applying the exponent 1-(1/change_denominator)
                // blockDiff - 1 times. Simulates multiple empty blocks. Clamp the resulting value
                // between min and max.
                newBaseFee = Arithmetic.clamp(
                    Arithmetic.cdexp(
                        newBaseFee,
                        BASE_FEE_MAX_CHANGE_DENOMINATOR,
                        int256(blockDiff - 1)
                    ),
                    MINIMUM_BASE_FEE,
                    MAXIMUM_BASE_FEE
                );
            }

            // Update new base fee, reset bought gas, and update block number.
            params.prevBaseFee = uint128(uint256(newBaseFee));
            params.prevBoughtGas = 0;
            params.prevBlockNum = uint64(block.number);
        }

        // Make sure we can actually buy the resource amount requested by the user.
        params.prevBoughtGas += _amount;
        require(
            int256(uint256(params.prevBoughtGas)) <= MAX_RESOURCE_LIMIT,
            "ResourceMetering: cannot buy more gas than available gas limit"
        );

        // Determine the amount of ETH to be paid.
        uint256 resourceCost = _amount * params.prevBaseFee;

        // We currently charge for this ETH amount as an L1 gas burn, so we convert the ETH amount
        // into gas by dividing by the L1 base fee. We assume a minimum base fee of 1 gwei to avoid
        // division by zero for L1s that don't support 1559 or to avoid excessive gas burns during
        // periods of extremely low L1 demand. One-day average gas fee hasn't dipped below 1 gwei
        // during any 1 day period in the last 5 years, so should be fine.
        uint256 gasCost = resourceCost / Math.max(block.basefee, 1 gwei);

        // Give the user a refund based on the amount of gas they used to do all of the work up to
        // this point. Since we're at the end of the modifier, this should be pretty accurate. Acts
        // effectively like a dynamic stipend (with a minimum value).
        uint256 usedGas = initialGas - gasleft();
        if (gasCost > usedGas) {
            Burn.gas(gasCost - usedGas);
        }
    }

    /**
     * @notice Sets initial resource parameter values. This function must either be called by the
     *         initializer function of an upgradeable child contract.
     */
    // solhint-disable-next-line func-name-mixedcase
    function __ResourceMetering_init() internal onlyInitializing {
        params = ResourceParams({
            prevBaseFee: INITIAL_BASE_FEE,
            prevBoughtGas: 0,
            prevBlockNum: uint64(block.number)
        });
    }
}
