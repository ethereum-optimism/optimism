//! Signer Actor

use crate::{UnsafePayloadGossipClient, UnsafePayloadGossipClientError};
use alloy_primitives::Address;
use alloy_signer::Signature;
use async_trait::async_trait;
use kona_sources::{BlockSignerError, BlockSignerHandler};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::future::Future;
use thiserror::Error;
use tokio::sync::{Semaphore, mpsc, watch};
use tokio_util::sync::CancellationToken;

/// A value in `1..=tokio::sync::Semaphore::MAX_PERMITS`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Capacity(usize);

impl Capacity {
    /// Returns the value.
    pub const fn get(self) -> usize {
        self.0
    }
}

impl TryFrom<usize> for Capacity {
    type Error = InvalidCapacity;

    fn try_from(value: usize) -> Result<Self, Self::Error> {
        if (1..=Semaphore::MAX_PERMITS).contains(&value) {
            Ok(Self(value))
        } else {
            Err(InvalidCapacity(value))
        }
    }
}

/// An integer outside the supported range.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("mailbox capacity {0} must be between 1 and {max}", max = Semaphore::MAX_PERMITS)]
pub struct InvalidCapacity(usize);

/// Queues unsigned payloads for signing and gossip.
///
/// Payloads are signed in order. Transient signing failures are retried, allowing the bounded
/// queue to apply backpressure to the sequencer while signing stalls.
#[derive(Debug, Clone)]
pub struct Handle {
    payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl Handle {
    /// Queues a payload, waiting for capacity when the signer is backed up.
    ///
    /// Success means the payload was queued, not that it was signed or gossiped.
    pub async fn send(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError> {
        self.payloads
            .send(payload.clone())
            .await
            .map_err(|_| UnsafePayloadGossipClientError::RequestError("request channel closed".to_string()))
            .inspect_err(|err| error!(target: "gossip_client", ?payload, ?err, "failed to request to gossip payload."))
    }

    /// Whether the input queue currently has room for another payload.
    pub fn has_capacity(&self) -> bool {
        self.payloads.capacity() > 0
    }
}

#[async_trait]
impl UnsafePayloadGossipClient for Handle {
    async fn schedule_execution_payload_gossip(
        &self,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadGossipClientError> {
        self.send(payload).await
    }

    fn has_capacity(&self) -> bool {
        self.has_capacity()
    }
}

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
}

impl Builder {
    /// Creates the builder.
    pub fn new(capacity: Capacity) -> Self {
        let (payloads_tx, payloads) = mpsc::channel(capacity.get());
        Self { handle: Handle { payloads: payloads_tx }, payloads }
    }

    /// Returns a handle that can be wired into other components before the signer starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the signer's lifetime future without spawning it.
    ///
    /// The caller must retain a handle to keep the input channel open.
    pub fn build(
        self,
        signer: BlockSignerHandler,
        chain_id: u64,
        unsafe_block_signer: watch::Receiver<Address>,
        signed: mpsc::Sender<Payload>,
        cancellation: CancellationToken,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let actor =
            Actor { signer, chain_id, unsafe_block_signer, payloads: self.payloads, signed };
        actor.run(cancellation)
    }
}

/// A payload and the signature to gossip it with.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Payload {
    /// The payload to gossip.
    pub payload: OpExecutionPayloadEnvelope,
    /// The signature over the payload hash.
    pub signature: Signature,
}

/// Signs each payload the sequencer produces, in order, and passes it to the
/// [`NetworkActor`](crate::NetworkActor) to gossip.
///
/// The signer retries its own transient failures, such as a remote signer that is unreachable or
/// does not answer in time, until it succeeds, so no payload is skipped or reordered. While a
/// payload waits, the sequencer's payload queue fills, and then the sequencer stops building blocks
/// until it drains, while still answering admin queries. Any error the signer returns is fatal.
#[derive(Debug)]
struct Actor {
    /// Signs the payloads.
    signer: BlockSignerHandler,
    /// The L2 chain ID the signatures commit to.
    chain_id: u64,
    /// The unsafe block signer currently set in `SystemConfig`, read for each payload.
    unsafe_block_signer: watch::Receiver<Address>,
    /// Payloads from the sequencer, in the order they were built.
    payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// Signed payloads for the network actor, in the same order.
    signed: mpsc::Sender<Payload>,
}

/// A fatal error from the signer actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// A channel to another actor closed.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
    /// The signer failed for a reason that retrying cannot fix.
    #[error("Failed to sign the payload: {0}")]
    Signing(#[from] BlockSignerError),
}

impl Actor {
    async fn run(mut self, cancellation: CancellationToken) -> Result<(), ActorError> {
        loop {
            tokio::select! {
                biased;
                _ = cancellation.cancelled() => return Ok(()),
                result = self.sign() => result?,
            }
        }
    }

    async fn sign(&mut self) -> Result<(), ActorError> {
        let payload = self.payloads.recv().await.ok_or(ActorError::ChannelClosed)?;
        // A remote signer rejects an address that is not its own, so a rotation seen before this
        // call fails with `InvalidAddress`. A local signer ignores the address.
        let sender = *self.unsafe_block_signer.borrow();
        let signature =
            self.signer.sign_block(payload.payload_hash(), self.chain_id, sender).await?;
        self.signed
            .send(Payload { payload, signature })
            .await
            .map_err(|_| ActorError::ChannelClosed)
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
    };
    use kona_sources::{RemoteSigner, RemoteSignerError};
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

