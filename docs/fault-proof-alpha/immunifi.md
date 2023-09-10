# Fault Proof Alpha Bounty

The fault proof alpha system will be deployed to the Goerli testnet as a sidecar to the current system. During the alpha, the outcome of fault dispute games will have no influence on the official bridge contracts nor the official `L2OutputOracle`.
During this early phase of ongoing development, we invite security researchers and developers to engage with the system and attempt to break its current components.

The current system is not production ready, however the core infrastructure for creating an instruction trace ([Cannon][cannon] + the [`op-program`][op-program]), the off-chain challenge agent ([`op-challenger`][op-challenger]),
and the on-chain infrastructure for the [Dispute Game][dispute-game] are all in place.

## Known Issues
The alpha system is not prepared for mainnet, and as such, there are a number of known issues that we are working on fixing and components of the system that must be improved prior to it being sustainable.

1. DoS attacks are currently likely to occur due to the lack of bonds in the alpha system as well as the lack of an extra layer of bisection in the dispute game to reduce the running time of [Cannon][cannon]. It is possible to
    DoS the network of honest challengers by creating a large number of invalid challenges.

### Reviewer Notes
1. **Any bug report without a PoC in the form of a test in `op-e2e` will not be considered a valid bug report.**
    1. *todo*: Provide Adrian's example op-e2e test
1. Exploits against the alpha system that take advantage of the aforementioned issues will not be considered valid bug reports.
1. The [AlphabetVM][alphabet-vm] is not equivalent to the MIPS thread context in behavior. Bug reports submitted against the [AlphabetVM][alphabet-vm] will not be considered valid bug reports, this mock VM is used solely for testing.

### Plans for the next iteration
Going past alpha, we have a number of plans for improving the system and fixing some of the aforementioned issues in preparation for full integration with the current system. These include:
1. Including an extra layer of bisection over output roots, enabling the off-chain challenge agents to only need to run [Cannon][cannon] over a single block rather than a string of blocks. This will heavily reduce the hardware cost of running the off
   chain challenge agent, which mitigates the DoS vector mentioned above.
1. Adding bonds to the system to preserve incentive compatibility. In the alpha, defenses of the honest L2 state are not incentivized, which also means that attacks on the honest L2 state are not disincentivized. Adding bonds to each claim
   made in the dispute game will preserve the incentives of the system as well as make it more costly to attack.
1. Improving the [Dispute Game][dispute-game]'s resolution algorithm to reduce the number of interactions that the off-chain challenge agents need to have with the on-chain dispute game. This will reduce the cost of running the off-chain challenge
   agent, ensure that an honest challenger's participation always results in a profitable move, and possibly prevent the need for challengers to respond to every invalid claim within the game.
1. The fault proof system will be integrated into the bridge contracts, specifically the `OptimismPortal`, in order to enable the system to be used in production and verify the correctness of output roots that withdrawals are proven against.

## Bounty Scope
The scope of the bounty is limited to the fault proof alpha system. This includes the following components, in order of security review priority:
1. **Cannon**: The [Cannon][cannon] binary and its dependencies.
1. **op-program**: The [`op-program`][op-program] binary and its dependencies.
1. **Smart Contracts**
    1. The [Cannon][cannon-contracts] contracts and their dependencies.
    1. The [Dispute Game][dispute-game] and their dependencies.
1. **op-challenger**: The [`op-challenger`][op-challenger] binary and its dependencies.

As mentioned above in the "[Plans for the next iteration](#plans-for-the-next-iteration)" section, there will soon be a number of large architectural changes to the [dispute smart contracts][dispute-game]
as well as the [`op-challenger`][op-challenger] in order to support the features that will bring the system to a production ready state. During this time, it is unlikely that [Cannon][cannon], the [Cannon contracts][cannon-contracts],
or the [`op-program`][op-program] will change significantly, and as such, we recommend focusing efforts primarily on these components.

There are several key invariants that must be maintained in order for the system to be considered secure:
1. **Cannon**
    1. [Cannon][cannon]'s `mipsevm` must be functionally equivalent to the [MIPS thread context][cannon-contracts] implemented in Solidity. Any disparities in behavior is considered a bug.
        1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must produce the same output given an identical setup state and input data.
    1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must produce a deterministic output given an identical setup state and input data.
    1. Both [Cannon][cannon] and the on-chain [MIPS thread context][cannon-contracts] must never panic on a valid state transition.
    1. The `PreimageOracle` contract's local data storage must not be able to be corrupted by an external party.
1. **op-program**
    1. The [`op-program`][op-program] must produce a deterministic output given an identical setup state and input data.
1. **Dispute Game Contracts**
    1. Assuming the presence of an `honest challenger` (defined by the behavior of the [`op-challenger`][op-challenger]) participating within the game, the `FaultDisputeGame` utilizing the `MIPS` VM **must always** resolve favorably towards the honest L2 state.
        1. *Note*: The only exception to this invariant in the alpha dispute game (with a `MIPS` VM) is the aforementioned DoS vector. This is not considered a valid bug report, however any other violation of this invariant is.
1. **op-challenger**
    1. The honest `op-challenger` must never make a claim that does not support the honest outcome of the dispute game (i.e., the outcome which favors the honest L2 state being considered canonical).

Any bug reports in the form of a PoC `op-e2e` test that demonstrates a violation of any of the above invariants will be considered valid bug reports and elligible for a reward.

### Resources
* [Cannon][cannon] & [Cannon Contracts][cannon-contracts]
    * [Cannon VM Specs][cannon-vm-specs]
* [`op-program`][op-program]
    * [Fault Proof Specs][fault-proof-specs]
* [Dispute Game][dispute-game]
    * [Fault Dispute Game Specs][fault-dispute-specs]
* [`op-challenger`][op-challenger]

### Bounty Rewards
*todo* - I don't define these.

<!-- LINKS -->
[cannon]: https://github.com/ethereum-optimism/optimism/tree/develop/cannon
[cannon-vm-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/cannon-fault-proof-vm.md
[dispute-game]: https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/src/dispute
[fault-dispute-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/fault-dispute-game.md
[cannon-contracts]: https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/src/cannon
[op-program]: https://github.com/ethereum-optimism/optimism/tree/develop/op-program
[op-challenger]: https://github.com/ethereum-optimism/optimism/tree/develop/op-challenger
[alphabet-vm]: https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/test/FaultDisputeGame.t.sol#L977-L1005
[fault-proof-specs]: https://github.com/ethereum-optimism/optimism/blob/develop/specs/fault-proof.md
