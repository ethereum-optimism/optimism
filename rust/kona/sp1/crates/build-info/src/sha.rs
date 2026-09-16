//! Shape of the embedded commit value. Shared by `build.rs` and the crate's tests via `#[path]`,
//! the same arrangement `kona-sp1-range-vkeys` uses for its codec.

/// Recorded when no commit was injected, so plain `cargo build`/`cargo test` stay buildable.
pub(crate) const UNKNOWN: &str = "unknown";

/// Accepts [`UNKNOWN`], or a 40-character lowercase hex sha carrying the optional `-dirty` and
/// `-custom` suffixes in that order. Uppercase is rejected: `git rev-parse` emits lowercase, and
/// the scanners in CI and the README match only lowercase.
pub(crate) fn is_valid(sha: &str) -> bool {
    if sha == UNKNOWN {
        return true;
    }
    let rest = sha.strip_suffix("-custom").unwrap_or(sha);
    let digits = rest.strip_suffix("-dirty").unwrap_or(rest);
    digits.len() == 40 && digits.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f'))
}
