//! Contains deposit transaction types and helper methods.

use alloc::vec::Vec;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, Log, TxKind, U256, b256};
use op_alloy_consensus::{TxDeposit, UserDepositSource};

/// Deposit log event abi signature.
pub const DEPOSIT_EVENT_ABI: &str = "TransactionDeposited(address,address,uint256,bytes)";

/// Deposit event abi hash.
///
/// This is the keccak256 hash of the deposit event ABI signature.
/// `keccak256("TransactionDeposited(address,address,uint256,bytes)")`
pub const DEPOSIT_EVENT_ABI_HASH: B256 =
    b256!("b3813568d9991fc951961fcb4c784893574240a28925604d09fc577c55bb7c32");

/// The initial version of the deposit event log.
pub const DEPOSIT_EVENT_VERSION_0: B256 = B256::ZERO;

/// An [`TxDeposit`] validation error.
#[derive(Debug, thiserror::Error, PartialEq, Eq)]
pub enum DepositError {
    /// Unexpected number of deposit event log topics.
    #[error("Unexpected number of deposit event log topics: {0}")]
    UnexpectedTopicsLen(usize),
    /// Invalid deposit event selector.
    /// Expected: [`B256`] (deposit event selector), Actual: [`B256`] (event log topic).
    #[error("Invalid deposit event selector: {1}, expected {0}")]
    InvalidSelector(B256, B256),
    /// Incomplete opaqueData slice header (incomplete length).
    #[error("Incomplete opaqueData slice header (incomplete length): {0}")]
    IncompleteOpaqueData(usize),
    /// The log data is not aligned to 32 bytes.
    #[error("Unaligned log data, expected multiple of 32 bytes, got: {0}")]
    UnalignedData(usize),
    /// Failed to decode the `from` field of the deposit event (the second topic).
    #[error("Failed to decode the `from` address of the deposit log topic: {0}")]
    FromDecode(B256),
    /// Failed to decode the `to` field of the deposit event (the third topic).
    #[error("Failed to decode the `to` address of the deposit log topic: {0}")]
    ToDecode(B256),
    /// Invalid opaque data content offset.
    #[error("Invalid u64 opaque data content offset: {0}")]
    InvalidOpaqueDataOffset(Bytes),
    /// Invalid opaque data content length.
    #[error("Invalid u64 opaque data content length: expected {expected}, actual {actual}")]
    InvalidOpaqueDataLength {
        /// Expected length.
        expected: usize,
        /// Actual length.
        actual: usize,
    },
    /// Invalid opaque data.
    #[error("Invalid opaque data padding. Not all zeros or incorrect length: {0}")]
    InvalidOpaqueDataPadding(Bytes),
    /// Opaque content length overflow.
    #[error("Opaque content length overflow: {0}")]
    OpaqueContentOverflow(Bytes),
    /// Opaque data length exceeds the deposit log event data length.
    /// Specified: [usize] (data length), Actual: [usize] (opaque data length).
    #[error("Specified opaque data length {1} exceeds the deposit log event data length {0}")]
    OpaqueDataOverflow(u64, usize),
    /// Opaque data padding overflow.
    #[error("Opaque data padding overflow")]
    OpaqueDataPaddingOverflow,
    /// Opaque data with padding exceeds the specified data length.
    /// Specified: [usize] (data length), Actual: [usize] (opaque data length).
    #[error("Opaque data with padding exceeds the specified data length: {1} > {0}")]
    PaddedOpaqueDataOverflow(usize, u64),
    /// An invalid deposit version.
    #[error("Invalid deposit version: {0}")]
    InvalidVersion(B256),
    /// Unexpected opaque data length.
    #[error("Unexpected opaque data length: {0}")]
    UnexpectedOpaqueDataLen(usize),
    /// Failed to decode the deposit mint value.
    #[error("Failed to decode the u128 deposit mint value: {0}")]
    MintDecode(Bytes),
    /// Failed to decode the deposit gas value.
    #[error("Failed to decode the u64 deposit gas value: {0}")]
    GasDecode(Bytes),
}

