use crate::primitives::PAYMENT_TX_TYPE_ID;
use alloy_consensus::{
    transaction::{RlpEcdsaDecodableTx, RlpEcdsaEncodableTx},
    SignableTransaction, Transaction,
};
use alloy_eips::{eip2930::AccessList, eip7702::SignedAuthorization, Typed2718};
use alloy_primitives::{Address, Bytes, ChainId, Signature, TxKind, B256, U256};
use alloy_rlp::{BufMut, Decodable, Encodable};
use reth_ethereum::primitives::{serde_bincode_compat::RlpBincode, InMemorySize};

/// A transaction with a priority fee ([EIP-1559](https://eips.ethereum.org/EIPS/eip-1559)).
#[derive(
    Clone,
    Debug,
    Default,
    PartialEq,
    Eq,
    Hash,
    serde::Serialize,
    serde::Deserialize,
    reth_codecs::Compact,
)]
#[serde(rename_all = "camelCase")]
#[doc(alias = "PaymentTransaction", alias = "TransactionPayment", alias = "PaymentTx")]
pub struct TxPayment {
    /// EIP-155: Simple replay attack protection
    #[serde(with = "alloy_serde::quantity")]
    pub chain_id: ChainId,
    /// A scalar value equal to the number of transactions sent by the sender; formally Tn.
    #[serde(with = "alloy_serde::quantity")]
    pub nonce: u64,
    /// A scalar value equal to the maximum
    /// amount of gas that should be used in executing
    /// this transaction. This is paid up-front, before any
    /// computation is done and may not be increased
    /// later; formally Tg.
    #[serde(with = "alloy_serde::quantity", rename = "gas", alias = "gasLimit")]
    pub gas_limit: u64,
    /// A scalar value equal to the maximum
    /// amount of gas that should be used in executing
    /// this transaction. This is paid up-front, before any
    /// computation is done and may not be increased
    /// later; formally Tg.
    ///
    /// As ethereum circulation is around 120mil eth as of 2022 that is around
    /// 120000000000000000000000000 wei we are safe to use u128 as its max number is:
    /// 340282366920938463463374607431768211455
    ///
    /// This is also known as `GasFeeCap`
    #[serde(with = "alloy_serde::quantity")]
    pub max_fee_per_gas: u128,
    /// Max Priority fee that transaction is paying
    ///
    /// As ethereum circulation is around 120mil eth as of 2022 that is around
    /// 120000000000000000000000000 wei we are safe to use u128 as its max number is:
    /// 340282366920938463463374607431768211455
    ///
    /// This is also known as `GasTipCap`
    #[serde(with = "alloy_serde::quantity")]
    pub max_priority_fee_per_gas: u128,
    /// The 160-bit address of the message call’s recipient.
    pub to: Address,
    /// A scalar value equal to the number of Wei to
    /// be transferred to the message call’s recipient or,
    /// in the case of contract creation, as an endowment
    /// to the newly created account; formally Tv.
    pub value: U256,
}

impl TxPayment {
    /// Get the transaction type
    #[doc(alias = "transaction_type")]
    pub const fn tx_type() -> super::tx::TxTypeCustom {
        super::tx::TxTypeCustom::Payment
    }

    /// Calculates a heuristic for the in-memory size of the [TxPayment] transaction.
    #[inline]
    pub fn size(&self) -> usize {
        size_of::<Self>()
    }
}

impl RlpEcdsaEncodableTx for TxPayment {
    /// Outputs the length of the transaction's fields, without a RLP header.
    fn rlp_encoded_fields_length(&self) -> usize {
        self.chain_id.length() +
            self.nonce.length() +
            self.max_priority_fee_per_gas.length() +
            self.max_fee_per_gas.length() +
            self.gas_limit.length() +
            self.to.length() +
            self.value.length()
    }

    /// Encodes only the transaction's fields into the desired buffer, without
    /// a RLP header.
    fn rlp_encode_fields(&self, out: &mut dyn alloy_rlp::BufMut) {
        self.chain_id.encode(out);
        self.nonce.encode(out);
        self.max_priority_fee_per_gas.encode(out);
        self.max_fee_per_gas.encode(out);
        self.gas_limit.encode(out);
        self.to.encode(out);
        self.value.encode(out);
    }
}

