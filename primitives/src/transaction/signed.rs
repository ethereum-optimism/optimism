//! A signed Optimism transaction.

use alloc::vec::Vec;
use alloy_consensus::{
    transaction::RlpEcdsaTx, Sealed, SignableTransaction, Signed, Transaction, TxEip1559,
    TxEip2930, TxEip7702, TxLegacy, Typed2718,
};
use alloy_eips::{
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718},
    eip2930::AccessList,
    eip7702::SignedAuthorization,
};
use alloy_evm::FromRecoveredTx;
use alloy_primitives::{
    keccak256, Address, Bytes, PrimitiveSignature as Signature, TxHash, TxKind, Uint, B256,
};
use alloy_rlp::Header;
use core::{
    hash::{Hash, Hasher},
    mem,
    ops::Deref,
};
use derive_more::{AsRef, Deref};
use op_alloy_consensus::{
    DepositTransaction, OpPooledTransaction, OpTxEnvelope, OpTypedTransaction, TxDeposit,
};
use op_revm::transaction::deposit::DepositTransactionParts;
#[cfg(any(test, feature = "reth-codec"))]
use proptest as _;
use reth_primitives_traits::{
    crypto::secp256k1::{recover_signer, recover_signer_unchecked},
    sync::OnceLock,
    transaction::{error::TransactionConversionError, signed::RecoveryError},
    InMemorySize, SignedTransaction,
};
use revm_context::TxEnv;

/// Signed transaction.
#[cfg_attr(any(test, feature = "reth-codec"), reth_codecs::add_arbitrary_tests(rlp))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[derive(Debug, Clone, Eq, AsRef, Deref)]
pub struct OpTransactionSigned {
    /// Transaction hash
    #[cfg_attr(feature = "serde", serde(skip))]
    hash: OnceLock<TxHash>,
    /// The transaction signature values
    signature: Signature,
    /// Raw transaction info
    #[deref]
    #[as_ref]
    transaction: OpTypedTransaction,
}

impl OpTransactionSigned {
    /// Creates a new signed transaction from the given transaction, signature and hash.
    pub fn new(transaction: OpTypedTransaction, signature: Signature, hash: B256) -> Self {
        Self { hash: hash.into(), signature, transaction }
    }

    /// Consumes the type and returns the transaction.
    #[inline]
    pub fn into_transaction(self) -> OpTypedTransaction {
        self.transaction
    }

    /// Returns the transaction.
    #[inline]
    pub const fn transaction(&self) -> &OpTypedTransaction {
        &self.transaction
    }

    /// Splits the `OpTransactionSigned` into its transaction and signature.
    pub fn split(self) -> (OpTypedTransaction, Signature) {
        (self.transaction, self.signature)
    }

    /// Creates a new signed transaction from the given transaction and signature without the hash.
    ///
    /// Note: this only calculates the hash on the first [`OpTransactionSigned::hash`] call.
    pub fn new_unhashed(transaction: OpTypedTransaction, signature: Signature) -> Self {
        Self { hash: Default::default(), signature, transaction }
    }

    /// Returns whether this transaction is a deposit.
    pub const fn is_deposit(&self) -> bool {
        matches!(self.transaction, OpTypedTransaction::Deposit(_))
    }

    /// Splits the transaction into parts.
    pub fn into_parts(self) -> (OpTypedTransaction, Signature, B256) {
        let hash = *self.hash.get_or_init(|| self.recalculate_hash());
        (self.transaction, self.signature, hash)
    }
}

impl SignedTransaction for OpTransactionSigned {
    fn tx_hash(&self) -> &TxHash {
        self.hash.get_or_init(|| self.recalculate_hash())
    }

    fn signature(&self) -> &Signature {
        &self.signature
    }

    fn recover_signer(&self) -> Result<Address, RecoveryError> {
        // Optimism's Deposit transaction does not have a signature. Directly return the
        // `from` address.
        if let OpTypedTransaction::Deposit(TxDeposit { from, .. }) = self.transaction {
            return Ok(from)
        }

        let Self { transaction, signature, .. } = self;
        let signature_hash = signature_hash(transaction);
        recover_signer(signature, signature_hash)
    }

