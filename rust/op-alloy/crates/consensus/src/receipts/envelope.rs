//! Receipt envelope types for Optimism.

use crate::{OpDepositReceipt, OpDepositReceiptWithBloom, OpTxType};
use alloc::vec::Vec;
use alloy_consensus::{Eip658Value, Receipt, ReceiptWithBloom, TxReceipt};
use alloy_eips::{
    Typed2718,
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718, IsTyped2718},
};
use alloy_primitives::{Bloom, Log, logs_bloom};
use alloy_rlp::{BufMut, Decodable, Encodable, length_of_length};

/// Receipt envelope, as defined in [EIP-2718], modified for OP Stack chains.
///
/// This enum distinguishes between tagged and untagged legacy receipts, as the
/// in-protocol merkle tree may commit to EITHER 0-prefixed or raw. Therefore
/// we must ensure that encoding returns the precise byte-array that was
/// decoded, preserving the presence or absence of the `TransactionType` flag.
///
/// Transaction receipt payloads are specified in their respective EIPs.
///
/// [EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(tag = "type"))]
pub enum OpReceiptEnvelope<T = Log> {
    /// Receipt envelope with no type flag.
    #[cfg_attr(feature = "serde", serde(rename = "0x0", alias = "0x00"))]
    Legacy(ReceiptWithBloom<Receipt<T>>),
    /// Receipt envelope with type flag 1, containing a [EIP-2930] receipt.
    ///
    /// [EIP-2930]: https://eips.ethereum.org/EIPS/eip-2930
    #[cfg_attr(feature = "serde", serde(rename = "0x1", alias = "0x01"))]
    Eip2930(ReceiptWithBloom<Receipt<T>>),
    /// Receipt envelope with type flag 2, containing a [EIP-1559] receipt.
    ///
    /// [EIP-1559]: https://eips.ethereum.org/EIPS/eip-1559
    #[cfg_attr(feature = "serde", serde(rename = "0x2", alias = "0x02"))]
    Eip1559(ReceiptWithBloom<Receipt<T>>),
    /// Receipt envelope with type flag 4, containing a [EIP-7702] receipt.
    ///
    /// [EIP-7702]: https://eips.ethereum.org/EIPS/eip-7702
    #[cfg_attr(feature = "serde", serde(rename = "0x4", alias = "0x04"))]
    Eip7702(ReceiptWithBloom<Receipt<T>>),
    /// Receipt envelope with type flag 126, containing a [deposit] receipt.
    ///
    /// [deposit]: https://specs.optimism.io/protocol/deposits.html
    #[cfg_attr(feature = "serde", serde(rename = "0x7e", alias = "0x7E"))]
    Deposit(ReceiptWithBloom<OpDepositReceipt<T>>),
}

impl OpReceiptEnvelope<Log> {
    /// Creates a new [`OpReceiptEnvelope`] from the given parts.
    pub fn from_parts<'a>(
        status: bool,
        cumulative_gas_used: u64,
        logs: impl IntoIterator<Item = &'a Log>,
        tx_type: OpTxType,
        deposit_nonce: Option<u64>,
        deposit_receipt_version: Option<u64>,
    ) -> Self {
        let logs = logs.into_iter().cloned().collect::<Vec<_>>();
        let logs_bloom = logs_bloom(&logs);
        let inner_receipt =
            Receipt { status: Eip658Value::Eip658(status), cumulative_gas_used, logs };
        match tx_type {
            OpTxType::Legacy => {
                Self::Legacy(ReceiptWithBloom { receipt: inner_receipt, logs_bloom })
            }
            OpTxType::Eip2930 => {
                Self::Eip2930(ReceiptWithBloom { receipt: inner_receipt, logs_bloom })
            }
            OpTxType::Eip1559 => {
                Self::Eip1559(ReceiptWithBloom { receipt: inner_receipt, logs_bloom })
            }
            OpTxType::Eip7702 => {
                Self::Eip7702(ReceiptWithBloom { receipt: inner_receipt, logs_bloom })
            }
            OpTxType::Deposit => {
                let inner = OpDepositReceiptWithBloom {
                    receipt: OpDepositReceipt {
                        inner: inner_receipt,
                        deposit_nonce,
                        deposit_receipt_version,
                    },
                    logs_bloom,
                };
                Self::Deposit(inner)
            }
        }
    }
}

