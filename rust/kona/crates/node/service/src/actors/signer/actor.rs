use crate::NodeActor;
use alloy_primitives::Address;
use alloy_signer::Signature;
use async_trait::async_trait;
use kona_sources::{BlockSignerError, BlockSignerHandler, RemoteSignerError};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::{
    sync::{mpsc, watch},
    time::{Duration, sleep, timeout},
};

/// Deadline for one signing attempt.
const SIGNING_TIMEOUT: Duration = Duration::from_secs(2);
/// Delay before the first retry of a failed signing attempt. It doubles up to
/// [`MAX_RETRY_DELAY`].
const MIN_RETRY_DELAY: Duration = Duration::from_millis(100);
/// Longest delay between signing attempts.
const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);

/// A payload and the signature to gossip it with.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SignedPayload {
    /// The payload to gossip.
    pub payload: OpExecutionPayloadEnvelope,
    /// The signature over the payload hash.
    pub signature: Signature,
}

/// Signs each payload the sequencer produces, in order, and passes it to the
/// [`NetworkActor`](crate::NetworkActor) to gossip.
///
/// A transient signer failure, such as a remote signer that is unreachable or does not answer in
/// time, is retried until it succeeds, so no payload is skipped or reordered. While a payload
/// waits, the sequencer's payload queue fills and then blocks block production: an outage that
/// outlasts the queue halts the sequencer instead of letting gossip fall behind. Any other signing
/// error is fatal.
///
/// Without a signer, payloads are dropped with a warning.
#[derive(Debug)]
pub struct SignerActor {
    /// Signs the payloads, if one is configured.
    signer: Option<BlockSignerHandler>,
    /// The L2 chain ID the signatures commit to.
    chain_id: u64,
    /// The unsafe block signer currently set in `SystemConfig`.
    unsafe_block_signer: watch::Receiver<Address>,
    /// Payloads from the sequencer, in the order they were built.
    payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// Signed payloads for the network actor, in the same order.
    signed: mpsc::Sender<SignedPayload>,
}

/// An error from the [`SignerActor`].
#[derive(Debug, Error)]
pub enum SignerActorError {
    /// A channel to another actor closed.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
    /// The signer failed for a reason that retrying cannot fix.
    #[error("Failed to sign the payload: {0}")]
    Signing(#[from] BlockSignerError),
}

impl SignerActor {
    /// Constructs a new [`SignerActor`].
    pub const fn new(
        signer: Option<BlockSignerHandler>,
        chain_id: u64,
        unsafe_block_signer: watch::Receiver<Address>,
        payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        signed: mpsc::Sender<SignedPayload>,
    ) -> Self {
        Self { signer, chain_id, unsafe_block_signer, payloads, signed }
    }

    /// Signs `payload` with `signer`, retrying transient failures until one attempt succeeds.
    async fn sign(
        &self,
        signer: &BlockSignerHandler,
        payload: &OpExecutionPayloadEnvelope,
    ) -> Result<Signature, BlockSignerError> {
        let payload_hash = payload.payload_hash();
        let mut delay = MIN_RETRY_DELAY;
        loop {
            // Read on each attempt: after a rotation, a remote signer then fails with
            // `InvalidAddress` instead of signing for a retired key. A local signer ignores it.
            let sender = *self.unsafe_block_signer.borrow();
            match timeout(SIGNING_TIMEOUT, signer.sign_block(payload_hash, self.chain_id, sender))
                .await
            {
                Ok(Ok(signature)) => return Ok(signature),
                Ok(Err(BlockSignerError::Remote(RemoteSignerError::SigningRPCError(err)))) => {
                    warn!(target: "signer", ?err, ?delay, "Remote signer unavailable; retrying");
                }
                Ok(Err(err)) => return Err(err),
                Err(_) => warn!(target: "signer", ?delay, "Signing attempt timed out; retrying"),
            }
            sleep(delay).await;
            delay = (delay * 2).min(MAX_RETRY_DELAY);
        }
    }
}

#[async_trait]
impl NodeActor for SignerActor {
    type Error = SignerActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let payload = self.payloads.recv().await.ok_or(SignerActorError::ChannelClosed)?;
        let Some(signer) = &self.signer else {
            warn!(target: "signer", "No block signer configured; not gossiping the payload");
            return Ok(());
        };
        let signature = self.sign(signer, &payload).await?;
        self.signed
            .send(SignedPayload { payload, signature })
            .await
            .map_err(|_| SignerActorError::ChannelClosed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV3};
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use arbitrary::Arbitrary;
    use jsonrpsee::{
        RpcModule,
        server::{ServerBuilder, ServerHandle},
        types::ErrorObjectOwned,
    };
    use kona_sources::RemoteSigner;
    use rand::Rng;
    use std::sync::{
        Arc,
        atomic::{AtomicUsize, Ordering},
    };

    const CHAIN_ID: u64 = 10;

    fn payload(number: u64) -> OpExecutionPayloadEnvelope {
        let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
        block.header.number = number;
        OpExecutionPayloadEnvelope::V1(alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(
            &block,
        ))
    }

    fn signature(key: &PrivateKeySigner, payload: &OpExecutionPayloadEnvelope) -> Signature {
        key.sign_hash_sync(&payload.payload_hash().signature_message(CHAIN_ID)).unwrap()
    }

