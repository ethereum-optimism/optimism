//! Helper function for Receipt root calculation for Optimism hardforks.

use alloc::vec::Vec;
use alloy_consensus::ReceiptWithBloom;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::B256;
use alloy_trie::root::ordered_trie_root_with_encoder;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::DepositReceipt;

/// Calculates the receipt root for a header.
pub(crate) fn calculate_receipt_root_optimism<R: DepositReceipt>(
    receipts: &[ReceiptWithBloom<&R>],
    chain_spec: impl OpHardforks,
    timestamp: u64,
) -> B256 {
    // There is a minor bug in op-geth and op-erigon where in the Regolith hardfork,
    // the receipt root calculation does not include the deposit nonce in the receipt
    // encoding. In the Regolith Hardfork, we must strip the deposit nonce from the
    // receipts before calculating the receipt root. This was corrected in the Canyon
    // hardfork.
    if chain_spec.is_regolith_active_at_timestamp(timestamp) &&
        !chain_spec.is_canyon_active_at_timestamp(timestamp)
    {
        let receipts = receipts
            .iter()
            .map(|receipt| {
                let mut receipt = receipt.clone().map_receipt(|r| r.clone());
                if let Some(receipt) = receipt.receipt.as_deposit_receipt_mut() {
                    receipt.deposit_nonce = None;
                }
                receipt
            })
            .collect::<Vec<_>>();

        return ordered_trie_root_with_encoder(receipts.as_slice(), |r, buf| r.encode_2718(buf))
    }

    ordered_trie_root_with_encoder(receipts, |r, buf| r.encode_2718(buf))
}

