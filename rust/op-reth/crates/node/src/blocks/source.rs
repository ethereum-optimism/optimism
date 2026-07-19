use super::{BlocksServerMetrics, StreamTermination};
use alloy_primitives::B256;
use async_stream::try_stream;
use futures_util::{FutureExt, Stream, StreamExt, stream::BoxStream};
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayloadEnvelope};
use reth_primitives_traits::{Block as _, NodePrimitives};
use reth_provider::{BlockReader, CanonStateNotification};
use std::{
    pin::Pin,
    task::{Context, Poll},
    time::Instant,
};

/// A stream of the canonical blocks a connection should send next.
///
/// Reorgs rewind the internal cursor, so the stream can yield a replacement block number that it
/// previously yielded. All block data is read from the canonical provider.
pub(super) struct CanonicalBlockStream {
    inner: BoxStream<'static, Result<OpExecutionPayloadEnvelope, StreamTermination>>,
}

impl CanonicalBlockStream {
    pub(super) fn new<P, S, N>(
        provider: P,
        notifications: S,
        start: u64,
        metrics: BlocksServerMetrics,
    ) -> Self
    where
        P: BlockReader + Send + Sync + 'static,
        S: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
        N: NodePrimitives,
    {
        Self::from_updates(provider, notifications.map(CanonicalUpdate::from), start, metrics)
    }

    fn from_updates<P, S>(
        provider: P,
        mut notifications: S,
        start: u64,
        metrics: BlocksServerMetrics,
    ) -> Self
    where
        P: BlockReader + Send + Sync + 'static,
        S: Stream<Item = CanonicalUpdate> + Send + Unpin + 'static,
    {
        let inner = try_stream! {
            let mut cursor = CanonicalCursor::new(start);
            let mut notifications_closed = false;

            loop {
                drain_ready_updates(
                    &mut notifications,
                    &mut cursor,
                    &mut notifications_closed,
                );

                match cursor.read_next(&provider, &metrics) {
                    Ok(Some(block)) => yield block,
                    Ok(None) => {
                        if notifications_closed {
                            Err(StreamTermination::new(
                                "notifications_closed",
                                "canonical state notification stream closed",
                            ))?;
                        }
                        let update = notifications.next().await.ok_or_else(|| {
                            StreamTermination::new(
                                "notifications_closed",
                                "canonical state notification stream closed",
                            )
                        })?;
                        cursor.apply_update(update);
                    }
                    Err(ReadNextError::ParentMismatch { block_number, actual, expected }) => {
                        if drain_ready_updates(
                            &mut notifications,
                            &mut cursor,
                            &mut notifications_closed,
                        ) {
                            continue
                        }
                        Err(StreamTermination::new(
                            "parent_mismatch",
                            format!(
                                "block {block_number} parent {actual} does not match previously sent block {expected}"
                            ),
                        ))?;
                    }
                    Err(ReadNextError::Termination(error)) => Err(error)?,
                }
            }
        };

        Self { inner: Box::pin(inner) }
    }

    #[cfg(test)]
    pub(super) fn for_test<P, S>(
        provider: P,
        notifications: S,
        start: u64,
        metrics: BlocksServerMetrics,
    ) -> Self
    where
        P: BlockReader + Send + Sync + 'static,
        S: Stream<Item = CanonicalUpdate> + Send + Unpin + 'static,
    {
        Self::from_updates(provider, notifications, start, metrics)
    }
}

