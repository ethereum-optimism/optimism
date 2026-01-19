//! Interop message primitives.
//!
//! <https://specs.optimism.io/interop/messaging.html#messaging>
//! <https://github.com/ethereum-optimism/optimism/blob/34d5f66ade24bd1f3ce4ce7c0a6cfc1a6540eca1/packages/contracts-bedrock/src/L2/CrossL2Inbox.sol>

use alloc::{vec, vec::Vec};
use alloy_primitives::{Bytes, ChainId, Log, keccak256};
use alloy_sol_types::{SolEvent, sol};
use derive_more::{AsRef, Constructor, From};
use kona_protocol::Predeploys;
use op_alloy_consensus::OpReceiptEnvelope;

sol! {
    /// @notice The struct for a pointer to a message payload in a remote (or local) chain.
    #[derive(Default, Debug, PartialEq, Eq)]
    #[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
    struct MessageIdentifier {
        address origin;
        uint256 blockNumber;
        uint256 logIndex;
        uint256 timestamp;
        #[cfg_attr(feature = "serde", serde(rename = "chainID"))]
        uint256 chainId;
    }

    /// @notice Emitted when a cross chain message is being executed.
    /// @param payloadHash Hash of message payload being executed.
    /// @param identifier Encoded Identifier of the message.
    ///
    /// Parameter names are derived from the `op-supervisor` JSON field names.
    /// See the relevant definition in the Optimism repository:
    /// [Ethereum-Optimism/op-supervisor](https://github.com/ethereum-optimism/optimism/blob/4ba2eb00eafc3d7de2c8ceb6fd83913a8c0a2c0d/op-supervisor/supervisor/types/types.go#L61-L64).
    #[derive(Default, Debug, PartialEq, Eq)]
    #[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
    event ExecutingMessage(bytes32 indexed payloadHash, MessageIdentifier identifier);

    /// @notice Executes a cross chain message on the destination chain.
    /// @param _id      Identifier of the message.
    /// @param _target  Target address to call.
    /// @param _message Message payload to call target with.
    function executeMessage(
        MessageIdentifier calldata _id,
        address _target,
        bytes calldata _message
    ) external;
}

/// A [RawMessagePayload] is the raw payload of an initiating message.
#[derive(Debug, Clone, From, AsRef, PartialEq, Eq)]
pub struct RawMessagePayload(Bytes);

impl From<&Log> for RawMessagePayload {
    fn from(log: &Log) -> Self {
        let mut data = vec![0u8; log.topics().len() * 32 + log.data.data.len()];
        for (i, topic) in log.topics().iter().enumerate() {
            data[i * 32..(i + 1) * 32].copy_from_slice(topic.as_ref());
        }
        data[(log.topics().len() * 32)..].copy_from_slice(log.data.data.as_ref());
        data.into()
    }
}

impl From<Vec<u8>> for RawMessagePayload {
    fn from(data: Vec<u8>) -> Self {
        Self(Bytes::from(data))
    }
}

impl From<executeMessageCall> for ExecutingMessage {
    fn from(call: executeMessageCall) -> Self {
        Self { identifier: call._id, payloadHash: keccak256(call._message.as_ref()) }
    }
}

/// An [`ExecutingDescriptor`] is a part of the payload to `supervisor_checkAccessList`
/// Spec: <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#executingdescriptor>
#[derive(Default, Debug, PartialEq, Eq, Clone, Constructor)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct ExecutingDescriptor {
    /// The timestamp used to enforce timestamp [invariant](https://github.com/ethereum-optimism/specs/blob/main/specs/interop/derivation.md#invariants)
    #[cfg_attr(feature = "serde", serde(with = "alloy_serde::quantity"))]
    pub timestamp: u64,
    /// The timeout that requests verification to still hold at `timestamp+timeout`
    /// (message expiry may drop previously valid messages).
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub timeout: Option<u64>,
    /// Chain ID of the chain that the message was executed on.
    #[cfg_attr(
        feature = "serde",
        serde(
            default,
            rename = "chainID",
            skip_serializing_if = "Option::is_none",
            with = "alloy_serde::quantity::opt"
        )
    )]
    pub chain_id: Option<ChainId>,
}

/// A wrapper type for [ExecutingMessage] containing the chain ID of the chain that the message was
/// executed on.
#[derive(Debug)]
pub struct EnrichedExecutingMessage {
    /// The inner [ExecutingMessage].
    pub inner: ExecutingMessage,
    /// The chain ID of the chain that the message was executed on.
    pub executing_chain_id: u64,
    /// The timestamp of the block that the executing message was included in.
    pub executing_timestamp: u64,
}

