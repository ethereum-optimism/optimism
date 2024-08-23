# supertoken properties

legend:

- `[ ]`: property not yet tested
- `**[ ]**`: property not yet tested, dev/research team has asked for extra focus on it
- `[X]`: tested/proven property
- `[~]`: partially tested/proven property
- `:(`: property won't be tested due to some limitation

## Unit test

| id  | description                                                                        | halmos | medusa |
| --- | ---------------------------------------------------------------------------------- | ------ | ------ |
| 0   | supertoken token address does not depend on the executing chain’s chainID          | [ ]    | [ ]    |
| 1   | supertoken token address depends on name, remote token, address and decimals       | [ ]    | [ ]    |
| 2   | convert() should only allow converting legacy tokens to supertoken and viceversa   | [ ]    | [ ]    |
| 3   | convert() only allows migrations between tokens representing the same remote asset | [ ]    | [ ]    |
| 4   | convert() only allows migrations from tokens with the same decimals                | [ ]    | [ ]    |
| 5   | convert() burns the same amount of one token that it mints of the other            | [ ]    | [ ]    |

## Valid state

| id  | description                                                                                | halmos  | medusa |
| --- | ------------------------------------------------------------------------------------------ | ------- | ------ |
| 6   | calls to sendERC20 succeed as long as caller has enough balance                            | [x]     | [ ]    |
| 7   | calls to relayERC20 always succeed as long as the sender and cross-domain caller are valid | **[~]** | [ ]    |

## Variable transition

| id  | description                                                                                       | halmos | medusa |
| --- | ------------------------------------------------------------------------------------------------- | ------ | ------ |
| 8   | sendERC20 with a value of zero does not modify accounting                                         | [x]    | [ ]    |
| 9   | relayERC20 with a value of zero does not modify accounting                                        | [x]    | [ ]    |
| 10  | sendERC20 decreases the token's totalSupply in the source chain exactly by the input amount       | [x]    | [ ]    |
| 11  | relayERC20 increases the token's totalSupply in the destination chain exactly by the input amount | [x]    | [ ]    |
| 12  | supertoken total supply only increases on calls to mint() by the L2toL2StandardBridge             | [x]    | [ ]    |
| 13  | supertoken total supply only decreases on calls to burn() by the L2toL2StandardBridge             | [x]    | [ ]    |
| 14  | supertoken total supply starts at zero                                                            | [x]    | [ ]    |
| 15  | deploying a supertoken registers its remote token in the factory                                  | [ ]    | [ ]    |
| 16  | deploying an OptimismMintableERC20 registers its remote token in the factory                      | [ ]    | [ ]    |

## High level

| id  | description                                                                                                                                                           | halmos | medusa |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 17  | only calls to convert(legacy, super) can increase a supertoken’s total supply across chains                                                                           | [ ]    | [ ]    |
| 18  | only calls to convert(super, legacy) can decrease a supertoken’s total supply across chains                                                                           | [ ]    | [ ]    |
| 19  | sum of total supply across all chains is always <= to convert(legacy, super)- convert(super, legacy)                                                                  | [ ]    | [ ]    |
| 20  | tokens sendERC20-ed on a source chain to a destination chain can be relayERC20-ed on it as long as the source chain is in the dependency set of the destination chain | [ ]    | [ ]    |
| 21  | sum of supertoken total supply across all chains is = to convert(legacy, super)- convert(super, legacy) when all cross-chain messages are processed                   | [ ]    | [ ]    |

## Atomic bridging pseudo-properties

As another layer of defense, the following properties are defined which assume bridging operations to be atomic (that is, the sequencer and L2Inbox and CrossDomainMessenger contracts are fully abstracted away, `sendERC20` triggering the `relayERC20` call on the same transaction)
It’s worth noting that these properties will not hold for a live system

| id  | description                                                                                                                        | halmos | medusa |
| --- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ |
| 22  | sendERC20 decreases sender balance in source chain and increases receiver balance in destination chain exactly by the input amount | [ ]    | [x]    |
| 23  | sendERC20 decreases total supply in source chain and increases it in destination chain exactly by the input amount                 | [ ]    | [x]    |
| 24  | sum of supertoken total supply across all chains is always equal to convert(legacy, super)- convert(super, legacy)                 | [ ]    | [~]    |

# Expected external interactions

- regular ERC20 operations between any accounts on the same chain, provided by [crytic ERC20 properties](https://github.com/crytic/properties?tab=readme-ov-file#erc20-tests)

# Invariant-breaking candidates (brain dump)

here we’ll list possible interactions that we intend the fuzzing campaign to support in order to help break invariants

- [ ] changing the decimals of tokens after deployment
- [ ] `convert()` ing between multiple (3+) representations of the same remote token, by having different names/symbols