/// Derives a deposit transaction from an EVM log event emitted by the deposit contract.
///
/// The emitted log must be in format:
/// ```solidity
/// event TransactionDeposited(
///    address indexed from,
///    address indexed to,
///    uint256 indexed version,
///    bytes opaqueData
/// );
/// ```
pub fn decode_deposit(block_hash: B256, index: usize, log: &Log) -> Result<Bytes, DepositError> {
    let topics = log.data.topics();
    if topics.len() != 4 {
        return Err(DepositError::UnexpectedTopicsLen(topics.len()));
    }
    if topics[0] != DEPOSIT_EVENT_ABI_HASH {
        return Err(DepositError::InvalidSelector(DEPOSIT_EVENT_ABI_HASH, topics[0]));
    }
    if log.data.data.len() < 64 {
        return Err(DepositError::IncompleteOpaqueData(log.data.data.len()));
    }
    if log.data.data.len() % 32 != 0 {
        return Err(DepositError::UnalignedData(log.data.data.len()));
    }

    // Validate the `from` address.
    let mut from_bytes = [0u8; 20];
    from_bytes.copy_from_slice(&topics[1].as_slice()[12..]);
    if topics[1].iter().take(12).any(|&b| b != 0) {
        return Err(DepositError::FromDecode(topics[1]));
    }

    // Validate the `to` address.
    let mut to_bytes = [0u8; 20];
    to_bytes.copy_from_slice(&topics[2].as_slice()[12..]);
    if topics[2].iter().take(12).any(|&b| b != 0) {
        return Err(DepositError::ToDecode(topics[2]));
    }

    let from = Address::from(from_bytes);
    let to = Address::from(to_bytes);
    let version = log.data.topics()[3];

    // Solidity serializes the event's Data field as follows:
    //
    // ```solidity
    // abi.encode(abi.encodPacked(uint256 mint, uint256 value, uint64 gasLimit, uint8 isCreation, bytes data))
    // ```
    //
    // The opaqueData will be packed as shown below:
    //
    // ------------------------------------------------------------
    // | offset | 256 byte content                                |
    // ------------------------------------------------------------
    // | 0      | [0; 24] . {U64 big endian, hex encoded offset}  |
    // ------------------------------------------------------------
    // | 32     | [0; 24] . {U64 big endian, hex encoded length}  |
    // ------------------------------------------------------------

    let opaque_content_offset: U256 = U256::from_be_slice(&log.data.data[0..32]);
    if opaque_content_offset != U256::from(32) {
        return Err(DepositError::InvalidOpaqueDataOffset(Bytes::copy_from_slice(
            &log.data.data[0..32],
        )));
    }

    // The next 32 bytes indicate the length of the opaqueData content.
    let opaque_content_len: U256 = U256::from_be_slice(&log.data.data[32..64]);
    let opaque_content_len: u64 = opaque_content_len.try_into().map_err(|_| {
        DepositError::OpaqueContentOverflow(Bytes::copy_from_slice(&log.data.data[32..64]))
    })?;

    let opaque_data_ceil_32: u64 = (opaque_content_len.saturating_add(31) / 32).saturating_mul(32);

    // Ensure that the remaining data is only zeros.
    // The padding ends at the next multiple of 32 after the opaque data.
    let Some(padding_end): Option<u64> = 64_u64.checked_add(opaque_data_ceil_32) else {
        return Err(DepositError::OpaqueDataPaddingOverflow);
    };

    // The remaining data is the opaqueData which is tightly packed and then padded to 32 bytes by
    // the EVM.
    let Some(opaque_data) = &log.data.data.get(64..64 + opaque_content_len as usize) else {
        return Err(DepositError::InvalidOpaqueDataLength {
            expected: opaque_content_len as usize,
            actual: log.data.data.len().saturating_sub(64),
        });
    };

    if !(opaque_content_len % 32 == 0 ||
        log.data
            .data
            .get((64 + opaque_content_len) as usize..padding_end as usize)
            .is_some_and(|data| data.iter().all(|&b| b == 0)))
    {
        return Err(DepositError::InvalidOpaqueDataPadding(Bytes::copy_from_slice(
            &log.data.data[(64 + opaque_content_len) as usize..],
        )));
    }

    let source = UserDepositSource::new(block_hash, index as u64);

    let mut deposit_tx = TxDeposit {
        from,
        is_system_transaction: false,
        source_hash: source.source_hash(),
        ..Default::default()
    };

    // Can only handle version 0 for now
    if !version.is_zero() {
        return Err(DepositError::InvalidVersion(version));
    }

    unmarshal_deposit_version0(&mut deposit_tx, to, opaque_data)?;

    // Re-encode the deposit transaction
    let mut buffer = Vec::with_capacity(deposit_tx.eip2718_encoded_length());
    deposit_tx.encode_2718(&mut buffer);
    Ok(Bytes::from(buffer))
}

