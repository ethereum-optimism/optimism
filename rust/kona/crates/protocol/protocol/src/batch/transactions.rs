//! This module contains the [`SpanBatchTransactions`] type and logic for encoding and decoding
//! transactions in a span batch.

use super::varint::read_uvarint;
use crate::{
    MAX_SPAN_BATCH_ELEMENTS, SpanBatchBits, SpanBatchError, SpanBatchPostExecTransactionData,
    SpanBatchTransactionData, SpanDecodingError,
};
use alloc::vec::Vec;
use alloy_consensus::{Transaction, TxEnvelope};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{Address, Bytes, Signature, U256, bytes};
use alloy_rlp::{Buf, Decodable, Encodable, Header};
use op_alloy_consensus::{OpTxEnvelope, OpTxType};

/// This struct contains the decoded information for transactions in a span batch.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct SpanBatchTransactions {
    /// The total number of transactions in a span batch. Must be manually set.
    pub total_block_tx_count: u64,
    /// The contract creation bits, standard span-batch bitlist.
    pub contract_creation_bits: SpanBatchBits,
    /// The transaction signatures.
    pub tx_sigs: Vec<Signature>,
    /// The transaction nonces
    pub tx_nonces: Vec<u64>,
    /// The transaction gas limits.
    pub tx_gases: Vec<u64>,
    /// The `to` addresses of the transactions.
    pub tx_tos: Vec<Address>,
    /// The transaction data.
    pub tx_data: Vec<Vec<u8>>,
    /// The protected bits, standard span-batch bitlist.
    pub protected_bits: SpanBatchBits,
    /// The types of the transactions.
    pub tx_types: Vec<OpTxType>,
    /// Total legacy transaction count in the span batch.
    pub legacy_tx_count: u64,
}

struct SpanBatchTransactionParts {
    data: SpanBatchTransactionData,
    signature: Signature,
    to: Option<Address>,
    nonce: u64,
    gas: u64,
    chain_id: Option<u64>,
    tx_type: OpTxType,
}

impl SpanBatchTransactionParts {
    fn from_op_tx(tx: &OpTxEnvelope) -> Result<Self, SpanBatchError> {
        let (data, signature, to, nonce, gas, chain_id, tx_type) = match tx {
            OpTxEnvelope::Legacy(signed) => (
                SpanBatchTransactionData::try_from(&TxEnvelope::Legacy(signed.clone()))?,
                *signed.signature(),
                signed.to(),
                signed.nonce(),
                signed.gas_limit(),
                signed.chain_id(),
                OpTxType::Legacy,
            ),
            OpTxEnvelope::Eip2930(signed) => (
                SpanBatchTransactionData::try_from(&TxEnvelope::Eip2930(signed.clone()))?,
                *signed.signature(),
                signed.to(),
                signed.nonce(),
                signed.gas_limit(),
                signed.chain_id(),
                OpTxType::Eip2930,
            ),
            OpTxEnvelope::Eip1559(signed) => (
                SpanBatchTransactionData::try_from(&TxEnvelope::Eip1559(signed.clone()))?,
                *signed.signature(),
                signed.to(),
                signed.nonce(),
                signed.gas_limit(),
                signed.chain_id(),
                OpTxType::Eip1559,
            ),
            OpTxEnvelope::Eip7702(signed) => (
                SpanBatchTransactionData::try_from(&TxEnvelope::Eip7702(signed.clone()))?,
                *signed.signature(),
                signed.to(),
                signed.nonce(),
                signed.gas_limit(),
                signed.chain_id(),
                OpTxType::Eip7702,
            ),
            OpTxEnvelope::PostExec(tx) => (
                SpanBatchTransactionData::PostExec(SpanBatchPostExecTransactionData {
                    data: tx.input.clone().into(),
                }),
                // PostExec transactions are unsigned, so we use an all-zero placeholder signature.
                Signature::new(U256::ZERO, U256::ZERO, false),
                None,
                0,
                0,
                None,
                OpTxType::PostExec,
            ),
            OpTxEnvelope::Deposit(_) => {
                return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
            }
        };

        Ok(Self { data, signature, to, nonce, gas, chain_id, tx_type })
    }
}