    /// Starts a remote signer for `key` that fails its first `failures` signing requests. It runs
    /// until the returned [`ServerHandle`] is dropped.
    async fn remote_signer(
        key: &PrivateKeySigner,
        failures: usize,
    ) -> (BlockSignerHandler, Arc<AtomicUsize>, ServerHandle) {
        let calls = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let endpoint = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        let mut module = RpcModule::new((calls.clone(), key.clone()));
        module.register_method("health_status", |_, _, _| "ok").unwrap();
        module
            .register_method("opsigner_signBlockPayload", move |params, ctx, _| {
                let (calls, key) = ctx;
                if calls.fetch_add(1, Ordering::SeqCst) < failures {
                    return Err(ErrorObjectOwned::owned(-32000, "unavailable", None::<()>));
                }
                // The client sends the arguments as a single object.
                let args: serde_json::Value = params.parse().unwrap();
                let hash: alloy_primitives::B256 =
                    serde_json::from_value(args["payloadHash"].clone()).unwrap();
                let message = op_alloy_rpc_types_engine::PayloadHash(hash).signature_message(CHAIN_ID);
                Ok(serde_json::json!({"signature": key.sign_hash_sync(&message).unwrap().to_string()}))
            })
            .unwrap();
        let server = server.start(module);
        let signer = RemoteSigner {
            endpoint,
            address: key.address(),
            client_cert: None,
            ca_cert: None,
            headers: Default::default(),
        }
        .start()
        .await
        .unwrap();
        (BlockSignerHandler::Remote(signer), calls, server)
    }

    struct Harness {
        payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
        signed: mpsc::Receiver<SignedPayload>,
        actor: SignerActor,
    }

    fn harness(signer: Option<BlockSignerHandler>, unsafe_block_signer: Address) -> Harness {
        let (payloads, payloads_rx) = mpsc::channel(8);
        let (signed_tx, signed) = mpsc::channel(8);
        let actor = SignerActor::new(
            signer,
            CHAIN_ID,
            watch::channel(unsafe_block_signer).1,
            payloads_rx,
            signed_tx,
        );
        Harness { payloads, signed, actor }
    }

    #[test]
    fn test_payload_signature_v1() {
        let mut bytes = [0u8; 4096];
        rand::rng().fill(bytes.as_mut_slice());

        let pubkey = PrivateKeySigner::random();
        let expected_address = pubkey.address();

        let block = OpExecutionPayloadEnvelope::V1(
            ExecutionPayloadV1::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
        );

        let payload_hash = block.payload_hash();
        let message = payload_hash.signature_message(CHAIN_ID);
        let signature = pubkey.sign_hash_sync(&message).unwrap();
        let msg_signer = signature.recover_address_from_prehash(&message).unwrap();

        assert_eq!(expected_address, msg_signer);
    }

    #[test]
    fn test_payload_signature_v3() {
        let mut bytes = [0u8; 4096];
        rand::rng().fill(bytes.as_mut_slice());

        let pubkey = PrivateKeySigner::random();
        let expected_address = pubkey.address();

        let block = OpExecutionPayloadEnvelope::V3 {
            payload: ExecutionPayloadV3::arbitrary(&mut arbitrary::Unstructured::new(&bytes))
                .unwrap(),
            parent_beacon_block_root: B256::random(),
        };

        let payload_hash = block.payload_hash();
        let message = payload_hash.signature_message(CHAIN_ID);
        let signature = pubkey.sign_hash_sync(&message).unwrap();
        let msg_signer = signature.recover_address_from_prehash(&message).unwrap();

        assert_eq!(expected_address, msg_signer);
    }

    #[tokio::test]
    async fn signs_payloads_in_order() {
        let key = PrivateKeySigner::random();
        let mut h = harness(Some(BlockSignerHandler::Local(key.clone())), key.address());
        for number in 1..=3 {
            h.payloads.send(payload(number)).await.unwrap();
        }
        for number in 1..=3 {
            h.actor.step().await.unwrap();
            let signed = h.signed.recv().await.unwrap();
            assert_eq!(signed.payload, payload(number));
            assert_eq!(signed.signature, signature(&key, &signed.payload));
        }
    }

    #[tokio::test]
    async fn transient_failures_are_retried_without_skipping_a_payload() {
        let key = PrivateKeySigner::random();
        let (signer, calls, _server) = remote_signer(&key, 3).await;
        let mut h = harness(Some(signer), key.address());
        h.payloads.send(payload(1)).await.unwrap();
        h.payloads.send(payload(2)).await.unwrap();

        h.actor.step().await.unwrap();
        // Three failed attempts, then the first payload is signed rather than skipped.
        assert_eq!(calls.load(Ordering::SeqCst), 4);
        let first = h.signed.recv().await.unwrap();
        assert_eq!(first.payload, payload(1));
        assert_eq!(first.signature, signature(&key, &first.payload));

        h.actor.step().await.unwrap();
        assert_eq!(h.signed.recv().await.unwrap().payload, payload(2));
        assert_eq!(calls.load(Ordering::SeqCst), 5);
    }

    #[tokio::test]
    async fn non_transient_signer_error_is_fatal() {
        let key = PrivateKeySigner::random();
        let (signer, calls, _server) = remote_signer(&key, 0).await;
        // `SystemConfig` names a different signer than the remote signer holds.
        let mut h = harness(Some(signer), Address::repeat_byte(1));
        h.payloads.send(payload(1)).await.unwrap();
        assert!(matches!(
            h.actor.step().await,
            Err(SignerActorError::Signing(BlockSignerError::Remote(
                RemoteSignerError::InvalidAddress { .. }
            )))
        ));
        assert_eq!(calls.load(Ordering::SeqCst), 0);
        assert!(h.signed.try_recv().is_err());
    }

    #[tokio::test]
    async fn without_a_signer_payloads_are_dropped() {
        let mut h = harness(None, Address::ZERO);
        h.payloads.send(payload(1)).await.unwrap();
        h.actor.step().await.unwrap();
        assert!(h.signed.try_recv().is_err());
    }
}
