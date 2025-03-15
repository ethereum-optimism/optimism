//! Codec for reading raw receipts from a file.

use alloy_consensus::Receipt;
use alloy_primitives::{
    bytes::{Buf, BytesMut},
    Address, Bloom, Bytes, Log, B256,
};
use alloy_rlp::{Decodable, RlpDecodable};
use op_alloy_consensus::{OpDepositReceipt, OpTxType};
use reth_optimism_primitives::OpReceipt;
use tokio_util::codec::Decoder;

use reth_downloaders::{file_client::FileClientError, receipt_file_client::ReceiptWithBlockNumber};

/// Codec for reading raw receipts from a file.
///
/// If using with [`FramedRead`](tokio_util::codec::FramedRead), the user should make sure the
/// framed reader has capacity for the entire receipts file. Otherwise, the decoder will return
/// [`InputTooShort`](alloy_rlp::Error::InputTooShort), because RLP receipts can only be
/// decoded if the internal buffer is large enough to contain the entire receipt.
///
/// Without ensuring the framed reader has capacity for the entire file, a receipt is likely to
/// fall across two read buffers, the decoder will not be able to decode the receipt, which will
/// cause it to fail.
///
/// It's recommended to use [`with_capacity`](tokio_util::codec::FramedRead::with_capacity) to set
/// the capacity of the framed reader to the size of the file.
#[derive(Debug)]
pub struct OpGethReceiptFileCodec<R = Receipt>(core::marker::PhantomData<R>);

impl<R> Default for OpGethReceiptFileCodec<R> {
    fn default() -> Self {
        Self(Default::default())
    }
}

impl<R> Decoder for OpGethReceiptFileCodec<R>
where
    R: TryFrom<OpGethReceipt, Error: Into<FileClientError>>,
{
    type Item = Option<ReceiptWithBlockNumber<R>>;
    type Error = FileClientError;

    fn decode(&mut self, src: &mut BytesMut) -> Result<Option<Self::Item>, Self::Error> {
        if src.is_empty() {
            return Ok(None)
        }

        let buf_slice = &mut src.as_ref();
        let receipt = OpGethReceiptContainer::decode(buf_slice)
            .map_err(|err| Self::Error::Rlp(err, src.to_vec()))?
            .0;
        src.advance(src.len() - buf_slice.len());

        Ok(Some(
            receipt
                .map(|receipt| {
                    let number = receipt.block_number;
                    receipt
                        .try_into()
                        .map_err(Into::into)
                        .map(|receipt| ReceiptWithBlockNumber { receipt, number })
                })
                .transpose()?,
        ))
    }
}

/// See <https://github.com/testinprod-io/op-geth/pull/1>
#[derive(Debug, PartialEq, Eq, RlpDecodable)]
pub struct OpGethReceipt {
    tx_type: u8,
    post_state: Bytes,
    status: u64,
    cumulative_gas_used: u64,
    bloom: Bloom,
    /// <https://github.com/testinprod-io/op-geth/blob/29062eb0fac595eeeddd3a182a25326405c66e05/core/types/log.go#L67-L72>
    logs: Vec<Log>,
    tx_hash: B256,
    contract_address: Address,
    gas_used: u64,
    block_hash: B256,
    block_number: u64,
    transaction_index: u32,
    l1_gas_price: u64,
    l1_gas_used: u64,
    l1_fee: u64,
    fee_scalar: String,
}

#[derive(Debug, PartialEq, Eq, RlpDecodable)]
#[rlp(trailing)]
struct OpGethReceiptContainer(Option<OpGethReceipt>);

impl TryFrom<OpGethReceipt> for OpReceipt {
    type Error = FileClientError;

    fn try_from(exported_receipt: OpGethReceipt) -> Result<Self, Self::Error> {
        let OpGethReceipt { tx_type, status, cumulative_gas_used, logs, .. } = exported_receipt;

        let tx_type = OpTxType::try_from(tx_type.to_be_bytes()[0])
            .map_err(|e| FileClientError::Rlp(e.into(), vec![tx_type]))?;

        let receipt =
            alloy_consensus::Receipt { status: (status != 0).into(), cumulative_gas_used, logs };

        match tx_type {
            OpTxType::Legacy => Ok(Self::Legacy(receipt)),
            OpTxType::Eip2930 => Ok(Self::Eip2930(receipt)),
            OpTxType::Eip1559 => Ok(Self::Eip1559(receipt)),
            OpTxType::Eip7702 => Ok(Self::Eip7702(receipt)),
            OpTxType::Deposit => Ok(Self::Deposit(OpDepositReceipt {
                inner: receipt,
                deposit_nonce: None,
                deposit_receipt_version: None,
            })),
        }
    }
}

#[cfg(test)]
pub(crate) mod test {
    use alloy_consensus::{Receipt, TxReceipt};
    use alloy_primitives::{address, b256, hex, LogData};

    use super::*;

