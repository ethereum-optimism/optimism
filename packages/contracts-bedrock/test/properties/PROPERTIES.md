# Supertoken advanced testing

## Overview

This document defines a set of properties global to the supertoken ecosystem, for which we will:

- run a [Medusa](https://github.com/crytic/medusa) fuzzing campaign, trying to break system invariants
- formally prove with [Halmos](https://github.com/a16z/halmos) whenever possible

## Milestones

The supertoken ecosystem consists of not just the supertoken contract, but the required changes to other contracts for liquidity to reach the former.

Considering only the supertoken contract is merged into the `develop` branch, and work for the other components is still in progress, we define three milestones for the testing campaign:

- SupERC20: concerned with only the supertoken contract, the first one to be implemented
- Factories: covers the above + the development of `OptimismSuperchainERC20Factory` and required changes to `OptimismMintableERC20Factory`
- Liquidity Migration: includes the `convert` function on the `L2StandardBridgeInterop` to migrate liquidity from legacy tokens into supertokens

## Where to place the testing campaign

Given the [OP monorepo](https://github.com/ethereum-optimism/optimism) already has invariant testing provided by foundry, it's not a trivial matter where to place this advanced testing campaign. Two alternatives are proposed:

- including it in the mainline OP monorepo, in a subdirectory of the existing test contracts such as `test/invariants/medusa/superc20/`
- keep the campaign in wonderland's fork of the repository, in its own feature branch, in which case the deliverable would consist primarily of:
    - a summary of the results, extending this document
    - PRs with extra unit tests replicating found issues to the main repo where applicable

## Contracts in scope

- [ ]  [OptimismMintableERC20Factory](https://github.com/defi-wonderland/optimism/blob/develop/packages/contracts-bedrock/src/universal/OptimismMintableERC20Factory.sol) (modifications to enable `convert` not yet merged)
- [ ]  [OptimismSuperchainERC20](https://github.com/defi-wonderland/optimism/blob/develop/packages/contracts-bedrock/src/L2/OptimismSuperchainERC20.sol1)
- [ ]  [OptimismSuperchainERC20Factory](https://github.com/defi-wonderland/optimism/pull/8/files#diff-09838f5703c353d0f7c5ff395acc04c1768ef58becac67404bc17e1fb0018517) (not yet merged)
- [ ]  [L2StandardBridgeInterop](https://github.com/defi-wonderland/optimism/pull/10/files#diff-56cf869412631eac0a04a03f7d026596f64a1e00fcffa713bc770d67c6856c2f) (not yet merged)

## Behavior assumed correct

- [ ]  inclusion of relay transactions
- [ ]  sequencer implementation
- [ ]  [OptimismMintableERC20](https://github.com/defi-wonderland/optimism/blob/develop/packages/contracts-bedrock/src/universal/OptimismMintableERC20.sol)
- [ ]  [L2ToL2CrossDomainMessenger](https://github.com/defi-wonderland/optimism/blob/develop/packages/contracts-bedrock/src/L2/L2CrossDomainMessenger.sol)
- [ ]  [CrossL2Inbox](https://github.com/defi-wonderland/src/L2/CrossL2Inbox.sol)

## Pain points

- existing fuzzing tools use the same EVM to run the tested contracts as they do for asserting invariants, tracking ghost variables and everything else necessary to provision a fuzzing campaign. While this is usually very convenient, it means that we can’t assert on the behaviour/state of *different* chains from within a fuzzing campaign. This means we will have to walk around the requirement of supertokens having the same address across all chains, and implement a way to mock tokens existing in different chains. We will strive to formally prove it in a unitary fashion to mitigate this in properties 0 and 1
- a buffer to represent 'in transit' messages should be implemented to assert on invariants relating to the non-atomicity of bridging from one chain to another. It is yet to be determined if it’ll be a FIFO queue (assuming ideal message ordering by sequencers) or it’ll have random-access capability to simulate messages arriving out of order

## Definitions

- *legacy token:*  an OptimismMintableERC20 or L2StandardERC20 token on the suprechain that has either been deployed by the factory after the liquidity migration upgrade to the latter, or has been deployed before it **but** added to factory’s `deployments` mapping as part of the upgrade. This testing campaign is not concerned with tokens on L1 or not listed in the factory’s `deployments` mapping.
- *supertoken:* a SuperchainERC20 contract deployed by the `OptimismSuperchainERC20Factory`

# Ecosystem properties

legend:

- `[ ]`: property not yet tested
- `**[ ]**`: property not yet tested, dev/research team has asked for extra focus on it
- `[X]`: tested/proven property
- `[~]`: partially tested/proven property
- `:(`: property won't be tested due to some limitation

## Unit test

| id  | milestone           | description                                                                                | halmos | medusa |
| --- | ---                 | ---                                                                                        | ---    | ---    |
| 0   | Factories           | supertoken token address does not depend on the executing chain’s chainID                  | [ ]    | [ ]    |
| 1   | Factories           | supertoken token address depends on remote token, name, symbol and decimals                | [ ]    | [ ]    |
| 2   | Liquidity Migration | convert() should only allow converting legacy tokens to supertoken and viceversa           | [ ]    | [ ]    |
| 3   | Liquidity Migration | convert() only allows migrations between tokens representing the same remote asset         | [ ]    | [ ]    |
| 4   | Liquidity Migration | convert() only allows migrations from tokens with the same decimals                        | [ ]    | [ ]    |
| 5   | Liquidity Migration | convert() burns the same amount of legacy token that it mints of supertoken, and viceversa | [ ]    | [ ]    |
| 25  | SupERC20            | supertokens can't be reinitialized                                                         | [ ]    | [x]    |

## Valid state

| id  | milestone | description                                                                    | halmos  | medusa |
| --- | ---       | ---                                                                            | ---     | ---    |
| 6   | SupERC20  | calls to sendERC20 succeed as long as caller has enough balance                | [x]     | [x]    |
| 7   | SupERC20  | calls to relayERC20 always succeed as long as the cross-domain caller is valid | **[~]** | [~]    |

## Variable transition

| id  | milestone           | description                                                                                       | halmos | medusa |
| --- | ---                 | ---                                                                                               | ---    | ---    |
| 8   | SupERC20            | sendERC20 with a value of zero does not modify accounting                                         | [x]    | [x]    |
| 9   | SupERC20            | relayERC20 with a value of zero does not modify accounting                                        | [x]    | [x]    |
| 10  | SupERC20            | sendERC20 decreases the token's totalSupply in the source chain exactly by the input amount       | [x]    | [x]    |
| 26  | SupERC20            | sendERC20 decreases the sender's balance in the source chain exactly by the input amount          | [ ]    | [x]    |
| 27  | SupERC20            | relayERC20 increases sender's balance in the destination chain exactly by the input amount        | [ ]    | [x]    |
| 11  | SupERC20            | relayERC20 increases the token's totalSupply in the destination chain exactly by the input amount | [x]    | [ ]    |
| 12  | Liquidity Migration | supertoken total supply only increases on calls to mint() by the L2toL2StandardBridge             | [x]    | [~]    |
| 13  | Liquidity Migration | supertoken total supply only decreases on calls to burn() by the L2toL2StandardBridge             | [x]    | [ ]    |
| 14  | SupERC20            | supertoken total supply starts at zero                                                            | [x]    | [x]    |
| 15  | Factories           | deploying a supertoken registers its remote token in the factory                                  | [ ]    | [ ]    |
| 16  | Factories           | deploying an OptimismMintableERC20 registers its remote token in the factory                      | [ ]    | [ ]    |

## High level

| id  | milestone           | description                                                                                                                                                           | halmos | medusa |
| --- | ---                 | ---                                                                                                                                                                   | ---    | ---    |
| 17  | Liquidity Migration | only calls to convert(legacy, super) can increase a supertoken’s  total supply across chains                                                                          | [ ]    | [ ]    |
| 18  | Liquidity Migration | only calls to convert(super, legacy) can decrease a supertoken’s  total supply across chains                                                                          | [ ]    | [ ]    |
| 19  | Liquidity Migration | sum of supertoken total supply across all chains is always <= to convert(legacy, super)- convert(super, legacy)                                                       | [ ]    | [~]    |
| 20  | SupERC20            | tokens sendERC20-ed on a source chain to a destination chain can be relayERC20-ed on it as long as the source chain is in the dependency set of the destination chain | [ ]    | [ ]    |
| 21  | Liquidity Migration | sum of supertoken total supply across all chains is = to convert(legacy, super)- convert(super, legacy) when all cross-chain messages are processed                   | [ ]    | [~]    |

## Atomic bridging pseudo-properties

As another layer of defense, the following properties are defined which assume bridging operations to be atomic (that is, the sequencer and L2Inbox and CrossDomainMessenger contracts are fully abstracted away, `sendERC20` triggering the `relayERC20` call on the same transaction)
It’s worth noting that these properties will not hold for a live system

| id  | milestone           | description                                                                                                                        | halmos | medusa |
| --- | ---                 | ---                                                                                                                                | ---    | ---    |
| 22  | SupERC20            | sendERC20 decreases sender balance in source chain and increases receiver balance in destination chain exactly by the input amount | [ ]    | [x]    |
| 23  | SupERC20            | sendERC20 decreases total supply in source chain and increases it in destination chain exactly by the input amount                 | [ ]    | [x]    |
| 24  | Liquidity Migration | sum of supertoken total supply across all chains is always equal to convert(legacy, super)- convert(super, legacy)                 | [ ]    | [~]    |

# Expected external interactions

- [x] regular ERC20 operations between any accounts on the same chain, provided by [crytic ERC20 properties](https://github.com/crytic/properties?tab=readme-ov-file#erc20-tests)

# Invariant-breaking candidates (brain dump)

here we’ll list possible interactions that we intend the fuzzing campaign to support in order to help break invariants

- [ ]  changing the decimals of tokens after deployment
- [ ]  `convert()` ing between multiple (3+) representations of the same remote token, by having different names/symbols

# Internal structure

## Medusa campaign

Fuzzing campaigns have both the need to push the contract into different states (state transitions) and assert properties are actually held. This defines a spectrum of function types:

- `handler_`: they only transition the protocol state, and don't perform any assertions.
- `fuzz_`: they both transition the protocol state and perform assertions on properties. They are further divided in:
    - unguided: they exist to let the fuzzer try novel or unexpected interactions, and potentially transition to unexpectedly invalid states
    - guided: they have the goal of quickly covering code and moving the protocol to known valid states (eg: by moving funds between valid users)
- `property_`: they only assert the protocol is in a valid state, without causing a state transition. Note that they still use assertion-mode testing for simplicity, but could be migrated to run in property-mode.