    fn recover_signer_unchecked(&self) -> Result<Address, RecoveryError> {
        // Optimism's Deposit transaction does not have a signature. Directly return the
        // `from` address.
        if let OpTypedTransaction::Deposit(TxDeposit { from, .. }) = &self.transaction {
            return Ok(*from)
        }

        let Self { transaction, signature, .. } = self;
        let signature_hash = signature_hash(transaction);
        recover_signer_unchecked(signature, signature_hash)
    }

    fn recover_signer_unchecked_with_buf(
        &self,
        buf: &mut Vec<u8>,
    ) -> Result<Address, RecoveryError> {
        match &self.transaction {
            // Optimism's Deposit transaction does not have a signature. Directly return the
            // `from` address.
            OpTypedTransaction::Deposit(tx) => return Ok(tx.from),
            OpTypedTransaction::Legacy(tx) => tx.encode_for_signing(buf),
            OpTypedTransaction::Eip2930(tx) => tx.encode_for_signing(buf),
            OpTypedTransaction::Eip1559(tx) => tx.encode_for_signing(buf),
            OpTypedTransaction::Eip7702(tx) => tx.encode_for_signing(buf),
        };
        recover_signer_unchecked(&self.signature, keccak256(buf))
    }

    fn recalculate_hash(&self) -> B256 {
        keccak256(self.encoded_2718())
    }
}

macro_rules! impl_from_signed {
    ($($tx:ident),*) => {
        $(
            impl From<Signed<$tx>> for OpTransactionSigned {
                fn from(value: Signed<$tx>) -> Self {
                    let(tx,sig,hash) = value.into_parts();
                    Self::new(tx.into(), sig, hash)
                }
            }
        )*
    };
}

impl_from_signed!(TxLegacy, TxEip2930, TxEip1559, TxEip7702, OpTypedTransaction);

impl From<OpTxEnvelope> for OpTransactionSigned {
    fn from(value: OpTxEnvelope) -> Self {
        match value {
            OpTxEnvelope::Legacy(tx) => tx.into(),
            OpTxEnvelope::Eip2930(tx) => tx.into(),
            OpTxEnvelope::Eip1559(tx) => tx.into(),
            OpTxEnvelope::Eip7702(tx) => tx.into(),
            OpTxEnvelope::Deposit(tx) => tx.into(),
        }
    }
}

impl From<OpTransactionSigned> for OpTxEnvelope {
    fn from(value: OpTransactionSigned) -> Self {
        let (tx, signature, hash) = value.into_parts();
        match tx {
            OpTypedTransaction::Legacy(tx) => Signed::new_unchecked(tx, signature, hash).into(),
            OpTypedTransaction::Eip2930(tx) => Signed::new_unchecked(tx, signature, hash).into(),
            OpTypedTransaction::Eip1559(tx) => Signed::new_unchecked(tx, signature, hash).into(),
            OpTypedTransaction::Deposit(tx) => Sealed::new_unchecked(tx, hash).into(),
            OpTypedTransaction::Eip7702(tx) => Signed::new_unchecked(tx, signature, hash).into(),
        }
    }
}

impl From<OpTransactionSigned> for Signed<OpTypedTransaction> {
    fn from(value: OpTransactionSigned) -> Self {
        let (tx, sig, hash) = value.into_parts();
        Self::new_unchecked(tx, sig, hash)
    }
}

impl From<Sealed<TxDeposit>> for OpTransactionSigned {
    fn from(value: Sealed<TxDeposit>) -> Self {
        let (tx, hash) = value.into_parts();
        Self::new(OpTypedTransaction::Deposit(tx), TxDeposit::signature(), hash)
    }
}

/// A trait that represents an optimism transaction, mainly used to indicate whether or not the
/// transaction is a deposit transaction.
pub trait OpTransaction {
    /// Whether or not the transaction is a dpeosit transaction.
    fn is_deposit(&self) -> bool;
}

impl OpTransaction for OpTransactionSigned {
    fn is_deposit(&self) -> bool {
        self.is_deposit()
    }
}

