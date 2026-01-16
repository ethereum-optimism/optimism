//! Interop [`MessageGraph`].

use crate::{
    MESSAGE_EXPIRY_WINDOW, RawMessagePayload,
    errors::{MessageGraphError, MessageGraphResult},
    message::{EnrichedExecutingMessage, extract_executing_messages},
    traits::InteropProvider,
};
use alloc::{string::ToString, vec::Vec};
use alloy_consensus::{Header, Sealed};
use alloy_primitives::keccak256;
use kona_genesis::RollupConfig;
use kona_registry::{HashMap, ROLLUP_CONFIGS};
use tracing::{info, warn};

/// The [`MessageGraph`] represents a set of blocks at a given timestamp and the interop
/// dependencies between them.
///
/// This structure is used to determine whether or not any interop messages are invalid within the
/// set of blocks within the graph. An "invalid message" is one that was relayed from one chain to
/// another, but the original [`MessageIdentifier`] is not present within the graph or from a
/// dependency referenced via the [`InteropProvider`] (or otherwise is invalid, such as being older
/// than the message expiry window).
///
/// Message validity rules: <https://specs.optimism.io/interop/messaging.html#invalid-messages>
///
/// [`MessageIdentifier`]: crate::MessageIdentifier
#[derive(Debug)]
pub struct MessageGraph<'a, P> {
    /// The edges within the graph.
    ///
    /// These are derived from the transactions within the blocks.
    messages: Vec<EnrichedExecutingMessage>,
    /// The data provider for the graph. Required for fetching headers, receipts and remote
    /// messages within history during resolution.
    provider: &'a P,
    /// Backup rollup configs for each chain.
    rollup_configs: &'a HashMap<u64, RollupConfig>,
}