/// Maximum EIP-2718 transaction-type byte. A larger leading byte is not a type identifier but the
/// start of a prefixless legacy transaction — a bare RLP list, whose header is `0xc0..=0xfe`. An
/// RLP string (`0x80..=0xbf`) fails the list check in [`read_tx_data`], and an overlong list header
/// (`0xff`) fails header decoding or the size bound.
const EIP2718_MAX_TX_TYPE: u8 = 0x7F;

/// Reads one span-batch transaction element from `r`, returning the element bytes (with any
/// EIP-2718 type prefix preserved) and its [`OpTxType`].
///
/// Its responsibility is deliberately narrow, matching op-node's `ReadTxData`: split the element
/// into its optional leading type byte and RLP payload, validating only the RLP *structure* — the
/// payload must be a list within the span-batch element-size bound. It does **not yet** discard
/// transactions with an invalid *type*: a representable-but-unsupported envelope — a leading `0x00`
/// (Legacy) or a Deposit (`0x7E`) prefix — is returned as-is and rejected once, downstream, by the
/// typed decoder ([`SpanBatchTransactionData`]) in [`SpanBatchTransactions::full_txs`]. The sole
/// type rejected here is a byte that maps to no [`OpTxType`] at all, which cannot be represented in
/// the returned type.
fn read_tx_data(r: &mut &[u8]) -> Result<(Vec<u8>, OpTxType), SpanBatchError> {
    let first_byte =
        *r.first().ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
    let has_type_prefix = first_byte <= EIP2718_MAX_TX_TYPE;
    let tx_type_id = if has_type_prefix {
        r.advance(1);
        first_byte
    } else {
        u8::from(OpTxType::Legacy)
    };

    // Read the RLP header with a different reader pointer. This prevents the initial pointer from
    // being advanced in the case that what we read is invalid.
    let rlp_header = Header::decode(&mut (**r).as_ref())
        .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
    if !rlp_header.list {
        return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    let payload_length_with_header = rlp_header.payload_length + rlp_header.length();
    if payload_length_with_header > MAX_SPAN_BATCH_ELEMENTS as usize {
        return Err(SpanBatchError::TooBigSpanBatchSize);
    }
    if r.len() < payload_length_with_header {
        return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    // A leading byte with no `OpTxType` at all is rejected here. Every representable type —
    // including `Legacy` (a leading `0x00`) and `Deposit`, neither a valid span-batch envelope — is
    // kept and rejected once by the typed decoder downstream, matching op-node's deferred
    // rejection.
    let tx_type = match OpTxType::try_from(tx_type_id) {
        Err(_) => {
            return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionType));
        }
        Ok(ty) => ty,
    };

    // Preserve whether a type prefix was physically present instead of inferring it from the
    // logical transaction type. `OpTxType::Legacy` is represented as zero for transaction-type
    // metadata, but a leading `0x00` is still a typed-envelope prefix and must not be discarded.
    // Keeping it lets the typed span-batch decoder reject the unsupported envelope, as op-node
    // does, rather than silently reinterpreting its payload as a prefixless legacy transaction.
    let tx_data_capacity = payload_length_with_header + usize::from(has_type_prefix);
    let mut tx_data = Vec::with_capacity(tx_data_capacity);
    if has_type_prefix {
        tx_data.push(first_byte);
    }
    tx_data.extend_from_slice(&r[..payload_length_with_header]);
    r.advance(payload_length_with_header);

    Ok((tx_data, tx_type))
}

