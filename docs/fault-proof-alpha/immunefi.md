# Fault Proof Alpha Bounty

The fault proof alpha system will be deployed to the Goerli testnet as a sidecar to the current system. During the alpha, the outcome of fault dispute games will have no influence on the official bridge contracts nor the official `L2OutputOracle`.
During this early phase of ongoing development, we invite security researchers and developers to engage with the system and attempt to break its current components.

The current system is not production ready, however the core infrastructure for creating an instruction trace ([Cannon][cannon] + the [`op-program`][op-program]), the off-chain challenge agent ([`op-challenger`][op-challenger]),
and the on-chain infrastructure for the [Dispute Game][dispute-game] are all in place.

For the Fault Proof Alpha security review, we've pinned `546fb2c7a5796b7fe50b0b7edc7666d3bd281d6f` as the commit hash in the monorepo. This commit hash was the head of the `develop` branch at the time of the alpha's launch. All
security reviews and PoCs should be derived from this commit hash, as the contracts and off-chain agents are being updated frequently at this stage of development.

### Resources

> **Note**
> Prior to moving forward, we recommended reading into the technical documentation for the components of Fault Proof Alpha.

* [Cannon][cannon] & [Cannon Contracts][cannon-contracts]
    * [Cannon VM Specs][cannon-vm-specs]
* [`op-program`][op-program]
    * [Fault Proof Specs][fault-proof-specs]
* [Dispute Game][dispute-game]
    * [Fault Dispute Game Specs][fault-dispute-specs]
* [`op-challenger`][op-challenger]

## Known Issues
The alpha system is not prepared for mainnet, and as such, there are a number of known issues that we are working on fixing and components of the system that must be improved prior to it being sustainable.

1. DoS attacks are currently likely to occur due to the lack of bonds in the alpha system as well as the lack of an extra layer of bisection in the dispute game to reduce the running time of [Cannon][cannon]. It is possible to
    DoS the network of honest challengers by creating a large number of invalid challenges.
1. Limitations of pre-image oracle inputs. The pre-image oracle currently does not support the full specified set of inputs.
    In particular, arbitrary pre-image value size and preimage key types other than `local` (type 1) `keccak256` (type 2) are not supported.
    The pre-image value size is limited to what the current oracle can verify: gas and calldata limits constrain this more than the pre-images are, rendering some state-transitions that include large pre-images impossible to prove with the oracle as-is. This does not affect most proofs. L1/L2 activity that breaks this pre-image size limitation does not qualify for the bounty.
    The remaining pre-images types are not supported, as the types are not used by the current op-program, but may be supported for future program proving, e.g. type 3 for application-specific proofs, and new types 4, 5, etc. for ethereum extensions like SHA2 and KZG point verification.
1. Non-standard rollup chain configurations do not qualify. Output roots span a range of L2 blocks derived from a range of L1 blocks, built on top of the previous agreed upon L2 state. By breaking time or input-range chain parameters, the proof program may not complete or fail in undefined ways.
### Reviewer Notes
1. **Any bug report without a proof-of-concept in the form of a test in `op-e2e` will not be considered a valid bug report.**
    1. A guide on creating an e2e test with an invalid output proposal to dispute can be found [here][invalid-proposal-doc].
1. Exploits against the alpha system that take advantage of the aforementioned issues will not be considered valid bug reports.
1. The [AlphabetVM][alphabet-vm] is not equivalent to the MIPS thread context in behavior. Bug reports submitted against the [AlphabetVM][alphabet-vm] will not be considered valid bug reports, this mock VM is used solely for testing.

### Plans for the next iteration
Going past alpha, we have a number of plans for improving the system and fixing some of the aforementioned issues in preparation for full integration with the current system. These include:
1. Including an extra layer of bisection over output roots prior to beginning execution trace bisection, enabling the off-chain challenge agents to only need to run [Cannon][cannon] over a single block rather than a string of blocks. This will heavily reduce the hardware cost of running the off
   chain challenge agent and provide an upper bound on what Cannon will have to execute, allowing for sparse proposals.
1. Adding bonds to the system to preserve incentive compatibility. In the alpha, defenses of the honest L2 state are not incentivized, which also means that attacks on the honest L2 state are not disincentivized. Adding bonds to each claim
   made in the dispute game will preserve the incentives of the system as well as make it more costly to attack.
1. Improving the [Dispute Game][dispute-game]'s resolution algorithm to reduce the number of interactions that the off-chain challenge agents need to have with the on-chain dispute game. This will reduce the cost of running the off-chain challenge
   agent, ensure that an honest challenger's participation always results in a profitable move, and possibly prevent the need for challengers to respond to every invalid claim within the game.
