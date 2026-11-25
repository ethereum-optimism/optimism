//! The blocks a cross-chain verifier denied, as the engine consults them.

use alloy_eips::BlockNumHash;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch;

/// The engine's view of the blocks a cross-chain verifier denied.
///
/// A denied block has an invalid executing message. Derivation never imports it: where derived
/// attributes produce it, the deposits-only block replaces it. The view is the verifier's last
/// answer since the node started, or unknown before the first, so the node keeps no record of its
/// own. While it is unknown, no derived block can be imported, since it might be a denied one.
///
/// A verifier only adds denials, so an answer that has since gone stale lacks only denials of
/// blocks this node has not replaced yet. It never lets derivation bring back a block the node
/// replaced: that replacement was made from an answer containing the denial.
#[derive(Debug, Clone)]
pub struct DeniedBlocks(watch::Receiver<Option<Arc<[BlockNumHash]>>>);

/// Publishes a cross-chain verifier's answers to a [`DeniedBlocks`] view.
#[derive(Debug)]
pub struct DeniedBlocksSender(watch::Sender<Option<Arc<[BlockNumHash]>>>);

/// The verifier's denied blocks are not known yet.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("the cross-chain verifier's denied blocks are not known yet")]
pub struct DeniedBlocksUnknown;

impl DeniedBlocks {
    /// The view of a node that follows no verifier: nothing is ever denied.
    pub fn none() -> Self {
        Self(watch::channel(Some(Arc::from([]))).1)
    }

    /// A view that is unknown until the returned sender publishes the verifier's first answer.
    pub fn from_verifier() -> (DeniedBlocksSender, Self) {
        let (sender, receiver) = watch::channel(None);
        (DeniedBlocksSender(sender), Self(receiver))
    }

    /// Whether `block` is denied.
    pub fn contains(&self, block: BlockNumHash) -> Result<bool, DeniedBlocksUnknown> {
        let denied = self.0.borrow();
        let denied = denied.as_ref().ok_or(DeniedBlocksUnknown)?;
        Ok(denied.contains(&block))
    }

    /// Fails unless the verifier's denied blocks are known.
    pub fn ensure_known(&self) -> Result<(), DeniedBlocksUnknown> {
        self.0.borrow().as_ref().map(|_| ()).ok_or(DeniedBlocksUnknown)
    }
}

impl DeniedBlocksSender {
    /// Publishes the verifier's latest answer: every block it denied above the finalized head.
    pub fn publish(&self, denied: Vec<BlockNumHash>) {
        self.0.send_replace(Some(denied.into()));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    fn block(number: u64, byte: u8) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(byte) }
    }

    #[test]
    fn a_node_without_a_verifier_denies_nothing() {
        let denied = DeniedBlocks::none();

        assert_eq!(denied.contains(block(1, 1)), Ok(false));
        assert_eq!(denied.ensure_known(), Ok(()));
    }

    /// Until the first answer nothing can be decided; afterwards the latest answer holds.
    #[test]
    fn a_verifier_view_is_unknown_until_the_first_answer() {
        let (sender, denied) = DeniedBlocks::from_verifier();
        assert_eq!(denied.contains(block(1, 1)), Err(DeniedBlocksUnknown));
        assert_eq!(denied.ensure_known(), Err(DeniedBlocksUnknown));

        sender.publish(vec![block(3, 3)]);
        assert_eq!(denied.contains(block(3, 3)), Ok(true));
        assert_eq!(denied.contains(block(3, 4)), Ok(false));

        drop(sender);
        assert_eq!(denied.contains(block(3, 3)), Ok(true), "the last answer outlives the sender");
    }
}
