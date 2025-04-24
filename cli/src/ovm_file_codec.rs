use alloy_consensus::{
    transaction::{from_eip155_value, RlpEcdsaDecodableTx, RlpEcdsaEncodableTx},
    Header, TxEip1559, TxEip2930, TxEip7702, TxLegacy,
};
use alloy_eips::{
    eip2718::{Decodable2718, Eip2718Error, Eip2718Result, Encodable2718},
    eip4895::Withdrawals,
    Typed2718,
};
use alloy_primitives::{
    bytes::{Buf, BytesMut},
    keccak256, Signature, TxHash, B256, U256,
};
use alloy_rlp::{Decodable, Error as RlpError, RlpDecodable};
use derive_more::{AsRef, Deref};
use op_alloy_consensus::{OpTxType, OpTypedTransaction, TxDeposit};
use reth_downloaders::file_client::FileClientError;
use serde::{Deserialize, Serialize};
use tokio_util::codec::Decoder;

#[expect(dead_code)]
/// Specific codec for reading raw block bodies from a file
/// with optimism-specific signature handling
pub(crate) struct OvmBlockFileCodec;

impl Decoder for OvmBlockFileCodec {
    type Item = OvmBlock;
    type Error = FileClientError;

    fn decode(&mut self, src: &mut BytesMut) -> Result<Option<Self::Item>, Self::Error> {
        if src.is_empty() {
            return Ok(None);
        }

        let buf_slice = &mut src.as_ref();
        let body =
            OvmBlock::decode(buf_slice).map_err(|err| FileClientError::Rlp(err, src.to_vec()))?;
        src.advance(src.len() - buf_slice.len());

        Ok(Some(body))
    }
}

/// OVM block, same as EVM block but with different transaction signature handling
/// Pre-bedrock system transactions on Optimism were sent from the zero address
/// with an empty signature,
#[derive(Debug, Clone, PartialEq, Eq, RlpDecodable)]
pub struct OvmBlock {
    /// Block header
    pub header: Header,
    /// Block body
    pub body: OvmBlockBody,
}

impl OvmBlock {
    /// Decodes a `Block` from the given byte slice.
    pub fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let header = Header::decode(buf)?;
        let body = OvmBlockBody::decode(buf)?;
        Ok(Self { header, body })
    }
}

/// The body of a block for OVM
#[derive(Debug, Clone, PartialEq, Eq, Default, RlpDecodable)]
#[rlp(trailing)]
pub struct OvmBlockBody {
    /// Transactions in the block
    pub transactions: Vec<OvmTransactionSigned>,
    /// Uncle headers for the given block
    pub ommers: Vec<Header>,
    /// Withdrawals in the block.
    pub withdrawals: Option<Withdrawals>,
}

/// Signed transaction pre bedrock.
#[derive(Debug, Clone, PartialEq, Eq, Hash, AsRef, Deref, Serialize, Deserialize)]
pub struct OvmTransactionSigned {
    /// Transaction hash
    pub hash: TxHash,
    /// The transaction signature values
    pub signature: Signature,
    /// Raw transaction info
    #[deref]
    #[as_ref]
    pub transaction: OpTypedTransaction,
}

impl AsRef<Self> for OvmTransactionSigned {
    fn as_ref(&self) -> &Self {
        self
    }
}

impl OvmTransactionSigned {
    /// Calculate transaction hash, eip2728 transaction does not contain rlp header and start with
    /// tx type.
    pub fn recalculate_hash(&self) -> B256 {
        keccak256(self.encoded_2718())
    }

    /// Create a new signed transaction from a transaction and its signature.
    ///
    /// This will also calculate the transaction hash using its encoding.
    pub fn from_transaction_and_signature(
        transaction: OpTypedTransaction,
        signature: Signature,
    ) -> Self {
        let mut initial_tx = Self { transaction, hash: Default::default(), signature };
        initial_tx.hash = initial_tx.recalculate_hash();
        initial_tx
    }

