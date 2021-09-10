# The Optimistic Ethereum Spec

This repository holds the open-source specification for Optimistic Ethereum, an ultra-minimal optimistic rollup protocol that maintains 1:1 compatibility with Ethereum.

## Work in Progress
Please note that this specification is currently heavily under construction.
You will find that several pages are incomplete or [under discussion](https://github.com/ethereum-optimism/optimistic-specs/issues).

## About the project

Early designs for Optimistic Ethereum were spearheaded by [Optimism](https://optimism.io/), which actively maintains an implementation of the protocol at the [optimism monorepo](https://github.com/ethereum-optimism/optimism).
Recent versions of the protocol have been vastly simplified to the point that we can now envision a future in which Optimistic Ethereum can act as a standardized optimistic rollup design.
We aim to make this specification open and accessible to anyone interesting in building their own ORU system.
Toward that end, this repository is distributed under the [Creative Commons Zero v1.0 Universal](https://github.com/ethereum-optimism/optimistic-specs/blob/main/LICENSE) license which dedicates this work to the public domain.

For those interested in contributing to this specification, please refer to [Contributing](#contributing).

## Specification structure

- [Overview](./overview.md)
- Components:
  - [Layer 1 Contracts](./components/layer1.md)
  - [Rollup Client](./components/rollup_client.md)
  - [Execution Engine](./components/exec_engine.md)
  - [Batch Submitter](./components/batch_submitter.md)
  - [Witness Generator](./components/witness_gen.md)
  - [Challenge Agent](./components/challenge_agent.md)

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
