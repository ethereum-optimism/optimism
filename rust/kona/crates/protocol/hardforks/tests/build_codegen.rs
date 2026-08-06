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

/// A bundle predating `extraGas` stays readable: its locked commit's generator emits the old
/// schema, and provenance verification regenerates it with exactly that generator.
#[test]
fn accepts_legacy_version_without_extra_gas() {
    let json = INPUT_JSON.replace("\"extraGas\": 50000,\n    ", "").replace("1.1.0", "1.0.0");
    let bundle = parse_bundle(&json).expect("legacy bundle should parse");
    assert_eq!(bundle.metadata.extra_gas, 0);
}

/// `extraGas` at the legacy version is a malformed bundle, not a silently-ignored field — a
/// reader that predates the field would under-reserve the activation block's gas limit.
#[test]
fn rejects_extra_gas_at_legacy_version() {
    let json = INPUT_JSON.replace("1.1.0", "1.0.0");
    let Err(err) = parse_bundle(&json) else { panic!("legacy version must not declare extraGas") };
    assert!(err.to_string().contains("must not declare extraGas"), "unexpected error: {err}");
}

#[test]
fn rejects_unknown_version() {
    let json = INPUT_JSON.replace("1.1.0", "2.0.0");
    let Err(err) = parse_bundle(&json) else { panic!("unknown version must be rejected") };
    assert!(err.to_string().contains("unsupported NUT bundle version"), "unexpected error: {err}");
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