    /// Decodes legacy transaction from the data buffer into a tuple.
    ///
    /// This expects `rlp(legacy_tx)`
    ///
    /// Refer to the docs for [`Self::decode_rlp_legacy_transaction`] for details on the exact
    /// format expected.
    pub(crate) fn decode_rlp_legacy_transaction_tuple(
        data: &mut &[u8],
    ) -> alloy_rlp::Result<(TxLegacy, TxHash, Signature)> {
        let original_encoding = *data;

        let header = alloy_rlp::Header::decode(data)?;
        let remaining_len = data.len();

        let transaction_payload_len = header.payload_length;

        if transaction_payload_len > remaining_len {
            return Err(RlpError::InputTooShort);
        }

        let mut transaction = TxLegacy {
            nonce: Decodable::decode(data)?,
            gas_price: Decodable::decode(data)?,
            gas_limit: Decodable::decode(data)?,
            to: Decodable::decode(data)?,
            value: Decodable::decode(data)?,
            input: Decodable::decode(data)?,
            chain_id: None,
        };

        let v = Decodable::decode(data)?;
        let r: U256 = Decodable::decode(data)?;
        let s: U256 = Decodable::decode(data)?;

        let tx_length = header.payload_length + header.length();
        let hash = keccak256(&original_encoding[..tx_length]);

        // Handle both pre-bedrock and regular cases
        let (signature, chain_id) = if v == 0 && r.is_zero() && s.is_zero() {
            // Pre-bedrock system transactions case
            (Signature::new(r, s, false), None)
        } else {
            // Regular transaction case
            let (parity, chain_id) = from_eip155_value(v)
                .ok_or(alloy_rlp::Error::Custom("invalid parity for legacy transaction"))?;
            (Signature::new(r, s, parity), chain_id)
        };

        // Set chain ID and verify length
        transaction.chain_id = chain_id;
        let decoded = remaining_len - data.len();
        if decoded != transaction_payload_len {
            return Err(RlpError::UnexpectedLength);
        }

        Ok((transaction, hash, signature))
    }

    /// Decodes legacy transaction from the data buffer.
    ///
    /// This should be used _only_ be used in general transaction decoding methods, which have
    /// already ensured that the input is a legacy transaction with the following format:
    /// `rlp(legacy_tx)`
    ///
    /// Legacy transactions are encoded as lists, so the input should start with a RLP list header.
    ///
    /// This expects `rlp(legacy_tx)`
    // TODO: make buf advancement semantics consistent with `decode_enveloped_typed_transaction`,
    // so decoding methods do not need to manually advance the buffer
    pub fn decode_rlp_legacy_transaction(data: &mut &[u8]) -> alloy_rlp::Result<Self> {
        let (transaction, hash, signature) = Self::decode_rlp_legacy_transaction_tuple(data)?;
        let signed = Self { transaction: OpTypedTransaction::Legacy(transaction), hash, signature };
        Ok(signed)
    }
}

impl Decodable for OvmTransactionSigned {
    /// This `Decodable` implementation only supports decoding rlp encoded transactions as it's used
    /// by p2p.
    ///
    /// The p2p encoding format always includes an RLP header, although the type RLP header depends
    /// on whether or not the transaction is a legacy transaction.
    ///
    /// If the transaction is a legacy transaction, it is just encoded as a RLP list:
    /// `rlp(tx-data)`.
    ///
    /// If the transaction is a typed transaction, it is encoded as a RLP string:
    /// `rlp(tx-type || rlp(tx-data))`
    ///
    /// This can be used for decoding all signed transactions in p2p `BlockBodies` responses.
    ///
    /// This cannot be used for decoding EIP-4844 transactions in p2p `PooledTransactions`, since
    /// the EIP-4844 variant of [`OvmTransactionSigned`] does not include the blob sidecar.
    ///
    /// For a method suitable for decoding pooled transactions, see \[`PooledTransaction`\].
    ///
    /// CAUTION: Due to a quirk in [`Header::decode`], this method will succeed even if a typed
    /// transaction is encoded in this format, and does not start with a RLP header:
    /// `tx-type || rlp(tx-data)`.
    ///
    /// This is because [`Header::decode`] does not advance the buffer, and returns a length-1
    /// string header if the first byte is less than `0xf7`.
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        Self::network_decode(buf).map_err(Into::into)
    }
}

