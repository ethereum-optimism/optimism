# `kona-preimage`

<a href="https://crates.io/crates/kona-preimage"><img src="https://img.shields.io/crates/v/kona-preimage.svg?label=kona-preimage&labelColor=2a2f35" alt="Kona Preimage ABI client"></a>
<a href="../../../../LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>

This crate offers a high-level API over the [`Preimage Oracle`][preimage-abi-spec]. It is `no_std` compatible to be used in
`client` programs, and the `host` handles are `async` colored to allow for the `host` programs to reach out to external
data sources to populate the `Preimage Oracle`.

[preimage-abi-spec]: https://specs.optimism.io/experimental/fault-proof/index.html#pre-image-oracle
