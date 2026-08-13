use super::*;
use alloy_consensus::{
    Block, BlockBody, Eip658Value, Header, Receipt, Sealable, SignableTransaction, TxEip7702,
    transaction::TransactionMeta,
};
use alloy_genesis::Genesis;
use alloy_op_hardforks::{
    ForkCondition, OP_MAINNET_ISTHMUS_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP,
    OP_MAINNET_KARST_TIMESTAMP, OpChainHardforks, OpHardfork,
};
use alloy_primitives::{Address, Bytes, Signature, U256, hex};
use op_alloy_consensus::{OpTypedTransaction, SDMGasEntry, build_post_exec_tx};
use op_alloy_network::eip2718::Decodable2718;
use reth_optimism_chainspec::{OP_MAINNET, OpChainSpecBuilder};
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_primitives_traits::{Recovered, SealedBlock};
use std::sync::Arc;

/// OP Mainnet transaction at index 0 in block 124665056.
///
/// <https://optimistic.etherscan.io/tx/0x312e290cf36df704a2217b015d6455396830b0ce678b860ebfcc30f41403d7b1>
const TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056: [u8; 251] = hex!(
    "7ef8f8a0683079df94aa5b9cf86687d739a60a9b4f0835e520ec4d664e2e415dca17a6df94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e200000146b000f79c500000000000000040000000066d052e700000000013ad8a3000000000000000000000000000000000000000000000000000000003ef1278700000000000000000000000000000000000000000000000000000000000000012fdf87b89884a61e74b322bbcf60386f543bfae7827725efaaf0ab1de2294a590000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985"
);

/// OP Mainnet transaction at index 1 in block 124665056.
///
/// <https://optimistic.etherscan.io/tx/0x1059e8004daff32caa1f1b1ef97fe3a07a8cf40508f5b835b66d9420d87c4a4a>
const TX_1_OP_MAINNET_BLOCK_124665056: [u8; 1176] = hex!(
    "02f904940a8303fba78401d6d2798401db2b6d830493e0943e6f4f7866654c18f536170780344aa8772950b680b904246a761202000000000000000000000000087000a300de7200382b55d40045000000e5d60e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003a0000000000000000000000000000000000000000000000000000000000000022482ad56cb0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000120000000000000000000000000dc6ff44d5d932cbd77b52e5612ba0529dc6226f1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000044095ea7b300000000000000000000000021c4928109acb0659a88ae5329b5374a3024694c0000000000000000000000000000000000000000000000049b9ca9a6943400000000000000000000000000000000000000000000000000000000000000000000000000000000000021c4928109acb0659a88ae5329b5374a3024694c000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000024b6b55f250000000000000000000000000000000000000000000000049b9ca9a694340000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000415ec214a3950bea839a7e6fbb0ba1540ac2076acd50820e2d5ef83d0902cdffb24a47aff7de5190290769c4f0a9c6fabf63012986a0d590b1b571547a8c7050ea1b00000000000000000000000000000000000000000000000000000000000000c080a06db770e6e25a617fe9652f0958bd9bd6e49281a53036906386ed39ec48eadf63a07f47cf51a4a40b4494cf26efc686709a9b03939e20ee27e59682f5faa536667e"
);

/// Timestamp of OP mainnet block 124665056.
///
/// <https://optimistic.etherscan.io/block/124665056>
const BLOCK_124665056_TIMESTAMP: u64 = 1724928889;

/// L1 block info for transaction at index 1 in block 124665056.
///
/// <https://optimistic.etherscan.io/tx/0x1059e8004daff32caa1f1b1ef97fe3a07a8cf40508f5b835b66d9420d87c4a4a>
const TX_META_TX_1_OP_MAINNET_BLOCK_124665056: OpTransactionReceiptFields =
    OpTransactionReceiptFields {
        l1_block_info: L1BlockInfo {
            l1_gas_price: Some(1055991687), // since bedrock l1 base fee
            l1_gas_used: Some(4471),
            l1_fee: Some(24681034813),
            l1_fee_scalar: None,
            l1_base_fee_scalar: Some(5227),
            l1_blob_base_fee: Some(1),
            l1_blob_base_fee_scalar: Some(1014213),
            operator_fee_scalar: None,
            operator_fee_constant: None,
            da_footprint_gas_scalar: None,
        },
        op_gas_refund: None,
        deposit_nonce: None,
        deposit_receipt_version: None,
    };

