use alloy_consensus::Header;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_client::single::{FaultProofProgramError, run};
use kona_preimage::{
    HintWriterClient, PreimageKey, PreimageOracleClient,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use std::{collections::HashMap, sync::Arc};
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

fn setup_preimages(
    agreed_root: B256,
    claimed_root: B256,
    claimed_block_number: u64,
    safe_head_hash: B256,
    safe_head_number: u64,
) -> HashMap<PreimageKey, Vec<u8>> {
    let mut preimages = HashMap::new();

    // BootInfo local keys (see `kona_proof::boot`):
    // 1: L1 head hash, 2: agreed output root, 3: claimed output root, 4: claimed block number,
    // 5: chain id.
    preimages.insert(PreimageKey::new_local(1), b256(0x11).as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(2), agreed_root.as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(3), claimed_root.as_slice().to_vec());
    preimages.insert(PreimageKey::new_local(4), claimed_block_number.to_be_bytes().to_vec());
    preimages.insert(PreimageKey::new_local(5), 10u64.to_be_bytes().to_vec());

    // Output root preimage for `fetch_safe_head_hash(...)`. The safe head hash is read from
    // bytes [96..128].
    let mut output_root_preimage = [0u8; 128];
    output_root_preimage[96..128].copy_from_slice(safe_head_hash.as_slice());
    preimages.insert(PreimageKey::new_keccak256(*agreed_root), output_root_preimage.to_vec());

    // L2 safe head header.
    let header = Header { number: safe_head_number, ..Default::default() };
    let header_rlp = alloy_rlp::encode(&header).to_vec();
    preimages.insert(PreimageKey::new_keccak256(*safe_head_hash), header_rlp);

    preimages
}

#[tokio::test(flavor = "multi_thread")]
async fn trace_extension_leaf_rejects_mismatched_output_root() {
    let safe_head_number = 3;
    let safe_head_hash = b256(0x22);
    let agreed_root = b256(0xAA);
    let claimed_root = b256(0xBB);

    let oracle = MockOracle::from_preimages(setup_preimages(
        agreed_root,
        claimed_root,
        safe_head_number,
        safe_head_hash,
        safe_head_number,
    ));
    let hints = MockHintWriter::default();

    let err = run(oracle, hints).await.unwrap_err();
    match err {
        FaultProofProgramError::InvalidClaim(expected, actual) => {
            assert_eq!(expected, agreed_root);
            assert_eq!(actual, claimed_root);
        }
        other => panic!("unexpected error: {other:?}"),
    }
}

#[tokio::test(flavor = "multi_thread")]
async fn trace_extension_leaf_accepts_matching_output_root() {
    let safe_head_number = 3;
    let safe_head_hash = b256(0x22);
    let agreed_root = b256(0xAA);

    let oracle = MockOracle::from_preimages(setup_preimages(
        agreed_root,
        agreed_root,
        safe_head_number,
        safe_head_hash,
        safe_head_number,
    ));
    let hints = MockHintWriter::default();

    run(oracle, hints).await.unwrap();
}

#[tokio::test(flavor = "multi_thread")]
async fn does_not_short_circuit_on_root_match_at_different_block() {
    let safe_head_number = 3;
    let safe_head_hash = b256(0x22);
    let agreed_root = b256(0xAA);

    // With the historical bug (checking only `agreed_root == claimed_root`), this would
    // incorrectly return `Ok(())` without attempting derivation.
    let oracle = MockOracle::from_preimages(setup_preimages(
        agreed_root,
        agreed_root,
        safe_head_number + 1,
        safe_head_hash,
        safe_head_number,
    ));
    let hints = MockHintWriter::default();

    assert!(run(oracle, hints).await.is_err());
}