impl SpanBatchTransactions {
    /// Encodes the [`SpanBatchTransactions`] into a writer.
    pub fn encode(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        self.encode_contract_creation_bits(w)?;
        self.encode_tx_sigs(w)?;
        self.encode_tx_tos(w)?;
        self.encode_tx_data(w)?;
        self.encode_tx_nonces(w)?;
        self.encode_tx_gases(w)?;
        self.encode_protected_bits(w)?;
        Ok(())
    }

    /// Decodes the [`SpanBatchTransactions`] from a reader.
    pub fn decode(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        self.decode_contract_creation_bits(r)?;
        self.decode_tx_sigs(r)?;
        self.decode_tx_tos(r)?;
        self.decode_tx_data(r)?;
        self.decode_tx_nonces(r)?;
        self.decode_tx_gases(r)?;
        self.decode_protected_bits(r)?;
        Ok(())
    }

    /// Encode the contract creation bits into a writer.
    pub fn encode_contract_creation_bits(
        &self,
        w: &mut dyn bytes::BufMut,
    ) -> Result<(), SpanBatchError> {
        SpanBatchBits::encode(w, self.total_block_tx_count as usize, &self.contract_creation_bits)?;
        Ok(())
    }

    /// Encode the protected bits into a writer.
    pub fn encode_protected_bits(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        SpanBatchBits::encode(w, self.legacy_tx_count as usize, &self.protected_bits)?;
        Ok(())
    }

    /// Encode the transaction signatures into a writer (excluding `v` field).
    pub fn encode_tx_sigs(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        let mut y_parity_bits = SpanBatchBits::default();
        for (i, sig) in self.tx_sigs.iter().enumerate() {
            y_parity_bits.set_bit(i, sig.v());
        }

        SpanBatchBits::encode(w, self.total_block_tx_count as usize, &y_parity_bits)?;
        for sig in &self.tx_sigs {
            w.put_slice(&sig.r().to_be_bytes::<32>());
            w.put_slice(&sig.s().to_be_bytes::<32>());
        }
        Ok(())
    }

    /// Encode the transaction nonces into a writer.
    pub fn encode_tx_nonces(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        let mut buf = [0u8; 10];
        for nonce in &self.tx_nonces {
            let slice = unsigned_varint::encode::u64(*nonce, &mut buf);
            w.put_slice(slice);
        }
        Ok(())
    }

    /// Encode the transaction gas limits into a writer.
    pub fn encode_tx_gases(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        let mut buf = [0u8; 10];
        for gas in &self.tx_gases {
            let slice = unsigned_varint::encode::u64(*gas, &mut buf);
            w.put_slice(slice);
        }
        Ok(())
    }

    /// Encode the `to` addresses of the transactions into a writer.
    pub fn encode_tx_tos(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        for to in &self.tx_tos {
            w.put_slice(to.as_ref());
        }
        Ok(())
    }

    /// Encode the transaction data into a writer.
    pub fn encode_tx_data(&self, w: &mut dyn bytes::BufMut) -> Result<(), SpanBatchError> {
        for data in &self.tx_data {
            w.put_slice(data);
        }
        Ok(())
    }

    /// Decode the contract creation bits from a reader.
    pub fn decode_contract_creation_bits(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if self.total_block_tx_count > MAX_SPAN_BATCH_ELEMENTS {
            return Err(SpanBatchError::TooBigSpanBatchSize);
        }

        self.contract_creation_bits = SpanBatchBits::decode(r, self.total_block_tx_count as usize)?;
        Ok(())
    }

    /// Decode the protected bits from a reader.
    pub fn decode_protected_bits(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        if self.legacy_tx_count > MAX_SPAN_BATCH_ELEMENTS {
            return Err(SpanBatchError::TooBigSpanBatchSize);
        }

        self.protected_bits = SpanBatchBits::decode(r, self.legacy_tx_count as usize)?;
        Ok(())
    }