    pub(crate) const HACK_RECEIPT_ENCODED_BLOCK_1: &[u8] = &hex!("f9030ff9030c8080018303183db9010000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000400000000000100000000000000200000000002000000000000001000000000000000000004000000000000000000000000000040000400000100400000000000000100000000000000000000000000000020000000000000000000000000000000000000000000000001000000000000000000000100000000000000000000000000000000000000000000000000000000000000088000000080000000000010000000000000000000000000000800008000120000000000000000000000000000000002000f90197f89b948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff863a00109fc6f55cf40689f02fbaad7af7fe7bbac8a3d2186600afc7d3e10cac60271a00000000000000000000000000000000000000000000000000000000000014218a000000000000000000000000070b17c0fe982ab4a7ac17a4c25485643151a1f2da000000000000000000000000000000000000000000000000000000000618d8837f89c948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff884a092e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68ca000000000000000000000000000000000000000000000000000000000d0e3ebf0a00000000000000000000000000000000000000000000000000000000000014218a000000000000000000000000070b17c0fe982ab4a7ac17a4c25485643151a1f2d80f85a948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff842a0fe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234fa000000000000000000000000000000000000000000000007edc6ca0bb6834800080a05e77a04531c7c107af1882d76cbff9486d0a9aa53701c30888509d4f5f2b003a9400000000000000000000000000000000000000008303183da0bee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e8754530180018212c2821c2383312e35");

    pub(crate) const HACK_RECEIPT_ENCODED_BLOCK_2: &[u8] = &hex!("f90271f9026e8080018301c60db9010000080000000200000000000000000008000000000000000000000100008000000000000000000000000000000000000000000000000000000000400000000000100000000000000000000000020000000000000000000000000000000000004000000000000000000000000000000000400000000400000000000000100000000000000000000000000000020000000000000000000000000000000000000000100000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000008400000000000000000010000000000000000020000000020000000000000000000000000000000000000000000002000f8faf89c948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff884a092e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68ca000000000000000000000000000000000000000000000000000000000d0ea0e40a00000000000000000000000000000000000000000000000000000000000014218a0000000000000000000000000e5e7492282fd1e3bfac337a0beccd29b15b7b24080f85a948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff842a0fe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234fa000000000000000000000000000000000000000000000007eda7867e0c7d4800080a0af6ed8a6864d44989adc47c84f6fe0aeb1819817505c42cde6cbbcd5e14dd3179400000000000000000000000000000000000000008301c60da045fd6ce41bb8ebb2bccdaa92dd1619e287704cb07722039901a7eba63dea1d130280018212c2821c2383312e35");

    pub(crate) const HACK_RECEIPT_ENCODED_BLOCK_3: &[u8] = &hex!("f90271f9026e8080018301c60db9010000000000000000000000000000000000000000400000000000000000008000000000000000000000000000000000004000000000000000000000400004000000100000000000000000000000000000000000000000000000000000000000004000000000000000000000040000000000400080000400000000000000100000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000000008100000000000000000000000000000000000004000000000000000000000000008000000000000000000010000000000000000000000000000400000000000000001000000000000000000000000002000f8faf89c948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff884a092e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68ca000000000000000000000000000000000000000000000000000000000d101e54ba00000000000000000000000000000000000000000000000000000000000014218a0000000000000000000000000fa011d8d6c26f13abe2cefed38226e401b2b8a9980f85a948ce8c13d816fe6daf12d6fd9e4952e1fc88850aff842a0fe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234fa000000000000000000000000000000000000000000000007ed8842f062774800080a08fab01dcec1da547e90a77597999e9153ff788fa6451d1cc942064427bd995019400000000000000000000000000000000000000008301c60da0da4509fe0ca03202ddbe4f68692c132d689ee098433691040ece18c3a45d44c50380018212c2821c2383312e35");

    fn hack_receipt_1() -> OpGethReceipt {
        let receipt = receipt_block_1();

        OpGethReceipt {
            tx_type: receipt.receipt.tx_type() as u8,
            post_state: Bytes::default(),
            status: receipt.receipt.status() as u64,
            cumulative_gas_used: receipt.receipt.cumulative_gas_used(),
            bloom: Bloom::from(hex!("00000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000400000000000100000000000000200000000002000000000000001000000000000000000004000000000000000000000000000040000400000100400000000000000100000000000000000000000000000020000000000000000000000000000000000000000000000001000000000000000000000100000000000000000000000000000000000000000000000000000000000000088000000080000000000010000000000000000000000000000800008000120000000000000000000000000000000002000")),
            logs: receipt.receipt.logs().to_vec(),
            tx_hash: b256!("0x5e77a04531c7c107af1882d76cbff9486d0a9aa53701c30888509d4f5f2b003a"), contract_address: address!("0x0000000000000000000000000000000000000000"), gas_used: 202813,
            block_hash: b256!("0xbee7192e575af30420cae0c7776304ac196077ee72b048970549e4f08e875453"),
            block_number: receipt.number,
            transaction_index: 0,
            l1_gas_price: 1,
            l1_gas_used: 4802,
            l1_fee: 7203,
            fee_scalar: String::from("1.5")
        }
    }