impl FromRecoveredTx<OpTransactionSigned> for op_revm::OpTransaction<TxEnv> {
    fn from_recovered_tx(tx: &OpTransactionSigned, sender: Address) -> Self {
        let envelope = tx.encoded_2718();

        let base = match &tx.transaction {
            OpTypedTransaction::Legacy(tx) => TxEnv {
                gas_limit: tx.gas_limit,
                gas_price: tx.gas_price,
                gas_priority_fee: None,
                kind: tx.to,
                value: tx.value,
                data: tx.input.clone(),
                chain_id: tx.chain_id,
                nonce: tx.nonce,
                access_list: Default::default(),
                blob_hashes: Default::default(),
                max_fee_per_blob_gas: Default::default(),
                authorization_list: Default::default(),
                tx_type: 0,
                caller: sender,
            },
            OpTypedTransaction::Eip2930(tx) => TxEnv {
                gas_limit: tx.gas_limit,
                gas_price: tx.gas_price,
                gas_priority_fee: None,
                kind: tx.to,
                value: tx.value,
                data: tx.input.clone(),
                chain_id: Some(tx.chain_id),
                nonce: tx.nonce,
                access_list: tx.access_list.clone(),
                blob_hashes: Default::default(),
                max_fee_per_blob_gas: Default::default(),
                authorization_list: Default::default(),
                tx_type: 1,
                caller: sender,
            },
            OpTypedTransaction::Eip1559(tx) => TxEnv {
                gas_limit: tx.gas_limit,
                gas_price: tx.max_fee_per_gas,
                gas_priority_fee: Some(tx.max_priority_fee_per_gas),
                kind: tx.to,
                value: tx.value,
                data: tx.input.clone(),
                chain_id: Some(tx.chain_id),
                nonce: tx.nonce,
                access_list: tx.access_list.clone(),
                blob_hashes: Default::default(),
                max_fee_per_blob_gas: Default::default(),
                authorization_list: Default::default(),
                tx_type: 2,
                caller: sender,
            },
            OpTypedTransaction::Eip7702(tx) => TxEnv {
                gas_limit: tx.gas_limit,
                gas_price: tx.max_fee_per_gas,
                gas_priority_fee: Some(tx.max_priority_fee_per_gas),
                kind: TxKind::Call(tx.to),
                value: tx.value,
                data: tx.input.clone(),
                chain_id: Some(tx.chain_id),
                nonce: tx.nonce,
                access_list: tx.access_list.clone(),
                blob_hashes: Default::default(),
                max_fee_per_blob_gas: Default::default(),
                authorization_list: tx.authorization_list.clone(),
                tx_type: 4,
                caller: sender,
            },
            OpTypedTransaction::Deposit(tx) => TxEnv {
                gas_limit: tx.gas_limit,
                gas_price: 0,
                kind: tx.to,
                value: tx.value,
                data: tx.input.clone(),
                chain_id: None,
                nonce: 0,
                access_list: Default::default(),
                blob_hashes: Default::default(),
                max_fee_per_blob_gas: Default::default(),
                authorization_list: Default::default(),
                gas_priority_fee: Default::default(),
                tx_type: 126,
                caller: sender,
            },
        };

        Self {
            base,
            enveloped_tx: Some(envelope.into()),
            deposit: if let OpTypedTransaction::Deposit(tx) = &tx.transaction {
                DepositTransactionParts {
                    is_system_transaction: tx.is_system_transaction,
                    source_hash: tx.source_hash,
                    // For consistency with op-geth, we always return `0x0` for mint if it is
                    // missing This is because op-geth does not distinguish
                    // between null and 0, because this value is decoded from RLP where null is
                    // represented as 0
                    mint: Some(tx.mint.unwrap_or_default()),
                }
            } else {
                Default::default()
            },
        }
    }
}

impl InMemorySize for OpTransactionSigned {
    #[inline]
    fn size(&self) -> usize {
        mem::size_of::<TxHash>() + self.transaction.size() + mem::size_of::<Signature>()
    }
}

impl alloy_rlp::Encodable for OpTransactionSigned {
    fn encode(&self, out: &mut dyn alloy_rlp::bytes::BufMut) {
        self.network_encode(out);
    }

    fn length(&self) -> usize {
        let mut payload_length = self.encode_2718_len();
        if !self.is_legacy() {
            payload_length += Header { list: false, payload_length }.length();
        }

        payload_length
    }
}

impl alloy_rlp::Decodable for OpTransactionSigned {
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::network_decode(buf).map_err(Into::into)
    }
}