    /// Decode the transaction signatures from a reader (excluding `v` field).
    pub fn decode_tx_sigs(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let y_parity_bits = SpanBatchBits::decode(r, self.total_block_tx_count as usize)?;
        let mut sigs = Vec::with_capacity(self.total_block_tx_count as usize);
        for i in 0..self.total_block_tx_count {
            let y_parity = y_parity_bits.get_bit(i as usize).expect("same length");
            if r.len() < 64 {
                return Err(SpanBatchError::Decoding(
                    SpanDecodingError::InvalidTransactionSignature,
                ));
            }
            let r_val = U256::from_be_slice(&r[..32]);
            let s_val = U256::from_be_slice(&r[32..64]);
            sigs.push(Signature::new(r_val, s_val, y_parity == 1));
            r.advance(64);
        }
        self.tx_sigs = sigs;
        Ok(())
    }

    /// Decode the transaction nonces from a reader.
    pub fn decode_tx_nonces(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let mut nonces = Vec::with_capacity(self.total_block_tx_count as usize);
        for _ in 0..self.total_block_tx_count {
            let (nonce, remaining) =
                read_uvarint(r).ok_or(SpanBatchError::Decoding(SpanDecodingError::TxNonces))?;
            nonces.push(nonce);
            *r = remaining;
        }
        self.tx_nonces = nonces;
        Ok(())
    }

    /// Decode the transaction gas limits from a reader.
    pub fn decode_tx_gases(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let mut gases = Vec::with_capacity(self.total_block_tx_count as usize);
        for _ in 0..self.total_block_tx_count {
            let (gas, remaining) =
                read_uvarint(r).ok_or(SpanBatchError::Decoding(SpanDecodingError::TxGases))?;
            gases.push(gas);
            *r = remaining;
        }
        self.tx_gases = gases;
        Ok(())
    }

    /// Decode the `to` addresses of the transactions from a reader.
    pub fn decode_tx_tos(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let mut tos = Vec::with_capacity(self.total_block_tx_count as usize);
        let contract_creation_count = self.contract_creation_count();
        for _ in 0..(self.total_block_tx_count - contract_creation_count) {
            if r.len() < 20 {
                return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
            }
            let to = Address::from_slice(&r[..20]);
            tos.push(to);
            r.advance(20);
        }
        self.tx_tos = tos;
        Ok(())
    }

    /// Decode the transaction data from a reader.
    pub fn decode_tx_data(&mut self, r: &mut &[u8]) -> Result<(), SpanBatchError> {
        let mut tx_data = Vec::new();
        let mut tx_types = Vec::new();

        // Do not need the transaction data header because the RLP stream already includes the
        // length information.
        for _ in 0..self.total_block_tx_count {
            let (tx_data_item, tx_type) = read_tx_data(r)?;
            tx_data.push(tx_data_item);
            tx_types.push(tx_type);
            if tx_type == OpTxType::Legacy {
                self.legacy_tx_count += 1;
            }
        }

        self.tx_data = tx_data;
        self.tx_types = tx_types;

        Ok(())
    }

    /// Returns the number of contract creation transactions in the span batch.
    pub fn contract_creation_count(&self) -> u64 {
        self.contract_creation_bits.as_ref().iter().map(|b| b.count_ones() as u64).sum()
    }