/// Unmarshals a deposit transaction from the opaque data.
pub(crate) fn unmarshal_deposit_version0(
    tx: &mut TxDeposit,
    to: Address,
    data: &[u8],
) -> Result<(), DepositError> {
    if data.len() < 32 + 32 + 8 + 1 {
        return Err(DepositError::UnexpectedOpaqueDataLen(data.len()));
    }

    let mut offset = 0;

    let raw_mint: [u8; 16] = data[offset + 16..offset + 32].try_into().map_err(|_| {
        DepositError::MintDecode(Bytes::copy_from_slice(&data[offset + 16..offset + 32]))
    })?;
    tx.mint = u128::from_be_bytes(raw_mint);
    offset += 32;

    // uint256 value
    tx.value = U256::from_be_slice(&data[offset..offset + 32]);
    offset += 32;

    // uint64 gas
    let raw_gas: [u8; 8] = data[offset..offset + 8]
        .try_into()
        .map_err(|_| DepositError::GasDecode(Bytes::copy_from_slice(&data[offset..offset + 8])))?;
    tx.gas_limit = u64::from_be_bytes(raw_gas);
    offset += 8;

    // uint8 isCreation
    // isCreation: If the boolean byte is 1 then dep.To will stay nil,
    // and it will create a contract using L2 account nonce to determine the created address.
    if data[offset] == 0 {
        tx.to = TxKind::Call(to);
    } else {
        tx.to = TxKind::Create;
    }
    offset += 1;

    // The remainder of the opaqueData is the transaction data (without length prefix).
    // The data may be padded to a multiple of 32 bytes
    let tx_data_len = data.len() - offset;

    // Remaining bytes fill the data
    tx.input = Bytes::copy_from_slice(&data[offset..offset + tx_data_len]);

    Ok(())
}

#[cfg(test)]
mod test {
    use super::*;
    use alloc::vec;
    use alloy_primitives::{LogData, U64, U128, address, b256, hex};

