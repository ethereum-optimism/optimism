//! Differential test of kona's interop message expiry rule against the shared vectors in
//! `packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json`, which
//! op-core's `LinkCheckerImpl.CanExecute` and op-supernode's `verifyExecutingMessage` are checked
//! against too.
//!
//! Spec: valid iff `init <= exec && exec - init <= W`, `W = override > 0 ? override : 604800`.
//!
//! The rule is exercised through the production composition: a [`MessageGraph`] derived from a
//! two-chain superchain (the initiating message in a block at `init`, the executing message in a
//! block at `exec`) and resolved, with the window the fault-proof program passes
//! (`DependencySet::get_message_expiry_window`) and the dependency set parsed from JSON, where an
//! override above 7 days is rejected. A second test applies `MessageRules` directly, in the order
//! `MessageGraph` uses (ordering, then expiry): `check_message_expiry` alone is not total, it
//! requires `init <= exec`.
//!
//! The vectors are read with `include_str!` from `packages/`, so this test needs a monorepo
//! checkout.

use alloy_primitives::{Bytes, keccak256, map::HashMap};
use kona_genesis::RollupConfig;
use kona_interop::{
    DependencySet, ExecutingMessageBuilder, MESSAGE_EXPIRY_WINDOW, MessageGraph, MessageGraphError,
    MessageRules, SuperchainBuilder,
};
use rand::{Rng, SeedableRng, rngs::StdRng};
use serde::Deserialize;

const VECTORS: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../../../../packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json"
));

const CHAIN_INIT: u64 = 1;
const CHAIN_EXEC: u64 = 2;
const MESSAGE: [u8; 4] = [0xde, 0xad, 0xbe, 0xef];

/// The outcome of one vector: the verdict on its message, or the rejection of its override.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "camelCase")]
enum Verdict {
    /// Valid.
    Ok,
    /// The initiating message is newer than the executing message.
    Future,
    /// The initiating message is older than the window allows.
    Expired,
    /// The dependency set rejects the override (above the cap), so no message is checked.
    ConfigRejected,
}

/// The vectors file.
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct VectorFile {
    default_window: u64,
    cap: u64,
    vectors: Vec<Vector>,
}

/// One vector.
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct Vector {
    name: String,
    init: u64,
    exec: u64,
    #[serde(rename = "override")]
    override_window: u64,
    window: u64,
    config_rejected: bool,
    valid: bool,
    reason: Verdict,
}

impl Vector {
    /// Parses the vector's dependency set and checks it is rejected exactly when the vector says
    /// so (an override above the cap is never clamped), and that an accepted override gives the
    /// vector's window. `None` if it was rejected.
    fn dep_set(&self) -> Option<DependencySet> {
        let name = &self.name;
        let override_window = self.override_window;
        let Some(ds) = parse_dep_set(Some(override_window)) else {
            assert!(self.config_rejected, "{name}: override {override_window} rejected");
            return None;
        };
        assert!(!self.config_rejected, "{name}: override {override_window} accepted");
        assert_eq!(ds.get_message_expiry_window(), self.window, "{name}: effective window");
        Some(ds)
    }
}

/// Parses a dependency set over both chains, with the given override or none, the way kona reads
/// one (serde JSON, camelCase). `None` if parsing rejects the override.
fn parse_dep_set(override_window: Option<u64>) -> Option<DependencySet> {
    let deps = format!(r#""dependencies":{{"{CHAIN_INIT}":{{}},"{CHAIN_EXEC}":{{}}}}"#);
    let override_field = override_window
        .map_or_else(String::new, |window| format!(r#","overrideMessageExpiryWindow":{window}"#));
    serde_json::from_str(&format!("{{{deps}{override_field}}}")).ok()
}

/// The verdict of a resolved [`MessageGraph`] for one message.
async fn graph_verdict(ds: &DependencySet, init: u64, exec: u64) -> Verdict {
    let mut superchain = SuperchainBuilder::new();
    superchain
        .chain(CHAIN_INIT)
        .with_timestamp(init)
        .with_block_time(1)
        .with_lagoon_activation_time(0)
        .add_initiating_message(Bytes::from(MESSAGE));
    superchain
        .chain(CHAIN_EXEC)
        .with_timestamp(exec)
        .with_block_time(1)
        .with_lagoon_activation_time(0)
        .add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MESSAGE))
                .with_origin_chain_id(CHAIN_INIT)
                .with_origin_timestamp(init),
        );
    let (headers, cfgs, provider) = superchain.build();
    let graph =
        MessageGraph::derive(&headers, &provider, &cfgs, ds, ds.get_message_expiry_window())
            .await
            .expect("derive");
    match graph.resolve().await {
        Ok(()) => Verdict::Ok,
        Err(MessageGraphError::InvalidMessages(invalid)) => match invalid.get(&CHAIN_EXEC) {
            Some(MessageGraphError::MessageInFuture { .. }) => Verdict::Future,
            Some(MessageGraphError::MessageExpired { .. }) => Verdict::Expired,
            other => panic!("unexpected invalid message {other:?} (init={init} exec={exec})"),
        },
        Err(e) => panic!("unexpected error {e:?} (init={init} exec={exec})"),
    }
}

/// The rule applied with `MessageRules` directly, in `MessageGraph`'s order.
fn rules_verdict(rules: &MessageRules<'_>, init: u64, exec: u64) -> Verdict {
    let res = MessageRules::check_message_ordering::<()>(init, exec)
        .and_then(|()| rules.check_message_expiry::<()>(init, exec));
    match res {
        Ok(()) => Verdict::Ok,
        Err(MessageGraphError::MessageInFuture { .. }) => Verdict::Future,
        Err(MessageGraphError::MessageExpired { .. }) => Verdict::Expired,
        Err(e) => panic!("unexpected error {e:?}"),
    }
}

