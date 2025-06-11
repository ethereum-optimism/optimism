# Fees

Fees on Boba are, for the most part, significantly lower than L1s. The cost of every transaction is the sum of two values:

1. Your L2 (execution) fee, and
2. Your L1 (security) fee.

At a high level, the L2 fee is the cost to execute your transaction in L2 and the L1 fee is the estimated cost to store your transaction's metadata on the canonical L1 chain from which the L2 blocks are derived.

Boba Anchorage is derived from Optimism Bedrock and inherits its fee framework. Details may be found [here](https://docs.optimism.io/stack/transactions/fees).

Prior to the Anchorage update, the Boba network offered an option to pay fees using the Boba token instead of ETH. Post-Anchorage, this is no longer provided by the core network but will instead be offered through an ERC-4337 Account Abstraction framework. For an overview of the AA paymaster concept, developers may refer to resources such as [this one](https://www.alchemy.com/blog/account-abstraction-paymasters).

To obtain ETH and BOBA on Boba Network you can deposit or bridge via [the gateway](https://gateway.boba.network) on both Sepolia or Mainnet.
