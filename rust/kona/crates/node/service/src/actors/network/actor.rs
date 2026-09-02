use alloy_primitives::{Address, ChainId};
use alloy_signer::Signature;
use async_trait::async_trait;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use kona_sources::{BlockSignerError, BlockSignerHandler, RemoteSignerError};
use libp2p::TransportError;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, PayloadHash};
use std::time::Duration;
use thiserror::Error;
use tokio::{
    self, select,
    sync::mpsc::{self, UnboundedReceiver, UnboundedSender},
};

use crate::{
    NetworkEngineClient, NodeActor,
    actors::network::{
        driver::NetworkDriverError, error::NetworkBuilderError, handler::NetworkHandler,
    },
};

/// Bounds a single attempt to sign an unsafe payload.
///
/// Generous against real signer latency, so a slow signer is never cut short. It exists for a
/// connection that hangs rather than errors, which would otherwise wedge this actor until TCP
/// gives up.
const BLOCK_SIGNING_TIMEOUT: Duration = Duration::from_secs(10);

/// Why signing an unsafe payload failed.
///
/// `Timeout` and `Signer` are transient: drop the payload and carry on. `NotAuthorized` is not,
/// and is handled separately - see [`NetworkActor::sign_payload`].
#[derive(Debug, thiserror::Error)]
enum SigningFailure {
    /// The signer did not answer within [`BLOCK_SIGNING_TIMEOUT`].
    #[error("signing timed out after {BLOCK_SIGNING_TIMEOUT:?}")]
    Timeout,
    /// The signer refused or failed the request.
    #[error(transparent)]
    Signer(BlockSignerError),
    /// This node's signer is not the chain's current unsafe block signer.
    #[error(transparent)]
    NotAuthorized(BlockSignerError),
}

impl SigningFailure {
    /// Low-cardinality label for [`Metrics::BLOCK_SIGNING_ERRORS`](crate::Metrics).
    const fn kind(&self) -> &'static str {
        match self {
            Self::Timeout => crate::Metrics::SIGNING_ERROR_TIMEOUT,
            Self::Signer(_) => crate::Metrics::SIGNING_ERROR_SIGNER,
            Self::NotAuthorized(_) => crate::Metrics::SIGNING_ERROR_MISMATCH,
        }
    }
}

/// The network actor handles two core networking components of the rollup node:
/// - *discovery*: Peer discovery over UDP using discv5.
/// - *gossip*: Block gossip over TCP using libp2p.
#[derive(Debug)]
pub struct NetworkActor<NetworkEngineClient_: NetworkEngineClient> {
    /// The live libp2p [`NetworkHandler`].
    handler: NetworkHandler,
    /// A channel to receive the unsafe block signer address.
    unsafe_block_signer_rx: mpsc::Receiver<Address>,
    /// A channel to receive p2p RPC requests.
    p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
    /// A channel to receive admin RPC queries.
    admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
    /// A channel to receive unsafe blocks and send them through the gossip layer.
    publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// A client to use to interact with the engine actor.
    engine_client: NetworkEngineClient_,
    // Purely-internal channel: loops gossip-swarm events back into this actor's own select. It
    // never crosses an actor boundary, so it lives here rather than being injected.
    unsafe_block_tx: UnboundedSender<OpExecutionPayloadEnvelope>,
    unsafe_block_rx: UnboundedReceiver<OpExecutionPayloadEnvelope>,
}

impl<NetworkEngineClient_: NetworkEngineClient> NetworkActor<NetworkEngineClient_> {
    /// Constructs a new [`NetworkActor`].
    ///
    /// `handler` must already be live — i.e. the libp2p swarm it wraps must already have been
    /// built and started — before being passed in. Passing an unstarted handler will cause
    /// `step()` to hang or fail on its first poll of the gossip swarm. Keeping the constructor
    /// sync and treating the "is this live?" invariant as the caller's responsibility is the
    /// deliberate trade-off over an `init()`-style trait method.
    pub fn new(
        engine_client: NetworkEngineClient_,
        handler: NetworkHandler,
        unsafe_block_signer_rx: mpsc::Receiver<Address>,
        p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
        admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
        publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    ) -> Self {
        let (unsafe_block_tx, unsafe_block_rx) = mpsc::unbounded_channel();
        Self {
            handler,
            unsafe_block_signer_rx,
            p2p_rpc_rx,
            admin_query_rx,
            publish_rx,
            engine_client,
            unsafe_block_tx,
            unsafe_block_rx,
        }
    }

