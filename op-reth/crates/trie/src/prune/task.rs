use crate::{prune::OpProofStoragePruner, OpProofsStorage, OpProofsStore};
use reth_provider::BlockHashReader;
use reth_tasks::shutdown::GracefulShutdown;
use tokio::{
    time,
    time::{Duration, MissedTickBehavior},
};
use tracing::info;

const PRUNE_BATCH_SIZE: u64 = 200;

/// Periodic pruner task: constructs the pruner and runs it every interval.
#[derive(Debug)]
pub struct OpProofStoragePrunerTask<P, H> {
    pruner: OpProofStoragePruner<P, H>,
    min_block_interval: u64,
    task_run_interval: Duration,
}

impl<P, H> OpProofStoragePrunerTask<P, H>
where
    P: OpProofsStore,
    H: BlockHashReader,
{
    /// Initialize a new [`OpProofStoragePrunerTask`]
    pub fn new(
        provider: OpProofsStorage<P>,
        hash_reader: H,
        min_block_interval: u64,
        task_run_interval: Duration,
    ) -> Self {
        let pruner =
            OpProofStoragePruner::new(provider, hash_reader, min_block_interval, PRUNE_BATCH_SIZE);
        Self { pruner, min_block_interval, task_run_interval }
    }

    /// Run forever (until `cancel`), executing one prune pass per `task_run_interval`.
    pub async fn run(self, mut signal: GracefulShutdown) {
        info!(
            target: "trie::pruner_task",
            min_block_interval = self.min_block_interval,
            interval_secs = self.task_run_interval.as_secs(),
            "Starting pruner task"
        );

        // Drive pruning with a periodic ticker
        let mut interval = time::interval(self.task_run_interval);
        interval.set_missed_tick_behavior(MissedTickBehavior::Delay);

        loop {
            tokio::select! {
                _ = &mut signal => {
                    info!(target: "trie::pruner_task", "Pruner task cancelled; exiting");
                    break;
                }
                _ = interval.tick() => {
                    self.pruner.run().await
                }
            }
        }

        info!(target: "trie::pruner_task", "Pruner task stopped");
    }
}
