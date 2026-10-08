//! Differential test of kona's interop message expiry rule against the shared vectors in
//! `packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json`, which op-core's
//! `LinkCheckerImpl.CanExecute` and op-supernode's `verifyExecutingMessage` are checked against too.
//!
//! Spec: valid iff `init <= exec && exec - init <= W`, `W = override > 0 ? override : 604800`.
//!
//! The rule is applied the way `MessageGraph` applies it (graph.rs): `check_message_ordering` first, then
//! `check_message_expiry` with the window from `DependencySet::get_message_expiry_window`, the dependency set
//! parsed from JSON (where an override above 7 days is rejected).
//! `check_message_expiry` on its own is NOT total: it requires `init <= exec` (documented precondition).

use kona_genesis::RollupConfig;
use kona_interop::{DependencySet, MESSAGE_EXPIRY_WINDOW, MessageGraphError, MessageRules};
use kona_registry::HashMap;
use serde_json::Value;

const VECTORS: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../../../../../packages/contracts-bedrock/test/formal/expiry/window-differential/vectors.json"
));

/// Parses a dependency set with the given override (0 = none, as the Go config encodes "unset") the way kona
/// reads one (serde JSON, camelCase). `None` if parsing rejects the override.
fn dep_set(override_window: u64) -> Option<DependencySet> {
    let json = if override_window == 0 {
        r#"{"dependencies":{}}"#.to_string()
    } else {
        format!(r#"{{"dependencies":{{}},"overrideMessageExpiryWindow":{override_window}}}"#)
    };
    serde_json::from_str::<DependencySet>(&json).ok()
}

/// The rule as `MessageGraph` applies it. Returns "ok", "future" or "expired".
fn check(rules: &MessageRules<'_>, init: u64, exec: u64) -> &'static str {
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

#[test]
fn window_differential_vectors() {
    let file: Value = serde_json::from_str(VECTORS).expect("vectors.json");
    assert_eq!(file["defaultWindow"].as_u64(), Some(MESSAGE_EXPIRY_WINDOW));
    let vectors = file["vectors"].as_array().expect("vectors");
    assert!(!vectors.is_empty());

    let configs: HashMap<u64, RollupConfig> = HashMap::default();
    for v in vectors {
        let name = v["name"].as_str().unwrap();
        let init = v["init"].as_u64().unwrap();
        let exec = v["exec"].as_u64().unwrap();
        let override_window = v["override"].as_u64().unwrap();
        let window = v["window"].as_u64().unwrap();
        let reason = v["reason"].as_str().unwrap();
        let valid = v["valid"].as_bool().unwrap();
        assert_eq!(valid, reason == "ok", "{name}: malformed vector");

        // An override above the cap is rejected when parsed (never clamped); every other one is accepted and
        // used verbatim.
        let rejected = v["configRejected"].as_bool().unwrap();
        let Some(ds) = dep_set(override_window) else {
            assert!(rejected, "{name}: override {override_window} rejected but not above the cap");
            continue;
        };
        assert!(!rejected, "{name}: override {override_window} above the cap was accepted");
        let effective = ds.get_message_expiry_window();
        assert_eq!(effective, window, "{name}: effective window");

        let rules = MessageRules::new(&configs, effective);
        assert_eq!(check(&rules, init, exec), reason, "{name}: init={init} exec={exec} W={window}");
    }
}

/// `SplitMix64`, so the property test needs no extra dependency and is reproducible.
struct SplitMix64(u64);

impl SplitMix64 {
    const fn next_u64(&mut self) -> u64 {
        self.0 = self.0.wrapping_add(0x9e37_79b9_7f4a_7c15);
        let mut z = self.0;
        z = (z ^ (z >> 30)).wrapping_mul(0xbf58_476d_1ce4_e5b9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94d0_49bb_1331_11eb);
        z ^ (z >> 31)
    }

    const fn below(&mut self, n: u64) -> u64 {
        self.next_u64() % n
    }

    /// Uniform, tiny, near `u64::MAX`, or mid-range.
    const fn pick(&mut self) -> u64 {
        match self.below(4) {
            0 => self.next_u64(),
            1 => self.below(4),
            2 => u64::MAX - self.below(4),
            _ => self.below(1 << 40),
        }
    }
}

#[test]
fn window_differential_property() {
    let configs: HashMap<u64, RollupConfig> = HashMap::default();
    let mut rng = SplitMix64(0x23259);
    for _ in 0..200_000 {
        let override_window = match rng.below(4) {
            0 => 0,
            1 => MESSAGE_EXPIRY_WINDOW + rng.below(3) - 1,
            _ => rng.pick(),
        };
        let Some(ds) = dep_set(override_window) else {
            assert!(override_window > MESSAGE_EXPIRY_WINDOW, "override {override_window} rejected");
            continue;
        };
        assert!(override_window <= MESSAGE_EXPIRY_WINDOW, "override {override_window} accepted");
        let window = ds.get_message_expiry_window();
        assert_eq!(
            window,
            if override_window == 0 { MESSAGE_EXPIRY_WINDOW } else { override_window }
        );

        let init = rng.pick();
        let exec = if rng.below(2) == 0 {
            let delta = window.wrapping_add(rng.below(3)).wrapping_sub(1);
            if rng.below(2) == 0 { init.wrapping_add(delta) } else { init.wrapping_sub(delta) }
        } else {
            rng.pick()
        };

        let rules = MessageRules::new(&configs, window);
        assert_eq!(
            check(&rules, init, exec),
            spec(init, exec, window),
            "init={init} exec={exec} W={window}"
        );
    }
}