    /// Retrieve all of the raw transactions from the [`SpanBatchTransactions`].
    pub fn full_txs(&self, chain_id: u64) -> Result<Vec<Vec<u8>>, SpanBatchError> {
        let mut txs = Vec::new();
        let mut to_idx = 0;
        let mut protected_bit_idx = 0;
        for idx in 0..self.total_block_tx_count as usize {
            let mut data = self.tx_data[idx].as_slice();
            let tx = SpanBatchTransactionData::decode(&mut data)
                .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let nonce = self
                .tx_nonces
                .get(idx)
                .ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let gas = self
                .tx_gases
                .get(idx)
                .ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let bit = self
                .contract_creation_bits
                .get_bit(idx)
                .ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let to = if bit == 0 {
                let to = *self
                    .tx_tos
                    .get(to_idx)
                    .ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
                to_idx += 1;
                Some(to)
            } else {
                None
            };
            let sig = *self
                .tx_sigs
                .get(idx)
                .ok_or(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let is_protected = if tx.tx_type() == OpTxType::Legacy {
                protected_bit_idx += 1;
                self.protected_bits.get_bit(protected_bit_idx - 1).unwrap_or_default() == 1
            } else {
                true
            };
            txs.push(tx.to_full_tx_bytes(*nonce, *gas, to, chain_id, sig, is_protected)?);
        }
        Ok(txs)
    }

    /// Add raw transactions into the [`SpanBatchTransactions`].
    pub fn add_txs(&mut self, txs: Vec<Bytes>, chain_id: u64) -> Result<(), SpanBatchError> {
        let offset = self.total_block_tx_count as usize;

        for (i, raw_tx) in txs.iter().enumerate() {
            let op_tx = OpTxEnvelope::decode_2718(&mut raw_tx.as_ref())
                .map_err(|_| SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))?;
            let parts = SpanBatchTransactionParts::from_op_tx(&op_tx)?;

            if parts.tx_type == OpTxType::Legacy {
                self.protected_bits
                    .set_bit(self.legacy_tx_count as usize, parts.chain_id.is_some());
                self.legacy_tx_count += 1;
            }
            if parts.tx_type != OpTxType::PostExec &&
                parts.chain_id.is_some_and(|id| id != chain_id)
            {
                return Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
            }

            let contract_creation_bit = match parts.to {
                Some(address) => {
                    self.tx_tos.push(address);
                    false
                }
                None => true,
            };
            let mut tx_data_buf = Vec::new();
            parts.data.encode(&mut tx_data_buf);

            self.tx_sigs.push(parts.signature);
            self.contract_creation_bits.set_bit(i + offset, contract_creation_bit);
            self.tx_nonces.push(parts.nonce);
            self.tx_data.push(tx_data_buf);
            self.tx_gases.push(parts.gas);
            self.tx_types.push(parts.tx_type);
        }
        self.total_block_tx_count += txs.len() as u64;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::vec;
    use alloy_consensus::{Signed, TxEip1559, TxEip2930, TxEip7702, TxEnvelope};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Signature, TxKind, address};
    use op_alloy_consensus::{
        OpTxEnvelope, POST_EXEC_PAYLOAD_VERSION, PostExecPayload, TxPostExec,
    };

