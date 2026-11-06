//! Builds blocks on a launched node the way a consensus client does, through the engine API.

use alloy_primitives::{B256, Bytes};
use alloy_rpc_types_engine::{ForkchoiceState, PayloadId, PayloadStatus};
use alloy_signer_local::PrivateKeySigner;
use jsonrpsee::{core::client::ClientT, rpc_params};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV3;
use reth_e2e_test_utils::{transaction::TransactionTestContext, wallet::Wallet};
use reth_node_api::ConsensusEngineHandle;
use reth_optimism_node::{OpEngineTypes, utils::optimism_payload_attributes};
use std::time::{Duration, Instant};

/// Bounds every engine call and wait, so a stalled engine fails the test instead of hanging it.
const ENGINE_CALL_TIMEOUT: Duration = Duration::from_secs(20);

/// A payload job the engine has started but not resolved.
pub(crate) struct StartedPayloadJob {
    number: u64,
    pub(crate) payload_id: PayloadId,
    started_at: Instant,
}

/// Builds blocks the way a consensus client does, through the engine API.
pub(crate) struct EngineDriver<C> {
    pub(crate) client: C,
    pub(crate) engine: ConsensusEngineHandle<OpEngineTypes>,
    pub(crate) head: B256,
    pub(crate) wallet: Wallet,
}

impl<C: ClientT + Sync> EngineDriver<C> {
    /// A funded account the driver never signs with: the L1 info transactions it signs use up
    /// the wallet's nonces.
    pub(crate) fn spare_signer(&self) -> PrivateKeySigner {
        Wallet::new(2).with_chain_id(self.wallet.chain_id).wallet_gen().remove(1)
    }

    pub(crate) async fn build_block(&mut self, number: u64) -> eyre::Result<Duration> {
        self.build_block_with(number, Vec::new()).await
    }

    /// Builds a block that carries the given sequencer transactions after the block's L1 info
    /// transaction.
    pub(crate) async fn build_block_with(
        &mut self,
        number: u64,
        sequencer_txs: Vec<Bytes>,
    ) -> eyre::Result<Duration> {
        let job = self.start_payload_job_with(number, sequencer_txs).await?;
        self.finish_payload_job(job).await
    }

    /// Starts a payload job on the head, with the block's L1 info transaction as its only
    /// sequencer transaction.
    pub(crate) async fn start_payload_job(&self, number: u64) -> eyre::Result<StartedPayloadJob> {
        self.start_payload_job_with(number, Vec::new()).await
    }

    /// Starts a payload job on the head whose sequencer transactions are the block's L1 info
    /// transaction followed by the given ones.
    async fn start_payload_job_with(
        &self,
        number: u64,
        sequencer_txs: Vec<Bytes>,
    ) -> eyre::Result<StartedPayloadJob> {
        let mut attrs = optimism_payload_attributes(number);
        let mut transactions = vec![
            TransactionTestContext::optimism_l1_block_info_tx(
                self.wallet.chain_id,
                self.wallet.inner.clone(),
                number - 1,
            )
            .await,
        ];
        transactions.extend(sequencer_txs);
        attrs.0.transactions = Some(transactions);
        let started_at = Instant::now();
        let fcu = within_timeout(
            "forkchoice update with attributes stalled",
            self.engine.fork_choice_updated(ForkchoiceState::same_hash(self.head), Some(attrs)),
        )
        .await?;
        assert!(fcu.payload_status.is_valid(), "forkchoice update for block {number}: {fcu:?}");
        let payload_id = fcu.payload_id.expect("a payload job must start");
        Ok(StartedPayloadJob { number, payload_id, started_at })
    }

    /// Fetches the job's payload and makes it canonical. Returns how long the job took, from the
    /// forkchoice update that started it to the payload being returned.
    pub(crate) async fn finish_payload_job(
        &mut self,
        job: StartedPayloadJob,
    ) -> eyre::Result<Duration> {
        let payload: OpExecutionPayloadEnvelopeV3 = within_timeout(
            "getPayload stalled",
            self.client.request("engine_getPayloadV3", rpc_params![job.payload_id]),
        )
        .await?;
        let job_duration = job.started_at.elapsed();

        let block = &payload.execution_payload.payload_inner.payload_inner;
        assert_eq!(block.block_number, job.number);
        assert!(
            !block.transactions.is_empty(),
            "block {} lacks its L1 info transaction",
            job.number
        );
        self.head = block.block_hash;
        let status: PayloadStatus = within_timeout(
            "newPayload stalled",
            self.client.request(
                "engine_newPayloadV3",
                rpc_params![
                    payload.execution_payload,
                    Vec::<B256>::new(),
                    payload.parent_beacon_block_root
                ],
            ),
        )
        .await?;
        assert!(status.is_valid(), "newPayload for block {}: {status:?}", job.number);
        let fcu = within_timeout(
            "canonical forkchoice update stalled",
            self.engine.fork_choice_updated(ForkchoiceState::same_hash(self.head), None),
        )
        .await?;
        assert!(
            fcu.payload_status.is_valid(),
            "forkchoice update to block {}: {fcu:?}",
            job.number
        );
        Ok(job_duration)
    }
}

pub(crate) async fn within_timeout<T>(message: &str, future: impl Future<Output = T>) -> T {
    tokio::time::timeout(ENGINE_CALL_TIMEOUT, future).await.unwrap_or_else(|_| panic!("{message}"))
}

pub(crate) async fn wait_until(message: &str, mut ready: impl FnMut() -> bool) {
    within_timeout(message, async {
        while !ready() {
            tokio::time::sleep(Duration::from_millis(10)).await;
        }
    })
    .await;
}
