use super::SafeChainService;
use crate::engine::{EngineClient, EngineRequest};
use async_trait::async_trait;
use kona_derive::{
    OriginProvider, Pipeline, PipelineErrorKind, PipelineResult, Signal, SignalReceiver, StepResult,
};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use std::sync::{Arc, Mutex};
use tokio::sync::watch;
use tokio_util::sync::CancellationToken;

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

#[tokio::test]
async fn administrative_reset_resets_only_derivation_state() {
    let signals = Arc::new(Mutex::new(Vec::new()));
    let pipeline = RecordingPipeline { config: RollupConfig::default(), signals: signals.clone() };
    let (engine, mut engine_rx) = EngineClient::test_pair(4);
    let engine_task = tokio::spawn(async move {
        while let Some(request) = engine_rx.recv().await {
            match request {
                EngineRequest::State { response } => {
                    let _ = response.send(Ok(Default::default()));
                }
                EngineRequest::Recover { .. } => {
                    panic!("routine derivation reset must not recover or rewind the engine")
                }
                request => panic!("unexpected engine request: {request:?}"),
            }
        }
    });

    let (_head_tx, head_rx) = watch::channel(None);
    let (_finalized_tx, finalized_rx) = watch::channel(None);
    let (service, handle) =
        SafeChainService::new(Box::new(pipeline), engine, head_rx, finalized_rx, None);
    let shutdown = CancellationToken::new();
    let service_shutdown = shutdown.clone();
    let service_task = tokio::spawn(service.run(service_shutdown));

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

    handle.reset().await.expect("administrative reset");
    assert_eq!(*signals.lock().unwrap(), ["reset", "reset"]);

    shutdown.cancel();
    service_task.await.unwrap().unwrap();
    drop(handle);
    engine_task.await.unwrap();
}
