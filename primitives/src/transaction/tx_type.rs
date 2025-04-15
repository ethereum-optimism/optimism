//! Optimism transaction type.

pub use op_alloy_consensus::OpTxType;

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::constants::EIP7702_TX_TYPE_ID;
    use op_alloy_consensus::DEPOSIT_TX_TYPE_ID;
    use reth_codecs::{txtype::*, Compact};
    use rstest::rstest;

    #[rstest]
    #[case(OpTxType::Legacy, COMPACT_IDENTIFIER_LEGACY, vec![])]
    #[case(OpTxType::Eip2930, COMPACT_IDENTIFIER_EIP2930, vec![])]
    #[case(OpTxType::Eip1559, COMPACT_IDENTIFIER_EIP1559, vec![])]
    #[case(OpTxType::Eip7702, COMPACT_EXTENDED_IDENTIFIER_FLAG, vec![EIP7702_TX_TYPE_ID])]
    #[case(OpTxType::Deposit, COMPACT_EXTENDED_IDENTIFIER_FLAG, vec![DEPOSIT_TX_TYPE_ID])]
    fn test_txtype_to_compact(
        #[case] tx_type: OpTxType,
        #[case] expected_identifier: usize,
        #[case] expected_buf: Vec<u8>,
    ) {
        let mut buf = vec![];
        let identifier = tx_type.to_compact(&mut buf);

        assert_eq!(
            identifier, expected_identifier,
            "Unexpected identifier for OpTxType {tx_type:?}",
        );
        assert_eq!(buf, expected_buf, "Unexpected buffer for OpTxType {tx_type:?}",);
    }

    #[rstest]
    #[case(OpTxType::Legacy, COMPACT_IDENTIFIER_LEGACY, vec![])]
    #[case(OpTxType::Eip2930, COMPACT_IDENTIFIER_EIP2930, vec![])]
    #[case(OpTxType::Eip1559, COMPACT_IDENTIFIER_EIP1559, vec![])]
    #[case(OpTxType::Eip7702, COMPACT_EXTENDED_IDENTIFIER_FLAG, vec![EIP7702_TX_TYPE_ID])]
    #[case(OpTxType::Deposit, COMPACT_EXTENDED_IDENTIFIER_FLAG, vec![DEPOSIT_TX_TYPE_ID])]
    fn test_txtype_from_compact(
        #[case] expected_type: OpTxType,
        #[case] identifier: usize,
        #[case] buf: Vec<u8>,
    ) {
        let (actual_type, remaining_buf) = OpTxType::from_compact(&buf, identifier);

        assert_eq!(actual_type, expected_type, "Unexpected TxType for identifier {identifier}");
        assert!(remaining_buf.is_empty(), "Buffer not fully consumed for identifier {identifier}");
    }
}
