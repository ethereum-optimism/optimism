# `ResourceMetering` Invariants

## The base fee should increase if the last block used more than the target amount of gas
**Test:** [`FuzzResourceMetering.sol#L139`](../contracts/echidna/FuzzResourceMetering.sol#L139)

If the last block used more than the target amount of gas (and there were no empty blocks in between), ensure this block's baseFee increased, but not by more than the max amount per block. 


## The base fee should decrease if the last block used less than the target amount of gas
**Test:** [`FuzzResourceMetering.sol#L150`](../contracts/echidna/FuzzResourceMetering.sol#L150)

If the previous block used less than the target amount of gas, the base fee should decrease, but not more than the max amount. 


## A block's base fee should never be below `MINIMUM_BASE_FEE`
**Test:** [`FuzzResourceMetering.sol#L160`](../contracts/echidna/FuzzResourceMetering.sol#L160)

This test asserts that a block's base fee can never drop below the `MINIMUM_BASE_FEE` threshold. 


## A block can never consume more than `MAX_RESOURCE_LIMIT` gas.
**Test:** [`FuzzResourceMetering.sol#L170`](../contracts/echidna/FuzzResourceMetering.sol#L170)

This test asserts that a block can never consume more than the `MAX_RESOURCE_LIMIT` gas threshold. 


## The base fee can never be raised more than the max base fee change.
**Test:** [`FuzzResourceMetering.sol#L181`](../contracts/echidna/FuzzResourceMetering.sol#L181)

After a block consumes more gas than the target gas, the base fee cannot be raised more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 


## The base fee can never be lowered more than the max base fee change.
**Test:** [`FuzzResourceMetering.sol#L192`](../contracts/echidna/FuzzResourceMetering.sol#L192)

After a block consumes less than the target gas, the base fee cannot be lowered more than the maximum amount allowed. The max base fee change (per-block) is derived as follows: `prevBaseFee / BASE_FEE_MAX_CHANGE_DENOMINATOR` 


## The `maxBaseFeeChange` calculation over multiple blocks can never underflow.
**Test:** [`FuzzResourceMetering.sol#L203`](../contracts/echidna/FuzzResourceMetering.sol#L203)

When calculating the `maxBaseFeeChange` after multiple empty blocks, the calculation should never be allowed to underflow. 
