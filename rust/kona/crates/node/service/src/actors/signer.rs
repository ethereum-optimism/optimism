//! Signer Actor

use crate::NodeActor;
use alloy_primitives::Address;
use alloy_signer::Signature;
use async_trait::async_trait;
use kona_sources::{BlockSignerError, BlockSignerHandler};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::{mpsc, watch};

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
/// The signer retries its own transient failures, such as a remote signer that is unreachable or
/// does not answer in time, until it succeeds, so no payload is skipped or reordered. While a
/// payload waits, the sequencer's payload queue fills, and then the sequencer stops building blocks
/// until it drains, while still answering admin queries. Any error the signer returns is fatal.
#[derive(Debug)]
pub struct SignerActor {
    /// Signs the payloads.
    signer: BlockSignerHandler,
    /// The L2 chain ID the signatures commit to.
    chain_id: u64,
    /// The unsafe block signer currently set in `SystemConfig`, read for each payload.
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
        signer: BlockSignerHandler,
        chain_id: u64,
        unsafe_block_signer: watch::Receiver<Address>,
        payloads: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        signed: mpsc::Sender<SignedPayload>,
    ) -> Self {
        Self { signer, chain_id, unsafe_block_signer, payloads, signed }
    }
}

#[async_trait]
impl NodeActor for SignerActor {
    type Error = SignerActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let payload = self.payloads.recv().await.ok_or(SignerActorError::ChannelClosed)?;
        // A remote signer rejects an address that is not its own, so a rotation seen before this
        // call fails with `InvalidAddress`. A local signer ignores the address.
        let sender = *self.unsafe_block_signer.borrow();
        let signature =
            self.signer.sign_block(payload.payload_hash(), self.chain_id, sender).await?;
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
        payloads: mpsc::Sender<OpExecutionPayloadEnvelope>,
        signed: mpsc::Receiver<SignedPayload>,
        actor: SignerActor,
    }

    fn harness(signer: BlockSignerHandler, unsafe_block_signer: Address) -> Harness {
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
        let mut h = harness(BlockSignerHandler::Local(key.clone()), key.address());
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
    async fn signer_error_is_fatal() {
        let key = PrivateKeySigner::random();
        let (signer, calls, _server) = remote_signer(&key).await;
        // `SystemConfig` names a different signer than the remote signer holds.
        let mut h = harness(signer, Address::repeat_byte(1));
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
}
