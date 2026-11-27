//! The interop authority epoch a node derives under.

use alloy_primitives::B256;
use async_trait::async_trait;
use jsonrpsee::core::RpcResult;
use kona_rpc::InteropNodeApiServer;
use tokio::sync::watch;

/// The interop authority epoch a node took on by deriving again from its finalized head.
///
/// It is held in memory only. A start derives again from the finalized head unless finality
/// stalled for longer than the sequencing window (see [`kona_engine::find_starting_forkchoice`]),
/// so the first answer after it takes the authority's epoch on with an undo that usually moves
/// nothing, see [`kona_engine::UndoTask`].
#[derive(Debug, Default)]
pub struct AppliedEpoch(watch::Sender<Option<B256>>);

impl AppliedEpoch {
    /// The epoch the node derives under, if it took one on since it started.
    pub fn get(&self) -> Option<B256> {
        *self.0.borrow()
    }

    /// A view of the epoch the node derives under.
    pub fn watch(&self) -> watch::Receiver<Option<B256>> {
        self.0.subscribe()
    }

    /// Records that the node derives under `epoch`.
    pub fn apply(&self, epoch: B256) {
        self.0.send_replace(Some(epoch));
    }
}

/// Serves the authority the epoch the node derives under, so it reads the node only once the
/// node took its epoch on.
#[derive(Debug)]
pub struct AppliedEpochRpc(pub watch::Receiver<Option<B256>>);

#[async_trait]
impl InteropNodeApiServer for AppliedEpochRpc {
    async fn applied_epoch(&self) -> RpcResult<Option<B256>> {
        Ok(*self.0.borrow())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn an_applied_epoch_is_seen_by_its_views() {
        let applied = AppliedEpoch::default();
        let view = applied.watch();
        assert_eq!(applied.get(), None);

        applied.apply(B256::repeat_byte(1));

        assert_eq!(applied.get(), Some(B256::repeat_byte(1)));
        assert_eq!(*view.borrow(), Some(B256::repeat_byte(1)));
    }
}