impl Encodable2718 for OpTransactionSigned {
    fn type_flag(&self) -> Option<u8> {
        if Typed2718::is_legacy(self) {
            None
        } else {
            Some(self.ty())
        }
    }

    fn encode_2718_len(&self) -> usize {
        match &self.transaction {
            OpTypedTransaction::Legacy(legacy_tx) => {
                legacy_tx.eip2718_encoded_length(&self.signature)
            }
            OpTypedTransaction::Eip2930(access_list_tx) => {
                access_list_tx.eip2718_encoded_length(&self.signature)
            }
            OpTypedTransaction::Eip1559(dynamic_fee_tx) => {
                dynamic_fee_tx.eip2718_encoded_length(&self.signature)
            }
            OpTypedTransaction::Eip7702(set_code_tx) => {
                set_code_tx.eip2718_encoded_length(&self.signature)
            }
            OpTypedTransaction::Deposit(deposit_tx) => deposit_tx.eip2718_encoded_length(),
        }
    }

    fn encode_2718(&self, out: &mut dyn alloy_rlp::BufMut) {
        let Self { transaction, signature, .. } = self;

        match &transaction {
            OpTypedTransaction::Legacy(legacy_tx) => {
                // do nothing w/ with_header
                legacy_tx.eip2718_encode(signature, out)
            }
            OpTypedTransaction::Eip2930(access_list_tx) => {
                access_list_tx.eip2718_encode(signature, out)
            }
            OpTypedTransaction::Eip1559(dynamic_fee_tx) => {
                dynamic_fee_tx.eip2718_encode(signature, out)
            }
            OpTypedTransaction::Eip7702(set_code_tx) => set_code_tx.eip2718_encode(signature, out),
            OpTypedTransaction::Deposit(deposit_tx) => deposit_tx.encode_2718(out),
        }
    }
}

impl Decodable2718 for OpTransactionSigned {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> Eip2718Result<Self> {
        match ty.try_into().map_err(|_| Eip2718Error::UnexpectedType(ty))? {
            op_alloy_consensus::OpTxType::Legacy => Err(Eip2718Error::UnexpectedType(0)),
            op_alloy_consensus::OpTxType::Eip2930 => {
                let (tx, signature, hash) = TxEip2930::rlp_decode_signed(buf)?.into_parts();
                let signed_tx = Self::new_unhashed(OpTypedTransaction::Eip2930(tx), signature);
                signed_tx.hash.get_or_init(|| hash);
                Ok(signed_tx)
            }
            op_alloy_consensus::OpTxType::Eip1559 => {
                let (tx, signature, hash) = TxEip1559::rlp_decode_signed(buf)?.into_parts();
                let signed_tx = Self::new_unhashed(OpTypedTransaction::Eip1559(tx), signature);
                signed_tx.hash.get_or_init(|| hash);
                Ok(signed_tx)
            }
            op_alloy_consensus::OpTxType::Eip7702 => {
                let (tx, signature, hash) = TxEip7702::rlp_decode_signed(buf)?.into_parts();
                let signed_tx = Self::new_unhashed(OpTypedTransaction::Eip7702(tx), signature);
                signed_tx.hash.get_or_init(|| hash);
                Ok(signed_tx)
            }
            op_alloy_consensus::OpTxType::Deposit => Ok(Self::new_unhashed(
                OpTypedTransaction::Deposit(TxDeposit::rlp_decode(buf)?),
                TxDeposit::signature(),
            )),
        }
    }

    fn fallback_decode(buf: &mut &[u8]) -> Eip2718Result<Self> {
        let (transaction, signature) = TxLegacy::rlp_decode_with_signature(buf)?;
        let signed_tx = Self::new_unhashed(OpTypedTransaction::Legacy(transaction), signature);

        Ok(signed_tx)
    }
}

impl Transaction for OpTransactionSigned {
    fn chain_id(&self) -> Option<u64> {
        self.deref().chain_id()
    }

    fn nonce(&self) -> u64 {
        self.deref().nonce()
    }

    fn gas_limit(&self) -> u64 {
        self.deref().gas_limit()
    }

    fn gas_price(&self) -> Option<u128> {
        self.deref().gas_price()
    }

