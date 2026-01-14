#[cfg(feature = "metrics")]
use std::time::Instant;
use std::time::SystemTime;

use alloy_consensus::Block;
use alloy_eips::eip7685::EMPTY_REQUESTS_HASH;
use alloy_primitives::{Address, B256};
use alloy_rpc_types_engine::{ExecutionPayloadV3, PayloadError};
use kona_genesis::RollupConfig;
use libp2p::gossipsub::MessageAcceptance;
use op_alloy_consensus::OpTxEnvelope;
use op_alloy_rpc_types_engine::{
    OpExecutionPayload, OpExecutionPayloadV4, OpNetworkPayloadEnvelope, OpPayloadError,
};

use super::BlockHandler;
#[cfg(feature = "metrics")]
use crate::Metrics;

/// Error that can occur when validating a block.
#[derive(Debug, thiserror::Error)]
pub enum BlockInvalidError {
    /// The block has an invalid timestamp.
    #[error("Invalid timestamp. Current: {current}, Received: {received}")]
    Timestamp {
        /// The current timestamp.
        current: u64,
        /// The received timestamp.
        received: u64,
    },
    /// The block has an invalid base fee per gas.
    #[error("Base fee per gas overflow")]
    BaseFeePerGasOverflow(#[from] PayloadError),
    /// The block has an invalid hash.
    #[error("Invalid block hash. Expected: {expected}, Received: {received}")]
    BlockHash {
        /// The expected block hash.
        expected: B256,
        /// The received block hash.
        received: B256,
    },
    /// The block has an invalid signature.
    #[error("Invalid signature.")]
    Signature,
    /// The block has an invalid signer.
    #[error("Invalid signer, expected: {expected}, received: {received}")]
    Signer {
        /// The expected signer.
        expected: Address,
        /// The received signer.
        received: Address,
    },
    /// Invalid block.
    #[error(transparent)]
    InvalidBlock(#[from] OpPayloadError),
    /// The block has an invalid parent beacon block root.
    #[error("Payload is on v3+ topic, but has empty parent beacon root")]
    ParentBeaconRoot,
    /// The block has an invalid blob gas used.
    #[error("Payload is on v3+ topic, but has non-zero blob gas used")]
    BlobGasUsed,
    /// The block has an invalid excess blob gas.
    #[error("Payload is on v3+ topic, but has non-zero excess blob gas")]
    ExcessBlobGas,
    /// The block has an invalid withdrawals root.
    #[error("Payload is on v4+ topic, but has non-empty withdrawals root")]
    WithdrawalsRoot,
    /// Too many blocks were validated for the same height.
    #[error("Too many blocks seen for height {height}")]
    TooManyBlocks {
        /// The height of the block.
        height: u64,
    },
    /// The block has already been seen.
    #[error("Block seen before")]
    BlockSeen {
        /// The hash of the block.
        block_hash: B256,
    },
}

impl From<BlockInvalidError> for MessageAcceptance {
    fn from(value: BlockInvalidError) -> Self {
        // We only want to ignore blocks that we have already seen.
        match value {
            BlockInvalidError::BlockSeen { block_hash: _ } => Self::Ignore,
            _ => Self::Reject,
        }
    }
}

impl BlockHandler {
    /// The maximum number of blocks to keep in the seen hashes map.
    ///
    /// Note: this value must be high enough to ensure we prevent replay attacks.
    /// Ie, the entries pruned must be old enough blocks to be considered invalid
    /// if new blocks for that height are received.
    ///
    /// This value is chosen to match `op-node` validator's lru cache size.
    /// See: <https://github.com/ethereum-optimism/optimism/blob/836d50be5d5f4ae14ffb2ea6106720a2b080cdae/op-node/p2p/gossip.go#L266>
    pub const SEEN_HASH_CACHE_SIZE: usize = 1_000;

    /// The maximum number of blocks to keep per height.
    /// This value is chosen according to the optimism specs:
    /// <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-validation>
    const MAX_BLOCKS_TO_KEEP: usize = 5;

    /// Determines if a block is valid.
    ///
    /// We validate the block according to the rules defined here:
    /// <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-validation>
    ///
    /// The block encoding/compression are assumed to be valid at this point (they are first checked
    /// in the handle).
    pub fn block_valid(
        &mut self,
        envelope: &OpNetworkPayloadEnvelope,
    ) -> Result<(), BlockInvalidError> {
        // Start timing for the validation duration
        #[cfg(feature = "metrics")]
        let validation_start = Instant::now();

        // Record total validation attempts
        #[cfg(feature = "metrics")]
        kona_macros::inc!(counter, Metrics::BLOCK_VALIDATION_TOTAL);

        // Record block version distribution
        #[cfg(feature = "metrics")]
        {
            let version = match &envelope.payload {
                OpExecutionPayload::V1(_) => "v1",
                OpExecutionPayload::V2(_) => "v2",
                OpExecutionPayload::V3(_) => "v3",
                OpExecutionPayload::V4(_) => "v4",
            };
            kona_macros::inc!(counter, Metrics::BLOCK_VERSION, "version" => version);
        }

        let validation_result = self.validate_block_internal(envelope);

        // Record validation duration
        #[cfg(feature = "metrics")]
        {
            let duration = validation_start.elapsed();
            kona_macros::record!(
                histogram,
                Metrics::BLOCK_VALIDATION_DURATION_SECONDS,
                duration.as_secs_f64()
            );
        }

        // Record success/failure metrics
        match &validation_result {
            Ok(()) => {
                #[cfg(feature = "metrics")]
                kona_macros::inc!(counter, Metrics::BLOCK_VALIDATION_SUCCESS);
            }
            Err(_err) => {
                #[cfg(feature = "metrics")]
                {
                    let reason = match _err {
                        BlockInvalidError::Timestamp { current, received } => {
                            if *received > *current + 5 {
                                "timestamp_future"
                            } else {
                                "timestamp_past"
                            }
                        }
                        BlockInvalidError::BlockHash { .. } => "invalid_hash",
                        BlockInvalidError::Signature => "invalid_signature",
                        BlockInvalidError::Signer { .. } => "invalid_signer",
                        BlockInvalidError::TooManyBlocks { .. } => "too_many_blocks",
                        BlockInvalidError::BlockSeen { .. } => "block_seen",
                        BlockInvalidError::InvalidBlock(_) => "invalid_block",
                        BlockInvalidError::ParentBeaconRoot => "parent_beacon_root",
                        BlockInvalidError::BlobGasUsed => "blob_gas_used",
                        BlockInvalidError::ExcessBlobGas => "excess_blob_gas",
                        BlockInvalidError::WithdrawalsRoot => "withdrawals_root",
                        BlockInvalidError::BaseFeePerGasOverflow(_) => "invalid_block",
                    };
                    kona_macros::inc!(counter, Metrics::BLOCK_VALIDATION_FAILED, "reason" => reason);
                }
            }
        }

        validation_result
    }

    /// Internal validation logic extracted for cleaner metrics instrumentation.
    fn validate_block_internal(
        &mut self,
        envelope: &OpNetworkPayloadEnvelope,
    ) -> Result<(), BlockInvalidError> {
        let current_timestamp =
            SystemTime::now().duration_since(SystemTime::UNIX_EPOCH).unwrap().as_secs();

        // The timestamp is at most 5 seconds in the future.
        let is_future = envelope.payload.timestamp() > current_timestamp + 5;
        // The timestamp is at most 60 seconds in the past.
        let is_past = envelope.payload.timestamp() < current_timestamp - 60;

        // CHECK: The timestamp is not too far in the future or past.
        if is_future || is_past {
            return Err(BlockInvalidError::Timestamp {
                current: current_timestamp,
                received: envelope.payload.timestamp(),
            });
        }

        // CHECK: Ensure the block hash is valid.
        let expected = envelope.payload.block_hash();
        let mut block: Block<OpTxEnvelope> = envelope.payload.clone().try_into_block()?;
        block.header.parent_beacon_block_root = envelope.parent_beacon_block_root;
        // If isthmus is active, set the requests hash to the empty hash.
        if self.rollup_config.is_isthmus_active(envelope.payload.timestamp()) {
            block.header.requests_hash = Some(EMPTY_REQUESTS_HASH);
        }
        let received = block.header.hash_slow();
        if received != expected {
            return Err(BlockInvalidError::BlockHash { expected, received });
        }

        // CHECK: The payload is valid for the specific version of this block.
        self.validate_version_specific_payload(envelope)?;

        if let Some(seen_hashes_at_height) =
            self.seen_hashes.get_mut(&envelope.payload.block_number())
        {
            // CHECK: If more than [`Self::MAX_BLOCKS_TO_KEEP`] different blocks have been received
            // for the same height, reject the block.
            if seen_hashes_at_height.len() > Self::MAX_BLOCKS_TO_KEEP {
                return Err(BlockInvalidError::TooManyBlocks {
                    height: envelope.payload.block_number(),
                });
            }

            // CHECK: If the block has already been seen, ignore it.
            if seen_hashes_at_height.contains(&envelope.payload.block_hash()) {
                return Err(BlockInvalidError::BlockSeen {
                    block_hash: envelope.payload.block_hash(),
                });
            }
        }

        // CHECK: The signature is valid.
        let msg = envelope.payload_hash.signature_message(self.rollup_config.l2_chain_id.id());
        let block_signer = *self.signer_recv.borrow();

        // The block has a valid signature.
        let Ok(msg_signer) = envelope.signature.recover_address_from_prehash(&msg) else {
            return Err(BlockInvalidError::Signature);
        };

        // The block is signed by the expected signer (the unsafe block signer).
        if msg_signer != block_signer {
            return Err(BlockInvalidError::Signer { expected: block_signer, received: msg_signer });
        }

        self.seen_hashes
            .entry(envelope.payload.block_number())
            .or_default()
            .insert(envelope.payload.block_hash());

        // Mark the block as seen.
        if self.seen_hashes.len() >= Self::SEEN_HASH_CACHE_SIZE {
            self.seen_hashes.pop_first();
        }

        Ok(())
    }

    /// Validate version specific contents of the payload.
    fn validate_version_specific_payload(
        &self,
        envelope: &OpNetworkPayloadEnvelope,
    ) -> Result<(), BlockInvalidError> {
        // Validation for v1 payloads are mostly ensured by type-safety, by decoding the
        // payload to the ExecutionPayloadV1 type:
        // 1. The block should not have any withdrawals
        // 2. The block should not have any withdrawals list
        // 3. The block should not have any blob gas used
        // 4. The block should not have any excess blob gas
        // 5. The block should not have any withdrawals root
        // 6. The block should not have any parent beacon block root (validated because ignored by
        //    the decoder, this causes a hash mismatch. See tests)

        // Same as v1, except:
        // 1. The block should have an empty withdrawals list. This is checked during the call to
        //    [`OpExecutionPayload::try_into_block`].

        // Same as v2, except:
        // 1. The block should have a zero blob gas used
        // 2. The block should have a zero excess blob gas
        // 3. The block should have a non empty parent beacon block root
        fn validate_v3(
            rollup_config: &RollupConfig,
            block: &ExecutionPayloadV3,
            parent_beacon_block_root: Option<B256>,
        ) -> Result<(), BlockInvalidError> {
            // If Jovian is not active, the blob gas used should be zero.
            if !rollup_config.is_jovian_active(block.timestamp()) && block.blob_gas_used != 0 {
                return Err(BlockInvalidError::BlobGasUsed);
            }

            if block.excess_blob_gas != 0 {
                return Err(BlockInvalidError::ExcessBlobGas);
            }

            if parent_beacon_block_root.is_none() {
                return Err(BlockInvalidError::ParentBeaconRoot);
            }

            Ok(())
        }

        // Same as v3, except:
        // 1. The block should have an non-empty withdrawals root (checked by type-safety)
        fn validate_v4(
            rollup_config: &RollupConfig,
            block: &OpExecutionPayloadV4,
            parent_beacon_block_root: Option<B256>,
        ) -> Result<(), BlockInvalidError> {
            validate_v3(rollup_config, &block.payload_inner, parent_beacon_block_root)
        }

        match &envelope.payload {
            OpExecutionPayload::V1(_) => Ok(()),
            OpExecutionPayload::V2(_) => Ok(()),
            OpExecutionPayload::V3(payload) => {
                validate_v3(&self.rollup_config, payload, envelope.parent_beacon_block_root)
            }
            OpExecutionPayload::V4(payload) => {
                validate_v4(&self.rollup_config, payload, envelope.parent_beacon_block_root)
            }
        }
    }
}

#[cfg(test)]
pub(crate) mod tests {

    use super::*;
    use alloy_chains::Chain;
    use alloy_consensus::{Block, EMPTY_OMMER_ROOT_HASH};
    use alloy_eips::{eip2718::Encodable2718, eip4895::Withdrawal};
    use alloy_primitives::{Address, B256, Bytes, Signature};
    use alloy_rlp::BufMut;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV2, ExecutionPayloadV3};
    use arbitrary::{Arbitrary, Unstructured};
    use kona_genesis::RollupConfig;
    use op_alloy_consensus::OpTxEnvelope;
    use op_alloy_rpc_types_engine::{OpExecutionPayload, OpExecutionPayloadV4, PayloadHash};

    fn valid_block() -> Block<OpTxEnvelope> {
        // Simulate some random data
        let mut data = vec![0; 1024 * 1024];
        let mut rng = rand::rng();
        rand::Rng::fill(&mut rng, &mut data[..]);

        // Create unstructured data with the random bytes
        let u = Unstructured::new(&data);

        // Generate a random instance of MyStruct
        let mut block: Block<OpTxEnvelope> = Block::arbitrary_take_rest(u).unwrap();

        let transactions: Vec<Bytes> =
            block.body.transactions().map(|tx| tx.encoded_2718().into()).collect();

        let transactions_root =
            alloy_consensus::proofs::ordered_trie_root_with_encoder(&transactions, |item, buf| {
                buf.put_slice(item)
            });

        block.header.transactions_root = transactions_root;

        // We always need to set the base fee per gas to a positive value to ensure the block is
        // valid.
        block.header.base_fee_per_gas =
            Some(block.header.base_fee_per_gas.unwrap_or_default().saturating_add(1));

        let current_timestamp =
            SystemTime::now().duration_since(SystemTime::UNIX_EPOCH).unwrap().as_secs();
        block.header.timestamp = current_timestamp;

        block
    }

    /// Make the block v1 compatible
    fn v1_valid_block() -> Block<OpTxEnvelope> {
        let mut block = valid_block();
        block.header.withdrawals_root = None;
        block.header.blob_gas_used = None;
        block.header.excess_blob_gas = None;
        block.header.parent_beacon_block_root = None;
        block.header.requests_hash = None;
        block.header.ommers_hash = EMPTY_OMMER_ROOT_HASH;
        block.header.difficulty = Default::default();
        block.header.nonce = Default::default();

        block
    }

    /// Make the block v2 compatible
    pub(crate) fn v2_valid_block() -> Block<OpTxEnvelope> {
        let mut block = v1_valid_block();

        block.body.withdrawals = Some(vec![].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );

        block.header.withdrawals_root = Some(withdrawals_root);

        block
    }

    /// Make the block v3 compatible
    pub(crate) fn v3_valid_block() -> Block<OpTxEnvelope> {
        let mut block = valid_block();

        block.body.withdrawals = Some(vec![].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );
        block.header.withdrawals_root = Some(withdrawals_root);

        block.header.blob_gas_used = Some(0);
        block.header.excess_blob_gas = Some(0);
        block.header.parent_beacon_block_root =
            Some(block.header.parent_beacon_block_root.unwrap_or_default());

        block.header.requests_hash = None;
        block.header.ommers_hash = EMPTY_OMMER_ROOT_HASH;
        block.header.difficulty = Default::default();
        block.header.nonce = Default::default();

        block
    }

    /// Make the block v4 compatible
    pub(crate) fn v4_valid_block() -> Block<OpTxEnvelope> {
        v3_valid_block()
    }

    /// Generates a random valid block and ensure it is v1 compatible
    #[test]
    fn test_block_valid() {
        let block = v1_valid_block();

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());
    }