#[test]
fn op_receipt_fields_from_block_and_tx() {
    // rig
    let tx_0 = OpTransactionSigned::decode_2718(
        &mut TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056.as_slice(),
    )
    .unwrap();

    let tx_1 =
        OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice()).unwrap();

    let block: Block<OpTransactionSigned> = Block {
        body: BlockBody { transactions: [tx_0, tx_1.clone()].to_vec(), ..Default::default() },
        ..Default::default()
    };

    let mut l1_block_info =
        reth_optimism_evm::extract_l1_info(&block.body).expect("should extract l1 info");

    // test
    assert!(OP_MAINNET.is_fjord_active_at_timestamp(BLOCK_124665056_TIMESTAMP));

    let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
        .l1_block_info(&*OP_MAINNET, &tx_1, &mut l1_block_info)
        .expect("should parse revm l1 info")
        .build();

    let L1BlockInfo {
        l1_gas_price,
        l1_gas_used,
        l1_fee,
        l1_fee_scalar,
        l1_base_fee_scalar,
        l1_blob_base_fee,
        l1_blob_base_fee_scalar,
        operator_fee_scalar,
        operator_fee_constant,
        da_footprint_gas_scalar,
    } = receipt_meta.l1_block_info;

    assert_eq!(
        l1_gas_price, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_gas_price,
        "incorrect l1 base fee (former gas price)"
    );
    assert_eq!(
        l1_gas_used, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_gas_used,
        "incorrect l1 gas used"
    );
    assert_eq!(
        l1_fee, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_fee,
        "incorrect l1 fee"
    );
    assert_eq!(
        l1_fee_scalar, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_fee_scalar,
        "incorrect l1 fee scalar"
    );
    assert_eq!(
        l1_base_fee_scalar,
        TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_base_fee_scalar,
        "incorrect l1 base fee scalar"
    );
    assert_eq!(
        l1_blob_base_fee, TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_blob_base_fee,
        "incorrect l1 blob base fee"
    );
    assert_eq!(
        l1_blob_base_fee_scalar,
        TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.l1_blob_base_fee_scalar,
        "incorrect l1 blob base fee scalar"
    );
    assert_eq!(
        operator_fee_scalar,
        TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.operator_fee_scalar,
        "incorrect operator fee scalar"
    );
    assert_eq!(
        operator_fee_constant,
        TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.operator_fee_constant,
        "incorrect operator fee constant"
    );
    assert_eq!(
        da_footprint_gas_scalar,
        TX_META_TX_1_OP_MAINNET_BLOCK_124665056.l1_block_info.da_footprint_gas_scalar,
        "incorrect da footprint gas scalar"
    );
}

#[test]
fn convert_receipts_extracts_post_exec_gas_refund_from_embedded_payload() {
    let tx_0 = OpTransactionSigned::decode_2718(
        &mut TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056.as_slice(),
    )
    .unwrap();
    let tx_1 =
        OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice()).unwrap();
    let post_exec = OpTransactionSigned::PostExec(
        build_post_exec_tx(124665056, vec![SDMGasEntry { index: 1, gas_refund: 77 }]).seal_slow(),
    );

    let block = SealedBlock::new_unhashed(Block::<OpTransactionSigned> {
        header: Header {
            number: 124665056,
            timestamp: BLOCK_124665056_TIMESTAMP,
            ..Default::default()
        },
        body: BlockBody { transactions: vec![tx_0, tx_1.clone(), post_exec], ..Default::default() },
    });

    let interop_active = Arc::new(
        OpChainSpecBuilder::default()
            .chain(OP_MAINNET.chain())
            .genesis(Genesis::default())
            .lagoon_activated()
            .build(),
    );
    let converter = OpReceiptConverter::new(
        reth_storage_api::noop::NoopProvider::<_, OpPrimitives>::new(interop_active),
    );
    let receipts =
        <OpReceiptConverter<_> as ReceiptConverter<OpPrimitives>>::convert_receipts_with_block(
            &converter,
            vec![ConvertReceiptInput::<OpPrimitives> {
                tx: Recovered::new_unchecked(&tx_1, Address::ZERO),
                receipt: OpReceipt::Eip1559(Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 100,
                    logs: vec![],
                }),
                gas_used: 100,
                next_log_index: 0,
                meta: TransactionMeta {
                    index: 1,
                    block_number: 124665056,
                    timestamp: BLOCK_124665056_TIMESTAMP,
                    ..Default::default()
                },
            }],
            &block,
        )
        .unwrap();

    assert_eq!(receipts.len(), 1);
    assert_eq!(receipts[0].op_gas_refund, Some(77));
}