impl<T> OpReceiptEnvelope<T> {
    /// Return the [`OpTxType`] of the inner receipt.
    pub const fn tx_type(&self) -> OpTxType {
        match self {
            Self::Legacy(_) => OpTxType::Legacy,
            Self::Eip2930(_) => OpTxType::Eip2930,
            Self::Eip1559(_) => OpTxType::Eip1559,
            Self::Eip7702(_) => OpTxType::Eip7702,
            Self::Deposit(_) => OpTxType::Deposit,
        }
    }

    /// Return true if the transaction was successful.
    pub const fn is_success(&self) -> bool {
        self.status()
    }

    /// Returns the success status of the receipt's transaction.
    pub const fn status(&self) -> bool {
        self.as_receipt().unwrap().status.coerce_status()
    }

    /// Returns the cumulative gas used at this receipt.
    pub const fn cumulative_gas_used(&self) -> u64 {
        self.as_receipt().unwrap().cumulative_gas_used
    }

    /// Converts the receipt's log type by applying a function to each log.
    ///
    /// Returns the receipt with the new log type.
    pub fn map_logs<U>(self, f: impl FnMut(T) -> U) -> OpReceiptEnvelope<U> {
        match self {
            Self::Legacy(r) => OpReceiptEnvelope::Legacy(r.map_logs(f)),
            Self::Eip2930(r) => OpReceiptEnvelope::Eip2930(r.map_logs(f)),
            Self::Eip1559(r) => OpReceiptEnvelope::Eip1559(r.map_logs(f)),
            Self::Eip7702(r) => OpReceiptEnvelope::Eip7702(r.map_logs(f)),
            Self::Deposit(r) => OpReceiptEnvelope::Deposit(r.map_receipt(|r| r.map_logs(f))),
        }
    }

    /// Return the receipt logs.
    pub fn logs(&self) -> &[T] {
        &self.as_receipt().unwrap().logs
    }

    /// Consumes the type and returns the logs.
    pub fn into_logs(self) -> Vec<T> {
        self.into_receipt().logs
    }

    /// Return the receipt's bloom.
    pub const fn logs_bloom(&self) -> &Bloom {
        match self {
            Self::Legacy(t) => &t.logs_bloom,
            Self::Eip2930(t) => &t.logs_bloom,
            Self::Eip1559(t) => &t.logs_bloom,
            Self::Eip7702(t) => &t.logs_bloom,
            Self::Deposit(t) => &t.logs_bloom,
        }
    }

    /// Return the receipt's deposit_nonce if it is a deposit receipt.
    pub fn deposit_nonce(&self) -> Option<u64> {
        self.as_deposit_receipt().and_then(|r| r.deposit_nonce)
    }

    /// Return the receipt's deposit version if it is a deposit receipt.
    pub fn deposit_receipt_version(&self) -> Option<u64> {
        self.as_deposit_receipt().and_then(|r| r.deposit_receipt_version)
    }

    /// Returns the deposit receipt if it is a deposit receipt.
    pub const fn as_deposit_receipt_with_bloom(&self) -> Option<&OpDepositReceiptWithBloom<T>> {
        match self {
            Self::Deposit(t) => Some(t),
            _ => None,
        }
    }

    /// Returns the deposit receipt if it is a deposit receipt.
    pub const fn as_deposit_receipt(&self) -> Option<&OpDepositReceipt<T>> {
        match self {
            Self::Deposit(t) => Some(&t.receipt),
            _ => None,
        }
    }

    /// Consumes the type and returns the underlying [`Receipt`].
    pub fn into_receipt(self) -> Receipt<T> {
        match self {
            Self::Legacy(t) | Self::Eip2930(t) | Self::Eip1559(t) | Self::Eip7702(t) => t.receipt,
            Self::Deposit(t) => t.receipt.into_inner(),
        }
    }

