//! Sync start algorithm for the OP Stack rollup node.

use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;

mod forkchoice;
pub use forkchoice::L2ForkchoiceState;

mod error;
pub use error::SyncStartError;

use tracing::info;

use crate::EngineClient;

/// Searches for the latest [`L2ForkchoiceState`] that we can use to start the sync process with.
///
///   - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a *plausible*
///     extension of the canonical L1 chain (as known to the rollup node).
///   - The *local-safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
///     complete within the canonical L1 chain (as known to the rollup node).
///   - The *finalized L2 block*: This is the L2 block which is known to be fully derived from
///     finalized L1 block data.
///
/// Plausible: meaning that the blockhash of the L2 block's L1 origin
/// (as reported in the L1 Attributes deposit within the L2 block) is not canonical at another
/// height in the L1 chain, and the same holds for all its ancestors.
pub async fn find_starting_forkchoice<EngineClient_: EngineClient>(
    cfg: &RollupConfig,
    engine_client: &EngineClient_,
) -> Result<L2ForkchoiceState, SyncStartError> {
    let mut current_fc = L2ForkchoiceState::current(cfg, engine_client).await?;
    info!(
        target: "sync_start",
        unsafe = %current_fc.un_safe.block_info.number,
        local_safe = %current_fc.local_safe.block_info.number,
        finalized = %current_fc.finalized.block_info.number,
        "Loaded current L2 EL forkchoice state"
    );

    // Search for the highest `unsafe` L2 block whose L1 origin is *plausible* — canonical in the
    // L1 chain at its height, or ahead of the L1 head — and whose ancestors' origins all are as
    // well. This mirrors op-node's `FindL2Heads` walkback (op-node/rollup/sync/start.go): the
    // origin is fetched **by number** and its hash is compared. A by-hash existence check cannot
    // detect an L1 reorg, because execution layers keep serving reorged-out blocks via
    // `eth_getBlockByHash`.
    //
    // `candidate` is the highest L2 block that is still plausible as the unsafe head. It is
    // discarded (moved down past `cursor`) whenever a walked block's origin turns out to be
    // non-canonical, and kept as-is while origins are merely ahead of the L1 head.
    let mut candidate = current_fc.un_safe;
    let mut cursor = current_fc.un_safe;
    current_fc.un_safe = loop {
        info!(
            target: "sync_start",
            l1_origin = %cursor.l1_origin.number,
            l2_block = %cursor.block_info.number,
            "Searching for L2 unsafe block with canonical L1 origin"
        );
        let canonical_l1_hash = engine_client
            .get_l1_block(cursor.l1_origin.number.into())
            .await?
            .map(|block| block.header.hash);

        match canonical_l1_hash {
            Some(hash) if hash == cursor.l1_origin.hash => {
                // The origin is canonical at its height. Every deeper origin is an L1 ancestor
                // of it, and is therefore canonical as well: the candidate holds.
                info!(
                    target: "sync_start",
                    l2_unsafe = %candidate.block_info.number,
                    "Found L2 unsafe block with canonical L1 origin"
                );
                break candidate;
            }
            canonical_l1_hash => {
                // `None`: the origin is ahead of our L1 view — a plausible unsafe head. Keep
                // the candidate, but keep walking to verify the ancestors' origins.
                //
                // `Some(_)`: a different L1 block is canonical at the origin's height, so the
                // origin was reorged out. Everything from the candidate down to `cursor` sits
                // on a dead L1 branch and is discarded below.
                let ahead = canonical_l1_hash.is_none();

                if cursor.block_info.number == cfg.genesis.l2.number {
                    // Never walk past the L2 genesis block. Its L2 hash was already validated
                    // against the genesis hash by `L2BlockInfo::from_block_and_genesis`, so a
                    // non-canonical genesis L1 origin means the L1 chain source is for a
                    // different chain (op-node's `WrongChainErr`).
                    if let Some(hash) = canonical_l1_hash {
                        return Err(SyncStartError::InvalidL1GenesisHash(
                            cfg.genesis.l1.hash,
                            hash,
                        ));
                    }
                    // The L1 view does not serve the genesis origin yet; nothing deeper exists
                    // to verify against.
                    break candidate;
                }

                if cursor.block_info.number == current_fc.finalized.block_info.number {
                    if cursor.block_info.hash != current_fc.finalized.block_info.hash {
                        return Err(SyncStartError::MismatchedFinalizedBlock(
                            current_fc.finalized.block_info.hash,
                            cursor.block_info.hash,
                        ));
                    }
                    // Never walk past the finalized block — op-node ends its walkback there.
                    // If even its origin is non-canonical, L1 reorged past finality and no
                    // better candidate exists below: keep the finalized block itself.
                    break if ahead { candidate } else { cursor };
                }

                // Walk back to the L2 parent block.
                let l2_parent_hash = cursor.block_info.parent_hash.into();
                let l2_parent = engine_client
                    .get_l2_block(l2_parent_hash)
                    .full()
                    .await?
                    .ok_or(SyncStartError::BlockNotFound(l2_parent_hash))?;
                cursor =
                    L2BlockInfo::from_block_and_genesis(&l2_parent.into_consensus(), &cfg.genesis)?;

                if !ahead {
                    // The blocks above `cursor` were discarded: the parent is now the highest
                    // block still plausible as the unsafe head.
                    candidate = cursor;
                }
            }
        }
    };

    // Search for the highest local-safe block whose L1 origin is at least older than the
    // sequencing window, relative to the L1 origin of the `unsafe` block.
    let mut local_safe_cursor = current_fc.un_safe;
    loop {
        info!(
            target: "sync_start",
            l1_origin = %local_safe_cursor.l1_origin.number,
            l2_local_safe = %local_safe_cursor.block_info.number,
            "Searching for L2 local-safe block beyond sequencing window"
        );

        let is_behind_sequence_window =
            current_fc.un_safe.l1_origin.number.saturating_sub(cfg.seq_window_size) >
                local_safe_cursor.l1_origin.number;
        let is_finalized =
            local_safe_cursor.block_info.hash == current_fc.finalized.block_info.hash;
        let is_genesis = local_safe_cursor.block_info.hash == cfg.genesis.l2.hash;
        if is_behind_sequence_window || is_finalized || is_genesis {
            info!(
                target: "sync_start",
                l2_local_safe = %local_safe_cursor.block_info.number,
                is_behind_sequence_window,
                is_finalized,
                is_genesis,
                "Found suitable L2 local-safe block"
            );
            current_fc.local_safe = local_safe_cursor;
            break;
        }
        let block = engine_client
            .get_l2_block(local_safe_cursor.block_info.parent_hash.into())
            .full()
            .await?
            .ok_or(SyncStartError::BlockNotFound(
                local_safe_cursor.block_info.parent_hash.into(),
            ))?;
        local_safe_cursor =
            L2BlockInfo::from_block_and_genesis(&block.into_consensus(), &cfg.genesis)?;
    }

    // Leave the finalized block as-is, and return the current forkchoice.
    Ok(current_fc)
}

