---
title: Security FAQs
lang: en-US
---

::: warning 🚧 Work in Progress

The OP Stack is a work in progress. Constantly pushing to improve the overall security and decentralization of the OP Stack is a top priority. 

:::

## Security in the decentralized context

The OP Stack is a decentralized development stack that powers Optimism. Components of the OP Stack may be maintained by various different teams within the Optimism Collective. It is generally easier to talk about the security model of specific chains built on the OP Stack rather than the security model of the stack itself. **The OP Stack security baseline is to create safe defaults while still giving developers the flexibility to make modifications and extend the stack.**

## FAQ

### Is every OP Stack chain safe?

The security model of an OP Stack based blockchain depends on the modules used for its components. Because of the flexibility provided by OP Stack, it is always possible to set up an insecure blockchain using OP Stack components. **The goal of the OP Stack is to provide safe defaults.**

Please also keep in mind that just like any other system, **the OP Stack may contain unknown bugs** that could lead to the loss of some or all of the assets held within an OP Stack based system. [Many components of the OP Stack codebase have been audited](https://github.com/ethereum-optimism/optimism/tree/develop/technical-documents/security-reviews) but **audits are not a stamp of approval** and **a completed audit does not mean that the audited codebase is free of bugs.** It’s important to understand that using the OP Stack inherently exposes you to the risk of bugs within the OP Stack codebase.

### Is the OP Stack safe to modify?

As with anything, modify the OP Stack at your own risk. There is no guarantee that modifications to the stack will be safe. If you aren’t entirely sure about what you’re doing, stick with the safer defaults that the OP Stack provides. At the moment, the OP Stack is not particularly amenable to modifications and **you should not expect any technical support for modifications that fall outside of the standard Rollup configuration of the stack**.

### Can I use fault proofs?

**Not yet.** The OP Stack does not currently have a fault proof system. **Note that fault proofs do not meaningfully improve the security of a system if that system can be upgraded within the 7 day challenge window (”fast upgrade keys”)**. A system with fast upgrade keys is fully dependent on the upgrade keys for security. 

Fault proofs are a key milestone and top priority for the OP Stack. In the meantime, the OP Stack can be shipped with several other excellent security options for systems that want to improve security before fault proofs are available in production.

### How can I help make the OP Stack more secure?

One of the easiest ways to help secure the OP Stack is to look for bugs and vulnerabilities. [Optimism Mainnet, a user of the OP Stack, has one of the biggest bug bounties (ever)](https://immunefi.com/bounty/optimism/). You can earn up to $2,000,042 by finding critical bugs in the Optimism Mainnet codebase (and by extension the OP Stack).

Don’t forget that the OP Stack is a decentralized development stack. Anyone can start to contribute to the OP Stack by building software that follows [the stack’s design principles](../understand/design-principles.md). You can always help make the OP Stack more secure by building components, like alternative client or proof implementations, that users of the OP Stack can take advantage of.

### Where do I report bugs?

[View the Security Policy for details about reporting vulnerabilities and available bug bounty programs](./policy.md)