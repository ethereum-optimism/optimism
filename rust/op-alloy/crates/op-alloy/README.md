## `op-alloy`

<a href="https://github.com/alloy-rs/op-alloy/actions/workflows/ci.yml"><img src="https://github.com/alloy-rs/op-alloy/actions/workflows/ci.yml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/op-alloy"><img src="https://img.shields.io/crates/v/op-alloy.svg" alt="op-alloy crate"></a>
<a href="https://github.com/alloy-rs/op-alloy/blob/main/LICENSE-APACHE"><img src="https://img.shields.io/badge/License-APACHE-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://github.com/alloy-rs/op-alloy/blob/main/LICENSE-MIT"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="License"></a>
<a href="https://alloy-rs.github.io/op-alloy"><img src="https://img.shields.io/badge/Book-854a15?logo=mdBook&labelColor=2a2f35" alt="Book"></a>


Built on [Alloy][alloy], `op-alloy` connects applications to the OP Stack.


### Usage

To use `op-alloy`, add the crate as a dependency to a `Cargo.toml`.

```toml
op-alloy = "0.6"
```

### Development Status

`op-alloy` is currently in active development, and is not yet ready for use in production.


### Supported Rust Versions (MSRV)

The current MSRV (minimum supported rust version) is 1.86.

Unlike Alloy, op-alloy may use the latest stable release,
to benefit from the latest features.

The MSRV is not increased automatically, and will be updated
only as part of a patch (pre-1.0) or minor (post-1.0) release.


### Contributing

op-alloy is built by open source contributors like you, thank you for improving the project!

A [contributing guide][contributing] is available that sets guidelines for contributing.

Pull requests will not be merged unless CI passes, so please ensure that your contribution follows the
linting rules and passes clippy.


### `no_std`

op-alloy is intended to be `no_std` compatible, initially for use in [kona][kona].

The following crates support `no_std`.
Notice, provider crates do not support `no_std` compatibility.

- [`op-alloy-consensus`][op-alloy-consensus]
- [`op-alloy-rpc-types-engine`][op-alloy-rpc-types-engine]
- [`op-alloy-rpc-types`][op-alloy-rpc-types]

If you would like to add no_std support to a crate,
please make sure to update [scripts/check_no_std.sh][check-no-std].


### Credits

op-alloy is inspired by the work of several teams and projects, most notably [the Alloy project][alloy].

This would not be possible without the hard work from open source contributors. Thank you.


### License

Licensed under either of <a href="LICENSE-APACHE">Apache License, Version
2.0</a> or <a href="LICENSE-MIT">MIT license</a> at your option.

Unless you explicitly state otherwise, any contribution intentionally submitted
for inclusion in these crates by you, as defined in the Apache-2.0 license,
shall be dual licensed as above, without any additional terms or conditions.


<!-- Hyperlinks -->

[check-no-std]: https://github.com/alloy-rs/op-alloy/blob/main/scripts/check_no_std.sh

[maili]: https://github.com/op-rs/maili
[kona]: https://github.com/op-rs/kona
[alloy]: https://github.com/alloy-rs/alloy
[contributing]: https://alloy-rs.github.io/op-alloy

[op-alloy-consensus]: https://crates.io/crates/op-alloy-consensus
[op-alloy-network]: https://crates.io/crates/op-alloy-network
[op-alloy-rpc-jsonrpsee]: https://crates.io/crates/op-alloy-rpc-jsonrpsee
[op-alloy-rpc-types-engine]: https://crates.io/crates/op-alloy-rpc-types-engine
[op-alloy-rpc-types]: https://crates.io/crates/op-alloy-rpc-types
