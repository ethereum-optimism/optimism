# Contributing to CONTRIBUTING.md

First off, thanks for taking the time to contribute! ❤️

We welcome and appreciate all kinds of contributions. Refer to the Table of Contents to discover various ways you can assist, as well as the procedures we follow for each type. Before you contribute, kindly review the appropriate section; this will streamline the process for both maintainers and contributors. We're excited to see your contributions! 🎉

## Table of Contents

- [I Have a Question](#i-have-a-question)
- [I Want To Contribute](#i-want-to-contribute)
- [Reporting Bugs](#reporting-bugs)
- [Suggesting Enhancements](#suggesting-enhancements)
- [Your First Code Contribution](#your-first-code-contribution)
- [Improving The Documentation](#improving-the-documentation)

## I Have a Question

> **Note**
> Before making an issue, please read the documentation and search the issues to see if your question has already been answered.

If you have any questions about the smart contracts, please feel free to ask them in the Optimism discord developer channels or in the issues of the monorepo.

## I Want To Contribute

### Reporting Bugs

**Any and all bug reports on production smart contract code should be submitted privately to the Optimism team so that we can mitigate the issue before it is exploited. Please see our security policy document [here](https://github.com/ethereum-optimism/.github/blob/master/SECURITY.md).**

### Suggesting Enhancements

#### Before Submitting an Enhancement

- Read the documentation and the smart contracts themselves to see if the feature already exists.
- Perform a search in the issues to see if the enhancement has already been suggested. If it has, add a comment to the existing issue instead of opening a new one.

#### How Do I Submit a Good Enhancement Suggestion?

Enhancement suggestions are tracked as [GitHub issues](/issues).

- Use a **clear and descriptive title** for the issue to identify the suggestion.
- Provide a **step-by-step description of the suggested enhancement** in as many details as possible.
- **Describe the current behavior** and **explain which behavior you expected to see instead** and why. At this point you can also tell which alternatives do not work for you.
- **Explain why this enhancement would be useful** in Optimism's smart contracts. You may also want to point out the other projects that solved it better and which could serve as inspiration.

### Your First Code Contribution

The best place to begin contributing is by looking through the issues with the `good first issue` label. These are issues that are relatively easy to implement and are a great way to get familiar with the codebase.

Optimism's smart contracts are written in Solidity and we use [foundry](https://github.com/foundry-rs/foundry) as our development framework. To get started, you'll need to install several dependencies:
1. [pnpm](https://pnpm.io)
1. [foundry](https://getfoundry.sh)
  1. Foundry is built with [rust](https://www.rust-lang.org/tools/install), so if you decide to build it from source, you'll need to install the rust toolchain via `rustup`.
1. [golang](https://golang.org/doc/install)
1. [python](https://www.python.org/downloads/)

Our [Style Guide](STYLE_GUIDE.md) contains information about the project structure, syntax preferences, naming conventions, and more. Please take a look at it before submitting a PR, and let us know if you spot inconcistencies!

Once you've read the styleguide and are ready to work on your PR, there are a plethora of useful `pnpm` scripts to know about that will help you with development:
1. `pnpm build` Builds the smart contracts.
1. `pnpm test` Runs the full `forge` test suite.
1  `pnpm gas-snapshot` Generates the gas snapshot for the smart contracts.
1. `pnpm semver-lock` Generates the semver lockfile.
1. `pnpm storage-snapshot` Generates the storage lockfile.
1. `pnpm autogen:invariant-docs` Generates the invariant test documentation.
1. `pnpm clean` Removes all build artifacts for `forge` and `go` compilations.
1. `pnpm validate-spacers` Validates the positions of the storage slot spacers.
1. `pnpm validate-deploy-configs` Validates the deployment configurations in `deploy-config`
1. `pnpm slither` Runs the slither static analysis tool on the smart contracts.
1. `pnpm lint` Runs the linter on the smart contracts and scripts.
1. `pnpm pre-pr` Runs most checks, generators, and linters prior to a PR. For most PRs, this is sufficient to pass CI if everything is in order.
1. `pnpm pre-pr:full` Runs all checks, generators, and linters prior to a PR.

### Improving The Documentation

Documentation improvements are more than welcome! If you see a typo or feel that a code comment describes something poorly or incorrectly, please submit a PR with a fix.
