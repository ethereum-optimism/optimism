---
title: Welcome to the OP Stack
lang: en-US
---

**The OP Stack is the standardized, shared, and open-source development stack that powers Optimism, maintained by the Optimism Collective.**

::: tip Staying up to date

[Stay up to date on the Superchain and the OP Stack by subscribing to the Optimism newsletter](https://optimism.us6.list-manage.com/subscribe/post?u=9727fa8bec4011400e57cafcb&id=ca91042234&f_id=002a19e3f0).

:::

The OP Stack consists of the many different software components managed and maintained by the Optimism Collective that, together, form the backbone of Optimism. 
The OP Stack is built as a public good for the Ethereum and Optimism ecosystems.

## The OP Stack powers Optimism

The OP Stack is the set of software that powers Optimism — currently in the form of the software behind Optimism Mainnet and eventually in the form of the Optimism Superchain and its governance.

With the advent of the Superchain concept, it has become increasingly important for Optimism to easily support the secure creation of new chains that can interoperate within the proposed Superchain ecosystem. 
As a result, the OP Stack is primarily focused around the creation of a shared, high-quality, and fully open-source system for creating new L2 blockchains. 
By coordinating on shared standards, the Optimism Collective can avoid rebuilding the same software in silos repeatedly.

Although the OP Stack today significantly simplifies the process of creating L2 blockchains, it’s important to note that this does not fundamentally define what the OP Stack **is**. 
The OP Stack is *all* of the software that powers Optimism. 
As Optimism evolves, so will the OP Stack.

**The OP Stack can be thought of as software components that either help define a specific layer of the Optimism ecosystem or fill a role as a module within an existing layer.**
Although the current heart of the OP Stack is infrastructure for running L2 blockchains, the OP Stack theoretically extends to layers on top of the underlying blockchain including tools like block explorers, message passing mechanisms, governance systems, and more.

Layers are generally more tightly defined towards the bottom of the stack (like the Data Availability Layer) but become more loosely defined towards the top of the stack (like the Governance Layer).

## The OP Stack today

Optimism Bedrock is the current iteration of the OP Stack. 
The Bedrock release provides the tools for launching a production-quality Optimistic Rollup blockchain. 
At this point in time, the APIs for the different layers of the OP Stack are still tightly coupled to this Rollup configuration of the stack. 

If you'd like to learn more about the current state of the OP Stack, check out [the page describing the Bedrock release](/docs/releases/bedrock/README.md).

The OP Stack of today was built to support [the Optimism Superchain](./docs/understand/explainer.md), a proposed network of L2s that share security, communication layers, and a common development stack (the OP Stack itself). 
The Bedrock release of the OP Stack makes it easy to spin up an L2 that will be compatible with the Superchain when it launches. 
If you'd like to launch a Superchain-ready L2, check out our guide for running a chain based on the Bedrock release of the OP Stack.

It is possible to modify components of the OP Stack to build novel L2 systems. 
If you're interested in experimenting with the OP Stack, check out [the OP Stack Hacks section of this site](/docs/build/hacks.md). 
Please note that, as of the Bedrock release, the OP Stack is *not* designed to support these modifications and you will very much be *hacking* on the codebase. 
As a result, **you should, for the moment, expect limited (if any) developer support for OP Stack Hacks.** 
OP Stack Hacks will likely make your chain incompatible with the Optimism Superchain. 
Have fun, but at your own risk and **stick to the Bedrock release if you're looking to join the Superchain!**

## The OP Stack tomorrow

The OP Stack is an evolving concept. 
As Optimism grows, so will the OP Stack. 
Today, the Bedrock Release of the OP Stack simplifies the process of deploying new L2 Rollups. 
As work on the stack continues, it should become easier to plug in and configure different modules. 
As the Superchain (link) begins to take shape, the OP Stack can evolve alongside it, to include the message-passing infrastructure that allows different chains to interoperate seamlessly. 
At the end of the day, the OP Stack becomes what Optimism needs.

## Dive Deeper into the OP Stack

Ready to dive into the world of the OP Stack?

- If you’re interested in learning more about the current release of the OP Stack, check out the Bedrock Release page.
- If you’re interested in understanding the OP Stack in more depth, start with the [Design Principles](/docs/understand/design-principles.md) and [Landscape Overview](/docs/understand/landscape.md).
- If you're excited to join the Superchain, launch your first Superchain-ready L2 with our [Getting Started guide](/docs/build/getting-started.md) or dive directly into the OP Stack codebase to learn more.

The OP Stack is the next frontier for Ethereum. You’re already here, so what are you waiting for?