#[cfg(test)]
mod test {
    use alloy_provider::Network;
    use alloy_rpc_types_eth::Block;
    use kona_protocol::L2BlockInfo;
    use kona_registry::ROLLUP_CONFIGS;
    use op_alloy_network::Optimism;

    mod walkback {
        use crate::{find_starting_forkchoice, test_utils::MockEngineClient};
        use alloy_eips::{BlockId, BlockNumHash, BlockNumberOrTag};
        use alloy_primitives::{B256, Bytes, Sealed};
        use alloy_rpc_types_eth::{Block, BlockTransactions, Header, Transaction};
        use kona_genesis::{ChainGenesis, RollupConfig};
        use op_alloy_consensus::{OpTxEnvelope, TxDeposit};
        use op_alloy_rpc_types::Transaction as OpTransaction;
        use std::sync::Arc;

        /// Canonical L1 chain hashes, by height.
        const L1_GENESIS: B256 = B256::repeat_byte(0xa0);
        const L1_ONE: B256 = B256::repeat_byte(0xa1);
        const L1_TWO: B256 = B256::repeat_byte(0xa2);
        /// A reorged-out L1 block at height 2: resolvable by hash, not canonical at its number.
        const L1_TWO_REORGED: B256 = B256::repeat_byte(0xb2);
        /// An L1 block at height 3, ahead of the L1 head (height 2): unresolvable entirely.
        const L1_THREE_AHEAD: B256 = B256::repeat_byte(0xb3);

        /// Encodes Bedrock L1-info deposit calldata pointing at the given L1 origin.
        fn l1_info_calldata(origin_number: u64, origin_hash: B256, sequence_number: u64) -> Bytes {
            let mut buf = vec![0u8; 260];
            buf[0..4].copy_from_slice(&[0x01, 0x5d, 0x8e, 0xb9]);
            buf[28..36].copy_from_slice(&origin_number.to_be_bytes());
            buf[100..132].copy_from_slice(origin_hash.as_slice());
            buf[156..164].copy_from_slice(&sequence_number.to_be_bytes());
            buf.into()
        }

