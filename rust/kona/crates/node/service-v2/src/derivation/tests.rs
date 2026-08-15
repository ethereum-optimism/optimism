use super::DerivationService;
use crate::{
    engine::{EngineHandle, EngineRequest},
    l1::{L1Reader, L1Snapshot},
};
use alloy_provider::RootProvider;
use async_trait::async_trait;
use kona_derive::{
    OriginProvider, Pipeline, PipelineErrorKind, PipelineResult, Signal, SignalReceiver, StepResult,
};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_providers_alloy::{AlloyChainProvider, OnlineBeaconClient, OnlineBlobProvider};
use std::sync::{Arc, Mutex};
use tokio::sync::{oneshot, watch};
use url::Url;

#[derive(Debug)]
struct RecordingPipeline {
    config: RollupConfig,
    signals: Arc<Mutex<Vec<&'static str>>>,
}

impl Iterator for RecordingPipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        None
    }
}

impl OriginProvider for RecordingPipeline {
    fn origin(&self) -> Option<BlockInfo> {
        None
    }
}

#[async_trait]
impl SignalReceiver for RecordingPipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        let name = match signal {
            Signal::Reset(_) => "reset",
            Signal::Activation(_) => "activation",
            Signal::FlushChannel => "flush",
            Signal::ProvideBlock(_) => "provide_block",
        };
        self.signals.lock().unwrap().push(name);
        Ok(())
    }
}

#[async_trait]
impl Pipeline for RecordingPipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        None
    }

    async fn step(&mut self, _cursor: L2BlockInfo) -> StepResult {
        panic!("pipeline must not step without an L1 head update")
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.config
    }

    async fn system_config_by_number(
        &mut self,
        _number: u64,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        Ok(SystemConfig::default())
    }
}

fn l1_reader() -> L1Reader {
    let url = Url::parse("http://127.0.0.1:1").unwrap();
    let provider = RootProvider::new_http(url.clone());
    let chain = AlloyChainProvider::new(provider.clone(), 1);
    let blobs = OnlineBlobProvider {
        beacon_client: OnlineBeaconClient::new_http(url.to_string()),
        genesis_time: 0,
        slot_interval: 12,
    };
    L1Reader::new(provider, chain, blobs)
}

#[tokio::test]
async fn administrative_reset_resets_only_derivation_state() {
    let signals = Arc::new(Mutex::new(Vec::new()));
    let pipeline = RecordingPipeline { config: RollupConfig::default(), signals: signals.clone() };
    let (engine, mut engine_rx) = EngineHandle::test_pair(4);
    let engine_task = tokio::spawn(async move {
        if let Some(request) = engine_rx.recv().await {
            match request {
                EngineRequest::UpdateSafe { .. } | EngineRequest::UpdateFinalized { .. } => {
                    panic!("routine derivation reset must not update Engine")
                }
                request => panic!("EngineHandle exposed an unexpected request: {request:?}"),
            }
        }
    });

    let (_snapshots_tx, snapshots) = watch::channel(L1Snapshot::default());
    let (service, admin) = DerivationService::new(
        Box::new(pipeline),
        engine,
        L2BlockInfo::default(),
        l1_reader(),
        snapshots,
    );
    let (shutdown, shutdown_rx) = oneshot::channel();
    let service_task = tokio::spawn(service.run(shutdown_rx));

    tokio::time::timeout(std::time::Duration::from_secs(1), async {
        loop {
            if signals.lock().unwrap().as_slice() == ["reset"] {
                break;
            }
            tokio::task::yield_now().await;
        }
    })
    .await
    .expect("startup derivation reset");

    admin.reset().await.expect("administrative reset");
    assert_eq!(*signals.lock().unwrap(), ["reset", "reset"]);

    let _ = shutdown.send(());
    service_task.await.unwrap().unwrap();
    drop(admin);
    engine_task.await.unwrap();
}
