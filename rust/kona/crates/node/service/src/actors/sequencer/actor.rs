//! The [`SequencerActor`] control plane.

use crate::{
    Conductor, NodeActor, SequencerAdminQuery, SequencerEngineClient, UnsafePayloadGossipClient,
    actors::sequencer::{
        error::SequencerActorError,
        origin_selector::OriginSelector,
        worker::{SequencerControl, SequencerWorker, SequencerWorkerStatus},
    },
};
use async_trait::async_trait;
use kona_derive::AttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::{
    select,
    sync::{mpsc, watch},
    task::{JoinError, JoinHandle},
};

/// The [`SequencerActor`] is the sequencing control plane.
///
/// Admin requests are handled here while a supervised `SequencerWorker` owns the linear block
/// lifecycle. Keeping the workflow in its own async task lets Rust retain typed build, sealed, and
/// committed values across `.await` points without manually encoding those phases in the actor.
#[derive(Debug)]
pub struct SequencerActor<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Receiver for admin API requests.
    pub admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
    /// Optional conductor client shared with admin operations.
    pub conductor: Option<Arc<Conductor_>>,
    /// Engine client shared with admin operations.
    pub engine_client: Arc<SequencerEngineClient_>,
    /// Whether sequencing is desired.
    pub is_active: bool,
    /// Whether recovery-mode blocks are desired.
    pub in_recovery_mode: bool,

    pub(super) control_tx: watch::Sender<SequencerControl>,
    pub(super) status_rx: watch::Receiver<SequencerWorkerStatus>,
    pub(super) worker: Option<
        SequencerWorker<
            AttributesBuilder_,
            Conductor_,
            OriginSelector_,
            SequencerEngineClient_,
            UnsafePayloadGossipClient_,
        >,
    >,
    worker_handle: Option<JoinHandle<Result<(), SequencerActorError>>>,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencerActor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Instantiates a sequencer control plane and its supervised worker.
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        is_active: bool,
        in_recovery_mode: bool,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        let conductor = conductor.map(Arc::new);
        let engine_client = Arc::new(engine_client);
        let (control_tx, control_rx) =
            watch::channel(SequencerControl { active: is_active, recovery_mode: in_recovery_mode });
        let (status_tx, status_rx) = watch::channel(SequencerWorkerStatus::Starting);

        let worker = SequencerWorker::new(
            attributes_builder,
            conductor.clone(),
            control_rx,
            engine_client.clone(),
            origin_selector,
            rollup_config,
            status_tx,
            unsafe_payload_gossip_client,
        );

        Self {
            admin_api_rx,
            conductor,
            engine_client,
            is_active,
            in_recovery_mode,
            control_tx,
            status_rx,
            worker: Some(worker),
            worker_handle: None,
        }
    }

    /// Updates the desired worker state.
    pub(super) fn update_control(&self) {
        self.control_tx.send_replace(SequencerControl {
            active: self.is_active,
            recovery_mode: self.in_recovery_mode,
        });
    }

    /// Returns whether the sequencing worker has been started.
    pub(super) const fn worker_started(&self) -> bool {
        self.worker_handle.is_some()
    }

    /// Starts the worker exactly once.
    fn start_worker(&mut self)
    where
        AttributesBuilder_: Send + Sync + 'static,
        Conductor_: Sync + 'static,
        OriginSelector_: Send + Sync + 'static,
        SequencerEngineClient_: Sync + 'static,
        UnsafePayloadGossipClient_: Send + Sync + 'static,
    {
        if self.worker_handle.is_some() {
            return;
        }

        let worker = self.worker.take().expect("sequencer worker can only be started once");
        self.worker_handle = Some(tokio::spawn(worker.run()));
    }

    fn worker_result(
        result: Result<Result<(), SequencerActorError>, JoinError>,
    ) -> Result<(), SequencerActorError> {
        match result {
            Ok(Ok(())) => Err(SequencerActorError::WorkerExited),
            Ok(Err(err)) => Err(err),
            Err(err) => Err(SequencerActorError::WorkerJoin(err.to_string())),
        }
    }
}

#[async_trait]
impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> NodeActor
    for SequencerActor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder + Send + Sync + 'static,
    Conductor_: Conductor + Sync + 'static,
    OriginSelector_: OriginSelector + Send + Sync + 'static,
    SequencerEngineClient_: SequencerEngineClient + Sync + 'static,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient + Send + Sync + 'static,
{
    type Error = SequencerActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        self.start_worker();

        enum StepEvent {
            Admin(Option<SequencerAdminQuery>),
            Worker(Result<Result<(), SequencerActorError>, JoinError>),
        }

        let event = {
            let worker = self.worker_handle.as_mut().expect("worker started above");
            select! {
                biased;
                query = self.admin_api_rx.recv() => StepEvent::Admin(query),
                result = worker => StepEvent::Worker(result),
            }
        };

        match event {
            StepEvent::Admin(Some(query)) => {
                self.handle_admin_query(query).await;
                Ok(())
            }
            StepEvent::Admin(None) => {
                // With no admin producer left, avoid a closed-channel hot loop and supervise only
                // the worker until service cancellation drops this actor.
                let result = self.worker_handle.as_mut().expect("worker started above").await;
                Self::worker_result(result)
            }
            StepEvent::Worker(result) => Self::worker_result(result),
        }
    }
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> Drop
    for SequencerActor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    fn drop(&mut self) {
        if let Some(handle) = &self.worker_handle {
            handle.abort();
        }
    }
}