impl Typed2718 for OvmTransactionSigned {
    fn ty(&self) -> u8 {
        self.transaction.tx_type() as u8
    }
}

impl Encodable2718 for OvmTransactionSigned {
    fn type_flag(&self) -> Option<u8> {
        match self.transaction.tx_type() {
            OpTxType::Legacy => None,
            tx_type => Some(tx_type as u8),
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
        match &self.transaction {
            OpTypedTransaction::Legacy(tx) => tx.eip2718_encode(&self.signature, out),
            OpTypedTransaction::Eip2930(tx) => tx.eip2718_encode(&self.signature, out),
            OpTypedTransaction::Eip1559(tx) => tx.eip2718_encode(&self.signature, out),
            OpTypedTransaction::Eip7702(tx) => tx.eip2718_encode(&self.signature, out),
            OpTypedTransaction::Deposit(tx) => tx.encode_2718(out),
        }
    }
}

impl Decodable2718 for OvmTransactionSigned {
    fn typed_decode(ty: u8, buf: &mut &[u8]) -> Eip2718Result<Self> {
        match ty.try_into().map_err(|_| Eip2718Error::UnexpectedType(ty))? {
            OpTxType::Legacy => Err(Eip2718Error::UnexpectedType(0)),
            OpTxType::Eip2930 => {
                let (tx, signature, hash) = TxEip2930::rlp_decode_signed(buf)?.into_parts();
                Ok(Self { transaction: OpTypedTransaction::Eip2930(tx), signature, hash })
            }
            OpTxType::Eip1559 => {
                let (tx, signature, hash) = TxEip1559::rlp_decode_signed(buf)?.into_parts();
                Ok(Self { transaction: OpTypedTransaction::Eip1559(tx), signature, hash })
            }
            OpTxType::Eip7702 => {
                let (tx, signature, hash) = TxEip7702::rlp_decode_signed(buf)?.into_parts();
                Ok(Self { transaction: OpTypedTransaction::Eip7702(tx), signature, hash })
            }
            OpTxType::Deposit => Ok(Self::from_transaction_and_signature(
                OpTypedTransaction::Deposit(TxDeposit::rlp_decode(buf)?),
                TxDeposit::signature(),
            )),
        }
    }

    fn fallback_decode(buf: &mut &[u8]) -> Eip2718Result<Self> {
        Ok(Self::decode_rlp_legacy_transaction(buf)?)
    }
}

#[cfg(test)]
mod tests {
    use crate::ovm_file_codec::OvmTransactionSigned;
    use alloy_consensus::Typed2718;
    use alloy_primitives::{address, b256, hex, TxKind, U256};
    use op_alloy_consensus::OpTypedTransaction;
    const DEPOSIT_FUNCTION_SELECTOR: [u8; 4] = [0xb6, 0xb5, 0x5f, 0x25];
    use alloy_rlp::Decodable;

