---
title: Meta Transactions
lang: en-US
---

## What are those?

Meta-transactions let users sign transactions that are then submitted (and the gas paid for) by somebody else. 
Optimism is a good place for meta-transactions because the low gas costs make it possible to explore business models that allow for payment for transactions by other means.

Here are some example use cases for meta-transactions:

- **On boarding**. 
  Users who aren't committed to web3 yet need to pay and go through [KYC](https://www.thalesgroup.com/en/markets/digital-identity-and-security/banking-payment/issuance/id-verification/know-your-customer) before they can do anything.
  This is a bad initial experience that can cause people to decide they prefer to investigate something else, especially for non-

- **Privacy**.
  It's easy to use a new address for privacy purposes.
  It is a lot harder to transfer ETH to the new address privately without creating a connection between it and your identity.
  
- **Transaction payment by other means**.
  With meta-transactions you can collect payment for transactions either using a different token (ERC-20) or a off-chain means (for example a credit card).

## OpenGSN

The [Gas Station Network](https://opengsn.org/) is distributed infrastructure for meta-transaction that allows you to create your own relay, or pay other relays to relay your users' traffic.
Relays have to post a bond, which they lose if they attempt to censor transactions (by pretending to accept them without sending them on chain).
See here for [their documentation](https://docs.opengsn.org/).

### Supported networks:

- [Optimism mainnet](https://docs.opengsn.org/networks/optimism/optimism.html)
- [Optimism Goerli](https://docs.opengsn.org/networks/optimism/goerli-optimism.html)


## Gelato

[Gelato](https://docs.gelato.network/developer-services/relay/what-is-relaying) uses a list of white-listed executors to relay transactions.

### Supported networks:

- Optimism mainnet
- Optimism Goerli
