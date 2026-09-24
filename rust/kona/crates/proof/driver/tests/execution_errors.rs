//! Regression tests for invalid payloads and execution failures in the derivation driver.

use alloy_consensus::{Header, Sealed};
use alloy_evm::block::{BlockExecutionError, BlockValidationError};
use alloy_primitives::{B256, Bytes};
use async_trait::async_trait;
use kona_derive::{
    OriginProvider, Pipeline, PipelineError, PipelineErrorKind, PipelineResult, Signal,
    SignalReceiver, StepResult,
};
use kona_driver::{Driver, DriverError, DriverPipeline, Executor, PipelineCursor, TipCursor};
use kona_executor::{BlockBuildingOutcome, ExecutorError, TrieDBError};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use spin::RwLock;
use std::sync::Arc;

#[derive(Debug)]
struct TestPipeline {
    config: RollupConfig,
    attributes: Option<OpAttributesWithParent>,
    flushes: usize,
}

impl Iterator for TestPipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        None
    }
}

impl OriginProvider for TestPipeline {
    fn origin(&self) -> Option<BlockInfo> {
        Some(BlockInfo::default())
    }
}

#[async_trait]
impl SignalReceiver for TestPipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        if matches!(signal, Signal::FlushChannel) {
            self.flushes += 1;
        }
        Ok(())
    }
}

#[async_trait]
impl Pipeline for TestPipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        None
    }

    async fn step(&mut self, _: L2BlockInfo) -> StepResult {
        unreachable!()
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.config
    }

    async fn system_config_by_l2_hash(
        &mut self,
        _: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        unreachable!()
    }
}

#[async_trait]
impl DriverPipeline<Self> for TestPipeline {
    fn flush(&mut self) {}

    async fn produce_payload(
        &mut self,
        _: L2BlockInfo,
    ) -> Result<OpAttributesWithParent, PipelineErrorKind> {
        self.attributes.take().ok_or_else(|| PipelineError::EndOfSource.crit())
    }
}

#[derive(Debug)]
struct TestExecutor {
    errors: Vec<ExecutorError>,
    calls: Vec<Vec<Bytes>>,
}

#[async_trait]
impl Executor for TestExecutor {
    type Error = ExecutorError;
    type Receipt = ();

    async fn wait_until_ready(&mut self) {}

    fn update_safe_head(&mut self, _: Sealed<Header>) {}

    async fn execute_payload(
        &mut self,
        attributes: OpPayloadAttributes,
    ) -> Result<BlockBuildingOutcome<Self::Receipt>, Self::Error> {
        self.calls.push(attributes.transactions.unwrap_or_default());
        Err(self.errors.remove(0))
    }

    fn is_invalid_payload_error(error: &Self::Error) -> bool {
        error.is_invalid_payload()
    }

    fn compute_output_root(&mut self) -> Result<B256, Self::Error> {
        unreachable!()
    }
}

fn driver_with_error(
    first_error: ExecutorError,
) -> (Driver<TestExecutor, TestPipeline, TestPipeline>, RollupConfig) {
    let mut config = RollupConfig::default();
    config.hardforks.holocene_time = Some(0);

    let attributes = OpPayloadAttributes {
        transactions: Some(vec![Bytes::from_static(&[0x7e]), Bytes::from_static(&[0x02])]),
        ..Default::default()
    };
    let pipeline = TestPipeline {
        config: config.clone(),
        attributes: Some(OpAttributesWithParent::new(
            attributes,
            L2BlockInfo::default(),
            Some(BlockInfo::default()),
            true,
        )),
        flushes: 0,
    };
    let executor = TestExecutor {
        errors: vec![first_error, ExecutorError::MissingExecutor],
        calls: Vec::new(),
    };
    let mut cursor = PipelineCursor::new(1, BlockInfo::default());
    cursor.advance(
        BlockInfo::default(),
        TipCursor::new(
            L2BlockInfo::default(),
            Sealed::new_unchecked(Header::default(), B256::ZERO),
            B256::ZERO,
        ),
    );
    (Driver::new(Arc::new(RwLock::new(cursor)), executor, pipeline), config)
}

#[tokio::test]
async fn invalid_payload_triggers_deposit_only_fallback() {
    let (mut driver, config) = driver_with_error(ExecutorError::BlockGasLimitExceeded);
    let result = driver.advance_to_target(&config, Some(1)).await;

    assert!(matches!(result, Err(DriverError::Executor(ExecutorError::MissingExecutor))));
    assert_eq!(driver.pipeline.flushes, 1);
    assert_eq!(driver.executor.calls.len(), 2);
    assert_eq!(driver.executor.calls[0].len(), 2);
    assert_eq!(driver.executor.calls[1], vec![Bytes::from_static(&[0x7e])]);
}

#[tokio::test]
async fn invalid_evm_validation_triggers_deposit_only_fallback() {
    let error = ExecutorError::ExecutionError(BlockExecutionError::Validation(
        BlockValidationError::TransactionGasLimitMoreThanAvailableBlockGas {
            transaction_gas_limit: 2,
            block_available_gas: 1,
        },
    ));
    let (mut driver, config) = driver_with_error(error);
    let result = driver.advance_to_target(&config, Some(1)).await;

    assert!(matches!(result, Err(DriverError::Executor(ExecutorError::MissingExecutor))));
    assert_eq!(driver.pipeline.flushes, 1);
    assert_eq!(driver.executor.calls[1], vec![Bytes::from_static(&[0x7e])]);
}

#[tokio::test]
async fn provider_error_must_not_invalidate_channel() {
    let error = ExecutorError::TrieDBError(TrieDBError::Provider("missing witness".into()));
    let (mut driver, config) = driver_with_error(error);
    let result = driver.advance_to_target(&config, Some(1)).await;

    assert!(matches!(result, Err(DriverError::Executor(ExecutorError::TrieDBError(_)))));
    assert_eq!(driver.pipeline.flushes, 0);
    assert_eq!(driver.executor.calls.len(), 1);
}

#[tokio::test]
async fn pre_holocene_provider_error_must_not_be_discarded() {
    let error = ExecutorError::TrieDBError(TrieDBError::Provider("missing witness".into()));
    let (mut driver, mut config) = driver_with_error(error);
    config.hardforks.holocene_time = None;
    driver.pipeline.config.hardforks.holocene_time = None;
    let result = driver.advance_to_target(&config, Some(1)).await;

    assert!(matches!(result, Err(DriverError::Executor(ExecutorError::TrieDBError(_)))));
    assert_eq!(driver.pipeline.flushes, 0);
    assert_eq!(driver.executor.calls.len(), 1);
}

#[tokio::test]
async fn validation_error_from_database_must_not_invalidate_channel() {
    let error = ExecutorError::ExecutionError(BlockExecutionError::Validation(
        BlockValidationError::IncrementBalanceFailed,
    ));
    let (mut driver, config) = driver_with_error(error);
    let result = driver.advance_to_target(&config, Some(1)).await;

    assert!(matches!(result, Err(DriverError::Executor(ExecutorError::ExecutionError(_)))));
    assert_eq!(driver.pipeline.flushes, 0);
    assert_eq!(driver.executor.calls.len(), 1);
}