#[test]
fn op_non_zero_operator_fee_params_included_in_receipt() {
    let tx_1 =
        OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice()).unwrap();

    let mut l1_block_info = op_revm::L1BlockInfo {
        operator_fee_scalar: Some(U256::ZERO),
        operator_fee_constant: Some(U256::from(2)),
        ..Default::default()
    };

    let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
        .l1_block_info(&*OP_MAINNET, &tx_1, &mut l1_block_info)
        .expect("should parse revm l1 info")
        .build();

    let L1BlockInfo { operator_fee_scalar, operator_fee_constant, .. } = receipt_meta.l1_block_info;

    assert_eq!(operator_fee_scalar, Some(0), "incorrect operator fee scalar");
    assert_eq!(operator_fee_constant, Some(2), "incorrect operator fee constant");
}

#[test]
fn op_zero_operator_fee_params_not_included_in_receipt() {
    let tx_1 =
        OpTransactionSigned::decode_2718(&mut TX_1_OP_MAINNET_BLOCK_124665056.as_slice()).unwrap();

    let mut l1_block_info = op_revm::L1BlockInfo {
        operator_fee_scalar: Some(U256::ZERO),
        operator_fee_constant: Some(U256::ZERO),
        ..Default::default()
    };

    let receipt_meta = OpReceiptFieldsBuilder::new(BLOCK_124665056_TIMESTAMP, 124665056)
        .l1_block_info(&*OP_MAINNET, &tx_1, &mut l1_block_info)
        .expect("should parse revm l1 info")
        .build();

    let L1BlockInfo { operator_fee_scalar, operator_fee_constant, .. } = receipt_meta.l1_block_info;

    assert_eq!(operator_fee_scalar, None, "incorrect operator fee scalar");
    assert_eq!(operator_fee_constant, None, "incorrect operator fee constant");
}

// <https://github.com/paradigmxyz/reth/issues/12177>
#[test]
fn base_receipt_gas_fields() {
    // https://basescan.org/tx/0x510fd4c47d78ba9f97c91b0f2ace954d5384c169c9545a77a373cf3ef8254e6e
    let system = hex!(
        "7ef8f8a0389e292420bcbf9330741f72074e39562a09ff5a00fd22e4e9eee7e34b81bca494deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e20000008dd00101c120000000000000004000000006721035b00000000014189960000000000000000000000000000000000000000000000000000000349b4dcdc000000000000000000000000000000000000000000000000000000004ef9325cc5991ce750960f636ca2ffbb6e209bb3ba91412f21dd78c14ff154d1930f1f9a0000000000000000000000005050f69a9786f081509234f1a7f4684b5e5b76c9"
    );
    let tx_0 = OpTransactionSigned::decode_2718(&mut &system[..]).unwrap();

    let block: alloy_consensus::Block<OpTransactionSigned> = Block {
        body: BlockBody { transactions: vec![tx_0], ..Default::default() },
        ..Default::default()
    };
    let mut l1_block_info =
        reth_optimism_evm::extract_l1_info(&block.body).expect("should extract l1 info");

    // https://basescan.org/tx/0xf9420cbaf66a2dda75a015488d37262cbfd4abd0aad7bb2be8a63e14b1fa7a94
    let tx = hex!(
        "02f86c8221058034839a4ae283021528942f16386bb37709016023232523ff6d9daf444be380841249c58bc080a001b927eda2af9b00b52a57be0885e0303c39dd2831732e14051c2336470fd468a0681bf120baf562915841a48601c2b54a6742511e535cf8f71c95115af7ff63bd"
    );
    let tx_1 = OpTransactionSigned::decode_2718(&mut &tx[..]).unwrap();

    let receipt_meta = OpReceiptFieldsBuilder::new(1730216981, 21713817)
        .l1_block_info(&*OP_MAINNET, &tx_1, &mut l1_block_info)
        .expect("should parse revm l1 info")
        .build();

    let L1BlockInfo {
        l1_gas_price,
        l1_gas_used,
        l1_fee,
        l1_fee_scalar,
        l1_base_fee_scalar,
        l1_blob_base_fee,
        l1_blob_base_fee_scalar,
        operator_fee_scalar,
        operator_fee_constant,
        da_footprint_gas_scalar,
    } = receipt_meta.l1_block_info;

    assert_eq!(l1_gas_price, Some(14121491676), "incorrect l1 base fee (former gas price)");
    assert_eq!(l1_gas_used, Some(1600), "incorrect l1 gas used");
    assert_eq!(l1_fee, Some(191150293412), "incorrect l1 fee");
    assert!(l1_fee_scalar.is_none(), "incorrect l1 fee scalar");
    assert_eq!(l1_base_fee_scalar, Some(2269), "incorrect l1 base fee scalar");
    assert_eq!(l1_blob_base_fee, Some(1324954204), "incorrect l1 blob base fee");
    assert_eq!(l1_blob_base_fee_scalar, Some(1055762), "incorrect l1 blob base fee scalar");
    assert_eq!(operator_fee_scalar, None, "incorrect operator fee scalar");
    assert_eq!(operator_fee_constant, None, "incorrect operator fee constant");
    assert_eq!(da_footprint_gas_scalar, None, "incorrect da footprint gas scalar");
}

