## `op-alloy`

Built on [Alloy][alloy], `op-alloy` connects applications to the OP Stack.


### Usage

To use `op-alloy`, add the crate as a dependency to a `Cargo.toml`.

```toml
op-alloy = "2.0"
```

### `no_std`

op-alloy is intended to be `no_std` compatible, initially for use in [kona][kona].

The following crates support `no_std`.
Notice, provider crates do not support `no_std` compatibility.

- [`op-alloy-consensus`][op-alloy-consensus]
- [`op-alloy-rpc-types-engine`][op-alloy-rpc-types-engine]
- [`op-alloy-rpc-types`][op-alloy-rpc-types]

If you would like to add `no_std` support to a crate,
please make sure to update [scripts/check_no_std.sh][check-no-std].


### Credits

op-alloy is inspired by the work of several teams and projects, most notably [the Alloy project][alloy].

This would not be possible without the hard work from open source contributors. Thank you.


### License

Licensed under either of <a href="../../LICENSE-APACHE">Apache License, Version
2.0</a> or <a href="../../LICENSE-MIT">MIT license</a> at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in these crates by you, as defined in the Apache-2.0 license,
shall be dual licensed as above, without any additional terms or conditions.


<!-- Hyperlinks -->

[check-no-std]: https://github.com/ethereum-optimism/optimism/blob/develop/rust/op-alloy/scripts/check_no_std.sh

[kona]: https://github.com/ethereum-optimism/optimism/tree/develop/rust/kona
[alloy]: https://github.com/alloy-rs/alloy

[op-alloy-consensus]: https://crates.io/crates/op-alloy-consensus
[op-alloy-network]: https://crates.io/crates/op-alloy-network
[op-alloy-rpc-jsonrpsee]: https://crates.io/crates/op-alloy-rpc-jsonrpsee
[op-alloy-rpc-types-engine]: https://crates.io/crates/op-alloy-rpc-types-engine
[op-alloy-rpc-types]: https://crates.io/crates/op-alloy-rpc-types