    /// Return the inner receipt. Currently this is infallible, however, future
    /// receipt types may be added.
    pub const fn as_receipt(&self) -> Option<&Receipt<T>> {
        match self {
            Self::Legacy(t) | Self::Eip2930(t) | Self::Eip1559(t) | Self::Eip7702(t) => {
                Some(&t.receipt)
            }
            Self::Deposit(t) => Some(&t.receipt.inner),
        }
    }
}

impl OpReceiptEnvelope {
    /// Get the length of the inner receipt in the 2718 encoding.
    pub fn inner_length(&self) -> usize {
        match self {
            Self::Legacy(t) => t.length(),
            Self::Eip2930(t) => t.length(),
            Self::Eip1559(t) => t.length(),
            Self::Eip7702(t) => t.length(),
            Self::Deposit(t) => t.length(),
        }
    }

    /// Calculate the length of the rlp payload of the network encoded receipt.
    pub fn rlp_payload_length(&self) -> usize {
        let length = self.inner_length();
        match self {
            Self::Legacy(_) => length,
            _ => length + 1,
        }
    }
}

impl<T> TxReceipt for OpReceiptEnvelope<T>
where
    T: Clone + core::fmt::Debug + PartialEq + Eq + Send + Sync,
{
    type Log = T;

    fn status_or_post_state(&self) -> Eip658Value {
        self.as_receipt().unwrap().status
    }

    fn status(&self) -> bool {
        self.as_receipt().unwrap().status.coerce_status()
    }

    /// Return the receipt's bloom.
    fn bloom(&self) -> Bloom {
        *self.logs_bloom()
    }

    fn bloom_cheap(&self) -> Option<Bloom> {
        Some(self.bloom())
    }

    /// Returns the cumulative gas used at this receipt.
    fn cumulative_gas_used(&self) -> u64 {
        self.as_receipt().unwrap().cumulative_gas_used
    }

    /// Return the receipt logs.
    fn logs(&self) -> &[T] {
        &self.as_receipt().unwrap().logs
    }
}

impl Encodable for OpReceiptEnvelope {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        self.network_encode(out)
    }

    fn length(&self) -> usize {
        let mut payload_length = self.rlp_payload_length();
        if !self.is_legacy() {
            payload_length += length_of_length(payload_length);
        }
        payload_length
    }
}

impl Decodable for OpReceiptEnvelope {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::network_decode(buf)
            .map_or_else(|_| Err(alloy_rlp::Error::Custom("Unexpected type")), Ok)
    }
}

impl Typed2718 for OpReceiptEnvelope {
    fn ty(&self) -> u8 {
        let ty = match self {
            Self::Legacy(_) => OpTxType::Legacy,
            Self::Eip2930(_) => OpTxType::Eip2930,
            Self::Eip1559(_) => OpTxType::Eip1559,
            Self::Eip7702(_) => OpTxType::Eip7702,
            Self::Deposit(_) => OpTxType::Deposit,
        };
        ty as u8
    }
}

impl IsTyped2718 for OpReceiptEnvelope {
    fn is_type(type_id: u8) -> bool {
        <OpTxType as IsTyped2718>::is_type(type_id)
    }
}

impl Encodable2718 for OpReceiptEnvelope {
    fn encode_2718_len(&self) -> usize {
        self.inner_length() + !self.is_legacy() as usize
    }

    fn encode_2718(&self, out: &mut dyn BufMut) {
        match self.type_flag() {
            None => {}
            Some(ty) => out.put_u8(ty),
        }
        match self {
            Self::Deposit(t) => t.encode(out),
            Self::Legacy(t) | Self::Eip2930(t) | Self::Eip1559(t) | Self::Eip7702(t) => {
                t.encode(out)
            }
        }
    }
}

