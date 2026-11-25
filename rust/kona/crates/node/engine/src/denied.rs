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
/// A verifier keeps a denial unless an L1 reorg removes the data it was decided on, so an answer
/// that has since gone stale lacks only denials of blocks this node has not replaced yet. Once a
/// rewind deleted a denial, derivation can build the denied block again; the verifier then checks
/// it again, denies it again if it still is invalid, and the node replaces it again. The safe
/// label never moves to it, since the verifier reports only blocks it verified.
///
/// A block before the verifier's activation has no cross-chain dependency, so it is never denied,
/// also while the view is unknown.
#[derive(Debug, Clone)]
pub struct DeniedBlocks {
    denied: watch::Receiver<Option<Arc<[BlockNumHash]>>>,
    /// The first L2 timestamp the verifier covers.
    activation_timestamp: u64,
}

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
        Self { denied: watch::channel(Some(Arc::from([]))).1, activation_timestamp: 0 }
    }

    /// The view of a verifier active from `activation_timestamp` on, unknown until the returned
    /// sender publishes the verifier's first answer.
    pub fn from_verifier(activation_timestamp: u64) -> (DeniedBlocksSender, Self) {
        let (sender, denied) = watch::channel(None);
        (DeniedBlocksSender(sender), Self { denied, activation_timestamp })
    }

    /// Whether `block`, of L2 time `timestamp`, is denied.
    pub fn contains(
        &self,
        block: BlockNumHash,
        timestamp: u64,
    ) -> Result<bool, DeniedBlocksUnknown> {
        if timestamp < self.activation_timestamp {
            return Ok(false);
        }
        let denied = self.denied.borrow();
        let denied = denied.as_ref().ok_or(DeniedBlocksUnknown)?;
        Ok(denied.contains(&block))
    }

    /// Fails unless whether a block of L2 time `timestamp` is denied is known.
    pub fn ensure_known(&self, timestamp: u64) -> Result<(), DeniedBlocksUnknown> {
        if timestamp < self.activation_timestamp {
            return Ok(());
        }
        self.denied.borrow().as_ref().map(|_| ()).ok_or(DeniedBlocksUnknown)
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

        assert_eq!(denied.contains(block(1, 1), 2), Ok(false));
        assert_eq!(denied.ensure_known(2), Ok(()));
    }

    /// Until the first answer nothing from activation on can be decided; afterwards the latest
    /// answer holds. A block before activation is never denied.
    #[test]
    fn a_verifier_view_is_unknown_until_the_first_answer() {
        let (sender, denied) = DeniedBlocks::from_verifier(4);
        assert_eq!(denied.contains(block(1, 1), 4), Err(DeniedBlocksUnknown));
        assert_eq!(denied.ensure_known(4), Err(DeniedBlocksUnknown));
        assert_eq!(denied.contains(block(1, 1), 3), Ok(false), "before activation");
        assert_eq!(denied.ensure_known(3), Ok(()), "before activation");

        sender.publish(vec![block(3, 3)]);
        assert_eq!(denied.contains(block(3, 3), 6), Ok(true));
        assert_eq!(denied.contains(block(3, 4), 6), Ok(false));

        drop(sender);
        assert_eq!(
            denied.contains(block(3, 3), 6),
            Ok(true),
            "the last answer outlives the sender"
        );
    }
}
