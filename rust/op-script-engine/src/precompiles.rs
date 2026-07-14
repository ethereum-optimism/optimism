//! Arbitrary-address precompiles for the OPCM `RunScript*` path (design §4, the
//! "unidirectional" input/output-precompile mechanism).
//!
//! In the Go host, `opcm.RunScriptSingle`/`RunScriptVoid` wrap the input struct `I` and output
//! struct `O` as reflection-backed precompiles installed at freshly-minted `NewScriptAddress()`
//! addresses (`script.WithPrecompileAtAddress`). The Solidity `run(input[, output])` entrypoint
//! receives those addresses and reads inputs via one CALL per field-getter, and writes results by
//! calling `output.set(bytes4,<T>)` on the field-setter precompile.
//!
//! The unidirectional design keeps all Go<->EVM traffic one-way: the Go side snapshots the input
//! getters (`selector -> ABI-return-bytes`) up front and ships them here as a
//! [`HostPrecompile::InputSnapshot`];
//! the Rust side captures the raw `set()` calldata into an [`OutputCapture`] for the Go side to
//! replay through the real `WithFieldSetter` precompile after the run. No Rust->Go callback.

use alloy_primitives::{
    Bytes, keccak256,
    map::{HashMap, HashSet},
};

/// A precompile installed at an arbitrary address, the revm-inspector replacement for op-geth's
/// per-address `PrecompileOverrides`.
#[derive(Debug, Clone)]
pub enum HostPrecompile {
    /// Input getter-snapshot: `getter-selector -> ABI-encoded return bytes`, pre-computed on the
    /// Go side from `NewPrecompile[*I]`. Answers each snapshotted getter and reverts loudly on any
    /// other selector, matching the Go `Precompile.Run` "unrecognized 4 byte signature".
    InputSnapshot(HashMap<[u8; 4], Bytes>),
    /// Output setter-capture: records raw `set(bytes4,<T>)` calldata (in call order) for Go-side
    /// replay through `WithFieldSetter`, and answers field getters with the last value that was set
    /// (mirroring the Go `WithFieldSetter` precompile, whose getters reflect the just-written
    /// struct field).
    OutputCapture(OutputCapture),
}

/// Result of dispatching a call to an installed precompile.
#[derive(Debug)]
pub enum PrecompileOutcome {
    /// ABI-encoded return data (success).
    Return(Bytes),
    /// A loud revert with the given reason (an unrecognized selector), mirroring the Go host.
    Revert(String),
}

/// Output setter-capture state (Family B, `RunScriptSingle` output precompile).
#[derive(Debug, Clone, Default)]
pub struct OutputCapture {
    /// Valid field-getter selectors of the output struct `O` (from `NewPrecompile[*O]`). A getter
    /// not in this set is an unrecognized selector and reverts, matching the Go host.
    getters: HashSet<[u8; 4]>,
    /// `field-getter-selector -> last value word` written by a `set()` call. The `bytes4` first
    /// argument of `set(bytes4,<T>)` *is* the field getter selector, so a getter reads back the
    /// value word its setter stored.
    stored: HashMap<[u8; 4], [u8; 32]>,
    /// Raw `set(...)` calldata (selector + params), in call order, for Go-side replay.
    pub captured: Vec<Bytes>,
}

impl OutputCapture {
    /// Build an empty capture that accepts the given output field-getter selectors.
    pub fn new(getters: HashSet<[u8; 4]>) -> Self {
        Self { getters, stored: HashMap::default(), captured: Vec::new() }
    }
}

/// keccak-derived 4-byte selector of a solidity function signature.
fn selector(sig: &str) -> [u8; 4] {
    keccak256(sig.as_bytes())[..4].try_into().unwrap()
}

/// The three `WithFieldSetter` setter selectors (`op-chain-ops/script/precompile.go`).
/// `set(bytes4,address)`, `set(bytes4,bool)`, `set(bytes4,uint32)` all take a 32-byte field
/// selector word followed by a 32-byte value word, so they share one capture path.
fn setter_selectors() -> [[u8; 4]; 3] {
    [selector("set(bytes4,address)"), selector("set(bytes4,bool)"), selector("set(bytes4,uint32)")]
}

impl HostPrecompile {
    /// Dispatch a call to this precompile. `input` is the full calldata (4-byte selector + params).
    pub fn run(&mut self, input: &[u8]) -> PrecompileOutcome {
        if input.len() < 4 {
            return PrecompileOutcome::Revert(format!(
                "expected at least 4 bytes, but got '{}'",
                crate::cheatcodes::hex(input)
            ));
        }
        let sel: [u8; 4] = input[0..4].try_into().unwrap();
        match self {
            Self::InputSnapshot(map) => map.get(&sel).map_or_else(
                || {
                    PrecompileOutcome::Revert(format!(
                        "unrecognized 4 byte signature: {}",
                        crate::cheatcodes::hex(&sel)
                    ))
                },
                |bytes| PrecompileOutcome::Return(bytes.clone()),
            ),
            Self::OutputCapture(out) => {
                if setter_selectors().contains(&sel) {
                    let params = &input[4..];
                    // set(bytes4 fieldSel, <T> value): field selector in the first word, value in
                    // the second. Address/bool/uint32 all encode as one right-aligned value word,
                    // which is exactly what the matching getter must return.
                    if params.len() < 64 {
                        return PrecompileOutcome::Revert(format!(
                            "cannot set field from {} bytes",
                            params.len()
                        ));
                    }
                    let field_sel: [u8; 4] = params[0..4].try_into().unwrap();
                    let value_word: [u8; 32] = params[32..64].try_into().unwrap();
                    out.stored.insert(field_sel, value_word);
                    out.captured.push(Bytes::copy_from_slice(input));
                    return PrecompileOutcome::Return(Bytes::new());
                }
                if out.getters.contains(&sel) {
                    let word = out.stored.get(&sel).copied().unwrap_or([0u8; 32]);
                    return PrecompileOutcome::Return(Bytes::copy_from_slice(&word));
                }
                PrecompileOutcome::Revert(format!(
                    "unrecognized 4 byte signature: {}",
                    crate::cheatcodes::hex(&sel)
                ))
            }
        }
    }
}