/// The spec on unbounded integers.
fn spec(init: u64, exec: u64, window: u64) -> Verdict {
    if init > exec {
        Verdict::Future
    } else if u128::from(exec) - u128::from(init) > u128::from(window) {
        Verdict::Expired
    } else {
        Verdict::Ok
    }
}

fn vectors() -> Vec<Vector> {
    let file: VectorFile = serde_json::from_str(VECTORS).expect("vectors.json");
    assert_eq!(file.default_window, MESSAGE_EXPIRY_WINDOW);
    // The override cap is the protocol window itself.
    assert_eq!(file.cap, MESSAGE_EXPIRY_WINDOW);
    assert!(!file.vectors.is_empty());
    for v in &file.vectors {
        assert_eq!(v.valid, v.reason == Verdict::Ok, "{}: malformed vector", v.name);
        assert_eq!(
            v.config_rejected,
            v.reason == Verdict::ConfigRejected,
            "{}: malformed vector",
            v.name
        );
    }
    file.vectors
}

#[tokio::test]
async fn window_differential_vectors_message_graph() {
    for v in vectors() {
        let Some(ds) = v.dep_set() else { continue };
        assert_eq!(graph_verdict(&ds, v.init, v.exec).await, v.reason, "{}", v.name);
    }
}

#[test]
fn window_differential_vectors_rules() {
    let configs: HashMap<u64, RollupConfig> = HashMap::default();
    for v in vectors() {
        let Some(ds) = v.dep_set() else { continue };
        let rules = MessageRules::new(&configs, ds.get_message_expiry_window());
        assert_eq!(rules_verdict(&rules, v.init, v.exec), v.reason, "{}", v.name);
    }
}

/// An omitted override and an explicit zero both mean the default window.
#[test]
fn window_differential_zero_override_omitted_or_explicit() {
    let omitted = parse_dep_set(None).expect("no override parses");
    assert_eq!(omitted.override_message_expiry_window, None);
    let explicit = parse_dep_set(Some(0)).expect("an explicit zero override parses");
    assert_eq!(explicit.override_message_expiry_window, Some(0));
    assert_eq!(omitted.get_message_expiry_window(), MESSAGE_EXPIRY_WINDOW);
    assert_eq!(explicit.get_message_expiry_window(), MESSAGE_EXPIRY_WINDOW);
}

/// Seeded case generator, biased towards the window boundary and the `u64` extremes.
struct Cases(StdRng);

impl Cases {
    fn new(seed: u64) -> Self {
        Self(StdRng::seed_from_u64(seed))
    }

    fn below(&mut self, n: u64) -> u64 {
        self.0.random_range(0..n)
    }

    /// Uniform, tiny, near `u64::MAX`, or mid-range.
    fn pick(&mut self) -> u64 {
        match self.below(4) {
            0 => self.0.random(),
            1 => self.below(4),
            2 => u64::MAX - self.below(4),
            _ => self.below(1 << 40),
        }
    }

    /// A random case: an override (rejected above the cap) and timestamps near the boundary.
    fn case(&mut self) -> Case {
        let override_window = match self.below(4) {
            0 => 0,
            1 => MESSAGE_EXPIRY_WINDOW + self.below(3) - 1,
            _ => self.pick(),
        };
        let window = if override_window == 0 { MESSAGE_EXPIRY_WINDOW } else { override_window };
        let init = self.pick();
        let exec = if self.below(2) == 0 {
            let delta = window.wrapping_add(self.below(3)).wrapping_sub(1);
            if self.below(2) == 0 { init.wrapping_add(delta) } else { init.wrapping_sub(delta) }
        } else {
            self.pick()
        };
        Case { override_window, init, exec }
    }
}

/// A random override and pair of timestamps.
#[derive(Debug, Clone, Copy)]
struct Case {
    override_window: u64,
    init: u64,
    exec: u64,
}

/// Checks the parse and the effective window, or returns `None` if the override was rejected.
fn checked_window(override_window: u64) -> Option<DependencySet> {
    let Some(ds) = parse_dep_set(Some(override_window)) else {
        assert!(override_window > MESSAGE_EXPIRY_WINDOW, "override {override_window} rejected");
        return None;
    };
    assert!(override_window <= MESSAGE_EXPIRY_WINDOW, "override {override_window} accepted");
    let window = if override_window == 0 { MESSAGE_EXPIRY_WINDOW } else { override_window };
    assert_eq!(ds.get_message_expiry_window(), window);
    Some(ds)
}

#[tokio::test]
async fn window_differential_property_message_graph() {
    let mut rng = Cases::new(0x23259);
    for _ in 0..5_000 {
        let Case { override_window, init, exec } = rng.case();
        let Some(ds) = checked_window(override_window) else { continue };
        let window = ds.get_message_expiry_window();
        assert_eq!(
            graph_verdict(&ds, init, exec).await,
            spec(init, exec, window),
            "init={init} exec={exec} W={window}"
        );
    }
}

#[test]
fn window_differential_property_rules() {
    let configs: HashMap<u64, RollupConfig> = HashMap::default();
    let mut rng = Cases::new(0x5eed);
    for _ in 0..200_000 {
        let Case { override_window, init, exec } = rng.case();
        let Some(ds) = checked_window(override_window) else { continue };
        let window = ds.get_message_expiry_window();
        let rules = MessageRules::new(&configs, window);
        assert_eq!(
            rules_verdict(&rules, init, exec),
            spec(init, exec, window),
            "init={init} exec={exec} W={window}"
        );
    }
}
