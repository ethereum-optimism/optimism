---
title: Security Policy, Vulnerability Reporting, and Bug Bounties
lang: en-US
---


## Reporting in the decentralized context

It's important to remember that the OP Stack is a decentralized software development stack built by the Optimism Collective. Different components of the OP Stack may be maintained by different teams that have different reporting processes. **This page describes general best practices for reporting bugs and provides specific reporting guidelines for the OP Stack code contained within the [ethereum-optimism](https://github.com/ethereum-optimism) GitHub organization**.

## Reporting bugs and vulnerabilities

::: danger 🚫 How NOT to disclose a vulnerability 

 Do *not* disclose vulnerabilities publicly or by executing them against a production network. If you do, will you not only be putting users at risk, but you will forfeit your right to a reward. Always follow the appropriate reporting pathways as described below.

- Do *not* disclose the vulnerability publicly, for example by filing a public ticket.
- Do *not* test the vulnerability on a publicly available network, either the testnet or the mainnet.

:::

### OP Stack bounty programs

The security of OP Stack smart contracts and blockchain infrastructure is paramount. Below are the various OP Stack-related bug bounty programs, as well as how to reach out if your bug is not covered by an existing bounty.

#### Optimism Mainnet bounty program

Optimism Mainnet is covered by a comprehensive [bug bounty program on Immunefi](https://immunefi.com/bounty/optimism/), which has already resulted in one of the [largest bounty payouts ever](https://medium.com/ethereum-optimism/disclosure-fixing-a-critical-bug-in-optimisms-geth-fork-a836ebdf7c94). In the listing you can find all the information relating to assets in scope, reporting, and the payout process. Because Optimism Mainnet is currently the primary user of the OP Stack, bugs in OP Stack software can generally be reported via the Optimism Mainnet bounty program.

#### Unscoped bugs

If you think you have found a significant bug or vulnerabilities in OP Stack smart contracts, infrastructure, etc., even if that component is not covered by an existing bug bounty, please report it to via the [Optimism Mainnet Immunefi program](https://immunefi.com/bounty/optimism/). The impact of any and all reported issues will be considered and the program has previously rewarded security researchers for bugs not within its stated scope.

### Other vulnerabilities

For vulnerabilities in any websites, email servers, or other non-critical infrastructure within the OP Stack, please email [OP Labs](https://www.oplabs.co/) at [security@oplabs.co](mailto:security@oplabs.co) and include detailed instructions for confirming and reproducing the vulnerability.

## Vulnerability disclosure

Each OP Stack component maintainer may determine its own process for vulnerability disclosure. However, the following describes a recommended process for disclosure that is currently in use by [OP Labs](https://www.oplabs.co/).

In the event that an OP Stack component maintainer learns of a critical security vulnerability, the maintainer reserves the right to silently fix it without immediately publicly disclosing the existence of nature of the vulnerability.

In such a scenario, the disclosure process used by [OP Labs](https://www.oplabs.co/) is as follows:

1. Silently fix the vulnerability and include the fix in release X.

1. After 4-8 weeks, disclose that release X contained a security fix.

1. After an additional 4-8 weeks, publish details of the vulnerability, along with credit to the reporter (with express permission from the reporter).

Alongside this policy, maintainers also reserve the right to:

- Bypass this policy and publish details on a shorter timeline.
- Directly notify a subset of downstream users prior to making a public announcement.

This policy is based the [Geth](https://geth.ethereum.org/) team’s [silent patch policy](https://geth.ethereum.org/docs/vulnerabilities/vulnerabilities#why-silent-patches).