    #[test]
    fn test_decode_deposit_invalid_first_topic() {
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![B256::default(), B256::default(), B256::default(), B256::default()],
                Bytes::default(),
            ),
        };
        let err: DepositError = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::InvalidSelector(DEPOSIT_EVENT_ABI_HASH, B256::default()));
    }

    #[test]
    fn test_decode_deposit_incomplete_data() {
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(vec![0u8; 63]),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::IncompleteOpaqueData(63));
    }

    #[test]
    fn test_decode_deposit_unaligned_data() {
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(vec![0u8; 65]),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::UnalignedData(65));
    }

    #[test]
    fn test_decode_deposit_invalid_from() {
        let invalid_from =
            b256!("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, invalid_from, B256::default(), B256::default()],
                Bytes::from(vec![0u8; 64]),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::FromDecode(invalid_from));
    }

    #[test]
    fn test_decode_deposit_invalid_to() {
        let invalid_to = b256!("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), invalid_to, B256::default()],
                Bytes::from(vec![0u8; 64]),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::ToDecode(invalid_to));
    }

    #[test]
    fn test_decode_deposit_invalid_opaque_data_offset() {
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(vec![0u8; 64]),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::InvalidOpaqueDataOffset(Bytes::from(vec![0u8; 32])));
    }

    #[test]
    fn test_decode_deposit_opaque_data_overflow() {
        let mut data = vec![0u8; 128];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        // The first 64 bytes of the data are identifiers so
        // if this test was to be valid, the data length would be 64 not 128.
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(data),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::InvalidOpaqueDataLength { expected: 128, actual: 64 });
    }

    #[test]
    fn test_decode_deposit_padded_overflow() {
        let mut data = vec![0u8; 256];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(64).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(data),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::UnexpectedOpaqueDataLen(64));
    }

    #[test]
    fn test_decode_deposit_invalid_version() {
        let mut data = vec![0u8; 128];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(64).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        let version = b256!("0000000000000000000000000000000000000000000000000000000000000001");
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), version],
                Bytes::from(data),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::InvalidVersion(version));
    }

    #[test]
    fn test_decode_deposit_empty_succeeds() {
        let valid_to = b256!("000000000000000000000000bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
        let valid_from = b256!("000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, valid_from, valid_to, B256::default()],
                Bytes::from(data),
            ),
        };
        let tx = decode_deposit(B256::default(), 0, &log).unwrap();
        let raw_hex = hex!(
            "7ef887a0ed428e1c45e1d9561b62834e1a2d3015a0caae3bfdc16b4da059ac885b01a14594ffffffffffffffffffffffffffffffffffffffff94bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb80808080b700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
        );
        let expected = Bytes::from(raw_hex);
        assert_eq!(tx, expected);
    }

    #[test]
    fn test_decode_deposit_invalid_offset() {
        let mut data = vec![0u8; 128];
        let offset: [u8; 16] = U128::MAX.to_be_bytes();
        data[16..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(data.clone()),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        let bytes = Bytes::from(data.get(0..32).unwrap().to_vec());
        assert_eq!(err, DepositError::InvalidOpaqueDataOffset(bytes));
    }

    #[test]
    fn test_decode_deposit_invalid_length() {
        let mut data = vec![0u8; 128];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 16] = U128::MAX.to_be_bytes();
        data[48..64].copy_from_slice(&len);
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![DEPOSIT_EVENT_ABI_HASH, B256::default(), B256::default(), B256::default()],
                Bytes::from(data.clone()),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(
            err,
            DepositError::OpaqueContentOverflow(Bytes::from(data.get(32..64).unwrap().to_vec()))
        );
    }

    #[test]
    fn test_invalid_opaque_data_length() {
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(129).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        let from = address!("1111111111111111111111111111111111111111");
        let mut from_bytes = vec![0u8; 32];
        from_bytes[12..32].copy_from_slice(from.as_slice());
        let to = address!("2222222222222222222222222222222222222222");
        let mut to_bytes = vec![0u8; 32];
        to_bytes[12..32].copy_from_slice(to.as_slice());
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![
                    DEPOSIT_EVENT_ABI_HASH,
                    B256::from_slice(&from_bytes),
                    B256::from_slice(&to_bytes),
                    B256::default(),
                ],
                Bytes::from(data),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::InvalidOpaqueDataLength { expected: 129, actual: 128 });
    }

    #[test]
    fn test_opaque_data_padding_overflow() {
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::MAX.to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        let from = address!("1111111111111111111111111111111111111111");
        let mut from_bytes = vec![0u8; 32];
        from_bytes[12..32].copy_from_slice(from.as_slice());
        let to = address!("2222222222222222222222222222222222222222");
        let mut to_bytes = vec![0u8; 32];
        to_bytes[12..32].copy_from_slice(to.as_slice());
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![
                    DEPOSIT_EVENT_ABI_HASH,
                    B256::from_slice(&from_bytes),
                    B256::from_slice(&to_bytes),
                    B256::default(),
                ],
                Bytes::from(data.clone()),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(err, DepositError::OpaqueDataPaddingOverflow);
    }

    #[test]
    fn test_invalid_opaque_data_padding() {
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(127).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        // Mess up the padding
        data[191] = 1;
        let from = address!("1111111111111111111111111111111111111111");
        let mut from_bytes = vec![0u8; 32];
        from_bytes[12..32].copy_from_slice(from.as_slice());
        let to = address!("2222222222222222222222222222222222222222");
        let mut to_bytes = vec![0u8; 32];
        to_bytes[12..32].copy_from_slice(to.as_slice());
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![
                    DEPOSIT_EVENT_ABI_HASH,
                    B256::from_slice(&from_bytes),
                    B256::from_slice(&to_bytes),
                    B256::default(),
                ],
                Bytes::from(data.clone()),
            ),
        };
        let err = decode_deposit(B256::default(), 0, &log).unwrap_err();
        assert_eq!(
            err,
            DepositError::InvalidOpaqueDataPadding(Bytes::from(data.get(191..).unwrap().to_vec()))
        );
    }

    #[test]
    fn test_decode_deposit_full_succeeds() {
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        let from = address!("1111111111111111111111111111111111111111");
        let mut from_bytes = vec![0u8; 32];
        from_bytes[12..32].copy_from_slice(from.as_slice());
        let to = address!("2222222222222222222222222222222222222222");
        let mut to_bytes = vec![0u8; 32];
        to_bytes[12..32].copy_from_slice(to.as_slice());
        let log = Log {
            address: Address::default(),
            data: LogData::new_unchecked(
                vec![
                    DEPOSIT_EVENT_ABI_HASH,
                    B256::from_slice(&from_bytes),
                    B256::from_slice(&to_bytes),
                    B256::default(),
                ],
                Bytes::from(data),
            ),
        };
        let tx = decode_deposit(B256::default(), 0, &log).unwrap();
        let raw_hex = hex!(
            "7ef875a0ed428e1c45e1d9561b62834e1a2d3015a0caae3bfdc16b4da059ac885b01a145941111111111111111111111111111111111111111800a648203e880b700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
        );
        let expected = Bytes::from(raw_hex);
        assert_eq!(tx, expected);
    }

    #[test]
    fn test_unmarshal_deposit_version0_invalid_len() {
        let data = vec![0u8; 72];
        let mut tx = TxDeposit::default();
        let to = address!("5555555555555555555555555555555555555555");
        let err = unmarshal_deposit_version0(&mut tx, to, &data).unwrap_err();
        assert_eq!(err, DepositError::UnexpectedOpaqueDataLen(72));

        // Data must have at least length 73
        let data = vec![0u8; 73];
        let mut tx = TxDeposit::default();
        let to = address!("5555555555555555555555555555555555555555");
        unmarshal_deposit_version0(&mut tx, to, &data).unwrap();
    }

    #[test]
    fn test_unmarshal_deposit_version0() {
        let mut data = vec![0u8; 192];
        let offset: [u8; 8] = U64::from(32).to_be_bytes();
        data[24..32].copy_from_slice(&offset);
        let len: [u8; 8] = U64::from(128).to_be_bytes();
        data[56..64].copy_from_slice(&len);
        // Copy the u128 mint value
        let mint: [u8; 16] = 10_u128.to_be_bytes();
        data[80..96].copy_from_slice(&mint);
        // Copy the tx value
        let value: [u8; 32] = U256::from(100).to_be_bytes();
        data[96..128].copy_from_slice(&value);
        // Copy the gas limit
        let gas: [u8; 8] = 1000_u64.to_be_bytes();
        data[128..136].copy_from_slice(&gas);
        // Copy the isCreation flag
        data[136] = 1;
        let mut tx = TxDeposit {
            from: address!("1111111111111111111111111111111111111111"),
            to: TxKind::Call(address!("2222222222222222222222222222222222222222")),
            value: U256::from(100),
            gas_limit: 1000,
            mint: 10,
            ..Default::default()
        };
        let to = address!("5555555555555555555555555555555555555555");
        unmarshal_deposit_version0(&mut tx, to, &data).unwrap();
        assert_eq!(tx.to, TxKind::Call(address!("5555555555555555555555555555555555555555")));
    }
}
