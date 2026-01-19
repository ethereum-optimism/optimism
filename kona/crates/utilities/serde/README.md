## `kona-serde`

<a href="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml"><img src="https://github.com/op-rs/kona/actions/workflows/rust_ci.yaml/badge.svg?label=ci" alt="CI"></a>
<a href="https://crates.io/crates/kona-serde"><img src="https://img.shields.io/crates/v/kona-serde.svg" alt="kona-serde crate"></a>
<a href="https://github.com/op-rs/kona/blob/main/LICENSE.md"><img src="https://img.shields.io/badge/License-MIT-d1d1f6.svg?label=license&labelColor=2a2f35" alt="MIT License"></a>
<a href="https://rollup.yoga"><img src="https://img.shields.io/badge/Docs-854a15?style=flat&labelColor=1C2C2E&color=BEC5C9&logo=mdBook&logoColor=BEC5C9" alt="Docs" /></a>

Serde related helpers for kona.

### Graceful Serialization

This crate extends the serialization and deserialization
functionality provided by [`alloy-serde`][alloy-serde] to
deserialize raw number quantity values.

This issue arose in `u128` toml deserialization where
deserialization of a raw number fails.
[This rust playground][invalid] demonstrates how toml fails to
deserialize a native `u128` internal value.

With `kona-serde`, tagging the inner `u128` field with `#[serde(with = "kona_serde::quantity")]`,
allows the `u128` or any other type within the following constraints to be deserialized by toml properly.

These are the supported native types:
- `bool`
- `u8`
- `u16`
- `u32`
- `u64`
- `u128`

Below demonstrates the use of the `#[serde(with = "kona_serde::quantity")]` attribute.

```rust
use serde::{Serialize, Deserialize};

/// My wrapper type.
#[derive(Debug, Serialize, Deserialize)]
pub struct MyStruct {
    /// The inner `u128` value.
    #[serde(with = "kona_serde::quantity")]
    pub inner: u128,
}

// Correctly deserializes a raw value.
let raw_toml = r#"inner = 120"#;
let b: MyStruct = toml::from_str(raw_toml).expect("failed to deserialize toml");
println!("{}", b.inner);

// Notice that a string value is also deserialized correctly.
let raw_toml = r#"inner = "120""#;
let b: MyStruct = toml::from_str(raw_toml).expect("failed to deserialize toml");
println!("{}", b.inner);
```

### Provenance

This code is heavily based on the [`alloy-serde`][alloy-serde] crate.


<!-- Hyperlinks -->

[invalid]: https://play.rust-lang.org/?version=stable&mode=debug&edition=2018&gist=d3c674d02a90c574e3f543144621418d
[alloy-serde]: https://crates.io/crates/alloy-serde