    /// Starts a remote signer for `key` that signs every request and counts them. It runs
    /// until the returned [`ServerHandle`] is dropped.
    async fn remote_signer(
        key: &PrivateKeySigner,
    ) -> (BlockSignerHandler, Arc<AtomicUsize>, ServerHandle) {
        let calls = Arc::new(AtomicUsize::new(0));
        let server = ServerBuilder::default().build("127.0.0.1:0").await.unwrap();
        let endpoint = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        let mut module = RpcModule::new((calls.clone(), key.clone()));
        module.register_method("health_status", |_, _, _| "ok").unwrap();
        module
            .register_method("opsigner_signBlockPayload", move |params, ctx, _| {
                let (calls, key) = ctx;
                calls.fetch_add(1, Ordering::SeqCst);
                // The client sends the arguments as a single object.
                let args: serde_json::Value = params.parse().unwrap();
                let hash: alloy_primitives::B256 =
                    serde_json::from_value(args["payloadHash"].clone()).unwrap();
                let message =
                    op_alloy_rpc_types_engine::PayloadHash(hash).signature_message(CHAIN_ID);
                serde_json::json!({"signature": key.sign_hash_sync(&message).unwrap().to_string()})
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
        payloads: Handle,
        signed: mpsc::Receiver<Payload>,
        task: tokio::task::JoinHandle<Result<(), ActorError>>,
        cancellation: CancellationToken,
    }

    impl Drop for Harness {
        fn drop(&mut self) {
            self.task.abort();
        }
    }

    fn harness(signer: BlockSignerHandler, unsafe_block_signer: Address) -> Harness {
        let builder = Builder::new(Capacity::try_from(8).unwrap());
        let payloads = builder.handle();
        let (signed_tx, signed) = mpsc::channel(8);
        let cancellation = CancellationToken::new();
        let task = tokio::spawn(builder.build(
            signer,
            CHAIN_ID,
            watch::channel(unsafe_block_signer).1,
            signed_tx,
            cancellation.clone(),
        ));
        Harness { payloads, signed, task, cancellation }
    }

    #[test]
    fn capacity_parsing_enforces_channel_bounds() {
        for value in [1, Semaphore::MAX_PERMITS] {
            assert_eq!(Capacity::try_from(value).map(Capacity::get), Ok(value));
        }
        for value in [0, Semaphore::MAX_PERMITS + 1, usize::MAX] {
            assert!(Capacity::try_from(value).is_err());
        }
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
        let mut h = harness(BlockSignerHandler::Local(key.clone()), key.address());
        for number in 1..=3 {
            h.payloads.send(payload(number)).await.unwrap();
        }
        for number in 1..=3 {
            let signed = h.signed.recv().await.unwrap();
            assert_eq!(signed.payload, payload(number));
            assert_eq!(signed.signature, signature(&key, &signed.payload));
        }
        h.cancellation.cancel();
        assert!((&mut h.task).await.unwrap().is_ok());
    }

    #[tokio::test]
    async fn signer_error_is_fatal() {
        let key = PrivateKeySigner::random();
        let (signer, calls, _server) = remote_signer(&key).await;
        // `SystemConfig` names a different signer than the remote signer holds.
        let mut h = harness(signer, Address::repeat_byte(1));
        h.payloads.send(payload(1)).await.unwrap();
        assert!(matches!(
            (&mut h.task).await.unwrap(),
            Err(ActorError::Signing(BlockSignerError::Remote(
                RemoteSignerError::InvalidAddress { .. }
            )))
        ));
        assert_eq!(calls.load(Ordering::SeqCst), 0);
        assert!(h.signed.try_recv().is_err());
    }

    #[tokio::test]
    async fn closing_input_is_fatal() {
        let builder = Builder::new(Capacity::try_from(8).unwrap());
        let handle = builder.handle();
        let key = PrivateKeySigner::random();
        let address = key.address();
        let (signed_tx, _signed_rx) = mpsc::channel(8);
        let task = builder.build(
            BlockSignerHandler::Local(key),
            CHAIN_ID,
            watch::channel(address).1,
            signed_tx,
            CancellationToken::new(),
        );
        drop(handle);

        assert!(matches!(task.await, Err(ActorError::ChannelClosed)));
    }

    #[tokio::test]
    async fn cancellation_interrupts_full_output_queue() {
        let builder = Builder::new(Capacity::try_from(2).unwrap());
        let handle = builder.handle();
        let key = PrivateKeySigner::random();
        let address = key.address();
        let (signed_tx, mut signed_rx) = mpsc::channel(1);
        let cancellation = CancellationToken::new();
        let task = builder.build(
            BlockSignerHandler::Local(key),
            CHAIN_ID,
            watch::channel(address).1,
            signed_tx,
            cancellation.clone(),
        );
        tokio::pin!(task);
        handle.send(payload(1)).await.unwrap();
        handle.send(payload(2)).await.unwrap();

        assert!(futures::poll!(&mut task).is_pending());
        assert_eq!(signed_rx.len(), 1);
        assert_eq!(handle.payloads.capacity(), 2);
        cancellation.cancel();

        assert!(task.await.is_ok());
        assert_eq!(signed_rx.recv().await.unwrap().payload, payload(1));
        assert!(signed_rx.recv().await.is_none());
    }
}
