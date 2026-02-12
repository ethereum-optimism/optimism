## `kona-genesis`

Genesis types for Optimism.

### Usage

_By default, `kona-genesis` enables both `std` and `serde` features._

If you're working in a `no_std` environment (like [`kona`][kona]), disable default features like so.

```toml
[dependencies]
kona-genesis = { version = "x.y.z", default-features = false, features = ["serde"] }
```

#### Rollup Config

`kona-genesis` exports a `RollupConfig`, the primary genesis type for Optimism Consensus.


<!-- Links -->

[alloy-genesis]: https://github.com/alloy-rs
[kona]: https://github.com/ethereum-optimism/optimism/blob/develop/rust/kona/Cargo.toml#L137