#[test]
fn da_footprint_gas_scalar_included_in_receipt_post_jovian() {
    const DA_FOOTPRINT_GAS_SCALAR: u16 = 10;

    let tx = TxEip7702 {
        chain_id: 1u64,
        nonce: 0,
        max_fee_per_gas: 0x28f000fff,
        max_priority_fee_per_gas: 0x28f000fff,
        gas_limit: 10,
        to: Address::default(),
        value: U256::from(3_u64),
        input: Bytes::from(vec![1, 2]),
        access_list: Default::default(),
        authorization_list: Default::default(),
    };

    let signature = Signature::new(U256::default(), U256::default(), true);

    let tx: OpTransactionSigned = OpTypedTransaction::Eip7702(tx).into_signed(signature).into();

    let mut l1_block_info = op_revm::L1BlockInfo {
        da_footprint_gas_scalar: Some(DA_FOOTPRINT_GAS_SCALAR),
        ..Default::default()
    };

    let op_hardforks = OpChainHardforks::op_mainnet();

    let receipt = OpReceiptFieldsBuilder::new(OP_MAINNET_JOVIAN_TIMESTAMP, u64::MAX)
        .l1_block_info(&op_hardforks, &tx, &mut l1_block_info)
        .expect("should parse revm l1 info")
        .build();

    assert_eq!(receipt.l1_block_info.da_footprint_gas_scalar, Some(DA_FOOTPRINT_GAS_SCALAR));
}

#[test]
fn blob_gas_used_included_in_receipt_post_jovian() {
    const DA_FOOTPRINT_GAS_SCALAR: u16 = 100;
    let tx = TxEip7702 {
        chain_id: 1u64,
        nonce: 0,
        max_fee_per_gas: 0x28f000fff,
        max_priority_fee_per_gas: 0x28f000fff,
        gas_limit: 10,
        to: Address::default(),
        value: U256::from(3_u64),
        access_list: Default::default(),
        authorization_list: Default::default(),
        input: Bytes::from(vec![0; 1_000_000]),
    };

    let signature = Signature::new(U256::default(), U256::default(), true);

    let tx: OpTransactionSigned = OpTypedTransaction::Eip7702(tx).into_signed(signature).into();

    let mut l1_block_info = op_revm::L1BlockInfo {
        da_footprint_gas_scalar: Some(DA_FOOTPRINT_GAS_SCALAR),
        ..Default::default()
    };

    let op_hardforks = OpChainHardforks::op_mainnet();

    let op_receipt = OpReceiptBuilder::new(
        &op_hardforks,
        ConvertReceiptInput::<OpPrimitives> {
            tx: Recovered::new_unchecked(&tx, Address::default()),
            receipt: OpReceipt::Eip7702(Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 100,
                logs: vec![],
            }),
            gas_used: 100,
            next_log_index: 0,
            meta: TransactionMeta { timestamp: OP_MAINNET_JOVIAN_TIMESTAMP, ..Default::default() },
        },
        &mut l1_block_info,
        None,
    )
    .unwrap();

    let expected_blob_gas_used = tx_da_footprint(&tx, DA_FOOTPRINT_GAS_SCALAR.into());

    assert_eq!(op_receipt.core_receipt.blob_gas_used, Some(expected_blob_gas_used));
}

/// Test-only Lagoon activation, one block after Karst.
const LAGOON_TIMESTAMP: u64 = OP_MAINNET_KARST_TIMESTAMP + 2;

