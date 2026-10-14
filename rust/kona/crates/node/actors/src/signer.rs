//! Signer Actor

use alloy_primitives::Address;
use alloy_signer::Signature;
use async_trait::async_trait;
use kona_sources::{BlockSignerError, BlockSignerHandler};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{fmt::Debug, future::Future};
use thiserror::Error;
use tokio::sync::{
    mpsc::{self, error::SendError},
    watch,
};

/// Queues unsigned payloads for signing and gossip.
///
/// Payloads are signed in order. Transient signing failures are retried, allowing the bounded
/// queue to apply backpressure to the sequencer while signing stalls.
#[derive(Debug, Clone)]
pub struct Handle {
    payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl Handle {
    /// Queues a payload, waiting for room when the signer is backed up.
    ///
    /// Success means the payload was queued, not that it was signed or gossiped.
    pub async fn send(&self, payload: OpExecutionPayloadEnvelope) -> Result<(), Stopped> {
        self.payloads.send(payload).await.map_err(|_| Stopped)
    }

    /// Waits until the queue has room for another payload.
    pub async fn ready(&self) -> Result<(), Stopped> {
        self.payloads.reserve().await.map(drop).map_err(|_| Stopped)
    }
}

/// The signer has stopped, so it takes no more payloads.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("the signer has stopped")]
pub struct Stopped;

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
}

impl Builder {
    /// Creates the builder.
    ///
    /// The queue holds one payload, so the sequencer stops building once the signer falls a
    /// payload behind.
    pub fn new() -> Self {
        let (payloads_tx, payloads) = mpsc::channel(1);
        Self { handle: Handle { payloads: payloads_tx }, payloads }
    }

    /// Returns a handle that can be wired into other components before the signer starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the signer's lifetime future without spawning it.
    ///
    /// The caller must retain a handle to keep the input channel open.
    /// Dropping the future stops the actor.
    pub fn build<Network_: Network>(
        self,
        signer: BlockSignerHandler,
        chain_id: u64,
        unsafe_block_signer: watch::Receiver<Address>,
        network: Network_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let actor =
            Actor { signer, chain_id, unsafe_block_signer, payloads: self.payloads, network };
        actor.run()
    }
}

impl Default for Builder {
    fn default() -> Self {
        Self::new()
    }
}

/// The network that publishes the payloads the signer signs.
#[async_trait]
pub trait Network: Debug + Send + Sync + 'static {
    /// An error queuing a payload.
    type Error: std::error::Error + Send + Sync + 'static;

    /// Queues a signed payload to publish, waiting for room in the queue.
    ///
    /// Success means the payload was queued, not that it was published.
    async fn publish(&self, payload: Payload) -> Result<(), Self::Error>;
}

#[async_trait]
impl Network for mpsc::Sender<Payload> {
    type Error = SendError<Payload>;