    pub(crate) fn receipt_block_1() -> ReceiptWithBlockNumber<OpReceipt> {
        let log_1 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0x0109fc6f55cf40689f02fbaad7af7fe7bbac8a3d2186600afc7d3e10cac60271"),
                    b256!("0x0000000000000000000000000000000000000000000000000000000000014218"),
                    b256!("0x00000000000000000000000070b17c0fe982ab4a7ac17a4c25485643151a1f2d"),
                ],
                Bytes::from(hex!(
                    "00000000000000000000000000000000000000000000000000000000618d8837"
                )),
            )
            .unwrap(),
        };

        let log_2 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0x92e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68c"),
                    b256!("0x00000000000000000000000000000000000000000000000000000000d0e3ebf0"),
                    b256!("0x0000000000000000000000000000000000000000000000000000000000014218"),
                    b256!("0x00000000000000000000000070b17c0fe982ab4a7ac17a4c25485643151a1f2d"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let log_3 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0xfe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234f"),
                    b256!("0x00000000000000000000000000000000000000000000007edc6ca0bb68348000"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let receipt = OpReceipt::Legacy(Receipt {
            status: true.into(),
            cumulative_gas_used: 202813,
            logs: vec![log_1, log_2, log_3],
        });

        ReceiptWithBlockNumber { receipt, number: 1 }
    }

    pub(crate) fn receipt_block_2() -> ReceiptWithBlockNumber<OpReceipt> {
        let log_1 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0x92e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68c"),
                    b256!("0x00000000000000000000000000000000000000000000000000000000d0ea0e40"),
                    b256!("0x0000000000000000000000000000000000000000000000000000000000014218"),
                    b256!("0x000000000000000000000000e5e7492282fd1e3bfac337a0beccd29b15b7b240"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let log_2 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0xfe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234f"),
                    b256!("0x00000000000000000000000000000000000000000000007eda7867e0c7d48000"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let receipt = OpReceipt::Legacy(Receipt {
            status: true.into(),
            cumulative_gas_used: 116237,
            logs: vec![log_1, log_2],
        });

        ReceiptWithBlockNumber { receipt, number: 2 }
    }

    pub(crate) fn receipt_block_3() -> ReceiptWithBlockNumber<OpReceipt> {
        let log_1 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0x92e98423f8adac6e64d0608e519fd1cefb861498385c6dee70d58fc926ddc68c"),
                    b256!("0x00000000000000000000000000000000000000000000000000000000d101e54b"),
                    b256!("0x0000000000000000000000000000000000000000000000000000000000014218"),
                    b256!("0x000000000000000000000000fa011d8d6c26f13abe2cefed38226e401b2b8a99"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let log_2 = Log {
            address: address!("0x8ce8c13d816fe6daf12d6fd9e4952e1fc88850af"),
            data: LogData::new(
                vec![
                    b256!("0xfe25c73e3b9089fac37d55c4c7efcba6f04af04cebd2fc4d6d7dbb07e1e5234f"),
                    b256!("0x00000000000000000000000000000000000000000000007ed8842f0627748000"),
                ],
                Bytes::default(),
            )
            .unwrap(),
        };

        let receipt = OpReceipt::Legacy(Receipt {
            status: true.into(),
            cumulative_gas_used: 116237,
            logs: vec![log_1, log_2],
        });

        ReceiptWithBlockNumber { receipt, number: 3 }
    }

    #[test]
    fn decode_hack_receipt() {
        let receipt = hack_receipt_1();

        let decoded = OpGethReceiptContainer::decode(&mut &HACK_RECEIPT_ENCODED_BLOCK_1[..])
            .unwrap()
            .0
            .unwrap();

        assert_eq!(receipt, decoded);
    }

    #[test]
    fn receipts_codec() {
        // rig

        let mut receipt_1_to_3 = HACK_RECEIPT_ENCODED_BLOCK_1.to_vec();
        receipt_1_to_3.extend_from_slice(HACK_RECEIPT_ENCODED_BLOCK_2);
        receipt_1_to_3.extend_from_slice(HACK_RECEIPT_ENCODED_BLOCK_3);

        let encoded = &mut BytesMut::from(&receipt_1_to_3[..]);

        let mut codec = OpGethReceiptFileCodec::default();

        // test

        let first_decoded_receipt = codec.decode(encoded).unwrap().unwrap().unwrap();

        assert_eq!(receipt_block_1(), first_decoded_receipt);

        let second_decoded_receipt = codec.decode(encoded).unwrap().unwrap().unwrap();

        assert_eq!(receipt_block_2(), second_decoded_receipt);

        let third_decoded_receipt = codec.decode(encoded).unwrap().unwrap().unwrap();

        assert_eq!(receipt_block_3(), third_decoded_receipt);
    }
}