/// Adds Lagoon to OP Mainnet's hardfork schedule.
fn lagoon_hardforks() -> OpChainHardforks {
    OpChainHardforks::new(
        OpHardfork::op_mainnet()
            .into_iter()
            .chain([(OpHardfork::Lagoon, ForkCondition::Timestamp(LAGOON_TIMESTAMP))]),
    )
}

/// A post-exec (`0x7D`) receipt reports zero DA footprint.
#[test]
fn blob_gas_used_zero_in_post_exec_receipt_post_lagoon() {
    const DA_FOOTPRINT_GAS_SCALAR: u16 = 100;

    let tx: OpTransactionSigned = OpTransactionSigned::PostExec(
        build_post_exec_tx(1, vec![SDMGasEntry { index: 1, gas_refund: 77 }]).seal_slow(),
    );

    let mut l1_block_info = op_revm::L1BlockInfo {
        da_footprint_gas_scalar: Some(DA_FOOTPRINT_GAS_SCALAR),
        ..Default::default()
    };

    let op_hardforks = lagoon_hardforks();
    // Verify appending Lagoon did not shift Karst's activation.
    assert_eq!(
        op_hardforks.op_fork_activation(OpHardfork::Lagoon),
        ForkCondition::Timestamp(LAGOON_TIMESTAMP)
    );
    assert_eq!(
        op_hardforks.op_fork_activation(OpHardfork::Karst),
        ForkCondition::Timestamp(OP_MAINNET_KARST_TIMESTAMP)
    );

    let op_receipt = OpReceiptBuilder::new(
        &op_hardforks,
        ConvertReceiptInput::<OpPrimitives> {
            tx: Recovered::new_unchecked(&tx, Address::default()),
            receipt: OpReceipt::PostExec(Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 100,
                logs: vec![],
            }),
            gas_used: 0,
            next_log_index: 0,
            meta: TransactionMeta { timestamp: LAGOON_TIMESTAMP, ..Default::default() },
        },
        &mut l1_block_info,
        None,
    )
    .unwrap();

    assert_eq!(op_receipt.core_receipt.blob_gas_used, Some(0));
}

#[test]
fn blob_gas_used_not_included_in_receipt_post_isthmus() {
    const DA_FOOTPRINT_GAS_SCALAR: u16 = 100;
    let tx = TxEip7702 {
        chain_id: 1u64,
        nonce: 0,
        max_fee_per_gas: 0x28f000fff,
        max_priority_fee_per_gas: 0x28f000fff,
        gas_limit: 10,
        to: Address::default(),
        value: U256::from(3_u64),
        access_list: Default::default(),
        authorization_list: Default::default(),
        input: Bytes::from(vec![0; 1_000_000]),
    };

    let signature = Signature::new(U256::default(), U256::default(), true);

    let tx: OpTransactionSigned = OpTypedTransaction::Eip7702(tx).into_signed(signature).into();

    let mut l1_block_info = op_revm::L1BlockInfo {
        da_footprint_gas_scalar: Some(DA_FOOTPRINT_GAS_SCALAR),
        ..Default::default()
    };

    let op_hardforks = OpChainHardforks::op_mainnet();

    let op_receipt = OpReceiptBuilder::new(
        &op_hardforks,
        ConvertReceiptInput::<OpPrimitives> {
            tx: Recovered::new_unchecked(&tx, Address::default()),
            receipt: OpReceipt::Eip7702(Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 100,
                logs: vec![],
            }),
            gas_used: 100,
            next_log_index: 0,
            meta: TransactionMeta { timestamp: OP_MAINNET_ISTHMUS_TIMESTAMP, ..Default::default() },
        },
        &mut l1_block_info,
        None,
    )
    .unwrap();

    assert_eq!(op_receipt.core_receipt.blob_gas_used, None);
}