1. The fault proof system will be integrated into the bridge contracts, specifically the `OptimismPortal`, in order to enable the system to be used in production and verify the correctness of output roots that withdrawals are proven against.
1. The pre-image oracle limitations related to pre-image size and typing support will be addressed to cover the full scope of valid onchain L1 and L2 activity.
## Bounty Scope
The scope of the bounty is limited to the fault proof alpha system. This includes the following components, in order of security review priority:
1. **Cannon**: The [Cannon][cannon] binary and its dependencies, as defined in the monorepo. The archived legacy version, and alternative implementations, do not qualify.
1. **op-program**: The [`op-program`][op-program] binary and its dependencies.
1. **Smart Contracts**
    1. The [Cannon][cannon-contracts] contracts and their dependencies.
    1. The [Dispute Game][dispute-game] and their dependencies.
1. **op-challenger**: The [`op-challenger`][op-challenger] binary and its dependencies.

As mentioned above in the "[Plans for the next iteration](#plans-for-the-next-iteration)" section, there will soon be a number of large architectural changes to the [dispute smart contracts][dispute-game]
as well as the [`op-challenger`][op-challenger] in order to support the features that will bring the system to a production ready state. During this time, it is unlikely that [Cannon][cannon], the [Cannon contracts][cannon-contracts],
or the [`op-program`][op-program] will change significantly, and as such, we recommend focusing efforts primarily on these components.

There are several key invariants that must be maintained in order for the system to be considered secure. A bounty report must demonstrate a bug which breaks one of these invariants.
1. **Cannon**
    1. [Cannon][cannon]'s `mipsevm` must be functionally equivalent to the [MIPS thread context][cannon-contracts] implemented in Solidity. Any disparities that result in different `op-program` execution are a bug.
        1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must produce the same output given an identical setup state and input data.
    1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must produce a deterministic output given an identical setup state and input data.
    1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must never panic on a state transition with honest input data / setup state.
        1. Note: There are a number of instructions from MIPS, and system calls in Linux, that Cannon does not support. Specifically, this invariant covers panic conditions within the realm of supported instructions and valid honest input data / setup state where cannon otherwise should have completed execution and produced a valid/invalid opinion about the state transition. The op-program may contain "dead code", non-reachable invalid instructions that do not affect the output.
    1. The `PreimageOracle` contract's local data storage must not be able to be corrupted by an external party.
1. **op-program**
    1. The [`op-program`][op-program] must produce a deterministic output given an identical setup state and input data.
1. **Dispute Game Contracts**
    1. Assuming the presence of an `honest challenger` (defined by the behavior of the [`op-challenger`][op-challenger]) participating within the game, the `FaultDisputeGame` utilizing the `MIPS` VM **must always** resolve favorably towards the honest L2 state.
        1. *Note (1)*: The presence of an honest challenger implies that the honest challenger has exhausted all moves it would have made - any game where the honest challenger was unable to exhaust its move set can resolve unfavorably to their desired outcome. The aforementioned DoS vector is one such reason the honest challenger may not perform all its moves.
1. **op-challenger**
    1. The honest `op-challenger` must never make a claim that does not support the honest outcome of the dispute game (i.e., the outcome which favors the honest L2 state being considered canonical).
        1. *Note:* Because of the rules in the current solving / resolution mechanism, the challenger will counter all claims that have a different view of the root claim's validity. While this is an inefficiency, it is not considered a violation of this invariant, as this behavior is necessary to ensure that all invalid claims have been countered.

Bug reports in the form of a proof-of-concept `op-e2e` test that demonstrates a violation of any of the above invariants will be considered valid bug reports and eligible for a reward*.

* All proof of concept reports should be configured to run against the parameters of the system deployed on the `goerli` testnet or with the environment defined in the `op-e2e` `faultproof_test.go` file. Bug reports that otherwise violate the above invariants
but use custom configurations will be assessed on a case by case basis, and their validity is not guaranteed.

### Bounty Rewards
See our bounty program on [Immunefi][immunefi] for information regarding reward sizes.

<!-- LINKS -->
[cannon]: https://github.com/ethereum-optimism/optimism/tree/develop/cannon
[cannon-vm-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/cannon-fault-proof-vm.md
[dispute-game]: https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/src/dispute
[fault-dispute-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/fault-dispute-game.md
[cannon-contracts]: https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/src/cannon
[op-program]: https://github.com/ethereum-optimism/optimism/tree/develop/op-program
[op-challenger]: https://github.com/ethereum-optimism/optimism/tree/develop/op-challenger
[alphabet-vm]: https://github.com/ethereum-optimism/optimism/blob/c1cbacef0097c28f999e3655200e6bd0d4dba9f2/packages/contracts-bedrock/test/FaultDisputeGame.t.sol#L977-L1005
[fault-proof-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/fault-proof.md
[immunefi]: https://immunefi.com/bounty/optimism/
[invalid-proposal-doc]: https://github.com/ethereum-optimism/optimism/blob/develop/docs/fault-proof-alpha/invalid-proposals.md