impl EnrichedExecutingMessage {
    /// Create a new [EnrichedExecutingMessage] from an [ExecutingMessage] and a chain ID.
    pub const fn new(
        inner: ExecutingMessage,
        executing_chain_id: u64,
        executing_timestamp: u64,
    ) -> Self {
        Self { inner, executing_chain_id, executing_timestamp }
    }
}

/// Extracts all [ExecutingMessage] events from list of [OpReceiptEnvelope]s.
///
/// See [`parse_log_to_executing_message`].
///
/// Note: filters out logs that don't contain executing message events.
pub fn extract_executing_messages(receipts: &[OpReceiptEnvelope]) -> Vec<ExecutingMessage> {
    receipts.iter().fold(Vec::new(), |mut acc, envelope| {
        let executing_messages = envelope.logs().iter().filter_map(parse_log_to_executing_message);

        acc.extend(executing_messages);
        acc
    })
}

/// Parses [`Log`]s to [`ExecutingMessage`]s.
///
/// See [`parse_log_to_executing_message`] for more details. Return iterator maps 1-1 with input.
pub fn parse_logs_to_executing_msgs<'a>(
    logs: impl Iterator<Item = &'a Log>,
) -> impl Iterator<Item = Option<ExecutingMessage>> {
    logs.map(parse_log_to_executing_message)
}

/// Parse [`Log`] to [`ExecutingMessage`], if any.
///
/// Max one [`ExecutingMessage`] event can exist per log. Returns `None` if log doesn't contain
/// executing message event.
pub fn parse_log_to_executing_message(log: &Log) -> Option<ExecutingMessage> {
    (log.address == Predeploys::CROSS_L2_INBOX && log.topics().len() == 2)
        .then(|| ExecutingMessage::decode_log_data(&log.data).ok())
        .flatten()
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, B256, LogData, U256};

    use super::*;

    // Test the serialization of ExecutingDescriptor
    #[cfg(feature = "serde")]
    #[test]
    fn test_serialize_executing_descriptor() {
        let descriptor = ExecutingDescriptor {
            timestamp: 1234567890,
            timeout: Some(3600),
            chain_id: Some(1000),
        };
        let serialized = serde_json::to_string(&descriptor).unwrap();
        let expected = r#"{"timestamp":"0x499602d2","timeout":"0xe10","chainID":"0x3e8"}"#;
        assert_eq!(serialized, expected);

        let deserialized: ExecutingDescriptor = serde_json::from_str(&serialized).unwrap();
        assert_eq!(descriptor, deserialized);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn test_deserialize_executing_descriptor_missing_chain_id() {
        let json = r#"{
            "timestamp": "0x499602d2",
            "timeout": "0xe10"
        }"#;

        let expected =
            ExecutingDescriptor { timestamp: 1234567890, timeout: Some(3600), chain_id: None };

        let deserialized: ExecutingDescriptor = serde_json::from_str(json).unwrap();
        assert_eq!(deserialized, expected);
    }

    #[cfg(feature = "serde")]
    #[test]
    fn test_deserialize_executing_descriptor_missing_timeout() {
        let json = r#"{
            "timestamp": "0x499602d2",
            "chainID": "0x3e8"
        }"#;

        let expected =
            ExecutingDescriptor { timestamp: 1234567890, timeout: None, chain_id: Some(1000) };

        let deserialized: ExecutingDescriptor = serde_json::from_str(json).unwrap();
        assert_eq!(deserialized, expected);
    }

    #[test]
    fn test_parse_logs_to_executing_msgs_iterator() {
        // One valid, one invalid log
        let identifier = MessageIdentifier {
            origin: Address::repeat_byte(0x77),
            blockNumber: U256::from(200),
            logIndex: U256::from(3),
            timestamp: U256::from(777777),
            chainId: U256::from(12),
        };
        let payload_hash = B256::repeat_byte(0x88);
        let event = ExecutingMessage { payloadHash: payload_hash, identifier };
        let data = ExecutingMessage::encode_log_data(&event);

        let valid_log = Log { address: Predeploys::CROSS_L2_INBOX, data };
        let invalid_log = Log {
            address: Address::repeat_byte(0x99),
            data: LogData::new_unchecked([B256::ZERO, B256::ZERO].to_vec(), Bytes::default()),
        };

        let logs = vec![&valid_log, &invalid_log];
        let mut iter = parse_logs_to_executing_msgs(logs.into_iter());
        assert_eq!(iter.next().unwrap().unwrap(), event);
        assert!(iter.next().unwrap().is_none());
    }
}
