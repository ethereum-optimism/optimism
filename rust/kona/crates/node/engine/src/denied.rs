//! The blocks a cross-chain verifier denied, as the engine consults them.

use crate::EngineClient;
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, map::HashMap};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch;

/// A block a cross-chain verifier denied, and the L1 block the denial was decided on.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct DeniedBlock {
    /// The denied block.
    pub block: BlockNumHash,
    /// The latest L1 block the decision depended on: the denial holds only on an L1 chain that
    /// includes it.
    pub l1: BlockNumHash,
}

/// The engine's view of the blocks a cross-chain verifier denied.
///
/// A denied block has an invalid executing message. Derivation never imports it: where derived
/// attributes produce it, the deposits-only block replaces it. The view is the verifier's last
/// answer since the node started, or unknown before the first, so the node keeps no record of its
/// own. While it is unknown, no derived block can be imported, since it might be a denied one.
///
/// A denial holds only on the L1 chain it was decided on, so the view keeps the verifier's answer
/// as it stands on the node's own L1 chain, see [`HeardDenials::judge`]. A denial whose L1
/// block a reorg removed does not apply, also when the answer is stale or the verifier cannot be
/// reached: derivation builds the block again, also where the node replaced it already, see
/// [`crate::DenyTask`], and the verifier decides again on the new L1 chain. A denial on an L1
/// block the node has not seen yet is pending: whether its block is denied is unknown until the
/// node sees that L1 block. The safe label never moves to a denied block, since the verifier
/// reports only blocks it verified.
///
/// A block before the verifier's activation has no cross-chain dependency, so it is never denied,
/// also while the view is unknown.
#[derive(Debug, Clone)]
pub struct DeniedBlocks {
    denied: watch::Receiver<Option<Arc<DeniedView>>>,
    /// The first L2 timestamp the verifier covers.
    activation_timestamp: u64,
}

/// A verifier's denials as they stand on the node's L1 chain.
#[derive(Debug, Default, PartialEq, Eq)]
struct DeniedView {
    /// The denials on canonical L1 blocks.
    applied: Vec<BlockNumHash>,
    /// The denials on L1 blocks the node has not seen yet.
    pending: Vec<BlockNumHash>,
}

/// Publishes a cross-chain verifier's answers to a [`DeniedBlocks`] view.
#[derive(Debug)]
pub struct DeniedBlocksSender(watch::Sender<Option<Arc<DeniedView>>>);

/// Whether a block is denied is not known yet: the verifier has not answered, or denied it on an
/// L1 block the node has not seen.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("whether the block is denied is not known yet")]
pub struct DeniedBlocksUnknown;

impl DeniedBlocks {
    /// The view of a node that follows no verifier: nothing is ever denied.
    pub fn none() -> Self {
        Self {
            denied: watch::channel(Some(Arc::new(DeniedView::default()))).1,
            activation_timestamp: 0,
        }
    }

    /// The view of a verifier active from `activation_timestamp` on, unknown until the returned
    /// sender publishes the verifier's first answer.
    pub fn from_verifier(activation_timestamp: u64) -> (DeniedBlocksSender, Self) {
        let (sender, denied) = watch::channel(None);
        (DeniedBlocksSender(sender), Self { denied, activation_timestamp })
    }

    /// A view of a verifier active from genesis on whose answer applies to `blocks`.
    #[cfg(test)]
    pub(crate) fn applied(blocks: Vec<BlockNumHash>) -> Self {
        let view = DeniedView { applied: blocks, pending: Vec::new() };
        Self { denied: watch::channel(Some(Arc::new(view))).1, activation_timestamp: 0 }
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
        if denied.pending.contains(&block) {
            return Err(DeniedBlocksUnknown);
        }
        Ok(denied.applied.contains(&block))
    }

    /// Fails unless the verifier answered, for a block of L2 time `timestamp`.
    pub fn ensure_known(&self, timestamp: u64) -> Result<(), DeniedBlocksUnknown> {
        if timestamp < self.activation_timestamp {
            return Ok(());
        }
        self.denied.borrow().as_ref().map(|_| ()).ok_or(DeniedBlocksUnknown)
    }
}

impl DeniedBlocksSender {
    /// Publishes the verifier's denials as they stand on the node's L1 chain.
    pub fn publish(&self, judged: &JudgedDenials) {
        let view = DeniedView { applied: judged.applied.clone(), pending: judged.pending.clone() };
        self.0.send_replace(Some(Arc::new(view)));
    }
}

