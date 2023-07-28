# `ResourceMetering` Invariants

## The base fee should increase if the last block used more than the target amount of gas.
**Test:** [`ResourceMetering.t.sol#L177`](../test/invariants/ResourceMetering.t.sol#L177)

If the last block used more than the target amount of gas (and there were no empty blocks in between), ensure this block's baseFee increased, but not by more than the max amount per block. 

## The base fee should decrease if the last block used less than the target amount of gas.
**Test:** [`ResourceMetering.t.sol#L186`](../test/invariants/ResourceMetering.t.sol#L186)

If the previous block used less than the target amount of gas, the base fee should decrease, but not more than the max amount. 

## A block's base fee should never be below `MINIMUM_BASE_FEE`.
**Test:** [`ResourceMetering.t.sol#L194`](../test/invariants/ResourceMetering.t.sol#L194)

This test asserts that a block's base fee can never drop below the `MINIMUM_BASE_FEE` threshold. 

## A block can never consume more than `MAX_RESOURCE_LIMIT` gas.
**Test:** [`ResourceMetering.t.sol#L202`](../test/invariants/ResourceMetering.t.sol#L202)

This test asserts that a block can never consume more than the `MAX_RESOURCE_LIMIT` gas threshold. 

## The base fee can never be raised more than the max base fee change.
**Test:** [`ResourceMetering.t.sol#L212`](../test/invariants/ResourceMetering.t.sol#L212)

After a block consumes more gas than the target gas, the base fee cannot be raised more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 

## The base fee can never be lowered more than the max base fee change.
**Test:** [`ResourceMetering.t.sol#L222`](../test/invariants/ResourceMetering.t.sol#L222)

After a block consumes less than the target gas, the base fee cannot be lowered more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 

## The `maxBaseFeeChange` calculation over multiple blocks can never underflow.
**Test:** [`ResourceMetering.t.sol#L231`](../test/invariants/ResourceMetering.t.sol#L231)

When calculating the `maxBaseFeeChange` after multiple empty blocks, the calculation should never be allowed to underflow. 