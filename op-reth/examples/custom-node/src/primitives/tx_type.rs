use crate::primitives::TxTypeCustom;
use alloy_primitives::bytes::{Buf, BufMut};
use reth_codecs::{txtype::COMPACT_EXTENDED_IDENTIFIER_FLAG, Compact};

pub const PAYMENT_TX_TYPE_ID: u8 = 42;

impl Compact for TxTypeCustom {
    fn to_compact<B>(&self, buf: &mut B) -> usize
    where
        B: BufMut + AsMut<[u8]>,
    {
        match self {
            Self::Op(ty) => ty.to_compact(buf),
            Self::Payment => {
                buf.put_u8(PAYMENT_TX_TYPE_ID);
                COMPACT_EXTENDED_IDENTIFIER_FLAG
            }
        }
    }

    fn from_compact(mut buf: &[u8], identifier: usize) -> (Self, &[u8]) {
        match identifier {
            COMPACT_EXTENDED_IDENTIFIER_FLAG => (
                {
                    let extended_identifier = buf.get_u8();
                    match extended_identifier {
                        PAYMENT_TX_TYPE_ID => Self::Payment,
                        _ => panic!("Unsupported TxType identifier: {extended_identifier}"),
                    }
                },
                buf,
            ),
            v => {
                let (inner, buf) = TxTypeCustom::from_compact(buf, v);
                (inner, buf)
            }
        }
    }
}