/// The denials a node heard from a cross-chain verifier, judged on the node's own L1 chain.
///
/// The latest answer is kept and judged again whenever the verifier cannot be reached, so a denial
/// does not outlive the L1 block it was decided on because its verifier is down.
#[derive(Debug, Default)]
pub struct HeardDenials {
    /// Whether the verifier answered since the node started.
    answered: bool,
    /// The denied blocks heard and not forgotten.
    heard: HashMap<BlockNumHash, Heard>,
}

/// A denial a node heard.
#[derive(Debug, Clone, Copy)]
struct Heard {
    /// The L1 block it was last decided on.
    l1: BlockNumHash,
    /// Whether the verifier's latest answer holds it.
    reported: bool,
    /// Whether a reorg removed `l1`: the denial is void, and is not judged again unless the
    /// verifier decides it again on another L1 block.
    voided: bool,
}

/// The denials a node heard, as they stand on its L1 chain.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct JudgedDenials {
    /// The blocks denied on canonical L1 blocks: they are denied.
    pub applied: Vec<BlockNumHash>,
    /// The blocks denied on L1 blocks the node has not seen yet, or could not look up: whether
    /// they are denied is not known yet.
    pub pending: Vec<BlockNumHash>,
    /// The blocks denied on L1 blocks a reorg replaced: they are not denied, and where the node
    /// replaced one, derivation must derive its height again. Each is reported once.
    pub reorged: Vec<BlockNumHash>,
}

/// Where an L1 block stands on the node's L1 chain.
enum OnL1 {
    Canonical,
    Reorged,
    NotSeen,
}

impl HeardDenials {
    /// Records the verifier's latest `answer`, every block it denied above the finalized head, or
    /// with `None` keeps the last one, and judges every denial heard on the L1 chain `client`
    /// reads. `None` until the verifier answered once.
    ///
    /// A denial on a canonical L1 block applies while the verifier reports it, and is forgotten
    /// once it does not. One on an L1 block another block replaced is void: it is reported as
    /// reorged once, also while a stale verifier keeps reporting it, until the verifier decides
    /// it again on another L1 block. One on an L1 block the node has not seen is pending while
    /// the verifier reports it. A denial at or below `finalized` is forgotten.
    pub async fn judge(
        &mut self,
        client: &EngineClient,
        answer: Option<Vec<DeniedBlock>>,
        finalized: u64,
    ) -> Option<JudgedDenials> {
        if let Some(answer) = answer {
            self.answered = true;
            self.heard.values_mut().for_each(|heard| heard.reported = false);
            for denial in answer {
                let voided = self
                    .heard
                    .get(&denial.block)
                    .is_some_and(|heard| heard.voided && heard.l1 == denial.l1);
                self.heard.insert(denial.block, Heard { l1: denial.l1, reported: true, voided });
            }
        }
        if !self.answered {
            return None;
        }
        let mut heard: Vec<_> = self.heard.iter().map(|(&block, &heard)| (block, heard)).collect();
        heard.sort_unstable_by_key(|(block, _)| (block.number, block.hash));
        let mut judged = JudgedDenials::default();
        let mut canonical: HashMap<u64, Option<B256>> = HashMap::default();
        for (block, heard) in heard {
            if block.number <= finalized || (heard.voided && !heard.reported) {
                self.heard.remove(&block);
                continue;
            }
            if heard.voided {
                continue;
            }
            match on_l1(client, heard.l1, &mut canonical).await {
                OnL1::Canonical if heard.reported => judged.applied.push(block),
                OnL1::NotSeen if heard.reported => judged.pending.push(block),
                OnL1::Reorged => {
                    debug!(target: "engine", denied = ?block, l1 = ?heard.l1, "A denial was decided on an L1 block a reorg removed");
                    judged.reorged.push(block);
                    match self.heard.get_mut(&block) {
                        Some(entry) if heard.reported => entry.voided = true,
                        _ => {
                            self.heard.remove(&block);
                        }
                    }
                }
                OnL1::Canonical | OnL1::NotSeen => {
                    self.heard.remove(&block);
                }
            }
        }
        Some(judged)
    }
}