    /// Generates a random block with an invalid timestamp and ensure it is rejected
    #[test]
    fn test_block_invalid_timestamp_early() {
        let mut block = v1_valid_block();

        block.header.timestamp -= 61;

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::Timestamp { .. })));
    }

    /// Generates a random block with an invalid timestamp and ensure it is rejected
    #[test]
    fn test_block_invalid_timestamp_too_far() {
        let mut block = v1_valid_block();

        block.header.timestamp += 6;

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::Timestamp { .. })));
    }

    /// Generates a random block with an invalid hash and ensure it is rejected
    #[test]
    fn test_block_invalid_hash() {
        let block = v1_valid_block();

        let mut v1 = ExecutionPayloadV1::from_block_slow(&block);

        v1.block_hash = B256::ZERO;

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::BlockHash { .. })));
    }

    #[test]
    fn test_cannot_validate_same_block_twice() {
        let block = v1_valid_block();

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());
        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::BlockSeen { .. })));
    }

    #[test]
    fn test_cannot_have_too_many_blocks_for_the_same_height() {
        let first_block = v1_valid_block();

        let initial_height = first_block.header.number;

        let v1 = ExecutionPayloadV1::from_block_slow(&first_block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());

        let next_payloads = (0..=BlockHandler::MAX_BLOCKS_TO_KEEP)
            .map(|_| {
                let mut block = v1_valid_block();
                // The blocks have the same height
                block.header.number = initial_height;

                let v1 = ExecutionPayloadV1::from_block_slow(&block);

                let payload = OpExecutionPayload::V1(v1);
                OpNetworkPayloadEnvelope {
                    payload,
                    signature: Signature::test_signature(),
                    payload_hash: PayloadHash(B256::ZERO),
                    parent_beacon_block_root: None,
                }
            })
            .collect::<Vec<_>>();

        for envelope in next_payloads[..next_payloads.len() - 1].iter() {
            assert!(handler.block_valid(envelope).is_ok());
        }

        // The last envelope should fail
        assert!(matches!(
            handler.block_valid(next_payloads.last().unwrap()),
            Err(BlockInvalidError::TooManyBlocks { .. })
        ));
    }

    /// Blocks with invalid signatures should be rejected.
    #[test]
    fn test_invalid_signature() {
        let block = v1_valid_block();

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let mut envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        let mut signature_bytes = envelope.signature.as_bytes();
        signature_bytes[0] = !signature_bytes[0];
        envelope.signature = Signature::from_raw_array(&signature_bytes).unwrap();

        assert!(handler.seen_hashes.is_empty());
        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::Signature)));
    }

    /// Blocks with invalid signers should be rejected.
    #[test]
    fn test_invalid_signer() {
        let block = v1_valid_block();

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let (_, unsafe_signer) = tokio::sync::watch::channel(Address::default());
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::Signer { .. })));
    }

    /// If we specify a non empty parent beacon block root for blocks with v1/v2 payloads we
    /// get a hash mismatch error because the decoder enforces that these versions of the execution
    /// payload don't contain the parent beacon block root.
    #[test]
    fn test_v1_v2_block_invalid_parent_beacon_block_root() {
        let block = v1_valid_block();

        let v1 = ExecutionPayloadV1::from_block_slow(&block);

        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(B256::ZERO),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::BlockHash { .. })));

        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(B256::ZERO),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::BlockHash { .. })));
    }

    #[test]
    fn test_v2_block() {
        let block = v2_valid_block();

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());
    }

    #[test]
    fn test_v2_non_empty_withdrawals() {
        let mut block = v2_valid_block();
        block.body.withdrawals = Some(vec![Withdrawal::default()].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );
        block.header.withdrawals_root = Some(withdrawals_root);

        let v2 = ExecutionPayloadV2::from_block_slow(&block);

        let payload = OpExecutionPayload::V2(v2);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(
            handler.block_valid(&envelope),
            Err(BlockInvalidError::InvalidBlock(OpPayloadError::NonEmptyL1Withdrawals))
        ));
    }

    #[test]
    fn test_v3_block() {
        let block = v3_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());
    }

    #[test]
    fn test_v3_non_empty_withdrawals() {
        let mut block = v3_valid_block();
        block.body.withdrawals = Some(vec![Withdrawal::default()].into());
        let withdrawals_root = alloy_consensus::proofs::calculate_withdrawals_root(
            &block.body.withdrawals.clone().unwrap_or_default(),
        );
        block.header.withdrawals_root = Some(withdrawals_root);

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(
            handler.block_valid(&envelope),
            Err(BlockInvalidError::InvalidBlock(OpPayloadError::NonEmptyL1Withdrawals))
        ));
    }

    #[test]
    fn test_v3_gas_params() {
        let mut block = v3_valid_block();
        block.header.blob_gas_used = Some(1);

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::BlobGasUsed)));

        block.header.blob_gas_used = Some(0);
        block.header.excess_blob_gas = Some(1);

        let v3 = ExecutionPayloadV3::from_block_slow(&block);

        let payload = OpExecutionPayload::V3(v3);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        assert!(matches!(handler.block_valid(&envelope), Err(BlockInvalidError::ExcessBlobGas)));
    }

    #[test]
    fn test_v4_block() {
        let block = v4_valid_block();

        let v3 = ExecutionPayloadV3::from_block_slow(&block);
        let v4 = OpExecutionPayloadV4::from_v3_with_withdrawals_root(
            v3,
            block.withdrawals_root.unwrap(),
        );

        let payload = OpExecutionPayload::V4(v4);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: Some(
                block.header.parent_beacon_block_root.unwrap_or_default(),
            ),
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        assert!(handler.block_valid(&envelope).is_ok());
    }

    #[test]
    #[cfg(feature = "metrics")]
    fn test_metrics_instrumentation() {
        // This test verifies that metrics code compiles and doesn't panic
        // The actual metric values would require a metrics registry setup in a real test
        // environment

        use crate::Metrics;

        // Initialize metrics (this should not panic)
        Metrics::init();

        let block = v1_valid_block();
        let v1 = ExecutionPayloadV1::from_block_slow(&block);
        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // Test successful validation metrics
        assert!(handler.block_valid(&envelope).is_ok());

        // Test failed validation metrics
        let mut invalid_block = v1_valid_block();
        invalid_block.header.timestamp = 0; // Invalid timestamp

        let v1_invalid = ExecutionPayloadV1::from_block_slow(&invalid_block);
        let payload_invalid = OpExecutionPayload::V1(v1_invalid);
        let envelope_invalid = OpNetworkPayloadEnvelope {
            payload: payload_invalid,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        // This should increment failure metrics
        assert!(handler.block_valid(&envelope_invalid).is_err());
    }

    #[test]
    fn test_metrics_feature_gating() {
        // Verify the code compiles and runs without panics even when metrics feature is disabled
        let block = v1_valid_block();
        let v1 = ExecutionPayloadV1::from_block_slow(&block);
        let payload = OpExecutionPayload::V1(v1);
        let envelope = OpNetworkPayloadEnvelope {
            payload,
            signature: Signature::test_signature(),
            payload_hash: PayloadHash(B256::ZERO),
            parent_beacon_block_root: None,
        };

        let msg = envelope.payload_hash.signature_message(10);
        let signer = envelope.signature.recover_address_from_prehash(&msg).unwrap();
        let (_, unsafe_signer) = tokio::sync::watch::channel(signer);
        let mut handler = BlockHandler::new(
            RollupConfig { l2_chain_id: Chain::optimism_mainnet(), ..Default::default() },
            unsafe_signer,
        );

        // Should work regardless of metrics feature
        assert!(handler.block_valid(&envelope).is_ok());
    }
}
