# `kona-preimage`

This crate offers a high-level API over the [`Preimage Oracle`][preimage-abi-spec]. It is `no_std` compatible to be used in
`client` programs, and the `host` handles are `async` colored to allow for the `host` programs to reach out to external
data sources to populate the `Preimage Oracle`.

[preimage-abi-spec]: https://specs.optimism.io/experimental/fault-proof/index.html#pre-image-oracle