/// Calculates the receipt root for a header for the reference type of an OP receipt.
///
/// NOTE: Prefer calculate receipt root optimism if you have log blooms memoized.
pub fn calculate_receipt_root_no_memo_optimism<R: DepositReceipt>(
    receipts: &[R],
    chain_spec: impl OpHardforks,
    timestamp: u64,
) -> B256 {
    // There is a minor bug in op-geth and op-erigon where in the Regolith hardfork,
    // the receipt root calculation does not include the deposit nonce in the receipt
    // encoding. In the Regolith Hardfork, we must strip the deposit nonce from the
    // receipts before calculating the receipt root. This was corrected in the Canyon
    // hardfork.
    if chain_spec.is_regolith_active_at_timestamp(timestamp) &&
        !chain_spec.is_canyon_active_at_timestamp(timestamp)
    {
        let receipts = receipts
            .iter()
            .map(|r| {
                let mut r = (*r).clone();
                if let Some(receipt) = r.as_deposit_receipt_mut() {
                    receipt.deposit_nonce = None;
                }
                r
            })
            .collect::<Vec<_>>();

        return ordered_trie_root_with_encoder(&receipts, |r, buf| {
            r.with_bloom_ref().encode_2718(buf);
        })
    }

    ordered_trie_root_with_encoder(receipts, |r, buf| {
        r.with_bloom_ref().encode_2718(buf);
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Receipt, ReceiptWithBloom, TxReceipt};
    use alloy_primitives::{b256, bloom, hex, Address, Bytes, Log, LogData};
    use op_alloy_consensus::OpDepositReceipt;
    use reth_optimism_chainspec::BASE_SEPOLIA;
    use reth_optimism_primitives::OpReceipt;

    /// Tests that the receipt root is computed correctly for the regolith block.
    /// This was implemented due to a minor bug in op-geth and op-erigon where in
    /// the Regolith hardfork, the receipt root calculation does not include the
    /// deposit nonce in the receipt encoding.
    /// To fix this an op-reth patch was applied to the receipt root calculation
    /// to strip the deposit nonce from each receipt before calculating the root.
    #[test]
    fn check_optimism_receipt_root() {
        let cases = [
            // Deposit nonces didn't exist in Bedrock; No need to strip. For the purposes of this
            // test, we do have them, so we should get the same root as Canyon.
            (
                "bedrock",
                1679079599,
                b256!("0xe255fed45eae7ede0556fe4fabc77b0d294d18781a5a581cab09127bc4cd9ffb"),
            ),
            // Deposit nonces introduced in Regolith. They weren't included in the receipt RLP,
            // so we need to strip them - the receipt root will differ.
            (
                "regolith",
                1679079600,
                b256!("0xe255fed45eae7ede0556fe4fabc77b0d294d18781a5a581cab09127bc4cd9ffb"),
            ),
            // Receipt root hashing bug fixed in Canyon. Back to including the deposit nonce
            // in the receipt RLP when computing the receipt root.
            (
                "canyon",
                1699981200,
                b256!("0x6eefbb5efb95235476654a8bfbf8cb64a4f5f0b0c80b700b0c5964550beee6d7"),
            ),
        ];

        for case in cases {
            let receipts = vec![
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
                                        b256!("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"),
                                        b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                        b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                        b256!("0x0000000000000000000000000000000000000000000000000000000000000000"),
                                    ],
                                    Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001"))
                                )
                            },
                            Log {
                                address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                                data: LogData::new_unchecked(
                                    vec![
                                        b256!("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"),
                                        b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                        b256!("0x0000000000000000000000000000000000000000000000000000000000000000"),
                                        b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                    ],
                                    Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001"))
                                )
                            },
                            Log {
                                address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                                data: LogData::new_unchecked(
                                vec![
                                    b256!("0x0eb774bb9698a73583fe07b6972cf2dcc08d1d97581a22861f45feb86b395820"),
                                    b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                    b256!("0x000000000000000000000000c498902843af527e674846bb7edefa8ad62b8fb9"),
                                ], Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000000000000000003")))
                            },
                        ]}),
                // 0x6c33676e8f6077f46a62eabab70bc6d1b1b18a624b0739086d77093a1ecf8266
                OpReceipt::Eip1559(Receipt {
                        status: true.into(),
                        cumulative_gas_used: 189253,
                        logs: vec![
                            Log {
                                address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                                data:  LogData::new_unchecked(vec![
                                    b256!("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                    b256!("0x0000000000000000000000000000000000000000000000000000000000000000"),
                                ],
                                Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001")))
                            },
                            Log {
                                address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                                data:  LogData::new_unchecked(vec![
                                    b256!("0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                    b256!("0x0000000000000000000000000000000000000000000000000000000000000000"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                ],
                                Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000000000000000000030000000000000000000000000000000000000000000000000000000000000001")))
                            },
                            Log {
                                address: hex!("ddb6dcce6b794415145eb5caa6cd335aeda9c272").into(),
                                data:  LogData::new_unchecked(vec![
                                    b256!("0x0eb774bb9698a73583fe07b6972cf2dcc08d1d97581a22861f45feb86b395820"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                    b256!("0x0000000000000000000000009d521a04bee134ff8136d2ec957e5bc8c50394ec"),
                                ],
                                Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000000000000000003")))
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
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
                                    b256!("0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"),
                                    b256!("0x0000000000000000000000002992607c1614484fe6d865088e5c048f0650afd4"),
                                ],
                                Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000018de76816d8000")))
                            },
                            Log {
                                address: hex!("cf8e7e6b26f407dee615fc4db18bf829e7aa8c09").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
                                    b256!("0x0000000000000000000000002992607c1614484fe6d865088e5c048f0650afd4"),
                                    b256!("0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"),
                                ],
                                Bytes::from_static(&hex!("000000000000000000000000000000000000000000000002d24d8e9ac1aa79e2")))
                            },
                            Log {
                                address: hex!("2992607c1614484fe6d865088e5c048f0650afd4").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"),
                                ],
                                Bytes::from_static(&hex!("000000000000000000000000000000000000000000000009bd50642785c15736000000000000000000000000000000000000000000011bb7ac324f724a29bbbf")))
                            },
                            Log {
                                address: hex!("2992607c1614484fe6d865088e5c048f0650afd4").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"),
                                    b256!("0x00000000000000000000000029843613c7211d014f5dd5718cf32bcd314914cb"),
                                    b256!("0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"),
                                ],
                                Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000018de76816d800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d24d8e9ac1aa79e2")))
                            },
                            Log {
                                address: hex!("6d0f8d488b669aa9ba2d0f0b7b75a88bf5051cd3").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
                                    b256!("0x0000000000000000000000008dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09"),
                                    b256!("0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"),
                                ],
                                Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000014bc73062aea8093")))
                            },
                            Log {
                                address: hex!("8dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"),
                                ],
                                Bytes::from_static(&hex!("00000000000000000000000000000000000000000000002f122cfadc1ca82a35000000000000000000000000000000000000000000000665879dc0609945d6d1")))
                            },
                            Log {
                                address: hex!("8dbffe4c8bf3caf5deae3a99b50cfcf3648cbc09").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"),
                                    b256!("0x00000000000000000000000029843613c7211d014f5dd5718cf32bcd314914cb"),
                                    b256!("0x000000000000000000000000c3feb4ef4c2a5af77add15c95bd98f6b43640cc8"),
                                ],
                                Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d24d8e9ac1aa79e200000000000000000000000000000000000000000000000014bc73062aea80930000000000000000000000000000000000000000000000000000000000000000")))
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
                                data:  LogData::new_unchecked( vec![
                                    b256!("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
                                    b256!("0x0000000000000000000000000000000000000000000000000000000000000000"),
                                    b256!("0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"),
                                    b256!("0x000000000000000000000000000000000000000000000000000000000011a1d3"),
                                ],
                                Default::default())
                            },
                            Log {
                                address: hex!("ac6564f3718837caadd42eed742d75c12b90a052").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0x9d89e36eadf856db0ad9ffb5a569e07f95634dddd9501141ecf04820484ad0dc"),
                                    b256!("0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"),
                                    b256!("0x000000000000000000000000000000000000000000000000000000000011a1d3"),
                                ],
                                Bytes::from_static(&hex!("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000037697066733a2f2f516d515141646b33736538396b47716577395256567a316b68643548375562476d4d4a485a62566f386a6d346f4a2f30000000000000000000")))
                            },
                            Log {
                                address: hex!("ac6564f3718837caadd42eed742d75c12b90a052").into(),
                                data:  LogData::new_unchecked( vec![
                                    b256!("0x110d160a1bedeea919a88fbc4b2a9fb61b7e664084391b6ca2740db66fef80fe"),
                                    b256!("0x00000000000000000000000084d47f6eea8f8d87910448325519d1bb45c2972a"),
                                    b256!("0x000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e"),
                                    b256!("0x000000000000000000000000000000000000000000000000000000000011a1d3"),
                                ],
                                Bytes::from_static(&hex!("0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000a4fa7f3fbf0677f254ebdb1646146864c305b76e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007717500762343034303661353035646234633961386163316433306335633332303265370000000000000000000000000000000000000000000000000000000000000037697066733a2f2f516d515141646b33736538396b47716577395256567a316b68643548375562476d4d4a485a62566f386a6d346f4a2f30000000000000000000")))
                            },
                        ],
                    }),
            ];
            let root = calculate_receipt_root_optimism(
                &receipts.iter().map(TxReceipt::with_bloom_ref).collect::<Vec<_>>(),
                BASE_SEPOLIA.as_ref(),
                case.1,
            );
            assert_eq!(root, case.2);
        }
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
        let root = calculate_receipt_root_optimism(&receipt, BASE_SEPOLIA.as_ref(), 0);
        assert_eq!(
            root,
            b256!("0xfe70ae4a136d98944951b2123859698d59ad251a381abc9960fa81cae3d0d4a0")
        );
    }
}