/// Builds a deposit `OpTransactionReceipt` for the given creation kind and receipt-extension
/// fields, returning `(from, contractAddress)` for assertions. `deposit_nonce` /
/// `deposit_receipt_version` model the per-fork receipt shape:
/// <https://specs.optimism.io/protocol/deposits.html#deposit-receipt>.
fn deposit_receipt_contract_address(
    to: alloy_primitives::TxKind,
    deposit_nonce: Option<u64>,
    deposit_receipt_version: Option<u64>,
) -> (Address, Option<Address>) {
    use op_alloy_consensus::{OpDepositReceipt, TxDeposit};

    let from = Address::with_last_byte(0x42);

    // A deposit tx's `nonce()` is always hard-coded to 0.
    let tx = TxDeposit {
        source_hash: Default::default(),
        from,
        to,
        mint: 0,
        value: U256::ZERO,
        gas_limit: 1_000_000,
        is_system_transaction: false,
        // init code: PUSH1 0x42, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
        input: Bytes::from_static(&hex!("604260005260206000f3")),
    };
    let signature = Signature::new(U256::ZERO, U256::ZERO, false);
    let tx: OpTransactionSigned =
        OpTransactionSigned::new_unhashed(OpTypedTransaction::Deposit(tx), signature);

    let mut l1_block_info = op_revm::L1BlockInfo::default();
    let op_hardforks = OpChainHardforks::op_mainnet();

    let op_receipt = OpReceiptBuilder::new(
        &op_hardforks,
        ConvertReceiptInput::<OpPrimitives> {
            tx: Recovered::new_unchecked(&tx, from),
            receipt: OpReceipt::Deposit(OpDepositReceipt {
                inner: Receipt {
                    status: Eip658Value::Eip658(true),
                    cumulative_gas_used: 100,
                    logs: vec![],
                },
                deposit_nonce,
                deposit_receipt_version,
            }),
            gas_used: 100,
            next_log_index: 0,
            meta: TransactionMeta { timestamp: OP_MAINNET_ISTHMUS_TIMESTAMP, ..Default::default() },
        },
        &mut l1_block_info,
        None,
    )
    .unwrap();

    (from, op_receipt.core_receipt.contract_address)
}

// Deposit contract-creation `contractAddress` across the three receipt eras defined by the
// spec. Per <https://specs.optimism.io/protocol/deposits.html#deposit-receipt>, the receipt
// extension fields are: omitted before Regolith; `depositNonce` from Regolith; and
// `depositNonce` + `depositReceiptVersion == 1` from Canyon. Address derivation depends only
// on `depositNonce` presence, so Regolith and Canyon must yield the same address, and only the
// pre-Regolith (no-nonce) case falls back to `CREATE(from, 0)`.

const DEPOSIT_NONCE: u64 = 35_964;

/// Pre-Regolith: no `depositNonce` (and no version). The override is a no-op and the address
/// stays `build_receipt`'s `CREATE(from, 0)` -- the guard's silent fall-through, the branch
/// most likely to regress.
#[test]
fn deposit_creation_pre_regolith_stays_zero() {
    let (from, addr) =
        deposit_receipt_contract_address(alloy_primitives::TxKind::Create, None, None);
    assert_eq!(
        addr,
        Some(from.create(0)),
        "without a deposit nonce the address must stay CREATE(from, 0)"
    );
}

/// Regolith: `depositNonce = Some`, version omitted. Address must be `CREATE(from, nonce)`.
#[test]
fn deposit_creation_regolith_uses_deposit_nonce() {
    let (from, addr) = deposit_receipt_contract_address(
        alloy_primitives::TxKind::Create,
        Some(DEPOSIT_NONCE),
        None,
    );
    assert_eq!(addr, Some(from.create(DEPOSIT_NONCE)));
    assert_ne!(
        addr,
        Some(from.create(0)),
        "must not derive the address from the deposit tx nonce (0)"
    );
}

/// Canyon: `depositNonce = Some` and `depositReceiptVersion = Some(1)`. The version must not
/// affect derivation -- the address is identical to the Regolith case.
#[test]
fn deposit_creation_canyon_matches_regolith() {
    let (from, addr) = deposit_receipt_contract_address(
        alloy_primitives::TxKind::Create,
        Some(DEPOSIT_NONCE),
        Some(1),
    );
    assert_eq!(
        addr,
        Some(from.create(DEPOSIT_NONCE)),
        "depositReceiptVersion must not change contract-address derivation"
    );
}

/// A deposit *call* (`to != Create`) must never receive a fabricated `contractAddress`, even
/// with a deposit nonce present -- the `is_some()` guard must hold it at `None`.
#[test]
fn deposit_call_keeps_contract_address_none() {
    let (_from, addr) = deposit_receipt_contract_address(
        alloy_primitives::TxKind::Call(Address::with_last_byte(0x99)),
        Some(DEPOSIT_NONCE),
        Some(1),
    );
    assert_eq!(addr, None, "a deposit call must not derive a contract address");
}
