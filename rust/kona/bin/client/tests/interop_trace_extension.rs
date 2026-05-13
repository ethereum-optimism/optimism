//! Integration tests for the interop client's trace-extension boundary on a
//! `PreState::TransitionState` agreed prestate. Mirrors the single-proof
//! `trace_extension.rs` tests but exercises the interop `run()` dispatch.

use alloy_primitives::B256;
use alloy_rlp::Encodable;
use async_trait::async_trait;
use kona_client::interop::{FaultProofProgramError, run};
use kona_genesis::{DependencySet, L1ChainConfig, RollupConfig};
use kona_interop::{OutputRootWithChain, SuperRoot};
use kona_preimage::{
    HintWriterClient, PreimageKey, PreimageOracleClient,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use kona_proof_interop::{PreState, TransitionState};
use std::{
    collections::{BTreeMap, HashMap},
    sync::Arc,
};
use tokio::sync::Mutex;

#[derive(Clone, Debug, Default)]
struct MockOracle {
    preimages: Arc<Mutex<HashMap<PreimageKey, Vec<u8>>>>,
}

impl MockOracle {
    fn from_preimages(preimages: HashMap<PreimageKey, Vec<u8>>) -> Self {
        Self { preimages: Arc::new(Mutex::new(preimages)) }
    }
}

#[async_trait]
impl PreimageOracleClient for MockOracle {
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        self.preimages.lock().await.get(&key).cloned().ok_or(PreimageOracleError::KeyNotFound)
    }

    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        let data = self.get(key).await?;
        if data.len() != buf.len() {
            return Err(PreimageOracleError::BufferLengthMismatch(buf.len(), data.len()));
        }
        buf.copy_from_slice(&data);
        Ok(())
    }
}

#[derive(Clone, Debug, Default)]
struct MockHintWriter {
    _hints: Arc<Mutex<Vec<String>>>,
}

#[async_trait]
impl HintWriterClient for MockHintWriter {
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        self._hints.lock().await.push(hint.to_string());
        Ok(())
    }
}

fn b256(fill: u8) -> B256 {
    B256::from([fill; 32])
}

/// Builds a synthetic interop boot fixture with a `PreState::TransitionState` agreed prestate
/// whose embedded `SuperRoot.timestamp == prestate_timestamp`.
///
/// Chain IDs `9901` and `9902` are intentionally absent from the embedded
/// `ROLLUP_CONFIGS`, `L1_CONFIGS`, and `DEPENDENCY_SETS` registries, forcing
/// `BootInfo::load` down the preimage-oracle fallback paths (keys 6, 7, 8).
fn setup_interop_preimages(
    prestate_timestamp: u64,
    claimed_l2_timestamp: u64,
    claimed_post_state: B256,
) -> (HashMap<PreimageKey, Vec<u8>>, B256) {
    const CHAIN_A: u64 = 9901;
    const CHAIN_B: u64 = 9902;
    const L1_CHAIN_ID: u64 = 9001;

    // Build the TransitionState prestate. `step = 1` is a plausible mid-transition value;
    // `PreState::decode` dispatches purely on the leading version byte, so any `step` in
    // `[0, TRANSITION_STATE_MAX_STEPS]` reaches the `TransitionState` arm.
    let pre_state = PreState::TransitionState(TransitionState::new(
        SuperRoot::new(
            prestate_timestamp,
            vec![
                OutputRootWithChain::new(CHAIN_A, b256(0xA1)),
                OutputRootWithChain::new(CHAIN_B, b256(0xB2)),
            ],
        ),
        Vec::new(),
        1,
    ));
    let mut pre_state_rlp = Vec::with_capacity(pre_state.length());
    pre_state.encode(&mut pre_state_rlp);
    let agreed_pre_state_commitment = pre_state.hash();

    let mut preimages = HashMap::new();

    // Local keys consumed by BootInfo::load.
    preimages.insert(PreimageKey::new_local(1), b256(0x11).as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(2), agreed_pre_state_commitment.as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(3), claimed_post_state.as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(4), claimed_l2_timestamp.to_be_bytes().to_vec());

    // RollupConfig fallback (key 6): chain IDs absent from ROLLUP_CONFIGS, so JSON deserialization
    // path is exercised. Default values are fine — `run()` short-circuits before any config field
    // is consumed.
    let mut rollup_configs: HashMap<u64, RollupConfig> = HashMap::new();
    for chain_id in [CHAIN_A, CHAIN_B] {
        let cfg = RollupConfig { l1_chain_id: L1_CHAIN_ID, ..RollupConfig::default() };
        rollup_configs.insert(chain_id, cfg);
    }
    preimages.insert(
        PreimageKey::new_local(6),
        serde_json::to_vec(&rollup_configs).expect("serialize rollup configs"),
    );

    // L1ChainConfig fallback (key 7): synthetic L1 chain ID 9001 is absent from L1_CONFIGS.
    let l1_config = L1ChainConfig { chain_id: L1_CHAIN_ID, ..L1ChainConfig::default() };
    preimages.insert(
        PreimageKey::new_local(7),
        serde_json::to_vec(&l1_config).expect("serialize l1 config"),
    );

    // DependencySet fallback (key 8): the two synthetic chain IDs are absent from
    // DEPENDENCY_SETS.
    let mut dependencies = BTreeMap::new();
    dependencies.insert(CHAIN_A, kona_genesis::ChainDependency {});
    dependencies.insert(CHAIN_B, kona_genesis::ChainDependency {});
    let depset = DependencySet { dependencies, override_message_expiry_window: None };
    preimages
        .insert(PreimageKey::new_local(8), serde_json::to_vec(&depset).expect("serialize depset"));

    // Keccak preimage for the agreed pre-state commitment (consumed by `read_raw_pre_state`).
    preimages.insert(PreimageKey::new_keccak256(*agreed_pre_state_commitment), pre_state_rlp);

    (preimages, agreed_pre_state_commitment)
}

