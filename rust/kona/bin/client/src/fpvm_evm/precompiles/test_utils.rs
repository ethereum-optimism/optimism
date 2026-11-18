//! Test utilities for accelerated precompiles.

use alloy_primitives::{Address, Bytes, keccak256};
use async_trait::async_trait;
use kona_preimage::{
    BidirectionalChannel, HintReader, HintReaderServer, HintRouter, HintWriter, NativeChannel,
    OracleReader, OracleServer, PreimageFetcher, PreimageKey, PreimageKeyType,
    PreimageOracleServer,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use kona_proof::{Hint, HintType};
use revm::precompile::PrecompileResult;
use std::{collections::HashMap, sync::Arc};
use tokio::sync::{Mutex, RwLock};

/// Runs a test with a mock host that serves [`HintType::L1Precompile`] hints and preimages. The
/// closure accepts the client's [`HintWriter`] and [`OracleReader`] as arguments.
pub(crate) async fn test_accelerated_precompile(
    f: impl Fn(&HintWriter<NativeChannel>, &OracleReader<NativeChannel>) + Send + Sync + 'static,
) {
    test_accelerated_precompile_inner(Arc::new(RwLock::new(None)), f).await;
}

/// The most recent `L1Precompile` hint payload captured by the mock host.
#[derive(Debug, Clone)]
pub(crate) struct CapturedHint {
    /// Raw hint payload: `address[20] || gas[8] || input[..]`.
    pub(crate) data: Vec<u8>,
}

impl CapturedHint {
    /// The 8-byte oracle-gas value the client embedded in the hint.
    pub(crate) fn oracle_gas(&self) -> u64 {
        u64::from_be_bytes(self.data[20..28].try_into().unwrap())
    }
}

/// Variant of [`test_accelerated_precompile`] that returns the most recent `L1Precompile` hint
/// payload after the closure completes. Tests use this to assert oracle-gas bytes at offset
/// 20..28 without reading the lock from inside the (sync) closure.
pub(crate) async fn test_accelerated_precompile_capture_hint(
    f: impl Fn(&HintWriter<NativeChannel>, &OracleReader<NativeChannel>) + Send + Sync + 'static,
) -> CapturedHint {
    let last_hint = Arc::new(RwLock::new(None));
    test_accelerated_precompile_inner(last_hint.clone(), f).await;

    let guard = last_hint.read().await;
    let raw = guard.as_ref().expect("client did not emit an L1Precompile hint");
    let parsed: Hint<HintType> = raw.parse().expect("failed to parse captured hint");
    CapturedHint { data: parsed.data.as_ref().to_vec() }
}

async fn test_accelerated_precompile_inner(
    last_hint: Arc<RwLock<Option<String>>>,
    f: impl Fn(&HintWriter<NativeChannel>, &OracleReader<NativeChannel>) + Send + Sync + 'static,
) {
    let (hint_chan, preimage_chan) =
        (BidirectionalChannel::new().unwrap(), BidirectionalChannel::new().unwrap());

    let host = tokio::task::spawn(precompile_host(
        OracleServer::new(preimage_chan.host),
        HintReader::new(hint_chan.host),
        last_hint,
    ));
    // The client closure blocks its thread on the oracle round trips, so it must not occupy a
    // runtime worker the host tasks need.
    let client = tokio::task::spawn_blocking(move || {
        let oracle_reader = OracleReader::new(preimage_chan.client);
        let hint_writer = HintWriter::new(hint_chan.client);

        (f)(&hint_writer, &oracle_reader)
    });

    tokio::try_join!(host, client).unwrap_or_else(|e| {
        panic!("Failed to join client/host: {e:?}");
    });
}

/// The claimed evaluation `y` at which the proof in [`kzg_point_eval_input`] verifies.
pub(crate) const VALID_KZG_EVALUATION: [u8; 32] =
    alloy_primitives::hex!("1522a4a7f34e1ea350ae07c29c96c7e79655aa926122e95fe69fcbd932ca49e9");

/// A KZG point-evaluation input claiming that the blob evaluates to `y`.
pub(crate) fn kzg_point_eval_input(y: [u8; 32]) -> Vec<u8> {
    use alloy_eips::eip4844::VERSIONED_HASH_VERSION_KZG;
    use alloy_primitives::hex;
    use sha2::{Digest, Sha256};

    let commitment = hex!("8f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7").to_vec();
    let mut versioned_hash = Sha256::digest(&commitment).to_vec();
    versioned_hash[0] = VERSIONED_HASH_VERSION_KZG;
    let z = hex!("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000").to_vec();
    let proof = hex!("a62ad71d14c5719385c0686f1871430475bf3a00f0aa3f7b8dd99a9abc2160744faf0070725e00b60ad9a026a15b1a8c").to_vec();
    [versioned_hash, z, y.to_vec(), commitment, proof].concat()
}

/// Executes a precompile on [`revm`].
pub(crate) fn execute_native_precompile<T: Into<Bytes>>(
    address: Address,
    input: T,
    gas: u64,
) -> PrecompileResult {
    let precompiles =
        revm::handler::EthPrecompiles::new(revm::primitives::hardfork::SpecId::PRAGUE);
    let Some(precompile) = precompiles.precompiles.get(&address) else {
        panic!("Precompile not found");
    };
    precompile.execute(&input.into(), gas, 0)
}

/// Starts a mock host thread that serves [`HintType::L1Precompile`] hints and preimages.
///
/// `last_hint` is shared with the caller so tests using
/// [`test_accelerated_precompile_capture_hint`] can read the most recently routed hint
/// after the client closure completes.
async fn precompile_host(
    oracle_server: OracleServer<NativeChannel>,
    hint_reader: HintReader<NativeChannel>,
    last_hint: Arc<RwLock<Option<String>>>,
) {
    let preimage_fetcher =
        PrecompilePreimageFetcher { map: Default::default(), last_hint: last_hint.clone() };
    let hint_router = PrecompileHintRouter { last_hint };

    let server = tokio::task::spawn(async move {
        loop {
            match oracle_server.next_preimage_request(&preimage_fetcher).await {
                Ok(_) => continue,
                Err(PreimageOracleError::IOError(_)) => return,
                Err(e) => {
                    panic!("Critical: Failed to serve preimage: {e:?}");
                }
            }
        }
    });
    let hint_reader = tokio::task::spawn(async move {
        loop {
            match hint_reader.next_hint(&hint_router).await {
                Ok(_) => continue,
                Err(PreimageOracleError::IOError(_)) => return,
                Err(e) => {
                    panic!("Critical: Failed to serve hint: {e:?}");
                }
            }
        }
    });

    tokio::try_join!(server, hint_reader).unwrap_or_else(|e| {
        panic!("Failed to join server/hint reader: {e:?}");
    });
}

#[derive(Default, Debug, Clone)]
struct PrecompilePreimageFetcher {
    /// Inner map of preimages.
    map: Arc<Mutex<HashMap<PreimageKey, Vec<u8>>>>,
    /// The previous hint received.
    last_hint: Arc<RwLock<Option<String>>>,
}

#[async_trait]
impl PreimageFetcher for PrecompilePreimageFetcher {
    async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let mut map_lock = self.map.lock().await;
        if let Some(preimage) = map_lock.get(&key) {
            return Ok(preimage.clone());
        }

        let last_hint = self.last_hint.read().await;
        let Some(last_hint) = last_hint.as_ref() else { unreachable!("Hint not queued") };

        let parsed_hint = last_hint.parse::<Hint<HintType>>().unwrap();
        if matches!(parsed_hint.ty, HintType::L1Precompile) {
            let address = Address::from_slice(&parsed_hint.data.as_ref()[..20]);
            let gas = u64::from_be_bytes(parsed_hint.data.as_ref()[20..28].try_into().unwrap());
            let input = parsed_hint.data[28..].to_vec();
            let input_hash = keccak256(parsed_hint.data.as_ref());

            let result = match execute_native_precompile(address, input, gas) {
                Ok(raw_res) if raw_res.is_success() => {
                    let mut res = Vec::with_capacity(1 + raw_res.bytes.len());
                    res.push(0x01);
                    res.extend_from_slice(&raw_res.bytes);
                    res
                }
                _ => vec![0u8; 1],
            };

            map_lock
                .insert(PreimageKey::new(*input_hash, PreimageKeyType::Precompile), result.clone());
            return Ok(result);
        } else {
            panic!("Unexpected hint type: {:?}", parsed_hint.ty);
        }
    }
}

#[derive(Default, Debug, Clone)]
struct PrecompileHintRouter {
    /// The latest hint received.
    last_hint: Arc<RwLock<Option<String>>>,
}

#[async_trait]
impl HintRouter for PrecompileHintRouter {
    async fn route_hint(&self, hint: String) -> PreimageOracleResult<()> {
        self.last_hint.write().await.replace(hint);
        Ok(())
    }
}
