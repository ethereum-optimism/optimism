//! Regression test for the build-script bundle code generator.
//!
//! Runs [`format_bundle`] on a committed fixture JSON and asserts the
//! generated Rust source matches the committed expected output byte-for-byte.
//! Any change to the code-generator's output format will break this test.
//!
//! To regenerate the expected fixture after an intentional codegen change:
//! `cargo test -p kona-hardforks --test build_codegen -- --ignored regenerate_expected`

#[path = "../build_helpers.rs"]
mod build_helpers;

use build_helpers::{capitalize, format_bundle, parse_bundle};

const INPUT_JSON: &str = include_str!("fixtures/test_bundle.json");
const EXPECTED_OUTPUT: &str = include_str!("fixtures/test_bundle_expected.rs");

#[test]
fn generates_expected_rust_source() {
    let bundle = parse_bundle(INPUT_JSON).expect("parse fixture bundle");
    let generated = format_bundle("test", &capitalize("test"), &bundle);
    assert_eq!(generated, EXPECTED_OUTPUT, "generated source does not match expected fixture");
}

/// Regenerate `tests/fixtures/test_bundle_expected.rs` from the current
/// generator output. Run manually after an intentional codegen change:
/// `cargo test -p kona-hardforks --test build_codegen -- --ignored regenerate_expected`
#[test]
#[ignore]
fn regenerate_expected() {
    let bundle = parse_bundle(INPUT_JSON).expect("parse fixture bundle");
    let generated = format_bundle("test", &capitalize("test"), &bundle);
    let path = concat!(env!("CARGO_MANIFEST_DIR"), "/tests/fixtures/test_bundle_expected.rs");
    std::fs::write(path, generated).expect("write expected fixture");
}