impl RlpEcdsaDecodableTx for TxPayment {
    const DEFAULT_TX_TYPE: u8 = { PAYMENT_TX_TYPE_ID };

    /// Decodes the inner [TxPayment] fields from RLP bytes.
    ///
    /// NOTE: This assumes a RLP header has already been decoded, and _just_
    /// decodes the following RLP fields in the following order:
    ///
    /// - `chain_id`
    /// - `nonce`
    /// - `max_priority_fee_per_gas`
    /// - `max_fee_per_gas`
    /// - `gas_limit`
    /// - `to`
    /// - `value`
    /// - `data` (`input`)
    /// - `access_list`
    fn rlp_decode_fields(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Ok(Self {
            chain_id: Decodable::decode(buf)?,
            nonce: Decodable::decode(buf)?,
            max_priority_fee_per_gas: Decodable::decode(buf)?,
            max_fee_per_gas: Decodable::decode(buf)?,
            gas_limit: Decodable::decode(buf)?,
            to: Decodable::decode(buf)?,
            value: Decodable::decode(buf)?,
        })
    }
}

impl Transaction for TxPayment {
    #[inline]
    fn chain_id(&self) -> Option<ChainId> {
        Some(self.chain_id)
    }

    #[inline]
    fn nonce(&self) -> u64 {
        self.nonce
    }

    #[inline]
    fn gas_limit(&self) -> u64 {
        self.gas_limit
    }

    #[inline]
    fn gas_price(&self) -> Option<u128> {
        None
    }

    #[inline]
    fn max_fee_per_gas(&self) -> u128 {
        self.max_fee_per_gas
    }

    #[inline]
    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        Some(self.max_priority_fee_per_gas)
    }

    #[inline]
    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        None
    }

    #[inline]
    fn priority_fee_or_price(&self) -> u128 {
        self.max_priority_fee_per_gas
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        base_fee.map_or(self.max_fee_per_gas, |base_fee| {
            // if the tip is greater than the max priority fee per gas, set it to the max
            // priority fee per gas + base fee
            let tip = self.max_fee_per_gas.saturating_sub(base_fee as u128);
            if tip > self.max_priority_fee_per_gas {
                self.max_priority_fee_per_gas + base_fee as u128
            } else {
                // otherwise return the max fee per gas
                self.max_fee_per_gas
            }
        })
    }

    #[inline]
    fn is_dynamic_fee(&self) -> bool {
        true
    }

    #[inline]
    fn kind(&self) -> TxKind {
        TxKind::Call(self.to)
    }

    #[inline]
    fn is_create(&self) -> bool {
        false
    }

    #[inline]
    fn value(&self) -> U256 {
        self.value
    }

    #[inline]
    fn input(&self) -> &Bytes {
        // No input data
        static EMPTY_BYTES: Bytes = Bytes::new();
        &EMPTY_BYTES
    }

    #[inline]
    fn access_list(&self) -> Option<&AccessList> {
        None
    }

    #[inline]
    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        None
    }

    #[inline]
    fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
        None
    }
}

impl Typed2718 for TxPayment {
    fn ty(&self) -> u8 {
        PAYMENT_TX_TYPE_ID
    }
}

impl SignableTransaction<Signature> for TxPayment {
    fn set_chain_id(&mut self, chain_id: ChainId) {
        self.chain_id = chain_id;
    }

    fn encode_for_signing(&self, out: &mut dyn alloy_rlp::BufMut) {
        out.put_u8(Self::tx_type().ty());
        self.encode(out)
    }

    fn payload_len_for_signature(&self) -> usize {
        self.length() + 1
    }
}

impl Encodable for TxPayment {
    fn encode(&self, out: &mut dyn BufMut) {
        self.rlp_encode(out);
    }

    fn length(&self) -> usize {
        self.rlp_encoded_length()
    }
}

impl Decodable for TxPayment {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::rlp_decode(buf)
    }
}

impl InMemorySize for TxPayment {
    fn size(&self) -> usize {
        TxPayment::size(self)
    }
}

impl RlpBincode for TxPayment {}