    #[test]
    fn test_decode_legacy_transactions() {
        // Test Case 1: contract deposit - regular L2 transaction calling deposit() function
        // tx: https://optimistic.etherscan.io/getRawTx?tx=0x7860252963a2df21113344f323035ef59648638a571eef742e33d789602c7a1c
        let deposit_tx_bytes = hex!(
            "f88881f0830f481c830c6e4594a75127121d28a9bf848f3b70e7eea26570aa770080a4b6b55f2500000000000000000000000000000000000000000000000000000000000710b238a0d5c622d92ddf37f9c18a3465a572f74d8b1aeaf50c1cfb10b3833242781fd45fa02c4f1d5819bf8b70bf651e7a063b7db63c55bd336799c6ae3e5bc72ad6ef3def"
        );
        let deposit_decoded = OvmTransactionSigned::decode(&mut &deposit_tx_bytes[..]).unwrap();

        // Verify deposit transaction
        let deposit_tx = match &deposit_decoded.transaction {
            OpTypedTransaction::Legacy(ref tx) => tx,
            _ => panic!("Expected legacy transaction for NFT deposit"),
        };

        assert_eq!(
            deposit_tx.to,
            TxKind::Call(address!("0xa75127121d28a9bf848f3b70e7eea26570aa7700"))
        );
        assert_eq!(deposit_tx.nonce, 240);
        assert_eq!(deposit_tx.gas_price, 1001500);
        assert_eq!(deposit_tx.gas_limit, 814661);
        assert_eq!(deposit_tx.value, U256::ZERO);
        assert_eq!(&deposit_tx.input.as_ref()[0..4], DEPOSIT_FUNCTION_SELECTOR);
        assert_eq!(deposit_tx.chain_id, Some(10));
        assert_eq!(
            deposit_decoded.signature.r(),
            U256::from_str_radix(
                "d5c622d92ddf37f9c18a3465a572f74d8b1aeaf50c1cfb10b3833242781fd45f",
                16
            )
            .unwrap()
        );
        assert_eq!(
            deposit_decoded.signature.s(),
            U256::from_str_radix(
                "2c4f1d5819bf8b70bf651e7a063b7db63c55bd336799c6ae3e5bc72ad6ef3def",
                16
            )
            .unwrap()
        );

        // Test Case 2: pre-bedrock system transaction from block 105235052
        // tx: https://optimistic.etherscan.io/getRawTx?tx=0xe20b11349681dd049f8df32f5cdbb4c68d46b537685defcd86c7fa42cfe75b9e
        let system_tx_bytes = hex!(
            "f9026c830d899383124f808302a77e94a0cc33dd6f4819d473226257792afe230ec3c67f80b902046c459a280000000000000000000000004d73adb72bc3dd368966edd0f0b2148401a178e2000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000647fac7f00000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000084704316e5000000000000000000000000000000000000000000000000000000000000006e10975631049de3c008989b0d8c19fc720dc556ca01abfbd794c6eb5075dd000d000000000000000000000000000000000000000000000000000000000000001410975631049de3c008989b0d8c19fc720dc556ca01abfbd794c6eb5075dd000d000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000082a39325251d44e11f3b6d92f9382438eb6c8b5068d4a488d4f177b26f2ca20db34ae53467322852afcc779f25eafd124c5586f54b9026497ba934403d4c578e3c1b5aa754c918ee2ecd25402df656c2419717e4017a7aecb84af3914fd3c7bf6930369c4e6ff76950246b98e354821775f02d33cdbee5ef6aed06c15b75691692d31c00000000000000000000000000000000000000000000000000000000000038a0e8991e95e66d809f4b6fb0af27c31368ca0f30e657165c428aa681ec5ea25bbea013ed325bd97365087ec713e9817d252b59113ea18430b71a5890c4eeb6b9efc4"
        );
        let system_decoded = OvmTransactionSigned::decode(&mut &system_tx_bytes[..]).unwrap();

        // Verify system transaction
        assert!(system_decoded.is_legacy());

        let system_tx = match &system_decoded.transaction {
            OpTypedTransaction::Legacy(ref tx) => tx,
            _ => panic!("Expected Legacy transaction"),
        };

        assert_eq!(system_tx.nonce, 887187);
        assert_eq!(system_tx.gas_price, 1200000);
        assert_eq!(system_tx.gas_limit, 173950);
        assert_eq!(
            system_tx.to,
            TxKind::Call(address!("0xa0cc33dd6f4819d473226257792afe230ec3c67f"))
        );
        assert_eq!(system_tx.value, U256::ZERO);
        assert_eq!(system_tx.chain_id, Some(10));

        assert_eq!(
            system_decoded.signature.r(),
            U256::from_str_radix(
                "e8991e95e66d809f4b6fb0af27c31368ca0f30e657165c428aa681ec5ea25bbe",
                16
            )
            .unwrap()
        );
        assert_eq!(
            system_decoded.signature.s(),
            U256::from_str_radix(
                "13ed325bd97365087ec713e9817d252b59113ea18430b71a5890c4eeb6b9efc4",
                16
            )
            .unwrap()
        );
        assert_eq!(
            system_decoded.hash,
            b256!("0xe20b11349681dd049f8df32f5cdbb4c68d46b537685defcd86c7fa42cfe75b9e")
        );
    }
}