    #[test]
    fn test_decode_tx_sigs_truncated() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        // Provide a valid y_parity bitfield (1 bit = 1 byte) but truncated signature data
        // SpanBatchBits::decode for 1 bit needs 1 byte for the bitfield
        let buf = vec![0u8]; // y_parity byte, but no r/s signature bytes
        let result = txs.decode_tx_sigs(&mut buf.as_slice());
        assert_eq!(
            result,
            Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionSignature))
        );
    }

    #[test]
    fn test_decode_tx_tos_truncated() {
        let mut txs = SpanBatchTransactions {
            total_block_tx_count: 1,
            contract_creation_bits: SpanBatchBits::default(),
            ..Default::default()
        };
        let buf = [0u8; 19]; // one byte short of a 20-byte address
        let result = txs.decode_tx_tos(&mut buf.as_slice());
        assert_eq!(
            result,
            Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))
        );
    }

    #[test]
    fn test_decode_tx_tos_empty() {
        let mut txs = SpanBatchTransactions {
            total_block_tx_count: 1,
            contract_creation_bits: SpanBatchBits::default(),
            ..Default::default()
        };
        let result = txs.decode_tx_tos(&mut [].as_slice());
        assert_eq!(
            result,
            Err(SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData))
        );
    }

    #[test]
    fn test_decode_tx_gases_truncated() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        let result = txs.decode_tx_gases(&mut [].as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::TxGases)));
    }

    // Conformance vectors for the accept-set documented in `batch::varint`.

    #[test]
    fn test_decode_tx_nonces_non_minimal() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        // `1` with a redundant trailing zero byte; the minimal form is `[0x01]`.
        let buf = [0x81, 0x00];
        let mut r = buf.as_slice();
        txs.decode_tx_nonces(&mut r).unwrap();
        assert_eq!(txs.tx_nonces, vec![1]);
        assert!(r.is_empty(), "both bytes must be consumed");
    }

    #[test]
    fn test_decode_tx_nonces_overflow_ten_byte_terminator() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        // The tenth byte terminates the varint but sets bits above 63.
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02];
        let result = txs.decode_tx_nonces(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::TxNonces)));
    }

    #[test]
    fn test_decode_tx_gases_non_minimal() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        let buf = [0x81, 0x00];
        let mut r = buf.as_slice();
        txs.decode_tx_gases(&mut r).unwrap();
        assert_eq!(txs.tx_gases, vec![1]);
        assert!(r.is_empty(), "both bytes must be consumed");
    }

    #[test]
    fn test_decode_tx_gases_overflow_ten_byte_terminator() {
        let mut txs = SpanBatchTransactions { total_block_tx_count: 1, ..Default::default() };
        let buf = [0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02];
        let result = txs.decode_tx_gases(&mut buf.as_slice());
        assert_eq!(result, Err(SpanBatchError::Decoding(SpanDecodingError::TxGases)));
    }

    #[test]
    fn test_span_batch_transactions_add_empty_txs() {
        let mut span_batch_txs = SpanBatchTransactions::default();
        let txs = vec![];
        let chain_id = 1;
        let result = span_batch_txs.add_txs(txs, chain_id);
        assert!(result.is_ok());
        assert_eq!(span_batch_txs.total_block_tx_count, 0);
    }

    #[test]
    fn test_span_batch_transactions_add_eip2930_tx_wrong_chain_id() {
        let sig = Signature::test_signature();
        let to = address!("0123456789012345678901234567890123456789");
        let tx = TxEnvelope::Eip2930(Signed::new_unchecked(
            TxEip2930 { to: TxKind::Call(to), ..Default::default() },
            sig,
            Default::default(),
        ));
        let mut span_batch_txs = SpanBatchTransactions::default();
        let mut buf = vec![];
        tx.encode_2718(&mut buf);
        let txs = vec![Bytes::from(buf)];
        let chain_id = 1;
        let err = span_batch_txs.add_txs(txs, chain_id).unwrap_err();
        assert_eq!(err, SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    #[test]
    fn test_span_batch_transactions_add_post_exec_tx_roundtrip() {
        let tx: OpTxEnvelope = TxPostExec::new(PostExecPayload {
            version: POST_EXEC_PAYLOAD_VERSION,
            block_number: 42,
            gas_refund_entries: vec![],
        })
        .into();
        let mut buf = vec![];
        tx.encode_2718(&mut buf);

        let mut span_batch_txs = SpanBatchTransactions::default();
        let result = span_batch_txs.add_txs(vec![Bytes::from(buf.clone())], 1);
        assert_eq!(result, Ok(()));
        assert_eq!(span_batch_txs.total_block_tx_count, 1);
        assert_eq!(span_batch_txs.tx_types, vec![OpTxType::PostExec]);
        assert!(span_batch_txs.tx_tos.is_empty());
        assert_eq!(span_batch_txs.contract_creation_bits.get_bit(0), Some(1));

        let full_txs = span_batch_txs.full_txs(1).unwrap();
        assert_eq!(full_txs, vec![buf]);
    }

    #[test]
    fn test_span_batch_transactions_add_eip2930_tx() {
        let sig = Signature::test_signature();
        let to = address!("0123456789012345678901234567890123456789");
        let tx = TxEnvelope::Eip2930(Signed::new_unchecked(
            TxEip2930 { to: TxKind::Call(to), chain_id: 1, ..Default::default() },
            sig,
            Default::default(),
        ));
        let mut span_batch_txs = SpanBatchTransactions::default();
        let mut buf = vec![];
        tx.encode_2718(&mut buf);
        let txs = vec![Bytes::from(buf)];
        let chain_id = 1;
        let result = span_batch_txs.add_txs(txs, chain_id);
        assert_eq!(result, Ok(()));
        assert_eq!(span_batch_txs.total_block_tx_count, 1);
    }

    #[test]
    fn test_span_batch_transactions_add_eip1559_tx() {
        let sig = Signature::test_signature();
        let to = address!("0123456789012345678901234567890123456789");
        let tx = TxEnvelope::Eip1559(Signed::new_unchecked(
            TxEip1559 { to: TxKind::Call(to), chain_id: 1, ..Default::default() },
            sig,
            Default::default(),
        ));
        let mut span_batch_txs = SpanBatchTransactions::default();
        let mut buf = vec![];
        tx.encode_2718(&mut buf);
        let txs = vec![Bytes::from(buf)];
        let chain_id = 1;
        let result = span_batch_txs.add_txs(txs, chain_id);
        assert_eq!(result, Ok(()));
        assert_eq!(span_batch_txs.total_block_tx_count, 1);
    }

    #[test]
    fn test_span_batch_transactions_add_eip7702_tx() {
        let sig = Signature::test_signature();
        let to = address!("0123456789012345678901234567890123456789");
        let tx = TxEnvelope::Eip7702(Signed::new_unchecked(
            TxEip7702 { to, chain_id: 1, ..Default::default() },
            sig,
            Default::default(),
        ));
        let mut span_batch_txs = SpanBatchTransactions::default();
        let mut buf = vec![];
        tx.encode_2718(&mut buf);
        let txs = vec![Bytes::from(buf)];
        let chain_id = 1;
        let result = span_batch_txs.add_txs(txs, chain_id);
        assert_eq!(result, Ok(()));
        assert_eq!(span_batch_txs.total_block_tx_count, 1);
    }

    #[test]
    fn test_read_tx_data_truncated_payload() {
        // RLP list header claiming 100 bytes of payload, but only 3 bytes actually present.
        // 0xf8 0x64 = list header with 1-byte length prefix, payload length 100
        let mut data: &[u8] = &[0xf8, 0x64, 0x00, 0x00, 0x00];
        let err = read_tx_data(&mut data).unwrap_err();
        assert_eq!(err, SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
    }

    #[test]
    fn test_read_tx_data_accepts_prefixless_legacy() {
        // A first byte > 0x7F is not a type id, so it is read as a prefixless legacy tx. `c3 80 80
        // 80` is a schema-valid legacy element — an RLP list of the three legacy fields (value,
        // gas_price, data), here all zero/empty: read_tx_data returns it verbatim and it decodes
        // end-to-end as a legacy tx.
        let mut data: &[u8] = &[0xc3, 0x80, 0x80, 0x80];
        let (tx_data, tx_type) = read_tx_data(&mut data).unwrap();
        assert_eq!(tx_type, OpTxType::Legacy);
        assert_eq!(tx_data, vec![0xc3, 0x80, 0x80, 0x80]);
        SpanBatchTransactionData::decode(&mut tx_data.as_slice())
            .expect("prefixless legacy payload decodes into a legacy tx");
    }

    #[test]
    fn test_read_tx_data_preserves_leading_zero_type_byte() {
        // The shared op-node<->kona conformance vector: the valid legacy element from
        // `test_read_tx_data_accepts_prefixless_legacy` with a `0x00` EIP-2718 type byte prepended.
        // The reader must preserve the prefix so the typed decoder rejects it, instead of
        // reinterpreting the remaining bytes as a prefixless legacy transaction.
        let mut data: &[u8] = &[0x00, 0xc3, 0x80, 0x80, 0x80];
        let (tx_data, tx_type) = read_tx_data(&mut data).unwrap();
        assert_eq!(tx_type, OpTxType::Legacy);
        assert_eq!(tx_data, vec![0x00, 0xc3, 0x80, 0x80, 0x80]);
        SpanBatchTransactionData::decode(&mut tx_data.as_slice())
            .expect_err("0x00 is not a supported typed span-batch transaction envelope");
    }

    #[test]
    fn test_read_tx_data_preserves_deposit_type_byte() {
        // Deposit (0x7E) is a valid OpTxType but never a valid span-batch envelope. Like a leading
        // `0x00`, the reader keeps the prefix and the typed decoder rejects it downstream, rather
        // than the reader making the type-validity call.
        let mut data: &[u8] = &[0x7E, 0xc3, 0x80, 0x80, 0x80];
        let (tx_data, tx_type) = read_tx_data(&mut data).unwrap();
        assert_eq!(tx_type, OpTxType::Deposit);
        assert_eq!(tx_data, vec![0x7E, 0xc3, 0x80, 0x80, 0x80]);
        SpanBatchTransactionData::decode(&mut tx_data.as_slice())
            .expect_err("deposit is not a supported span-batch transaction envelope");
    }

    #[test]
    fn test_read_tx_data_rejects_unknown_type_byte() {
        // A type byte <= 0x7F that maps to no `OpTxType` (e.g. 0x03, 0x7F) cannot be represented,
        // so — unlike Legacy/Deposit, which defer to the typed decoder — it is rejected at
        // the reader.
        for first in [0x03u8, 0x7F] {
            let mut data: &[u8] = &[first, 0xc1, 0x05];
            let err = read_tx_data(&mut data).unwrap_err();
            assert_eq!(err, SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionType));
        }
    }

    #[test]
    fn test_read_tx_data_rejects_non_list_high_first_byte() {
        // Not every byte > 0x7F is a legacy list header: 0x80..=0xbf are RLP strings (rejected by
        // the list check) and 0xff is an overlong list header (rejected by header decoding). Only
        // genuine list headers (0xc0..=0xfe) survive.
        for first in [0x80u8, 0xbf, 0xff] {
            let mut data: &[u8] = &[first];
            let err = read_tx_data(&mut data).unwrap_err();
            assert_eq!(err, SpanBatchError::Decoding(SpanDecodingError::InvalidTransactionData));
        }
    }

    #[test]
    fn test_read_tx_data_accepts_typed() {
        // A valid EIP-2718 type byte (0x02 = EIP-1559) is preserved before the payload.
        let mut data: &[u8] = &[0x02, 0xc1, 0x05];
        let (tx_data, tx_type) = read_tx_data(&mut data).unwrap();
        assert_eq!(tx_type, OpTxType::Eip1559);
        assert_eq!(tx_data, vec![0x02, 0xc1, 0x05]);
    }

    #[test]
    fn test_read_tx_data_exceeds_max_span_batch_elements() {
        // RLP list header claiming MAX_SPAN_BATCH_ELEMENTS + 1 bytes of payload.
        // 0xfa = list with 3-byte length prefix (0xf7 + 3), then 0x989681 = 10_000_001.
        // Header::decode validates the claimed length against the buffer, so we must provide
        // a buffer large enough for decoding to succeed before the max size check triggers.
        let mut data = vec![0u8; MAX_SPAN_BATCH_ELEMENTS as usize + 5];
        data[0] = 0xfa;
        data[1] = 0x98;
        data[2] = 0x96;
        data[3] = 0x81;
        let mut slice: &[u8] = &data;
        let err = read_tx_data(&mut slice).unwrap_err();
        assert_eq!(err, SpanBatchError::TooBigSpanBatchSize);
    }
}