        /// Builds an L2 RPC block. Non-genesis blocks carry an L1-info deposit as their first
        /// transaction, pointing at `origin` with the given sequence number.
        fn l2_block(
            number: u64,
            parent_hash: B256,
            origin: Option<(BlockNumHash, u64)>,
        ) -> Block<OpTransaction> {
            let transactions = origin.map_or_else(Vec::new, |(origin, seq)| {
                let deposit = TxDeposit {
                    input: l1_info_calldata(origin.number, origin.hash, seq),
                    ..Default::default()
                };
                let inner = Transaction {
                    inner: alloy_consensus::transaction::Recovered::new_unchecked(
                        OpTxEnvelope::Deposit(Sealed::new(deposit)),
                        Default::default(),
                    ),
                    block_hash: None,
                    block_number: Some(number),
                    effective_gas_price: Some(0),
                    transaction_index: Some(0),
                    block_timestamp: None,
                };
                vec![OpTransaction { inner, deposit_nonce: None, deposit_receipt_version: None }]
            });
            let inner = alloy_consensus::Header {
                number,
                parent_hash,
                timestamp: number * 2,
                ..Default::default()
            };
            Block {
                header: Header { hash: inner.hash_slow(), inner, ..Default::default() },
                transactions: BlockTransactions::Full(transactions),
                ..Default::default()
            }
        }

        /// Builds an L1 RPC block with the given RPC-reported hash.
        fn l1_block(number: u64, hash: B256) -> Block {
            Block {
                header: Header {
                    hash,
                    inner: alloy_consensus::Header { number, ..Default::default() },
                    ..Default::default()
                },
                ..Default::default()
            }
        }

        /// Builds a 4-block L2 chain over a 3-block canonical L1 chain and runs the walkback.
        ///
        /// - L1: canonical blocks 0..=2. Height 2 is post-reorg: [`L1_TWO`] is canonical while
        ///   [`L1_TWO_REORGED`] still *resolves by hash* (as real execution layers do for
        ///   side-chain blocks) but is served for neither `2` nor any other height.
        ///   [`L1_THREE_AHEAD`] resolves to nothing: it is beyond the L1 head.
        /// - L2: genesis, then blocks 1..=3 whose L1 origins are given by `origins` (`(origin,
        ///   sequence_number)` for blocks 1, 2, 3). Finalized = genesis, safe = block 1, latest
        ///   (unsafe) = block 3.
        ///
        /// Returns the L2 block number `find_starting_forkchoice` selected as the unsafe head.
        async fn walkback_unsafe_head(origins: [(BlockNumHash, u64); 3]) -> u64 {
            let genesis = l2_block(0, B256::ZERO, None);
            let genesis_hash = genesis.header.hash;
            let b1 = l2_block(1, genesis_hash, Some(origins[0]));
            let b2 = l2_block(2, b1.header.hash, Some(origins[1]));
            let b3 = l2_block(3, b2.header.hash, Some(origins[2]));

            let cfg = RollupConfig {
                genesis: ChainGenesis {
                    l1: BlockNumHash { number: 0, hash: L1_GENESIS },
                    l2: BlockNumHash { number: 0, hash: genesis_hash },
                    ..Default::default()
                },
                seq_window_size: 3600,
                ..Default::default()
            };

            let client = MockEngineClient::builder()
                .with_config(Arc::new(cfg.clone()))
                // Canonical L1 blocks resolve both by number and by hash.
                .with_l1_block(BlockId::from(0u64), l1_block(0, L1_GENESIS))
                .with_l1_block(BlockId::from(L1_GENESIS), l1_block(0, L1_GENESIS))
                .with_l1_block(BlockId::from(1u64), l1_block(1, L1_ONE))
                .with_l1_block(BlockId::from(L1_ONE), l1_block(1, L1_ONE))
                .with_l1_block(BlockId::from(2u64), l1_block(2, L1_TWO))
                .with_l1_block(BlockId::from(L1_TWO), l1_block(2, L1_TWO))
                // The reorged-out L1 block resolves ONLY by hash.
                .with_l1_block(BlockId::from(L1_TWO_REORGED), l1_block(2, L1_TWO_REORGED))
                // L2 forkchoice labels.
                .with_l2_block(BlockNumberOrTag::Finalized.into(), genesis.clone())
                .with_l2_block(BlockNumberOrTag::Safe.into(), b1.clone())
                .with_l2_block(BlockNumberOrTag::Latest.into(), b3)
                // L2 blocks by hash, for parent traversal.
                .with_l2_block(BlockId::from(genesis.header.hash), genesis)
                .with_l2_block(BlockId::from(b1.header.hash), b1)
                .with_l2_block(BlockId::from(b2.header.hash), b2)
                .build();

            let fc = find_starting_forkchoice(&cfg, &client).await.unwrap();
            fc.un_safe.block_info.number
        }

