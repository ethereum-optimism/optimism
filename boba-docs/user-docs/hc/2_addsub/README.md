# Add-Subtract Example

The purpose of our next smart contract example is to explore more in-depth technical detail in a smart contract using Hybrid Compute. That depth includes writing and calling an off-chain handler. This contract can do two things:

1. Intentionally burn (or "waste") ETH gas to simulate a more true-to-life Hybrid Compute request.
2. Increment a counter variable each time the contract is called based on certain inputs and errors.

The contract is also an extension of an [example provided](https://github.com/eth-infinitism/account-abstraction/blob/develop/contracts/test/TestCounter.sol) by the upstream Account Abstraction framework on which we based our Hybrid Compute extensions.

You can find the needed `HybridAccount` contract along with its dependencies in our [repository](https://github.com/bobanetwork/account-abstraction-hc/contracts/samples/HybridAccount.sol). Additionally, the contract is pulled in as a submodule when you checkout our [`rundler-hc` repository](https://github.com/bobanetwork/rundler-hc/crates/types/contracts/lib/account-abstraction/).

The code blocks in the following sections are edited to show the lines being discussed. We recommend opening another window containing the full contract so that you can see them in the proper context.
