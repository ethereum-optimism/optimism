# Guaranteed Gas Fee Market

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Gas Stipend](#gas-stipend)
- [Default Values](#default-values)
- [Limiting Guaranteed Gas](#limiting-guaranteed-gas)
- [Rationale for burning L1 Gas](#rationale-for-burning-l1-gas)
- [On Preventing Griefing Attacks](#on-preventing-griefing-attacks)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

[Deposited transactions](./glossary.md#deposited-transaction) are transactions on L2 that are
initiated on L1. The gas that they use on L2 is bought on L1 via a gas burn (or a direct payment
in the future). We maintain a fee market and hard cap on the amount of gas provided to all deposits
in a single L1 block.

The gas provided to deposited transactions is sometimes called "guaranteed gas". The gas provided to
deposited transactions is unique in the regard that it is not refundable. It cannot be refunded as
it is sometimes paid for with a gas burn and there may not be any ETH left to refund.

The **guaranteed gas** is composed of a gas stipend, and of any guaranteed gas the user would like
to purchase (on L1) on top of that.

Guaranteed gas on L2 is bought in the following manner. An L2 gas price is calculated via an
EIP-1559-style algorithm. The total amount of ETH required to buy that gas is then calculated as
(`guaranteed gas * L2 deposit basefee`). The contract then accepts that amount of ETH (in a future
upgrade) or (only method right now), burns an amount of L1 gas that corresponds to the L2 cost
(`L2 cost / L1 Basefee`). The L2 gas price for guaranteed gas is not synchronized with the basefee
on L2 and will likely be different.

## Gas Stipend

To offset the gas spent on the deposit event, we credit `gas spent * L1 basefee` ETH to the cost of
the L2 gas, where `gas spent` is the amount of L1 gas spent processing the deposit. If the ETH value
of this credit is greater than the ETH value of the requested guaranteed gas
(`requested guaranteed gas * L2 gas price`), no L1 gas is burnt.

## Default Values

| Variable                         | Value                                          |
| -------------------------------- | ---------------------------------------------- |
| `MAX_RESOURCE_LIMIT`             | 20,000,000                                     |
| `ELASTICITY_MULTIPLIER`          | 10                                             |
| `BASEFEE_MAX_CHANGE_DENOMINATOR` | 8                                              |
| `MINIMUM_BASEFEE`                | 1 gwei                                         |
| `MAXIMUM_BASEFEE`                | type(uint128).max                              |
| `SYSTEM_TX_MAX_GAS`              | 1,000,000                                      |
| `TARGET_RESOURCE_LIMIT`          | `MAX_RESOURCE_LIMIT` / `ELASTICITY_MULTIPLIER` |

## Limiting Guaranteed Gas

The total amount of guaranteed gas that can be bought in a single L1 block must be limited to
prevent a denial of service attack against L2 as well as ensure the total amount of guaranteed gas
stays below the L2 block gas limit.

We set a guaranteed gas limit of `MAX_RESOURCE_LIMIT` gas per L1 block and a target of
`MAX_RESOURCE_LIMIT` / `ELASTICITY_MULTIPLIER` gas per L1 block. These numbers enabled
occasional large transactions while staying within our target and maximum gas usage on L2.

Because the amount of guaranteed L2 gas that can be purchased in a single block is now limited,
we implement an EIP-1559-style fee market to reduce congestion on deposits. By setting the limit
at a multiple of the target, we enable deposits to temporarily use more L2 gas at a greater cost.

```python
# Pseudocode to update the L2 Deposit Basefee and cap the amount of guaranteed gas
# bought in a block. Calling code must handle the gas burn and validity checks on
# the ability of the account to afford this gas.

# prev_basefee is a u128, prev_bought_gas and prev_num are u64s
prev_basefee, prev_bought_gas, prev_num = <values from previous update>
now_num = block.number

# Clamp the full basefee to a specific range. The minimum value in the range should be around 100-1000
# to enable faster responses in the basefee. This replaces the `max` mechanism in the ethereum 1559
# implementation (it also serves to enable the basefee to increase if it is very small).
def clamp(v: i256, min: u128, max: u128) -> u128:
    if v < i256(min):
        return min
    elif v > i256(max):
        return max
    else:
        return u128(v)

# If this is a new block, update the basefee and reset the total gas
# If not, just update the total gas
if prev_num == now_num:
    now_basefee = prev_basefee
    now_bought_gas = prev_bought_gas + requested_gas
elif prev_num != now_num:
    # Width extension and conversion to signed integer math
    gas_used_delta = int128(prev_bought_gas) - int128(TARGET_RESOURCE_LIMIT)
    # Use truncating (round to 0) division - solidity's default.
    # Sign extend gas_used_delta & prev_basefee to 256 bits to avoid overflows here.
    base_fee_per_gas_delta = prev_basefee * gas_used_delta / TARGET_RESOURCE_LIMIT / BASEFEE_MAX_CHANGE_DENOMINATOR
    now_basefee_wide = prev_basefee + base_fee_per_gas_delta

    now_basefee = clamp(now_basefee_wide, min=MINIMUM_BASEFEE, max=UINT_128_MAX_VALUE)
    now_bought_gas =  requested_gas

    # If we skipped multiple blocks between the previous block and now update the basefee again.
    # This is not exactly the same as iterating the above function, but quite close for reasonable
    # gas target values. It is also constant time wrt the number of missed blocks which is important
    # for keeping gas usage stable.
    if prev_num + 1 < now_num:
        n = now_num - prev_num - 1
        # Apply 7/8 reduction to prev_basefee for the n empty blocks in a row.
        now_basefee_wide = now_basefee * pow(1-(1/BASEFEE_MAX_CHANGE_DENOMINATOR), n)
        now_basefee = clamp(now_basefee_wide, min=MINIMUM_BASEFEE, max=type(uint128).max)

require(now_bought_gas < MAX_RESOURCE_LIMIT)

store_values(now_basefee, now_bought_gas, now_num)
```

## Rationale for burning L1 Gas

There must be a sybil resistance mechanism for usage of the network. If it is very cheap to get
guaranteed gas on L2, then it would be possible to spam the network. Burning a dynamic amount
of gas on L1 acts as a sybil resistance mechanism as it becomes more expensive with more demand.

If we collect ETH directly to pay for L2 gas, every (indirect) caller of the deposit function will need
to be marked with the payable selector. This won't be possible for many existing projects. Unfortunately
this is quite wasteful. As such, we will provide two options to buy L2 gas:

1. Burn L1 Gas
2. Send ETH to the Optimism Portal (Not yet supported)

The payable version (Option 2) will likely have discount applied to it (or conversely, #1 has a
premium applied to it).

For the initial release of bedrock, only #1 is supported.

## On Preventing Griefing Attacks

The cost of purchasing all of the deposit gas in every block must be expensive
enough to prevent attackers from griefing all deposits to the network.
An attacker would observe a deposit in the mempool and frontrun it with a deposit
that purchases enough gas such that the other deposit reverts.
The smaller the max resource limit is, the easier this attack is to pull off.
This attack is mitigated by having a large resource limit as well as a large
elasticity multiplier. This means that the target resource usage is kept small,
giving a lot of room for the deposit base fee to rise when the max resource limit
is being purchased.

This attack should be too expensive to pull off in practice, but if an extremely
wealthy adversary does decide to grief network deposits for an extended period
of time, efforts will be placed to ensure that deposits are able to be processed
on the network.
