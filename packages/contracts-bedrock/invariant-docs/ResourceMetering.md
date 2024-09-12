# `ResourceMetering` Invariants

## The base fee should increase if the last block used more than the target amount of gas.
**Test:** [`ResourceMetering.t.sol#L166`](../test/invariants/ResourceMetering.t.sol#L166)

If the last block used more than the target amount of gas (and there were no empty blocks in between), ensure this block's baseFee increased, but not by more than the max amount per block. 

## The base fee should decrease if the last block used less than the target amount of gas.
**Test:** [`ResourceMetering.t.sol#L175`](../test/invariants/ResourceMetering.t.sol#L175)

If the previous block used less than the target amount of gas, the base fee should decrease, but not more than the max amount. 

## A block's base fee should never be below `MINIMUM_BASE_FEE`.
**Test:** [`ResourceMetering.t.sol#L183`](../test/invariants/ResourceMetering.t.sol#L183)

This test asserts that a block's base fee can never drop below the `MINIMUM_BASE_FEE` threshold. 

## A block can never consume more than `MAX_RESOURCE_LIMIT` gas.
**Test:** [`ResourceMetering.t.sol#L191`](../test/invariants/ResourceMetering.t.sol#L191)

This test asserts that a block can never consume more than the `MAX_RESOURCE_LIMIT` gas threshold. 

## The base fee can never be raised more than the max base fee change.
**Test:** [`ResourceMetering.t.sol#L201`](../test/invariants/ResourceMetering.t.sol#L201)

After a block consumes more gas than the target gas, the base fee cannot be raised more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 

## The base fee can never be lowered more than the max base fee change.
**Test:** [`ResourceMetering.t.sol#L211`](../test/invariants/ResourceMetering.t.sol#L211)

After a block consumes less than the target gas, the base fee cannot be lowered more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 

## The `maxBaseFeeChange` calculation over multiple blocks can never underflow.
**Test:** [`ResourceMetering.t.sol#L220`](../test/invariants/ResourceMetering.t.sol#L220)

When calculating the `maxBaseFeeChange` after multiple empty blocks, the calculation should never be allowed to underflow. 