/// Sub-case A: `T == GT AND C == P`.
///
/// Baseline (`fbbf9089d0`) returns `Err(InvalidClaim(expected, actual))` with
/// `expected == actual == agreed_pre_state_commitment`. After the fix, returns `Ok(())`,
/// matching the existing `PreState::SuperRoot` arm at `mod.rs:86-98`.
#[tokio::test(flavor = "multi_thread")]
async fn trace_extension_transition_state_at_game_timestamp_accepts_matching_claim() {
    let t: u64 = 1000;
    let (preimages, agreed_commit) = setup_interop_preimages(t, t, B256::ZERO);
    // Overwrite key 3 with the matching commitment (sub-case A: C == P).
    let mut preimages = preimages;
    preimages.insert(PreimageKey::new_local(3), agreed_commit.as_slice().to_vec());

    let oracle = MockOracle::from_preimages(preimages);
    let hints = MockHintWriter::default();

    let result = run(oracle, hints).await;
    match result {
        Ok(()) => {}
        Err(FaultProofProgramError::InvalidClaim(expected, actual)) => {
            panic!(
                "expected Ok(()); got InvalidClaim(expected={expected}, actual={actual}) — \
                 trace-extension parity bug not yet fixed"
            );
        }
        Err(other) => panic!("unexpected error variant: {other:?}"),
    }
}

/// Sub-case B: `T == GT AND C != P`. Must remain a fail-closed `InvalidClaim` on baseline and
/// after the fix.
#[tokio::test(flavor = "multi_thread")]
async fn trace_extension_transition_state_at_game_timestamp_rejects_mismatched_claim() {
    let t: u64 = 1000;
    let mismatched_claim = b256(0xCC);
    let (preimages, agreed_commit) = setup_interop_preimages(t, t, mismatched_claim);

    let oracle = MockOracle::from_preimages(preimages);
    let hints = MockHintWriter::default();

    let err = run(oracle, hints).await.unwrap_err();
    match err {
        FaultProofProgramError::InvalidClaim(expected, actual) => {
            assert_eq!(expected, agreed_commit);
            assert_eq!(actual, mismatched_claim);
            assert_ne!(expected, actual);
        }
        other => panic!("unexpected error variant: {other:?}"),
    }
}

/// Sub-case C-eq: `T > GT AND C == P`. Baseline returns `Err(InvalidClaim)`; after the fix the
/// symmetric `>=` arm accepts and returns `Ok(())`.
#[tokio::test(flavor = "multi_thread")]
async fn trace_extension_transition_state_past_game_timestamp_accepts_matching_claim() {
    let t: u64 = 1000;
    let game_timestamp: u64 = t - 1; // T > GT
    let (preimages, agreed_commit) = setup_interop_preimages(t, game_timestamp, B256::ZERO);
    let mut preimages = preimages;
    preimages.insert(PreimageKey::new_local(3), agreed_commit.as_slice().to_vec());

    let oracle = MockOracle::from_preimages(preimages);
    let hints = MockHintWriter::default();

    let result = run(oracle, hints).await;
    match result {
        Ok(()) => {}
        Err(FaultProofProgramError::InvalidClaim(expected, actual)) => {
            panic!(
                "expected Ok(()); got InvalidClaim(expected={expected}, actual={actual}) — \
                 symmetric `>=` arm not yet covering the strict-`>` half"
            );
        }
        Err(other) => panic!("unexpected error variant: {other:?}"),
    }
}
