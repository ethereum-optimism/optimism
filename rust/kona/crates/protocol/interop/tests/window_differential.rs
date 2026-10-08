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

use alloy_primitives::{Bytes, keccak256, map::HashMap};
use kona_genesis::RollupConfig;
use kona_interop::{
    DependencySet, ExecutingMessageBuilder, MESSAGE_EXPIRY_WINDOW, MessageGraph, MessageGraphError,
    MessageRules, SuperchainBuilder,
};
use rand::{Rng, SeedableRng, rngs::StdRng};
use serde_json::Value;

const VECTORS: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../../../../packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json"
));

const CHAIN_INIT: u64 = 1;
const CHAIN_EXEC: u64 = 2;
const MESSAGE: [u8; 4] = [0xde, 0xad, 0xbe, 0xef];

/// Parses a dependency set over both chains with the given override, the way kona reads one
/// (serde JSON, camelCase). `0` is tried both omitted and explicit (`Some(0)`), which must agree.
/// `None` if parsing rejects the override.
fn dep_set(override_window: u64) -> Option<DependencySet> {
    let deps = format!(r#""dependencies":{{"{CHAIN_INIT}":{{}},"{CHAIN_EXEC}":{{}}}}"#);
    let explicit = serde_json::from_str::<DependencySet>(&format!(
        r#"{{{deps},"overrideMessageExpiryWindow":{override_window}}}"#
    ))
    .ok();
    if override_window == 0 {
        let omitted = serde_json::from_str::<DependencySet>(&format!("{{{deps}}}")).unwrap();
        assert_eq!(omitted.override_message_expiry_window, None);
        let explicit = explicit.expect("an explicit zero override parses");
        assert_eq!(explicit.override_message_expiry_window, Some(0));
        assert_eq!(explicit.get_message_expiry_window(), omitted.get_message_expiry_window());
        return Some(explicit);
    }
    explicit
}

/// The verdict of a resolved [`MessageGraph`] for one message: "ok", "future" or "expired".
async fn graph_verdict(ds: &DependencySet, init: u64, exec: u64) -> &'static str {
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
        Ok(()) => "ok",
        Err(MessageGraphError::InvalidMessages(invalid)) => match invalid.get(&CHAIN_EXEC) {
            Some(MessageGraphError::MessageInFuture { .. }) => "future",
            Some(MessageGraphError::MessageExpired { .. }) => "expired",
            other => panic!("unexpected invalid message {other:?} (init={init} exec={exec})"),
        },
        Err(e) => panic!("unexpected error {e:?} (init={init} exec={exec})"),
    }
}

/// The rule applied with `MessageRules` directly, in `MessageGraph`'s order.
fn rules_verdict(rules: &MessageRules<'_>, init: u64, exec: u64) -> &'static str {
    let res = MessageRules::check_message_ordering::<()>(init, exec)
        .and_then(|()| rules.check_message_expiry::<()>(init, exec));
    match res {
        Ok(()) => "ok",
        Err(MessageGraphError::MessageInFuture { .. }) => "future",
        Err(MessageGraphError::MessageExpired { .. }) => "expired",
        Err(e) => panic!("unexpected error {e:?}"),
    }
}

/// The spec on unbounded integers.
fn spec(init: u64, exec: u64, window: u64) -> &'static str {
    if init > exec {
        "future"
    } else if u128::from(exec) - u128::from(init) > u128::from(window) {
        "expired"
    } else {
        "ok"
    }
}

/// One vector: (name, init, exec, override, window, rejected, reason).
type Vector = (String, u64, u64, u64, u64, bool, String);

fn vectors() -> Vec<Vector> {
    let file: Value = serde_json::from_str(VECTORS).expect("vectors.json");
    assert_eq!(file["defaultWindow"].as_u64(), Some(MESSAGE_EXPIRY_WINDOW));
    let vectors = file["vectors"].as_array().expect("vectors");
    assert!(!vectors.is_empty());
    vectors
        .iter()
        .map(|v| {
            let name = v["name"].as_str().unwrap().to_string();
            let reason = v["reason"].as_str().unwrap().to_string();
            assert_eq!(v["valid"].as_bool().unwrap(), reason == "ok", "{name}: malformed vector");
            (
                name,
                v["init"].as_u64().unwrap(),
                v["exec"].as_u64().unwrap(),
                v["override"].as_u64().unwrap(),
                v["window"].as_u64().unwrap(),
                v["configRejected"].as_bool().unwrap(),
                reason,
            )
        })
        .collect()
}

#[tokio::test]
async fn window_differential_vectors_message_graph() {
    for (name, init, exec, override_window, window, rejected, reason) in vectors() {
        // An override above the cap is rejected when parsed (never clamped); every other one is
        // accepted and used verbatim.
        let Some(ds) = dep_set(override_window) else {
            assert!(rejected, "{name}: override {override_window} rejected but not above the cap");
            continue;
        };
        assert!(!rejected, "{name}: override {override_window} above the cap was accepted");
        assert_eq!(ds.get_message_expiry_window(), window, "{name}: effective window");
        assert_eq!(graph_verdict(&ds, init, exec).await, reason, "{name}");
    }
}

#[test]
fn window_differential_vectors_rules() {
    let configs: HashMap<u64, RollupConfig> = HashMap::default();
    for (name, init, exec, override_window, window, rejected, reason) in vectors() {
        let Some(ds) = dep_set(override_window) else {
            assert!(rejected, "{name}");
            continue;
        };
        assert_eq!(ds.get_message_expiry_window(), window, "{name}");
        let rules = MessageRules::new(&configs, window);
        assert_eq!(rules_verdict(&rules, init, exec), reason, "{name}");
    }
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
    fn case(&mut self) -> (u64, u64, u64) {
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
        (override_window, init, exec)
    }
}

/// Checks the parse and returns the effective window, or `None` if the override was rejected.
fn checked_window(override_window: u64) -> Option<(DependencySet, u64)> {
    let Some(ds) = dep_set(override_window) else {
        assert!(override_window > MESSAGE_EXPIRY_WINDOW, "override {override_window} rejected");
        return None;
    };
    assert!(override_window <= MESSAGE_EXPIRY_WINDOW, "override {override_window} accepted");
    let window = ds.get_message_expiry_window();
    assert_eq!(window, if override_window == 0 { MESSAGE_EXPIRY_WINDOW } else { override_window });
    Some((ds, window))
}

#[tokio::test]
async fn window_differential_property_message_graph() {
    let mut rng = Cases::new(0x23259);
    for _ in 0..5_000 {
        let (override_window, init, exec) = rng.case();
        let Some((ds, window)) = checked_window(override_window) else { continue };
        if init == 0 || exec == 0 {
            continue; // the activation rule (activation 0, block time 1), not under test
        }
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
        let (override_window, init, exec) = rng.case();
        let Some((_, window)) = checked_window(override_window) else { continue };
        let rules = MessageRules::new(&configs, window);
        assert_eq!(
            rules_verdict(&rules, init, exec),
            spec(init, exec, window),
            "init={init} exec={exec} W={window}"
        );
    }
}