impl<'a, P> MessageGraph<'a, P>
where
    P: InteropProvider,
{
    /// Derives the edges from the blocks within the graph by scanning all receipts within the
    /// blocks and searching for [`ExecutingMessage`]s.
    ///
    /// [`ExecutingMessage`]: crate::ExecutingMessage
    pub async fn derive(
        blocks: &HashMap<u64, Sealed<Header>>,
        provider: &'a P,
        rollup_configs: &'a HashMap<u64, RollupConfig>,
    ) -> MessageGraphResult<Self, P> {
        info!(
            target: "message_graph",
            num_chains = blocks.len(),
            "Deriving message graph",
        );

        let mut messages = Vec::with_capacity(blocks.len());
        for (chain_id, header) in blocks.iter() {
            let receipts = provider.receipts_by_hash(*chain_id, header.hash()).await?;
            let executing_messages = extract_executing_messages(receipts.as_slice());

            messages.extend(executing_messages.into_iter().map(|message| {
                EnrichedExecutingMessage::new(message, *chain_id, header.timestamp)
            }));
        }

        info!(
            target: "message_graph",
            num_chains = blocks.len(),
            num_messages = messages.len(),
            "Derived message graph successfully",
        );
        Ok(Self { messages, provider, rollup_configs })
    }

    /// Checks the validity of all messages within the graph.
    ///
    /// _Note_: This function does not account for cascading dependency failures. When
    /// [`MessageGraphError::InvalidMessages`] is returned by this function, the consumer must
    /// re-execute the bad blocks with deposit transactions only per the [interop derivation
    /// rules][int-block-replacement]. Once the bad blocks have been replaced, a new
    /// [`MessageGraph`] should be constructed and resolution should be re-attempted.
    /// This process should repeat recursively until no invalid dependencies remain, with the
    /// terminal case being all blocks reduced to deposits-only.
    ///
    /// [int-block-replacement]: https://specs.optimism.io/interop/derivation.html#replacing-invalid-blocks
    pub async fn resolve(self) -> MessageGraphResult<(), P> {
        info!(
            target: "message_graph",
            "Checking the message graph for invalid messages"
        );

        // Create a new vector to store invalid edges
        let mut invalid_messages = HashMap::default();

        // Prune all valid messages, collecting errors for any chain whose block contains an invalid
        // message. Errors are de-duplicated by chain ID in a map, since a single invalid
        // message is cause for invalidating a block.
        for message in self.messages.iter() {
            if let Err(e) = self.check_single_dependency(message).await {
                warn!(
                    target: "message_graph",
                    executing_chain_id = message.executing_chain_id,
                    message_hash = ?message.inner.payloadHash,
                    err = %e,
                    "Invalid ExecutingMessage found",
                );
                invalid_messages.insert(message.executing_chain_id, e);
            }
        }

        info!(
            target: "message_graph",
            num_invalid_messages = invalid_messages.len(),
            "Successfully reduced the message graph",
        );

        // Check if the graph is now empty. If not, there are invalid messages.
        if !invalid_messages.is_empty() {
            warn!(
                target: "message_graph",
                bad_chain_ids = %invalid_messages
                    .keys()
                    .map(ToString::to_string)
                    .collect::<Vec<_>>()
                    .join(", "),
                "Failed to reduce the message graph entirely",
            );

            // Return an error with the chain IDs of the blocks containing invalid messages.
            return Err(MessageGraphError::InvalidMessages(invalid_messages));
        }

        Ok(())
    }

    /// Checks the dependency of a single [`EnrichedExecutingMessage`]. If the message's
    /// dependencies are unavailable, the message is considered invalid and an [`Err`] is
    /// returned.
    async fn check_single_dependency(
        &self,
        message: &EnrichedExecutingMessage,
    ) -> MessageGraphResult<(), P> {
        // ChainID Invariant: The chain id of the initiating message MUST be in the dependency set
        // This is enforced implicitly by the graph constructor and the provider.

        let initiating_chain_id = message.inner.identifier.chainId.saturating_to();
        let initiating_timestamp = message.inner.identifier.timestamp.saturating_to::<u64>();

        // Attempt to fetch the rollup config for the initiating chain from the registry. If the
        // rollup config is not found, fall back to the local rollup configs.
        let rollup_config = ROLLUP_CONFIGS
            .get(&initiating_chain_id)
            .or_else(|| self.rollup_configs.get(&initiating_chain_id))
            .ok_or(MessageGraphError::MissingRollupConfig(initiating_chain_id))?;

        // Timestamp invariant: The timestamp at the time of inclusion of the initiating message
        // MUST be less than or equal to the timestamp of the executing message as well as greater
        // than the Interop activation block's timestamp.
        if initiating_timestamp > message.executing_timestamp {
            return Err(MessageGraphError::MessageInFuture {
                max: message.executing_timestamp,
                actual: initiating_timestamp,
            });
        } else if initiating_timestamp <
            rollup_config.hardforks.interop_time.unwrap_or_default() + rollup_config.block_time
        {
            return Err(MessageGraphError::InitiatedTooEarly {
                activation_time: rollup_config.hardforks.interop_time.unwrap_or_default(),
                initiating_message_time: initiating_timestamp,
            });
        }

        // Message expiry invariant: The timestamp of the initiating message must be no more than
        // `MESSAGE_EXPIRY_WINDOW` seconds in the past, relative to the timestamp of the executing
        // message.
        if initiating_timestamp < message.executing_timestamp.saturating_sub(MESSAGE_EXPIRY_WINDOW)
        {
            return Err(MessageGraphError::MessageExpired {
                initiating_timestamp,
                executing_timestamp: message.executing_timestamp,
            });
        }

        // Fetch the header & receipts for the message's claimed origin block on the remote chain.
        let remote_header = self
            .provider
            .header_by_number(
                message.inner.identifier.chainId.saturating_to(),
                message.inner.identifier.blockNumber.saturating_to(),
            )
            .await?;
        let remote_receipts = self
            .provider
            .receipts_by_number(
                message.inner.identifier.chainId.saturating_to(),
                message.inner.identifier.blockNumber.saturating_to(),
            )
            .await?;

        // Find the log that matches the message's claimed log index. Note that the
        // log index is global to the block, so we chain the full block's logs together
        // to find it.
        let remote_log = remote_receipts
            .iter()
            .flat_map(|receipt| receipt.logs())
            .nth(message.inner.identifier.logIndex.saturating_to())
            .ok_or(MessageGraphError::RemoteMessageNotFound {
                chain_id: message.inner.identifier.chainId.to(),
                message_hash: message.inner.payloadHash,
            })?;

        // Validate the message's origin is correct.
        if remote_log.address != message.inner.identifier.origin {
            return Err(MessageGraphError::InvalidMessageOrigin {
                expected: message.inner.identifier.origin,
                actual: remote_log.address,
            });
        }

        // Validate that the message hash is correct.
        let remote_message = RawMessagePayload::from(remote_log);
        let remote_message_hash = keccak256(remote_message.as_ref());
        if remote_message_hash != message.inner.payloadHash {
            return Err(MessageGraphError::InvalidMessageHash {
                expected: message.inner.payloadHash,
                actual: remote_message_hash,
            });
        }

        // Validate that the timestamp of the block header containing the log is correct.
        if remote_header.timestamp != initiating_timestamp {
            return Err(MessageGraphError::InvalidMessageTimestamp {
                expected: initiating_timestamp,
                actual: remote_header.timestamp,
            });
        }

        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::{MESSAGE_EXPIRY_WINDOW, MessageGraph};
    use crate::{
        MessageGraphError,
        test_util::{ExecutingMessageBuilder, SuperchainBuilder},
    };
    use alloy_primitives::{Address, hex, keccak256};

    const MOCK_MESSAGE: [u8; 4] = hex!("deadbeef");
    const CHAIN_A_ID: u64 = 1;
    const CHAIN_B_ID: u64 = 2;

    /// Returns a [`SuperchainBuilder`] with two chains (ids: `CHAIN_A_ID` and `CHAIN_B_ID`),
    /// configured with interop activating at timestamp `0`, the current block at timestamp `2`,
    /// and a block time of `2` seconds.
    fn default_superchain() -> SuperchainBuilder {
        let mut superchain = SuperchainBuilder::new();
        superchain
            .chain(CHAIN_A_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);

        superchain
    }

    #[tokio::test]
    async fn test_derive_and_resolve_simple_graph_no_cycles() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        graph.resolve().await.unwrap();
    }

    #[tokio::test]
    async fn test_derive_and_resolve_simple_graph_with_cycles() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;
        let chain_b_time = superchain.chain(CHAIN_B_ID).header.timestamp;

        superchain
            .chain(CHAIN_A_ID)
            .add_initiating_message(MOCK_MESSAGE.into())
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_B_ID)
                    .with_origin_timestamp(chain_b_time),
            );
        superchain
            .chain(CHAIN_B_ID)
            .add_initiating_message(MOCK_MESSAGE.into())
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_A_ID)
                    .with_origin_timestamp(chain_a_time),
            );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        graph.resolve().await.unwrap();
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_in_future() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time + 1),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::MessageInFuture { max: 2, actual: chain_a_time + 1 }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_before_interop() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(50)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly {
                activation_time: 50,
                initiating_message_time: chain_a_time
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_before_interop_unaligned_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        // Chain A activates @ `1s`, which is unaligned with the block time of `2s`. The first
        // block, at `2s`, should be the activation block.
        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(1)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly {
                activation_time: 1,
                initiating_message_time: chain_a_time
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_at_interop_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(chain_a_time)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly { activation_time: 2, initiating_message_time: 2 }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_expired() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(chain_a_time + MESSAGE_EXPIRY_WINDOW + 1)
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_A_ID)
                    .with_origin_timestamp(chain_a_time),
            );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::MessageExpired {
                initiating_timestamp: chain_a_time,
                executing_timestamp: chain_a_time + MESSAGE_EXPIRY_WINDOW + 1
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_remote_message_not_found() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::RemoteMessageNotFound {
                chain_id: CHAIN_A_ID,
                message_hash: keccak256(MOCK_MESSAGE)
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_origin_address() {
        let mut superchain = default_superchain();
        let mock_address = Address::left_padding_from(&[0xFF]);

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_address(mock_address)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageOrigin {
                expected: mock_address,
                actual: Address::ZERO
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_message_hash() {
        let mut superchain = default_superchain();
        let mock_message_hash = keccak256([0xBE, 0xEF]);

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(mock_message_hash)
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageHash {
                expected: mock_message_hash,
                actual: keccak256(MOCK_MESSAGE)
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_timestamp() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).with_timestamp(4).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).with_timestamp(4).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time - 1),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(&headers, &provider, &cfgs).await.unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageTimestamp {
                expected: chain_a_time - 1,
                actual: chain_a_time
            }
        );
    }
}
