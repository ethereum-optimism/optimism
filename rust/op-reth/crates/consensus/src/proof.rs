//! Helper function for Receipt root calculation for Optimism hardforks.

use alloc::vec::Vec;
use alloy_consensus::ReceiptWithBloom;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::B256;
use alloy_trie::root::ordered_trie_root_with_encoder;
use reth_optimism_primitives::DepositReceipt;

/// Whether a receipt must be cloned and normalized before computing the receipts trie root.
///
/// This mirrors op-geth `core/types/receipt.go` (`Receipts.EncodeIndex`) and Nethermind
/// `OptimismReceiptMessageDecoder` / `OptimismReceiptTrieDecoder`: for deposit receipts,
/// `deposit_nonce` is included in the trie value only when `deposit_receipt_version` is set
/// (post-Canyon). When the version is unset, the trie uses the same RLP shape as a non-deposit
/// typed receipt (status, cumulative gas, bloom, logs) and omits the nonce—even if it is
/// present on the in-memory receipt after Regolith.
fn deposit_receipt_needs_trie_normalization<R: DepositReceipt>(receipt: &R) -> bool {
    receipt.as_deposit_receipt().is_some_and(|d| {
        d.deposit_nonce.is_some() && d.deposit_receipt_version.is_none()
    })
}

/// Calculates the receipt root for a header.
pub(crate) fn calculate_receipt_root_optimism<R: DepositReceipt>(
    receipts: &[ReceiptWithBloom<&R>],
) -> B256 {
    if !receipts.iter().any(|r| deposit_receipt_needs_trie_normalization(r.receipt)) {
        return ordered_trie_root_with_encoder(receipts, |r, buf| r.encode_2718(buf));
    }

    let receipts: Vec<ReceiptWithBloom<R>> = receipts
        .iter()
        .map(|receipt| {
            let mut receipt = receipt.clone().map_receipt(|r| r.clone());
            if let Some(deposit) = receipt.receipt.as_deposit_receipt_mut() {
                if deposit.deposit_receipt_version.is_none() {
                    deposit.deposit_nonce = None;
                }
            }
            receipt
        })
        .collect();

    ordered_trie_root_with_encoder(receipts.as_slice(), |r, buf| r.encode_2718(buf))
}