impl Stream for CanonicalBlockStream {
    type Item = Result<OpExecutionPayloadEnvelope, StreamTermination>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        self.inner.as_mut().poll_next(cx)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum CanonicalUpdate {
    Wake,
    Reorg { fork_number: u64, fork_hash: B256 },
}

impl<N: NodePrimitives> From<CanonStateNotification<N>> for CanonicalUpdate {
    fn from(notification: CanonStateNotification<N>) -> Self {
        match notification {
            CanonStateNotification::Commit { .. } => Self::Wake,
            CanonStateNotification::Reorg { old, .. } => {
                let fork = old.fork_block();
                Self::Reorg { fork_number: fork.number, fork_hash: fork.hash }
            }
        }
    }
}

struct CanonicalCursor {
    next: u64,
    previous_hash: Option<B256>,
}

impl CanonicalCursor {
    const fn new(next: u64) -> Self {
        Self { next, previous_hash: None }
    }

    fn apply_update(&mut self, update: CanonicalUpdate) -> bool {
        let CanonicalUpdate::Reorg { fork_number, fork_hash } = update else { return false };
        let replacement_start = fork_number.saturating_add(1);
        if replacement_start >= self.next {
            return false;
        }

        tracing::debug!(target: "reth::blocks", fork_number, %fork_hash, "Rewinding blocks stream after canonical reorg");
        self.next = replacement_start;
        self.previous_hash = Some(fork_hash);
        true
    }

    fn read_next<P>(
        &mut self,
        provider: &P,
        metrics: &BlocksServerMetrics,
    ) -> Result<Option<OpExecutionPayloadEnvelope>, ReadNextError>
    where
        P: BlockReader,
    {
        let head = provider.best_block_number().map_err(|error| {
            StreamTermination::new("provider", format!("failed to read canonical head: {error}"))
        })?;
        if self.next > head {
            metrics.replay_distance.set(0.0);
            return Ok(None);
        }
        metrics.replay_distance.set(head.saturating_sub(self.next).saturating_add(1) as f64);

        let block_number = self.next;
        let read_started = Instant::now();
        let block = provider.block_by_number(block_number).map_err(|error| {
            StreamTermination::new(
                "provider",
                format!("failed to read canonical block {block_number}: {error}"),
            )
        })?;
        metrics.database_read_latency.record(read_started.elapsed().as_secs_f64());
        let Some(block) = block else {
            metrics.missing_block_errors.increment(1);
            return Err(StreamTermination::new(
                "missing_block",
                format!("canonical block {block_number} is unavailable"),
            )
            .into());
        };

        let execution_data = OpExecutionData::from_block_slow(&block.into_ethereum_block());
        if execution_data.block_number() != block_number {
            return Err(StreamTermination::new(
                "invalid_block_number",
                format!(
                    "requested canonical block {block_number}, provider returned {}",
                    execution_data.block_number()
                ),
            )
            .into());
        }
        if let Some(expected) = self.previous_hash &&
            execution_data.parent_hash() != expected
        {
            return Err(ReadNextError::ParentMismatch {
                block_number,
                actual: execution_data.parent_hash(),
                expected,
            });
        }

        self.previous_hash = Some(execution_data.block_hash());
        self.next = self.next.checked_add(1).ok_or_else(|| {
            StreamTermination::new("offset_overflow", "stream reached block number u64::MAX")
        })?;
        Ok(Some(OpExecutionPayloadEnvelope {
            parent_beacon_block_root: execution_data.parent_beacon_block_root(),
            execution_payload: execution_data.payload,
        }))
    }
}

enum ReadNextError {
    Termination(StreamTermination),
    ParentMismatch { block_number: u64, actual: B256, expected: B256 },
}

impl From<StreamTermination> for ReadNextError {
    fn from(error: StreamTermination) -> Self {
        Self::Termination(error)
    }
}

fn drain_ready_updates<S>(
    notifications: &mut S,
    cursor: &mut CanonicalCursor,
    notifications_closed: &mut bool,
) -> bool
where
    S: Stream<Item = CanonicalUpdate> + Unpin,
{
    let mut rewound = false;
    loop {
        match notifications.next().now_or_never() {
            Some(Some(update)) => rewound |= cursor.apply_update(update),
            Some(None) => {
                *notifications_closed = true;
                return rewound;
            }
            None => return rewound,
        }
    }
}