    async fn publish(&self, payload: Payload) -> Result<(), Self::Error> {
        self.send(payload).await
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
/// [network actor](crate::network) to gossip.
///
/// The signer retries its own transient failures, such as a remote signer that is unreachable or
/// does not answer in time, until it succeeds, so no payload is skipped or reordered. While a
/// payload waits, the sequencer's payload queue fills, and then the sequencer stops building blocks
/// until it drains, while still answering admin queries. The signer likewise waits for room in the
/// network's queue. Any error the signer returns is fatal.
#[derive(Debug)]
struct Actor<Network_> {
    /// Signs the payloads.
    signer: BlockSignerHandler,
    /// The L2 chain ID the signatures commit to.
    chain_id: u64,
    /// The unsafe block signer currently set in `SystemConfig`, read for each payload.
    unsafe_block_signer: watch::Receiver<Address>,
    /// Payloads from the sequencer, in the order they were built.
    payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// Publishes the signed payloads, in the same order.
    network: Network_,
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

impl<Network_: Network> Actor<Network_> {
    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            let payload = self.payloads.recv().await.ok_or(ActorError::ChannelClosed)?;
            // A remote signer rejects an address that is not its own, so a rotation seen before
            // this call fails with `InvalidAddress`. A local signer ignores the
            // address.
            let sender = *self.unsafe_block_signer.borrow();
            let signature =
                self.signer.sign_block(payload.payload_hash(), self.chain_id, sender).await?;
            self.network
                .publish(Payload { payload, signature })
                .await
                .map_err(|_| ActorError::ChannelClosed)?;
        }
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
    }

    impl Drop for Harness {
        fn drop(&mut self) {
            self.task.abort();
        }
    }

    fn harness(signer: BlockSignerHandler, unsafe_block_signer: Address) -> Harness {
        let builder = Builder::new();
        let payloads = builder.handle();
        let (signed_tx, signed) = mpsc::channel(8);
        let task = tokio::spawn(builder.build(
            signer,
            CHAIN_ID,
            watch::channel(unsafe_block_signer).1,
            signed_tx,
        ));
        Harness { payloads, signed, task }
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
            h.payloads.send(crate::test_payload(number)).await.unwrap();
        }
        for number in 1..=3 {
            let signed = h.signed.recv().await.unwrap();
            assert_eq!(signed.payload, crate::test_payload(number));
            assert_eq!(signed.signature, signature(&key, &signed.payload));
        }
    }

    #[tokio::test]
    async fn signer_error_is_fatal() {
        let key = PrivateKeySigner::random();
        let (signer, calls, _server) = remote_signer(&key).await;
        // `SystemConfig` names a different signer than the remote signer holds.
        let mut h = harness(signer, Address::repeat_byte(1));
        h.payloads.send(crate::test_payload(1)).await.unwrap();
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
        let builder = Builder::new();
        let handle = builder.handle();
        let key = PrivateKeySigner::random();
        let address = key.address();
        let (signed_tx, _signed_rx) = mpsc::channel(8);
        let task = builder.build(
            BlockSignerHandler::Local(key),
            CHAIN_ID,
            watch::channel(address).1,
            signed_tx,
        );
        drop(handle);

        assert!(matches!(task.await, Err(ActorError::ChannelClosed)));
    }

    #[tokio::test]
    async fn dropping_lifetime_closes_a_full_output_queue() {
        let builder = Builder::new();
        let handle = builder.handle();
        let key = PrivateKeySigner::random();
        let address = key.address();
        let (signed_tx, mut signed_rx) = mpsc::channel(1);
        let mut task = Box::pin(builder.build(
            BlockSignerHandler::Local(key),
            CHAIN_ID,
            watch::channel(address).1,
            signed_tx,
        ));
        // The first payload fills the output queue.
        handle.send(crate::test_payload(1)).await.unwrap();
        assert!(futures::poll!(task.as_mut()).is_pending());
        assert_eq!(signed_rx.len(), 1);
        // The second is taken from the input queue, and waits for room in the output queue.
        handle.send(crate::test_payload(2)).await.unwrap();
        assert!(futures::poll!(task.as_mut()).is_pending());
        assert!(futures::poll!(Box::pin(handle.ready())).is_ready());
        drop(task);
        assert_eq!(signed_rx.recv().await.unwrap().payload, crate::test_payload(1));
        assert!(signed_rx.recv().await.is_none());
    }

    #[tokio::test]
    async fn ready_waits_for_room_and_fails_once_the_signer_stops() {
        let builder = Builder::new();
        let handle = builder.handle();
        handle.send(crate::test_payload(1)).await.unwrap();
        let ready = handle.ready();
        tokio::pin!(ready);
        assert!(futures::poll!(ready.as_mut()).is_pending());

        // Dropping the builder drops the signer's end of the queue.
        drop(builder);
        assert_eq!(ready.await, Err(Stopped));
        assert_eq!(handle.send(crate::test_payload(2)).await, Err(Stopped));
    }
}
