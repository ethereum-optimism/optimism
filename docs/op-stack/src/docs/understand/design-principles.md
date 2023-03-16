---
title: Design Principles for USEful Software
lang: en-US
---


::: tip The OP Stack is USEful software

The OP Stack is a set of software components for building L2 blockchain ecosystems, built by the Optimism Collective to power Optimism. 
Components to be added to the OP Stack should be built according to three key design principles: 
- **U**tility
- **S**implicity
- **E**xtensibility. 

Software that follows these principles is **USE**ful software for the Optimism Collective!

::: 

## Utility

For something to be part of the OP Stack, it should help power the Optimism Collective. 
This condition helps guide the type of software that can be included in the stack. 
For instance, a powerful open-source block explorer that makes it easier for users to inspect [the Superchain](https://app.optimism.io/superchain/) would be a great addition to the OP Stack.

Although utility is important for inclusion in the OP Stack, you shouldn’t be afraid to experiment. 
Do something crazy. 
Build something that’s never been built before, even if it doesn’t have any clear utility. Make a blockchain for Emojis, or whatever. Have fun! 

## Simplicity

Complex code does not scale. 
Code that makes it into the OP Stack should be simple.

Simplicity reduces engineering overhead, which in turn means the Collective can spend its time working on new features instead of re-creating existing ones. 
The OP Stack prefers to use existing battle-tested code and infrastructure where possible. 
The most visible example of this philosophy in practice is the choice to use Geth as the OP Stack’s default execution engine.

When dealing with critical infrastructure, simplicity is also security and maintainability. 
Every line of code written is an opportunity to introduce bugs and vulnerabilities. 
A simple protocol means there's less code to write and, as a result, less surface area for potential mistakes. 
A clean and minimal codebase is also more accessible to external contributors and auditors. 
All of this serves to maximize the security and correctness of the OP Stack.

## Extensibility

Good OP Stack code is inherently open, collaborative, and extensible. 
Collaboration allows us to break out of siloed development. 
Collaboration allows us spend more time building on top of one another's work and less time rebuilding the same components over and over again. 
Collaboration is how we win, *together*.

Extensible code should be designed with the mindset that others will want to build with and on top of that code. 
In practice, this means that the code should be open source (under a permissive license), expose clean APIs, and generally be modular such that another developer can relatively easily extend the functionality of the code. 
Extensibility is a key design principle that unlocks the superpower of collaboration within the Optimism Collective ecosystem.

## Contributing to the OP Stack

The OP Stack is a decentralized software stack that anyone can contribute to. 
If you're interested in contributing to the OP Stack, check out [the Contributing section of these docs](../contribute.md).
Of course, software that has impact for the Optimism Collective can receive [Retroactive Public Goods Funding](https://app.optimism.io/retropgf). 
Build for the OP Stack — get rewarded for writing great open source software. What's not to love?