impl Decodable2718 for OpReceiptEnvelope {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> Eip2718Result<Self> {
        match ty.try_into().map_err(|_| Eip2718Error::UnexpectedType(ty))? {
            OpTxType::Legacy => {
                Err(alloy_rlp::Error::Custom("type-0 eip2718 transactions are not supported")
                    .into())
            }
            OpTxType::Eip1559 => Ok(Self::Eip1559(Decodable::decode(buf)?)),
            OpTxType::Eip7702 => Ok(Self::Eip7702(Decodable::decode(buf)?)),
            OpTxType::Eip2930 => Ok(Self::Eip2930(Decodable::decode(buf)?)),
            OpTxType::Deposit => Ok(Self::Deposit(Decodable::decode(buf)?)),
        }
    }

    fn fallback_decode(buf: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::Legacy(Decodable::decode(buf)?))
    }
}

impl<T> From<T> for OpReceiptEnvelope<T>
where
    T: Into<ReceiptWithBloom<OpDepositReceipt<T>>>,
{
    fn from(value: T) -> Self {
        Self::Deposit(value.into())
    }
}

impl<T> From<OpReceiptEnvelope<T>> for Receipt<T> {
    fn from(receipt: OpReceiptEnvelope<T>) -> Self {
        receipt.into_receipt()
    }
}

#[cfg(all(test, feature = "arbitrary"))]
impl<'a, T> arbitrary::Arbitrary<'a> for OpReceiptEnvelope<T>
where
    T: arbitrary::Arbitrary<'a>,
{
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        match u.int_in_range(0..=4)? {
            0 => Ok(Self::Legacy(ReceiptWithBloom::arbitrary(u)?)),
            1 => Ok(Self::Eip2930(ReceiptWithBloom::arbitrary(u)?)),
            2 => Ok(Self::Eip1559(ReceiptWithBloom::arbitrary(u)?)),
            _ => Ok(Self::Deposit(OpDepositReceiptWithBloom::arbitrary(u)?)),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Receipt, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Log, LogData, address, b256, bytes, hex};
    use alloy_rlp::Encodable;

    #[cfg(not(feature = "std"))]
    use alloc::vec;

    // Test vector from: https://eips.ethereum.org/EIPS/eip-2481
    #[test]
    fn encode_legacy_receipt() {
        let expected = hex!(
            "f901668001b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000f85ff85d940000000000000000000000000000000000000011f842a0000000000000000000000000000000000000000000000000000000000000deada0000000000000000000000000000000000000000000000000000000000000beef830100ff"
        );

        let mut data = vec![];
        let receipt = OpReceiptEnvelope::Legacy(ReceiptWithBloom {
            receipt: Receipt {
                status: false.into(),
                cumulative_gas_used: 0x1,
                logs: vec![Log {
                    address: address!("0000000000000000000000000000000000000011"),
                    data: LogData::new_unchecked(
                        vec![
                            b256!(
                                "000000000000000000000000000000000000000000000000000000000000dead"
                            ),
                            b256!(
                                "000000000000000000000000000000000000000000000000000000000000beef"
                            ),
                        ],
                        bytes!("0100ff"),
                    ),
                }],
            },
            logs_bloom: [0; 256].into(),
        });

        receipt.network_encode(&mut data);

        // check that the rlp length equals the length of the expected rlp
        assert_eq!(receipt.length(), expected.len());
        assert_eq!(data, expected);
    }

    #[test]
    fn legacy_receipt_from_parts() {
        let receipt =
            OpReceiptEnvelope::from_parts(true, 100, vec![], OpTxType::Legacy, None, None);
        assert!(receipt.status());
        assert_eq!(receipt.cumulative_gas_used(), 100);
        assert_eq!(receipt.logs().len(), 0);
        assert_eq!(receipt.tx_type(), OpTxType::Legacy);
    }

    #[test]
    fn deposit_receipt_from_parts() {
        let receipt =
            OpReceiptEnvelope::from_parts(true, 100, vec![], OpTxType::Deposit, Some(1), Some(2));
        assert!(receipt.status());
        assert_eq!(receipt.cumulative_gas_used(), 100);
        assert_eq!(receipt.logs().len(), 0);
        assert_eq!(receipt.tx_type(), OpTxType::Deposit);
        assert_eq!(receipt.deposit_nonce(), Some(1));
        assert_eq!(receipt.deposit_receipt_version(), Some(2));
    }
}