    fn max_fee_per_gas(&self) -> u128 {
        self.deref().max_fee_per_gas()
    }

    fn max_priority_fee_per_gas(&self) -> Option<u128> {
        self.deref().max_priority_fee_per_gas()
    }

    fn max_fee_per_blob_gas(&self) -> Option<u128> {
        self.deref().max_fee_per_blob_gas()
    }

    fn priority_fee_or_price(&self) -> u128 {
        self.deref().priority_fee_or_price()
    }

    fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
        self.deref().effective_gas_price(base_fee)
    }

    fn effective_tip_per_gas(&self, base_fee: u64) -> Option<u128> {
        self.deref().effective_tip_per_gas(base_fee)
    }

    fn is_dynamic_fee(&self) -> bool {
        self.deref().is_dynamic_fee()
    }

    fn kind(&self) -> TxKind {
        self.deref().kind()
    }

    fn is_create(&self) -> bool {
        self.deref().is_create()
    }

    fn value(&self) -> Uint<256, 4> {
        self.deref().value()
    }

    fn input(&self) -> &Bytes {
        self.deref().input()
    }

    fn access_list(&self) -> Option<&AccessList> {
        self.deref().access_list()
    }

    fn blob_versioned_hashes(&self) -> Option<&[B256]> {
        self.deref().blob_versioned_hashes()
    }

    fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
        self.deref().authorization_list()
    }
}

impl Typed2718 for OpTransactionSigned {
    fn ty(&self) -> u8 {
        self.deref().ty()
    }
}

impl PartialEq for OpTransactionSigned {
    fn eq(&self, other: &Self) -> bool {
        self.signature == other.signature &&
            self.transaction == other.transaction &&
            self.tx_hash() == other.tx_hash()
    }
}

impl Hash for OpTransactionSigned {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.signature.hash(state);
        self.transaction.hash(state);
    }
}

#[cfg(feature = "reth-codec")]
impl reth_codecs::Compact for OpTransactionSigned {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: bytes::BufMut + AsMut<[u8]>,
    {
        let start = buf.as_mut().len();

        // Placeholder for bitflags.
        // The first byte uses 4 bits as flags: IsCompressed[1bit], TxType[2bits], Signature[1bit]
        buf.put_u8(0);

        let sig_bit = self.signature.to_compact(buf) as u8;
        let zstd_bit = self.transaction.input().len() >= 32;

        let tx_bits = if zstd_bit {
            let mut tmp = Vec::with_capacity(256);
            if cfg!(feature = "std") {
                reth_zstd_compressors::TRANSACTION_COMPRESSOR.with(|compressor| {
                    let mut compressor = compressor.borrow_mut();
                    let tx_bits = self.transaction.to_compact(&mut tmp);
                    buf.put_slice(&compressor.compress(&tmp).expect("Failed to compress"));
                    tx_bits as u8
                })
            } else {
                let mut compressor = reth_zstd_compressors::create_tx_compressor();
                let tx_bits = self.transaction.to_compact(&mut tmp);
                buf.put_slice(&compressor.compress(&tmp).expect("Failed to compress"));
                tx_bits as u8
            }
        } else {
            self.transaction.to_compact(buf) as u8
        };

        // Replace bitflags with the actual values
        buf.as_mut()[start] = sig_bit | (tx_bits << 1) | ((zstd_bit as u8) << 3);

        buf.as_mut().len() - start
    }

    fn from_compact(mut buf: &[u8], _len: usize) -> (Self, &[u8]) {
        use bytes::Buf;

        // The first byte uses 4 bits as flags: IsCompressed[1], TxType[2], Signature[1]
        let bitflags = buf.get_u8() as usize;

        let sig_bit = bitflags & 1;
        let (signature, buf) = Signature::from_compact(buf, sig_bit);

        let zstd_bit = bitflags >> 3;
        let (transaction, buf) = if zstd_bit != 0 {
            if cfg!(feature = "std") {
                reth_zstd_compressors::TRANSACTION_DECOMPRESSOR.with(|decompressor| {
                    let mut decompressor = decompressor.borrow_mut();

                    // TODO: enforce that zstd is only present at a "top" level type
                    let transaction_type = (bitflags & 0b110) >> 1;
                    let (transaction, _) = OpTypedTransaction::from_compact(
                        decompressor.decompress(buf),
                        transaction_type,
                    );

                    (transaction, buf)
                })
            } else {
                let mut decompressor = reth_zstd_compressors::create_tx_decompressor();
                let transaction_type = (bitflags & 0b110) >> 1;
                let (transaction, _) = OpTypedTransaction::from_compact(
                    decompressor.decompress(buf),
                    transaction_type,
                );

                (transaction, buf)
            }
        } else {
            let transaction_type = bitflags >> 1;
            OpTypedTransaction::from_compact(buf, transaction_type)
        };

        (Self { signature, transaction, hash: Default::default() }, buf)
    }
}