    /// Signs `payload_hash`, bounded by [`BLOCK_SIGNING_TIMEOUT`].
    ///
    /// `Ok(None)` means drop this payload and carry on: returning `Err` for a transient signing
    /// failure would be read by [`NodeActor`] as fatal, and the supervisor would tear down every
    /// other actor over a blip on the signing endpoint.
    ///
    /// `Err` is reserved for the one failure no retry fixes: the configured signer is not the
    /// chain's unsafe block signer, so this node cannot produce a block any peer will accept.
    /// Nothing else stops it sequencing - the sequencer canonicalises the block and commits it to
    /// the conductor before the payload ever reaches this actor, so the unsafe head keeps
    /// advancing and conductor health checks keep passing while nothing is ever published.
    async fn sign_payload(
        signer: &BlockSignerHandler,
        payload_hash: PayloadHash,
        chain_id: ChainId,
        sender_address: Address,
    ) -> Result<Option<Signature>, NetworkActorError> {
        let signed = tokio::time::timeout(
            BLOCK_SIGNING_TIMEOUT,
            signer.sign_block(payload_hash, chain_id, sender_address),
        )
        .await
        .map_err(|_| SigningFailure::Timeout)
        .and_then(|result| {
            result.map_err(|e| {
                if matches!(e, BlockSignerError::Remote(RemoteSignerError::InvalidAddress { .. })) {
                    SigningFailure::NotAuthorized(e)
                } else {
                    SigningFailure::Signer(e)
                }
            })
        });

        let failure = match signed {
            Ok(signature) => return Ok(Some(signature)),
            Err(failure) => failure,
        };
        // Read the label before the match moves `failure`. It is also logged, so log-based
        // alerting can match the metric's label rather than parsing the message.
        let kind = failure.kind();
        kona_macros::inc!(counter, crate::Metrics::BLOCK_SIGNING_ERRORS, "kind" => kind);

        match failure {
            SigningFailure::NotAuthorized(e) => {
                error!(
                    target: "network",
                    kind,
                    err = %e,
                    "This node is not the chain's unsafe block signer and cannot sequence"
                );
                Err(NetworkActorError::FailedToSignPayload(e))
            }
            failure => {
                warn!(
                    target: "network",
                    kind,
                    err = %failure,
                    "Failed to sign the unsafe payload, dropping it"
                );
                Ok(None)
            }
        }
    }
}

