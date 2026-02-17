use alloy_primitives::{Address, B256, Bytes, Log, keccak256};

/// Computes the log hash from a payload hash and log address.
///
/// This is done by:
/// 1. Concatenating the raw 20-byte address with the 32-byte payload hash,
/// 2. Hashing the result with keccak256.
///
/// This log hash is stored in the log storage and is used to map
/// an executing message back to the original initiating log.
pub fn payload_hash_to_log_hash(payload_hash: B256, addr: Address) -> B256 {
    let mut buf = Vec::with_capacity(64);
    buf.extend_from_slice(addr.as_slice()); // 20 bytes
    buf.extend_from_slice(payload_hash.as_slice()); // 32 bytes
    keccak256(&buf)
}

/// Converts an L2 log into its raw message payload for hashing.
///
/// This payload is defined as the concatenation of all log topics followed by the log data,
/// in accordance with the OP stack interop messaging spec.
///
/// This data is what is hashed to produce the `payloadHash`.
pub fn log_to_message_payload(log: &Log) -> Bytes {
    let mut payload = Vec::with_capacity(log.topics().len() * 32 + log.data.data.len());

    // Append each topic in order
    for topic in log.topics() {
        payload.extend_from_slice(topic.as_slice());
    }

    // Append the raw data
    payload.extend_from_slice(&log.data.data);

    payload.into()
}

/// Computes the full log hash from a log using the OP Stack convention.
pub fn log_to_log_hash(log: &Log) -> B256 {
    let payload = log_to_message_payload(log);
    let payload_hash = keccak256(&payload);
    payload_hash_to_log_hash(payload_hash, log.address)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, Log, address, b256};

    /// Creates a dummy log with fixed topics and data for testing.
    fn sample_log() -> Log {
        Log::new_unchecked(
            address!("0xe0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9"),
            vec![
                b256!("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
                b256!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
            ],
            Bytes::from_static(b"example payload"),
        )
    }

    #[test]
    fn test_log_to_message_payload_is_correct() {
        let log = sample_log();
        let payload = log_to_message_payload(&log);

        // Expect: topics + data
        let mut expected = Vec::new();
        expected.extend_from_slice(&log.topics()[0].0);
        expected.extend_from_slice(&log.topics()[1].0);
        expected.extend_from_slice(&log.data.data);

        assert_eq!(payload.as_ref(), expected.as_slice());
    }

    #[test]
    fn test_payload_hash_to_log_hash_with_known_value() {
        let address = address!("0xe0e1e2e3e4e5e6e7e8e9f0f1f2f3f4f5f6f7f8f9");
        let payload_hash = keccak256(Bytes::from_static(b"example payload"));
        let log_hash = payload_hash_to_log_hash(payload_hash, address);
        let expected_hash =
            b256!("0xf9ed05990c887d3f86718aabd7e940faaa75d6a5cd44602e89642586ce85f2aa");

        assert_eq!(log_hash, expected_hash);
    }

    #[test]
    fn test_log_to_log_hash_with_known_value() {
        let log = sample_log();
        let actual_log_hash = log_to_log_hash(&log);
        let expected_log_hash =
            b256!("0x20b21f284fb0286571fbf1cbfc20cdb1d50ea5c74c914478aee4a47b0a82a170");
        assert_eq!(actual_log_hash, expected_log_hash);
    }
}