#[cfg(any(test, feature = "arbitrary"))]
impl<'a> arbitrary::Arbitrary<'a> for OpTransactionSigned {
    fn arbitrary(u: &mut arbitrary::Unstructured<'a>) -> arbitrary::Result<Self> {
        #[allow(unused_mut)]
        let mut transaction = OpTypedTransaction::arbitrary(u)?;

        let secp = secp256k1::Secp256k1::new();
        let key_pair = secp256k1::Keypair::new(&secp, &mut rand::thread_rng());
        let signature = reth_primitives_traits::crypto::secp256k1::sign_message(
            B256::from_slice(&key_pair.secret_bytes()[..]),
            signature_hash(&transaction),
        )
        .unwrap();

        // Both `Some(0)` and `None` values are encoded as empty string byte. This introduces
        // ambiguity in roundtrip tests. Patch the mint value of deposit transaction here, so that
        // it's `None` if zero.
        if let OpTypedTransaction::Deposit(ref mut tx_deposit) = transaction {
            if tx_deposit.mint == Some(0) {
                tx_deposit.mint = None;
            }
        }

        let signature = if is_deposit(&transaction) { TxDeposit::signature() } else { signature };

        Ok(Self::new_unhashed(transaction, signature))
    }
}

/// Calculates the signing hash for the transaction.
fn signature_hash(tx: &OpTypedTransaction) -> B256 {
    match tx {
        OpTypedTransaction::Legacy(tx) => tx.signature_hash(),
        OpTypedTransaction::Eip2930(tx) => tx.signature_hash(),
        OpTypedTransaction::Eip1559(tx) => tx.signature_hash(),
        OpTypedTransaction::Eip7702(tx) => tx.signature_hash(),
        OpTypedTransaction::Deposit(_) => B256::ZERO,
    }
}

/// Returns `true` if transaction is deposit transaction.
pub const fn is_deposit(tx: &OpTypedTransaction) -> bool {
    matches!(tx, OpTypedTransaction::Deposit(_))
}

impl From<OpPooledTransaction> for OpTransactionSigned {
    fn from(value: OpPooledTransaction) -> Self {
        match value {
            OpPooledTransaction::Legacy(tx) => tx.into(),
            OpPooledTransaction::Eip2930(tx) => tx.into(),
            OpPooledTransaction::Eip1559(tx) => tx.into(),
            OpPooledTransaction::Eip7702(tx) => tx.into(),
        }
    }
}

impl TryFrom<OpTransactionSigned> for OpPooledTransaction {
    type Error = TransactionConversionError;

    fn try_from(value: OpTransactionSigned) -> Result<Self, Self::Error> {
        let hash = *value.tx_hash();
        let OpTransactionSigned { hash: _, signature, transaction } = value;

        match transaction {
            OpTypedTransaction::Legacy(tx) => {
                Ok(Self::Legacy(Signed::new_unchecked(tx, signature, hash)))
            }
            OpTypedTransaction::Eip2930(tx) => {
                Ok(Self::Eip2930(Signed::new_unchecked(tx, signature, hash)))
            }
            OpTypedTransaction::Eip1559(tx) => {
                Ok(Self::Eip1559(Signed::new_unchecked(tx, signature, hash)))
            }
            OpTypedTransaction::Eip7702(tx) => {
                Ok(Self::Eip7702(Signed::new_unchecked(tx, signature, hash)))
            }
            OpTypedTransaction::Deposit(_) => Err(TransactionConversionError::UnsupportedForP2P),
        }
    }
}