/// Where `l1` stands on the L1 chain `client` reads, with the canonical hashes looked up so far.
/// A failed lookup counts as not seen.
async fn on_l1(
    client: &EngineClient,
    l1: BlockNumHash,
    canonical: &mut HashMap<u64, Option<B256>>,
) -> OnL1 {
    if l1.hash.is_zero() {
        return OnL1::Canonical;
    }
    let hash = match canonical.get(&l1.number) {
        Some(hash) => *hash,
        None => match client.l1_block_hash_by_number(l1.number).await {
            Ok(hash) => {
                canonical.insert(l1.number, hash);
                hash
            }
            Err(err) => {
                warn!(target: "engine", %err, ?l1, "Failed to look up the L1 block of a denial");
                return OnL1::NotSeen;
            }
        },
    };
    match hash {
        Some(hash) if hash == l1.hash => OnL1::Canonical,
        Some(_) => OnL1::Reorged,
        None => OnL1::NotSeen,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{RpcMock, test_engine_client};
    use alloy_rpc_types_eth::{Block as EthBlock, Transaction as EthTransaction};
    use kona_genesis::RollupConfig;

    fn block(number: u64, byte: u8) -> BlockNumHash {
        BlockNumHash { number, hash: B256::repeat_byte(byte) }
    }

    /// L1 block `number` of fork `fork`.
    fn l1_block(number: u64, fork: u8) -> EthBlock<EthTransaction> {
        let mut block = EthBlock::<EthTransaction>::default();
        block.header.inner.number = number;
        block.header.inner.extra_data = vec![fork].into();
        block.header.hash = block.header.inner.hash_slow();
        block
    }

    fn l1_id(number: u64, fork: u8) -> BlockNumHash {
        BlockNumHash { number, hash: l1_block(number, fork).header.hash }
    }

    /// Expects the lookup of L1 block `number`, answered with `fork`'s block or with null.
    fn expect_l1(l1: &RpcMock, number: u64, fork: Option<u8>) {
        let params = serde_json::json!([format!("{number:#x}"), false]);
        match fork {
            Some(fork) => l1.expect_params("eth_getBlockByNumber", params, l1_block(number, fork)),
            None => l1.expect_params("eth_getBlockByNumber", params, serde_json::Value::Null),
        }
    }

    fn denial(block: BlockNumHash, l1: BlockNumHash) -> DeniedBlock {
        DeniedBlock { block, l1 }
    }

    fn judged(
        applied: Vec<BlockNumHash>,
        pending: Vec<BlockNumHash>,
        reorged: Vec<BlockNumHash>,
    ) -> Option<JudgedDenials> {
        Some(JudgedDenials { applied, pending, reorged })
    }

    #[test]
    fn a_node_without_a_verifier_denies_nothing() {
        let denied = DeniedBlocks::none();

        assert_eq!(denied.contains(block(1, 1), 2), Ok(false));
        assert_eq!(denied.ensure_known(2), Ok(()));
    }

    /// Until the first answer nothing from activation on can be decided; afterwards the latest
    /// answer holds: an applied denial is denied, a pending one unknown. A block before
    /// activation is never denied.
    #[test]
    fn a_verifier_view_is_unknown_until_the_first_answer() {
        let (sender, denied) = DeniedBlocks::from_verifier(4);
        assert_eq!(denied.contains(block(1, 1), 4), Err(DeniedBlocksUnknown));
        assert_eq!(denied.ensure_known(4), Err(DeniedBlocksUnknown));
        assert_eq!(denied.contains(block(1, 1), 3), Ok(false), "before activation");
        assert_eq!(denied.ensure_known(3), Ok(()), "before activation");

        sender.publish(&JudgedDenials {
            applied: vec![block(3, 3)],
            pending: vec![block(5, 5)],
            reorged: vec![block(6, 6)],
        });
        assert_eq!(denied.contains(block(3, 3), 6), Ok(true));
        assert_eq!(denied.contains(block(3, 4), 6), Ok(false));
        assert_eq!(denied.contains(block(5, 5), 6), Err(DeniedBlocksUnknown));
        assert_eq!(denied.contains(block(6, 6), 6), Ok(false));
        assert_eq!(denied.ensure_known(6), Ok(()));

        drop(sender);
        assert_eq!(
            denied.contains(block(3, 3), 6),
            Ok(true),
            "the last answer outlives the sender"
        );
    }

    /// A denial applies only while its L1 block is canonical on the node's L1 chain: one on a
    /// block a reorg replaced is reorged, once, one on a block the node has not seen is pending,
    /// and each L1 block is looked up once per judgement.
    #[tokio::test]
    async fn a_denial_applies_only_on_the_l1_chain_it_was_decided_on() {
        let (client, l1, _l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let mut heard = HeardDenials::default();
        assert_eq!(heard.judge(&client, None, 0).await, None, "nothing heard yet");
        let answer = vec![
            denial(block(3, 3), l1_id(5, 0)),
            denial(block(4, 4), l1_id(6, 0)),
            denial(block(7, 7), l1_id(9, 0)),
            denial(block(8, 8), l1_id(5, 0)),
        ];
        expect_l1(&l1, 5, Some(0));
        expect_l1(&l1, 6, Some(1));
        expect_l1(&l1, 9, None);

        let first = heard.judge(&client, Some(answer.clone()), 0).await;

        assert_eq!(
            first,
            judged(vec![block(3, 3), block(8, 8)], vec![block(7, 7)], vec![block(4, 4)])
        );
        // A stale verifier reporting the void denial again does not make it apply, nor reorged
        // again, until it decides it again on another L1 block.
        expect_l1(&l1, 5, Some(0));
        expect_l1(&l1, 9, None);
        let again = heard.judge(&client, Some(answer), 0).await;
        assert_eq!(
            again,
            judged(vec![block(3, 3), block(8, 8)], vec![block(7, 7)], vec![]),
            "reorged once"
        );
        // Denials the verifier stopped reporting are looked up once more, as a reorg of their L1
        // blocks voids replacements made on them, and forgotten.
        expect_l1(&l1, 5, Some(0));
        expect_l1(&l1, 7, Some(1));
        expect_l1(&l1, 9, None);
        let decided_again =
            heard.judge(&client, Some(vec![denial(block(4, 4), l1_id(7, 1))]), 0).await;
        assert_eq!(decided_again, judged(vec![block(4, 4)], vec![], vec![]));
        l1.assert_finished();
    }

    /// While the verifier cannot be reached, its last answer is judged again: a denial whose L1
    /// block a reorg then removes stops applying although the answer still holds it. A denial
    /// the verifier stopped reporting on the canonical chain, or that is final, is forgotten.
    #[tokio::test]
    async fn a_stale_answer_is_judged_again_on_the_l1_chain() {
        let (client, l1, _l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let mut heard = HeardDenials::default();
        expect_l1(&l1, 5, Some(0));
        expect_l1(&l1, 6, Some(0));
        expect_l1(&l1, 7, Some(0));
        let answer = vec![
            denial(block(3, 3), l1_id(5, 0)),
            denial(block(4, 4), l1_id(6, 0)),
            denial(block(9, 9), l1_id(7, 0)),
        ];
        let first = heard.judge(&client, Some(answer), 0).await;
        assert_eq!(first, judged(vec![block(3, 3), block(4, 4), block(9, 9)], vec![], vec![]));

        // L1 reorgs from block 6 while the verifier is down.
        expect_l1(&l1, 6, Some(1));
        expect_l1(&l1, 7, Some(1));
        let stale = heard.judge(&client, None, 3).await;
        assert_eq!(
            stale,
            judged(vec![], vec![], vec![block(4, 4), block(9, 9)]),
            "block 3 is final"
        );

        // The verifier is back without the reorged denials.
        assert_eq!(heard.judge(&client, Some(vec![]), 3).await, judged(vec![], vec![], vec![]));
        l1.assert_finished();
    }

    /// A denial the verifier withdraws while its L1 block stays canonical is forgotten, not
    /// reorged: the node keeps whatever block it has.
    #[tokio::test]
    async fn a_withdrawn_denial_is_forgotten() {
        let (client, l1, _l2) = test_engine_client(Arc::new(RollupConfig::default()));
        let mut heard = HeardDenials::default();
        expect_l1(&l1, 5, Some(0));
        heard.judge(&client, Some(vec![denial(block(3, 3), l1_id(5, 0))]), 0).await;
        expect_l1(&l1, 5, Some(0));

        assert_eq!(heard.judge(&client, Some(vec![]), 0).await, judged(vec![], vec![], vec![]));
        assert_eq!(heard.judge(&client, None, 0).await, judged(vec![], vec![], vec![]));
        l1.assert_finished();
    }
}