        /// Baseline: the unsafe head's L1 origin is canonical at its height, so the head is kept.
        #[tokio::test]
        async fn test_unsafe_head_kept_when_origin_canonical() {
            let unsafe_head = walkback_unsafe_head([
                (BlockNumHash { number: 1, hash: L1_ONE }, 0),
                (BlockNumHash { number: 1, hash: L1_ONE }, 1),
                (BlockNumHash { number: 2, hash: L1_TWO }, 0),
            ])
            .await;
            assert_eq!(unsafe_head, 3);
        }

        /// After an L1 reorg the unsafe head's origin still *exists by hash* but is no longer
        /// canonical at its height. The walkback must not be fooled by hash existence: it has to
        /// fetch the origin by number, compare hashes, and walk back past the poisoned head —
        /// mirroring op-node's `FindL2Heads`.
        #[tokio::test]
        async fn test_unsafe_head_walks_back_past_reorged_out_origin() {
            let unsafe_head = walkback_unsafe_head([
                (BlockNumHash { number: 1, hash: L1_ONE }, 0),
                (BlockNumHash { number: 1, hash: L1_ONE }, 1),
                (BlockNumHash { number: 2, hash: L1_TWO_REORGED }, 0),
            ])
            .await;
            assert_eq!(unsafe_head, 2);
        }

        /// An origin beyond the L1 head is plausible: the head is kept, but its ancestors are
        /// still verified for canonicality.
        #[tokio::test]
        async fn test_unsafe_head_kept_when_origin_ahead_of_l1_head() {
            let unsafe_head = walkback_unsafe_head([
                (BlockNumHash { number: 1, hash: L1_ONE }, 0),
                (BlockNumHash { number: 1, hash: L1_ONE }, 1),
                (BlockNumHash { number: 3, hash: L1_THREE_AHEAD }, 0),
            ])
            .await;
            assert_eq!(unsafe_head, 3);
        }

        /// An ahead origin only keeps the head *tentatively*: when a deeper ancestor's origin
        /// turns out to be reorged out, the head (and everything down to that ancestor) is
        /// discarded.
        #[tokio::test]
        async fn test_ahead_unsafe_head_discarded_when_ancestor_origin_reorged_out() {
            let unsafe_head = walkback_unsafe_head([
                (BlockNumHash { number: 1, hash: L1_ONE }, 0),
                (BlockNumHash { number: 2, hash: L1_TWO_REORGED }, 0),
                (BlockNumHash { number: 3, hash: L1_THREE_AHEAD }, 0),
            ])
            .await;
            assert_eq!(unsafe_head, 1);
        }
    }

    const OP_SEPOLIA_CHAIN_ID: u64 = 11155420;
    const OP_SEPOLIA_GENESIS_RPC_RESPONSE: &str = "{\"hash\":\"0x102de6ffb001480cc9b8b548fd05c34cd4f46ae4aa91759393db90ea0409887d\",\"parentHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0x4200000000000000000000000000000000000011\",\"stateRoot\":\"0x06787a17a3ed87c339a39dbbeeb311578a0c83ed29daa2db95da62b28efce8a9\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x0\",\"number\":\"0x0\",\"gasLimit\":\"0x1c9c380\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x64d6dbac\",\"extraData\":\"0x424544524f434b\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"baseFeePerGas\":\"0x3b9aca00\",\"size\":\"0x209\",\"uncles\":[],\"transactions\":[]}";

    /// Sanity regression test - `alloy_rpc_types`' `Block::into_consensus` failed to saturate the
    /// header of the `alloy_consensus::Header` type on an old version. This test covers the
    /// conversion to ensure an OP genesis block's conversion to the consensus type works for
    /// the sake of `L2BlockInfo::from_block_and_genesis`.
    #[tokio::test]
    async fn test_genesis_block_hash() {
        let rollup_config = ROLLUP_CONFIGS.get(&OP_SEPOLIA_CHAIN_ID).unwrap();
        let genesis_block: Block<<Optimism as Network>::TransactionResponse> =
            serde_json::from_str(OP_SEPOLIA_GENESIS_RPC_RESPONSE).unwrap();

        let rpc_reported_hash = genesis_block.header.hash;
        let consensus_block = genesis_block.into_consensus();

        // Check that the genesis block's RPC-reported hash is equal to the manually computed hash.
        assert_eq!(rpc_reported_hash, consensus_block.hash_slow());

        // Convert to `L2BlockInfo` and check the same.
        let l2_block_info =
            L2BlockInfo::from_block_and_genesis(&consensus_block, &rollup_config.genesis).unwrap();
        assert_eq!(rpc_reported_hash, l2_block_info.block_info.hash);
    }
}