/// An error from the network actor.
#[derive(Debug, Error)]
pub enum NetworkActorError {
    /// Network builder error.
    #[error(transparent)]
    NetworkBuilder(#[from] NetworkBuilderError),
    /// Network driver error.
    #[error(transparent)]
    NetworkDriver(#[from] NetworkDriverError),
    /// Driver startup failed.
    #[error(transparent)]
    DriverStartup(#[from] TransportError<std::io::Error>),
    /// The network driver was missing its unsafe block receiver.
    #[error("Missing unsafe block receiver in network driver")]
    MissingUnsafeBlockReceiver,
    /// The network driver was missing its unsafe block signer sender.
    #[error("Missing unsafe block signer in network driver")]
    MissingUnsafeBlockSigner,
    /// Channel closed unexpectedly.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
    /// This node cannot sign unsafe payloads for this chain.
    ///
    /// Only raised when the configured signer is not the chain's unsafe block signer, which no
    /// retry can fix. Transient signing failures are logged against
    /// `kona_node_block_signing_errors` and the payload dropped, rather than surfacing here.
    #[error("Failed to sign the payload: {0}")]
    FailedToSignPayload(#[from] BlockSignerError),
}

#[async_trait]
impl<NetworkEngineClient_: NetworkEngineClient + 'static> NodeActor
    for NetworkActor<NetworkEngineClient_>
{
    type Error = NetworkActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
            block = self.unsafe_block_rx.recv() => {
                let Some(block) = block else {
                    error!(target: "node::p2p", "The unsafe block receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if self.engine_client.send_unsafe_block(block).await.is_err() {
                    warn!(target: "network", "Failed to forward unsafe block to engine");
                    return Err(NetworkActorError::ChannelClosed);
                }
                Ok(())
            }
            unsafe_block_signer = self.unsafe_block_signer_rx.recv() => {
                let Some(unsafe_block_signer) = unsafe_block_signer else {
                    warn!(
                        target: "network",
                        "Found no unsafe block signer on receive"
                    );
                    return Err(NetworkActorError::ChannelClosed);
                };
                if self.handler.unsafe_block_signer_sender.send(unsafe_block_signer).is_err() {
                    warn!(
                        target: "network",
                        "Failed to send unsafe block signer to network handler",
                    );
                }
                Ok(())
            }
            Some(block) = self.publish_rx.recv(), if !self.publish_rx.is_closed() => {
                let timestamp = block.timestamp();
                let selector = |handler: &kona_gossip::BlockHandler| {
                    handler.topic(timestamp)
                };
                let Some(signer) = self.handler.signer.as_ref() else {
                    warn!(target: "network", "No local signer available to sign the payload");
                    return Ok(());
                };

                let chain_id = self.handler.discovery.chain_id;

                let sender_address = *self.handler.unsafe_block_signer_sender.borrow();

                let payload_hash = block.payload_hash();

                let Some(signature) =
                    Self::sign_payload(signer, payload_hash, chain_id, sender_address).await?
                else {
                    return Ok(());
                };

                match self.handler.gossip.publish(selector, block, signature) {
                    Ok(id) => info!("Published unsafe payload | {:?}", id),
                    Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
                }
                Ok(())
            }
            event = self.handler.gossip.next() => {
                let Some(event) = event else {
                    error!(target: "node::p2p", "The gossip swarm stream has ended");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if let Some(payload) = self.handler.gossip.handle_event(event) &&
                    self.unsafe_block_tx.send(payload).is_err()
                {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            enr = self.handler.enr_receiver.recv() => {
                let Some(enr) = enr else {
                    error!(target: "node::p2p", "The enr receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };
                self.handler.gossip.dial(enr);
                Ok(())
            }
            _ = self.handler.peer_score_inspector.tick(), if self.handler.gossip.peer_monitoring.as_ref().is_some() => {
                self.handler.handle_peer_monitoring().await;
                Ok(())
            }
            Some(NetworkAdminQuery::PostUnsafePayload { payload }) = self.admin_query_rx.recv(), if !self.admin_query_rx.is_closed() => {
                debug!(target: "node::p2p", "Broadcasting unsafe payload from admin api");
                if self.unsafe_block_tx.send(payload).is_err() {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            Some(req) = self.p2p_rpc_rx.recv(), if !self.p2p_rpc_rx.is_closed() => {
                req.handle(&mut self.handler.gossip, &self.handler.discovery);
                Ok(())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::network::{
        engine_client::MockNetworkEngineClient,
        test_utils::{
            SignerBehaviour, spawn_mock_signer, test_actor, test_actor_with_unsafe_signer,
        },
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV3};
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use arbitrary::Arbitrary;
    use rand::Rng;

    /// Whether the driving runtime starts with a paused clock.
    #[derive(Debug, Clone, Copy)]
    enum Clock {
        Real,
        Paused,
    }

    fn arbitrary_payload() -> OpExecutionPayloadEnvelope {
        let mut bytes = [0u8; 4096];
        rand::rng().fill(bytes.as_mut_slice());
        OpExecutionPayloadEnvelope::V1(
            ExecutionPayloadV1::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
        )
    }

    fn runtime(clock: Clock) -> tokio::runtime::Runtime {
        tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .start_paused(matches!(clock, Clock::Paused))
            .build()
            .unwrap()
    }

    /// Schedules one payload against a signer behaving as `behaviour`, then runs a single step.
    async fn drive_one_payload(
        behaviour: SignerBehaviour,
    ) -> (Result<(), NetworkActorError>, Duration) {
        let signer = spawn_mock_signer(behaviour).await;
        let mut fixture = test_actor(&signer, MockNetworkEngineClient::new()).await;
        fixture.publish_tx.send(arbitrary_payload()).await.unwrap();

        let start = tokio::time::Instant::now();
        let result = fixture.actor.step().await;
        (result, start.elapsed())
    }

    fn step_once(
        clock: Clock,
        behaviour: SignerBehaviour,
    ) -> (Result<(), NetworkActorError>, Duration) {
        runtime(clock).block_on(drive_one_payload(behaviour))
    }

    /// `spawn_and_wait!` calls `step()` in a loop and treats `Err` as fatal: the task returns, its
    /// drop guard fires, and the supervisor cancels every other actor. So "a signer failure does
    /// not take the node down" and "this step returns `Ok`" are the same claim.
    ///
    /// That chain was established by reading the macro, not by exercising it here.
    #[test]
    fn a_signer_error_is_not_fatal() {
        let (result, _) = step_once(Clock::Real, SignerBehaviour::Error);
        assert!(result.is_ok(), "a signer error reported a fatal condition: {result:?}");
    }

    /// A rotated-away signer key is the one signing failure that must stay fatal.
    ///
    /// Nothing else stops a sequencer with the wrong signer key: it canonicalises and commits to
    /// the conductor before the payload reaches this actor, so its unsafe head advances and
    /// conductor health checks pass while it publishes nothing.
    #[test]
    fn a_signer_address_mismatch_is_fatal() {
        let result = runtime(Clock::Real).block_on(async {
            let signer = spawn_mock_signer(SignerBehaviour::Sign).await;
            let mut fixture = test_actor_with_unsafe_signer(
                &signer,
                MockNetworkEngineClient::new(),
                Address::repeat_byte(0xAA),
            )
            .await;
            fixture.publish_tx.send(arbitrary_payload()).await.unwrap();
            fixture.actor.step().await
        });

        assert!(
            matches!(result, Err(NetworkActorError::FailedToSignPayload(_))),
            "a signer address mismatch was not fatal: {result:?}"
        );
    }

    #[test]
    fn signing_is_bounded_by_the_deadline() {
        // Paused clock, so the deadline elapses in virtual time. `tokio::time::Instant` advances
        // with it, which is what makes the asserted duration meaningful rather than incidental.
        let (result, elapsed) = step_once(Clock::Paused, SignerBehaviour::Hang);
        assert!(result.is_ok(), "a hung signer reported a fatal condition: {result:?}");
        assert!(
            elapsed >= BLOCK_SIGNING_TIMEOUT,
            "gave up after {elapsed:?}, before the {BLOCK_SIGNING_TIMEOUT:?} deadline"
        );
    }

    #[cfg(feature = "metrics")]
    #[test]
    fn signing_failures_are_counted_by_kind() {
        use metrics_util::debugging::{DebugValue, DebuggingRecorder};

        // The `Sign` row is the negative control: without it, every assertion here would hold
        // just as well if signing had failed, since a failure also returns `Ok`.
        for (behaviour, clock, expected) in [
            (SignerBehaviour::Sign, Clock::Real, None),
            (SignerBehaviour::Error, Clock::Real, Some(crate::Metrics::SIGNING_ERROR_SIGNER)),
            (SignerBehaviour::Hang, Clock::Paused, Some(crate::Metrics::SIGNING_ERROR_TIMEOUT)),
        ] {
            let recorder = DebuggingRecorder::new();
            let snapshotter = recorder.snapshotter();
            let runtime = runtime(clock);
            let (result, _) = metrics::with_local_recorder(&recorder, || {
                runtime.block_on(drive_one_payload(behaviour))
            });
            assert!(result.is_ok(), "{behaviour:?} reported a fatal condition: {result:?}");

            // `snapshot()` drains, so take one and count from it. Calling it per kind would
            // make every lookup after the first read zero, and the test would pass regardless.
            let snapshot = snapshotter.snapshot().into_vec();
            let counted = |kind: &str| -> u64 {
                snapshot
                    .iter()
                    .filter(|(key, _, _, _)| {
                        let key = key.key();
                        key.name() == crate::Metrics::BLOCK_SIGNING_ERRORS &&
                            key.labels().any(|l| l.key() == "kind" && l.value() == kind)
                    })
                    .map(|(_, _, _, value)| match value {
                        DebugValue::Counter(count) => *count,
                        other => panic!("expected a counter, got {other:?}"),
                    })
                    .sum()
            };

            for kind in [
                crate::Metrics::SIGNING_ERROR_TIMEOUT,
                crate::Metrics::SIGNING_ERROR_SIGNER,
                crate::Metrics::SIGNING_ERROR_MISMATCH,
            ] {
                let want = u64::from(expected == Some(kind));
                assert_eq!(
                    counted(kind),
                    want,
                    "{behaviour:?} recorded the wrong count for {kind}"
                );
            }
        }
    }

    #[test]
    fn test_payload_signature_v1() {
        let pubkey = PrivateKeySigner::random();
        let expected_address = pubkey.address();
        const CHAIN_ID: u64 = 1337;

        let block = arbitrary_payload();

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
        const CHAIN_ID: u64 = 1337;

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
}
