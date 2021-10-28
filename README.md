# The Optimistic Ethereum Spec

This repository holds the open-source specification for Optimistic Ethereum, an ultra-minimal optimistic rollup protocol that maintains 1:1 compatibility with Ethereum.

## Work in Progress

Please note that this specification is currently heavily under construction.
You will find that several pages are incomplete or [under discussion](https://github.com/ethereum-optimism/optimistic-specs/issues).

## Specification structure

- [Introduction](./introduction.md)
- [Overview](./overview.md)
- [Roadmap](./roadmap.md)
- Components:
  - [Layer 1 Contracts](./components/layer1.md)
  - [Rollup Client](./components/rollup_client.md)
  - [Execution Engine](./components/exec_engine.md)
  - [Batch Submitter](./components/batch_submitter.md)
  - [Witness Generator](./components/witness_gen.md)
  - [Challenge Agent](./components/challenge_agent.md)

## Design Philosophy

We believe that **the best optimistic rollup design needs to be minimal, open, and accessible.**

### Minimalism

Ethereum-focused optimistic rollups should be minimal to best take advantage of the battle-tested infrastructure (like Geth) that already runs Ethereum.
An ideal optimistic rollup design should be representable as a *diff* against Ethereum client software.
We imagine a world in which any Ethereum client can, with only minor modifications, participate in an Optimistic Ethereum network.

### Openness

We think it's time to coordinate the Ethereum community around a well-specified optimistic rollup design.
We acknowledge that this is only possible if the design process remains open to the feedback of the many teams already working on optimistic rollup architectures.
We aim to make this both this specification and the process by which this specification is built available to anyone interesting in building their own ORU system.

Anyone interested in contributing to this specification should refer to the [Contributing](#contributing) section.
You will find multiple options for contributing to this project.

This repository is distributed under the [Creative Commons Zero v1.0 Universal](https://github.com/ethereum-optimism/optimistic-specs/blob/main/LICENSE) license which dedicates this work to the public domain.
An MIT licensed implementation of this protocol can be found [here](https://github.com/ethereum-optimism/optimism).

### Accessibility

Users, developers, and protocol designers need to be confident that a given optimistic rollup is robust and secure.
We believe that this confidence can only truly come from an accessible specification and codebase that developers can reasonably be expected to understand.
Without this accessibility we'll always fundamentally have to trust the knowledge and competence of a very small group of core developers, a fact antithetical to the ideal decentralized nature of these systems.

## Contributing
### Basic Contributions
Contributing to the Optimistic Ethereum specification is easy.
You'll find a list of open questions and active research topics over on the [Fellowship of Ethereum Magicians](https://ethereum-magicians.org) forum.
Specific tasks and TODOs can be found on the [Issues](https://github.com/ethereum-optimism/optimistic-specs/issues) page.
You can edit content or add new pages by creating a [Pull Request](https://github.com/ethereum-optimism/optimistic-specs/pulls).

### R&D Calls
We hold weekly R&D calls that are open to anyone interested in contributing to the Optimistic Ethereum spec.
Contact [@karlfloersch](https://twitter.com/karl_dot_tech/), [@protolambda](https://github.com/protolambda/), or [@kelvinfichter](https://twitter.com/kelvinfichter) if you'd like to join these calls.
Please note that these calls may be recorded and shared publicly (we will ask for consent before recording).

## License

CC0 1.0 Universal, see [`LICENSE`](./LICENSE) file.
