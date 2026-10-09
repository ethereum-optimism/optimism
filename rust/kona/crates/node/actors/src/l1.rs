//! L1 block polling and publication.

use alloy_eips::BlockNumberOrTag;
use alloy_provider::Provider;
use futures::StreamExt;
use kona_protocol::BlockInfo;
use std::{future::Future, time::Duration};
use thiserror::Error;
use tokio::sync::watch;

mod blockstream;
pub use blockstream::BlockStream;

/// The L1 block tag to observe.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Tag {
    /// The latest L1 head.
    Head,
    /// The safe L1 head.
    Safe,
    /// The finalized L1 head.
    Finalized,
}

impl From<Tag> for BlockNumberOrTag {
    fn from(tag: Tag) -> Self {
        match tag {
            Tag::Head => Self::Latest,
            Tag::Safe => Self::Safe,
            Tag::Finalized => Self::Finalized,
        }
    }
}

/// Constructs the state publisher and task.
#[derive(Debug)]
pub struct Builder {
    published: watch::Sender<BlockInfo>,
}

impl Builder {
    /// Creates the builder with its initial block.
    pub fn new(initial: BlockInfo) -> Self {
        let (published, _) = watch::channel(initial);
        Self { published }
    }

    /// Subscribes to the latest block before the actor starts.
    pub fn handle(&self) -> watch::Receiver<BlockInfo> {
        self.published.subscribe()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Runtime work begins when the future is polled.
    /// Dropping the future stops the actor and closes its publisher.
    pub fn build<L1Provider: Provider + 'static>(
        self,
        provider: L1Provider,
        tag: Tag,
        interval: Duration,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        Actor { provider, tag, interval, published: self.published }.run()
    }
}

/// A fatal error from the L1 actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// The block stream ended.
    #[error("block stream ended")]
    StreamEnded,
}

#[derive(Debug)]
struct Actor<L1Provider> {
    provider: L1Provider,
    tag: Tag,
    interval: Duration,
    published: watch::Sender<BlockInfo>,
}

impl<L1Provider: Provider> Actor<L1Provider> {
    async fn run(self) -> Result<(), ActorError> {
        let mut blocks =
            BlockStream::new(&self.provider, self.tag.into(), self.interval).into_stream();
        while let Some(block) = blocks.next().await {
            // Publish reorgs as well as advances, and suppress unchanged observations.
            self.published.send_if_modified(|current| std::mem::replace(current, block) != block);
        }
        Err(ActorError::StreamEnded)
    }
}

#[cfg(test)]
mod tests;
