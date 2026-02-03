# Contributing

Thank you for wanting to contribute! Before contributing to this repository,
please read through this document and discuss the change you wish to make via issue.


## Dependencies

Before working with this repository locally, you'll need to install a few dependencies:

- [Just](https://github.com/casey/just) for our command-runner scripts.
- [The Rust toolchain](https://rustup.rs/).

**Optional**

- [mdbook](https://github.com/rust-lang/mdBook) to contribute to the [book](/)
  - [mdbook-template](https://github.com/sgoudham/mdbook-template)
  - [mdbook-mermaid](https://github.com/badboy/mdbook-mermaid)


## Pull Request Process

1. [Create an issue][new-issue] for any significant changes. Trivial changes may skip this step.
1. Once the change is implemented, ensure that all checks are passing before creating a PR.
   The full CI pipeline can be run locally via the `Justfile`s in the repository.
1. Be sure to update any documentation that has gone stale as a result of the change,
   in the `README` files, the [book][book], and in rustdoc comments.
1. Once your PR is approved by a maintainer, you may merge your pull request yourself
   if you have permissions to do so. Otherwise, the maintainer who approves your pull
   request will add it to the merge queue.


## Working with OP Stack Specs

The [OP Stack][op-stack] is a set of standardized open-source specifications
that powers Optimism, developed by the Optimism Collective.

`op-alloy` is a rust implementation of core OP Stack types, transports,
middleware and more. Not all types and implementation details in `op-alloy`
are present in the OP Stack [specs][specs], and on the flipside, not all
specifications are implemented by `op-alloy`. That said, `op-alloy` is
entirely _based off_ of the [specs][specs], and new functionality or
core modifications to `op-alloy` must be reflected in the [specs][specs].

As such, the first step for introducing changes to the OP Stack is to
[open a pr][specs-pr] in the [specs repository][specs-repo]. These
changes should target a [protocol upgrade][upgrades] so that all
implementations of the OP Stack are able to synchronize and implement
the changes.

Once changes are merged in the OP Stack [specs][specs] repo, they
may be added to `op-alloy` in a **backwards-compatible** way such
that pre-upgrade functionality persists. The primary way to enable
backwards-compatibility is by using timestamp-based activation for
protocol upgrades.


<!-- Links -->

[upgrades]: https://specs.optimism.io/protocol/isthmus/overview.html
[specs-repo]: https://github.com/ethereum-optimism/specs
[specs-pr]: https://github.com/ethereum-optimism/specs/pulls
[specs]: https://specs.optimism.io/
[op-stack]: https://docs.optimism.io/stack/getting-started
[book]: https://github.com/alloy-rs/op-alloy/tree/main/book
[new-issue]: https://github.com/alloy-rs/op-alloy/issues/new