impl DepositTransaction for OpTransactionSigned {
    fn source_hash(&self) -> Option<B256> {
        match &self.transaction {
            OpTypedTransaction::Deposit(tx) => Some(tx.source_hash),
            _ => None,
        }
    }

    fn mint(&self) -> Option<u128> {
        match &self.transaction {
            OpTypedTransaction::Deposit(tx) => tx.mint,
            _ => None,
        }
    }

    fn is_system_transaction(&self) -> bool {
        self.is_deposit()
    }
}

/// Bincode-compatible transaction type serde implementations.
#[cfg(feature = "serde-bincode-compat")]
pub mod serde_bincode_compat {
    use alloy_consensus::transaction::serde_bincode_compat::{
        TxEip1559, TxEip2930, TxEip7702, TxLegacy,
    };
    use alloy_primitives::{PrimitiveSignature as Signature, TxHash};
    use reth_primitives_traits::{serde_bincode_compat::SerdeBincodeCompat, SignedTransaction};
    use serde::{Deserialize, Serialize};

    /// Bincode-compatible [`super::OpTypedTransaction`] serde implementation.
    #[derive(Debug, Serialize, Deserialize)]
    #[allow(missing_docs)]
    enum OpTypedTransaction<'a> {
        Legacy(TxLegacy<'a>),
        Eip2930(TxEip2930<'a>),
        Eip1559(TxEip1559<'a>),
        Eip7702(TxEip7702<'a>),
        Deposit(op_alloy_consensus::serde_bincode_compat::TxDeposit<'a>),
    }

    impl<'a> From<&'a super::OpTypedTransaction> for OpTypedTransaction<'a> {
        fn from(value: &'a super::OpTypedTransaction) -> Self {
            match value {
                super::OpTypedTransaction::Legacy(tx) => Self::Legacy(TxLegacy::from(tx)),
                super::OpTypedTransaction::Eip2930(tx) => Self::Eip2930(TxEip2930::from(tx)),
                super::OpTypedTransaction::Eip1559(tx) => Self::Eip1559(TxEip1559::from(tx)),
                super::OpTypedTransaction::Eip7702(tx) => Self::Eip7702(TxEip7702::from(tx)),
                super::OpTypedTransaction::Deposit(tx) => {
                    Self::Deposit(op_alloy_consensus::serde_bincode_compat::TxDeposit::from(tx))
                }
            }
        }
    }

    impl<'a> From<OpTypedTransaction<'a>> for super::OpTypedTransaction {
        fn from(value: OpTypedTransaction<'a>) -> Self {
            match value {
                OpTypedTransaction::Legacy(tx) => Self::Legacy(tx.into()),
                OpTypedTransaction::Eip2930(tx) => Self::Eip2930(tx.into()),
                OpTypedTransaction::Eip1559(tx) => Self::Eip1559(tx.into()),
                OpTypedTransaction::Eip7702(tx) => Self::Eip7702(tx.into()),
                OpTypedTransaction::Deposit(tx) => Self::Deposit(tx.into()),
            }
        }
    }

    /// Bincode-compatible [`super::OpTransactionSigned`] serde implementation.
    #[derive(Debug, Serialize, Deserialize)]
    pub struct OpTransactionSigned<'a> {
        hash: TxHash,
        signature: Signature,
        transaction: OpTypedTransaction<'a>,
    }

    impl<'a> From<&'a super::OpTransactionSigned> for OpTransactionSigned<'a> {
        fn from(value: &'a super::OpTransactionSigned) -> Self {
            Self {
                hash: *value.tx_hash(),
                signature: value.signature,
                transaction: OpTypedTransaction::from(&value.transaction),
            }
        }
    }

    impl<'a> From<OpTransactionSigned<'a>> for super::OpTransactionSigned {
        fn from(value: OpTransactionSigned<'a>) -> Self {
            Self {
                hash: value.hash.into(),
                signature: value.signature,
                transaction: value.transaction.into(),
            }
        }
    }

    impl SerdeBincodeCompat for super::OpTransactionSigned {
        type BincodeRepr<'a> = OpTransactionSigned<'a>;

        fn as_repr(&self) -> Self::BincodeRepr<'_> {
            self.into()
        }

        fn from_repr(repr: Self::BincodeRepr<'_>) -> Self {
            repr.into()
        }
    }
}