/// Calculates the receipt root for a header for the reference type of an OP receipt.
///
/// NOTE: Prefer calculate receipt root optimism if you have log blooms memoized.
pub fn calculate_receipt_root_no_memo_optimism<R: DepositReceipt>(receipts: &[R]) -> B256 {
    if !receipts.iter().any(deposit_receipt_needs_trie_normalization) {
        return ordered_trie_root_with_encoder(receipts, |r, buf| {
            r.with_bloom_ref().encode_2718(buf);
        });
    }

    let receipts: Vec<R> = receipts
        .iter()
        .map(|r| {
            let mut r = (*r).clone();
            if let Some(deposit) = r.as_deposit_receipt_mut() {
                if deposit.deposit_receipt_version.is_none() {
                    deposit.deposit_nonce = None;
                }
            }
            r
        })
        .collect();

    ordered_trie_root_with_encoder(&receipts, |r, buf| {
        r.with_bloom_ref().encode_2718(buf);
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Receipt, ReceiptWithBloom, TxReceipt};
    use alloy_primitives::{Address, Bytes, Log, LogData, b256, bloom, hex};
    use op_alloy_consensus::OpDepositReceipt;
    use reth_optimism_primitives::OpReceipt;

    /// Regression test for OP Stack receipts trie hashing (op-geth `Receipts.EncodeIndex`).
    ///
    /// When `deposit_receipt_version` is unset, the trie omits `deposit_nonce` even if it is
    /// populated on the receipt. See [`canyon_deposit_includes_nonce_in_trie`] for post-Canyon
    /// behavior.
    #[test]
    fn check_optimism_receipt_root() {
        let expected_root =
            b256!("0xe255fed45eae7ede0556fe4fabc77b0d294d18781a5a581cab09127bc4cd9ffb");

        for name in ["bedrock_compat", "regolith"] {
            let receipts = [
                // 0xb0d6ee650637911394396d81172bd1c637d568ed1fbddab0daddfca399c58b53
                OpReceipt::Deposit(OpDepositReceipt {
                    inner: Receipt {
                        status: true.into(),
                        cumulative_gas_used: 46913,
                        logs: vec![],
                    },
                    deposit_nonce: Some(4012991u64),
                    deposit_receipt_version: None,
                }),
                // 0x2f433586bae30573c393adfa02bc81d2a1888a3d6c9869f473fb57245166bd9a
                OpReceipt::Eip1559(Receipt {
                    status: true.into(),
                    cumulative_gas_used: 118083,
                    logs: vec![
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000000000000000000000000000000000000000000000"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000000000000000000000000000000000000000000000"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0x0eb774bb9698a73583fe07b6972cf2dcc08d1d97581a22861f45feb86b395820"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000000000000000003"
                                )),
                            ),
                        },
                    ],
                }),
                // 0x6c33676e8f6077f46a62eabab70bc6d1b1b18a624b0739086d77093a1ecf8266
                OpReceipt::Eip1559(Receipt {
                    status: true.into(),
                    cumulative_gas_used: 189253,
                    logs: vec![
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000000000000000000000000000000000000000000000"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000000000000000000000000000000000000000000000"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0x0eb774bb9698a73583fe07b6972cf2dcc08d1d97581a22861f45feb86b395820"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000000000000000003"
                                )),
                            ),
                        },
                    ],
                }),
                // 0x4d3ecbef04ba7ce7f5ab55be0c61978ca97c117d7da448ed9771d4ff0c720a3f
                OpReceipt::Eip1559(Receipt {
                    status: true.into(),
                    cumulative_gas_used: 346969,
                    logs: vec![
                        Log {
                            address: hex!("4200000000000000000000000000000000000006").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000002992607c1614484fe6d865088e5c048f0650afd4"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000018de76816d8000"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("cf8e7e6b26f407dee615fc4db18bf829e7aa8c09").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000002992607c1614484fe6d865088e5c048f0650afd4"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "000000000000000000000000000000000000000000000002d24d8e9ac1aa79e2"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("2992607c1614484fe6d865088e5c048f0650afd4").into(),
                            data: LogData::new_unchecked(
                                vec![b256!(
                                    "0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"
                                )],
                                Bytes::from_static(&hex!(
                                    "000000000000000000000000000000000000000000000009bd50642785c15736000000000000000000000000000000000000000000011bb7ac324f724a29bbbf"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("2992607c1614484fe6d865088e5c048f0650afd4").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"
                                    ),
                                    b256!(
                                        "0x00000000000000000000000029843613c7211d014f5dd5718cf32bcd314914cb"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000018de76816d800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d24d8e9ac1aa79e2"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("6d0f8d488b669aa9ba2d0f0b7b75a88bf5051cd3").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000014bc73062aea8093"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("8dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09").into(),
                            data: LogData::new_unchecked(
                                vec![b256!(
                                    "0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"
                                )],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000002f122cfadc1ca82a35000000000000000000000000000000000000000000000665879dc0609945d6d1"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("8dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"
                                    ),
                                    b256!(
                                        "0x00000000000000000000000029843613c7211d014f5dd5718cf32bcd314914cb"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d24d8e9ac1aa79e200000000000000000000000000000000000000000000000014bc73062aea80930000000000000000000000000000000000000000000000000000000000000000"
                                )),
                            ),
                        },
                    ],
                }),
                // 0xf738af5eb00ba23dbc1be2dbce41dbc0180f0085b7fb46646e90bf737af90351
                OpReceipt::Eip1559(Receipt {
                    status: true.into(),
                    cumulative_gas_used: 623249,
                    logs: vec![
                        Log {
                            address: hex!("ac6564f3718837caadd42eed742d75c12b90a052").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
                                    ),
                                    b256!(
                                        "0x0000000000000000000000000000000000000000000000000000000000000000"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000000000000000000000000000000000000011a1d3"
                                    ),
                                ],
                                Default::default(),
                            ),
                        },
                        Log {
                            address: hex!("ac6564f3718837caadd42eed742d75c12b90a052").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0x9d89e36eadf856db0ad9ffb5a569e07f95634dddd9501141ecf04820484ad0dc"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000000000000000000000000000000000000011a1d3"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000037697066733a2f2f516d515141646b33736538396b47716577395256567a316b68643548375562476d4d4a485a62566f386a6d346f4a2f30000000000000000000"
                                )),
                            ),
                        },
                        Log {
                            address: hex!("ac6564f3718837caadd42eed742d75c12b90a052").into(),
                            data: LogData::new_unchecked(
                                vec![
                                    b256!(
                                        "0x110d160a1bedeea919a88fbc4b2a9fb61b7e664084391b6ca2740db66fef80fe"
                                    ),
                                    b256!(
                                        "0x00000000000000000000000084d47f6eea8f8d87910448325519d1bb45c2972a"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"
                                    ),
                                    b256!(
                                        "0x000000000000000000000000000000000000000000000000000000000011a1d3"
                                    ),
                                ],
                                Bytes::from_static(&hex!(
                                    "0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007717500762343034303661353035646234633961386163316433306335633332303265370000000000000000000000000000000000000000000000000000000000000037697066733a2f2f516d515141646b33736538396b47716577395256567a316b68643548375562476d4d4a485a62566f386a6d346f4a2f30000000000000000000"
                                )),
                            ),
                        },
                    ],
                }),
            ];
            let root = calculate_receipt_root_optimism(
                &receipts.iter().map(TxReceipt::with_bloom_ref).collect::<Vec<_>>(),
            );
            assert_eq!(root, expected_root, "{name}");
        }
    }

    /// Post-Canyon deposit receipts set `deposit_receipt_version` and then include `deposit_nonce`
    /// in the trie (op-geth `EncodeIndex` uses full `depositReceiptRLP`).
    #[test]
    fn canyon_deposit_includes_nonce_in_trie() {
        let inner = Receipt {
            status: true.into(),
            cumulative_gas_used: 46913,
            logs: vec![],
        };
        let with_nonce_no_version = OpReceipt::Deposit(OpDepositReceipt {
            inner: inner.clone(),
            deposit_nonce: Some(4012991u64),
            deposit_receipt_version: None,
        });
        let with_nonce_stripped = OpReceipt::Deposit(OpDepositReceipt {
            inner: inner.clone(),
            deposit_nonce: None,
            deposit_receipt_version: None,
        });
        let post_canyon = OpReceipt::Deposit(OpDepositReceipt {
            inner,
            deposit_nonce: Some(4012991u64),
            deposit_receipt_version: Some(1),
        });

        let root_stripped = calculate_receipt_root_no_memo_optimism(&[with_nonce_no_version]);
        assert_eq!(root_stripped, calculate_receipt_root_no_memo_optimism(&[with_nonce_stripped]));

        let root_canyon = calculate_receipt_root_no_memo_optimism(&[post_canyon]);
        assert_ne!(root_canyon, root_stripped);
    }

    #[test]
    fn check_receipt_root_optimism() {
        let logs = vec![Log {
            address: Address::ZERO,
            data: LogData::new_unchecked(vec![], Default::default()),
        }];
        let logs_bloom = bloom!(
            "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001"
        );
        let inner =
            OpReceipt::Eip2930(Receipt { status: true.into(), cumulative_gas_used: 102068, logs });
        let receipt = ReceiptWithBloom { receipt: &inner, logs_bloom };
        let receipt = vec![receipt];
        tracing::info!("blablabla2322");
        let root = calculate_receipt_root_optimism(&receipt);
        assert_eq!(
            root,
            b256!("0xfe70ae4a136d98944951b2123859698d59ad251a381abc9960fa81cae3d0d4a0")
        );
    }

    /// Manta Sepolia block 863731: single Regolith deposit (`depositNonce` set, no
    /// `depositReceiptVersion`). Header `receiptsRoot` from a live RPC; trie encoding must match
    /// op-geth `Receipts.EncodeIndex` (nonce omitted) not `Receipt.MarshalBinary` (nonce present).
    ///
    /// Reproduces the pre-fix op-reth failure: computed root `0x49f9…` vs header `0x3c71…`.
    #[test]
    fn manta_sepolia_863731_regolith_deposit_receipt_root_matches_chain_header() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xf9f5, logs: vec![] },
            deposit_nonce: Some(0xd2df2),
            deposit_receipt_version: None,
        });
        let root = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        assert_eq!(
            root,
            b256!("0x3c715dd96d2597ccd46fde046da5e4b13e0a5b7d0a2ff60c3ee6c92fee9600ea")
        );
    }

    /// Manta Sepolia block 549192 (another single Regolith deposit-only block; vectors from RPC).
    ///
    /// Pre-fix op-reth logged: got `0xac185bb4402e70ccef2c1188aea24fa4055a0334ffa0f703b6b0d50881c20da5`,
    /// expected header `0xdbf9def3e8d930433596aa91b3ad3279652031ca45b1a3ca92785703f84aa6b0`.
    #[test]
    fn manta_sepolia_549192_regolith_deposit_receipt_root_matches_chain_header() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xccfd, logs: vec![] },
            deposit_nonce: Some(0x86147),
            deposit_receipt_version: None,
        });
        let root = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        assert_eq!(
            root,
            b256!("0xdbf9def3e8d930433596aa91b3ad3279652031ca45b1a3ca92785703f84aa6b0")
        );
    }

    /// Trie built from raw `encode_2718` (nonce included) for block 549192 matches the faulty
    /// `got` root from pre-fix nodes.
    #[test]
    fn regolith_deposit_549192_wrong_root_if_nonce_encoded_in_trie_value() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xccfd, logs: vec![] },
            deposit_nonce: Some(0x86147),
            deposit_receipt_version: None,
        });
        let correct = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        let wrong = ordered_trie_root_with_encoder(
            std::slice::from_ref(&deposit),
            |r, buf| r.with_bloom_ref().encode_2718(buf),
        );
        assert_ne!(wrong, correct);
        assert_eq!(
            wrong,
            b256!("0xac185bb4402e70ccef2c1188aea24fa4055a0334ffa0f703b6b0d50881c20da5")
        );
    }

    /// Manta Sepolia block 697672 (StateRootTask / `receipt_root_bloom` path exposed wrong trie).
    #[test]
    fn manta_sepolia_697672_regolith_deposit_receipt_root_matches_chain_header() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xc545, logs: vec![] },
            deposit_nonce: Some(0xaa547),
            deposit_receipt_version: None,
        });
        let root = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        assert_eq!(
            root,
            b256!("0x0e7c255df3e2b7ca4d55b82047531f7f3e7437ab1eb33e22eca70653c874982d")
        );
    }

    #[test]
    fn regolith_deposit_697672_wrong_root_matches_precomputed_ethereum_trie_got() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xc545, logs: vec![] },
            deposit_nonce: Some(0xaa547),
            deposit_receipt_version: None,
        });
        let correct = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        let wrong = ordered_trie_root_with_encoder(
            std::slice::from_ref(&deposit),
            |r, buf| r.with_bloom_ref().encode_2718(buf),
        );
        assert_ne!(wrong, correct);
        assert_eq!(
            wrong,
            b256!("0xe0afede48ad81b1163eaa1f659972d7eefa65a5be63e6a98a534516b390fbb79")
        );
    }

    /// Including `deposit_nonce` in the typed RLP payload (MarshalBinary-style) must not match the
    /// canonical receipts trie for pre-Canyon deposit receipts.
    #[test]
    fn regolith_deposit_wrong_root_if_nonce_encoded_in_trie_value() {
        let deposit = OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xf9f5, logs: vec![] },
            deposit_nonce: Some(0xd2df2),
            deposit_receipt_version: None,
        });
        let correct = calculate_receipt_root_no_memo_optimism(std::slice::from_ref(&deposit));
        let wrong = ordered_trie_root_with_encoder(
            std::slice::from_ref(&deposit),
            |r, buf| r.with_bloom_ref().encode_2718(buf),
        );
        assert_ne!(wrong, correct);
        assert_eq!(
            wrong,
            b256!("0x49f9ab1e7322d0075c5783ee199ece50eb2e25291a680476bdf5043a0c6e68bd")
        );
    }

    /// Manta Sepolia block 836765: Regolith deposit + EIP-1559 user tx. Verifies normalization
    /// interacts correctly when only the first receipt is a deposit.
    ///
    /// Reproduces the pre-fix failure: got `0xa43c…` vs header `0x11e4…`.
    #[test]
    fn manta_sepolia_836765_mixed_deposit_and_eip1559_receipt_root_matches_chain_header() {
        let transfer_log = Log {
            address: hex!("9c76c6304885661cdb97f3984b13114b6d4b5248").into(),
            data: LogData::new_unchecked(
                vec![
                    b256!("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
                    b256!("0x00000000000000000000000070ce2d9bb8502e302b9cbe5c56a5d0f1067da713"),
                    b256!("0x000000000000000000000000717c7f9822e8c2ae6a64773f7a92727755a7e2fc"),
                ],
                Bytes::from_static(&hex!(
                    "00000000000000000000000000000000000000000000003635c9adc5dea00000"
                )),
            ),
        };
        let receipts = [
            OpReceipt::Deposit(OpDepositReceipt {
                inner: Receipt {
                    status: true.into(),
                    cumulative_gas_used: 0xccfd,
                    logs: vec![],
                },
                deposit_nonce: Some(0xcc49c),
                deposit_receipt_version: None,
            }),
            OpReceipt::Eip1559(Receipt {
                status: true.into(),
                cumulative_gas_used: 0x18336,
                logs: vec![transfer_log],
            }),
        ];
        let root = calculate_receipt_root_no_memo_optimism(&receipts);
        assert_eq!(
            root,
            b256!("0x11e432ab48d8ffcb3278758a25fb2e10ba1267feae336da97a0f7bc861c74bbe")
        );
    }
}
