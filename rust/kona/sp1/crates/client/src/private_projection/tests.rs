use super::*;
use alloy_consensus::{Signed, TxEip1559};
use alloy_primitives::{Signature, U256, hex};
use alloy_rpc_types_engine::PayloadAttributes;
use alloy_trie::{EMPTY_ROOT_HASH, TrieAccount};
use kona_genesis::{HardForkConfig, PrivateProjectionConfig};
use kona_mpt::{Nibbles, NoopTrieProvider};
use kona_protocol::{L1BlockInfoIsthmus, SingleBatch};
use op_alloy_consensus::{L1InfoDepositSource, TxDeposit};

fn signed(to: Address, data: Bytes, nonce: u64) -> Bytes {
    let access_list = if to == INBOX {
        projection::validateMessageCall::abi_decode_validate(&data)
            .ok()
            .map(|call| {
                vec![alloy_eips::eip2930::AccessListItem {
                    address: INBOX,
                    storage_keys: projection::import_keys(&call).unwrap(),
                }]
                .into()
            })
            .unwrap_or_default()
    } else {
        Default::default()
    };
    TxEnvelope::Eip1559(Signed::new_unhashed(
        TxEip1559 {
            chain_id: 12345,
            nonce,
            gas_limit: 500_000,
            to: to.into(),
            input: data,
            access_list,
            ..Default::default()
        },
        Signature::test_signature(),
    ))
    .encoded_2718()
    .into()
}

fn save_trie(node: &TrieNode, out: &mut BTreeMap<B256, Bytes>) {
    let raw = alloy_rlp::encode(node);
    out.insert(keccak256(&raw), raw.into());
    match node {
        TrieNode::Branch { stack } => {
            for child in stack {
                save_trie(child, out);
            }
        }
        TrieNode::Extension { node, .. } => save_trie(node, out),
        _ => {}
    }
}

/// A real EVM transfer/call with a messenger event. Tiny synthetic genesis keeps
/// guest execution/proving cheap; no operator-supplied receipt is trusted.
fn fixture(replacements: usize) -> (PublicInputs, Witness) {
    fixture_with_sdm(replacements, false)
}

fn fixture_with_sdm(replacements: usize, sdm: bool) -> (PublicInputs, Witness) {
    let event = SentMessage {
        destination: U256::from(100),
        target: Address::with_last_byte(8),
        nonce: U256::from(7),
        sender: Address::with_last_byte(9),
        message: Bytes::from_static(b"private hello"),
    };
    let log = event.encode_log_data();
    // Copy calldata, then emit LOG4 with the exact messenger event's topics.
    let mut code = vec![0x36, 0x5f, 0x5f, 0x37];
    for topic in log.topics().iter().rev() {
        code.push(0x7f);
        code.extend_from_slice(topic.as_slice());
    }
    code.extend_from_slice(&[0x36, 0x5f, 0xa4, 0x00]);
    let private_tx = signed(MESSENGER, log.data.clone(), 0);
    use alloy_consensus::transaction::SignerRecoverable;
    let sender = TxEnvelope::decode_2718_exact(&private_tx).unwrap().recover_signer().unwrap();
    let import = ExecutingMessage {
        payloadHash: B256::repeat_byte(0x42),
        identifier: kona_interop::MessageIdentifier {
            origin: MESSENGER,
            blockNumber: U256::from(50),
            logIndex: U256::from(2),
            timestamp: U256::from(900),
            chainId: U256::from(100),
        },
    }
    .encode_log_data();
    let mut inbox_code = vec![0x36, 0x5f, 0x5f, 0x37];
    for topic in import.topics().iter().rev() {
        inbox_code.push(0x7f);
        inbox_code.extend_from_slice(topic.as_slice());
    }
    inbox_code.extend_from_slice(&[0x36, 0x5f, 0xa2, 0x00]);
    let import_tx = signed(INBOX, import.data.clone(), 0);
    let importer = TxEnvelope::decode_2718_exact(&import_tx).unwrap().recover_signer().unwrap();
    let mut trie = TrieNode::Empty;
    for (addr, account) in [
        (
            importer,
            TrieAccount { balance: U256::from(1_000_000_000_000_000_000u64), ..Default::default() },
        ),
        (INBOX, TrieAccount { code_hash: keccak256(&inbox_code), ..Default::default() }),
        (
            sender,
            TrieAccount { balance: U256::from(1_000_000_000_000_000_000u64), ..Default::default() },
        ),
        (
            address!("4200000000000000000000000000000000000016"),
            TrieAccount { nonce: 1, ..Default::default() },
        ),
        (MESSENGER, TrieAccount { code_hash: keccak256(&code), ..Default::default() }),
    ] {
        trie.insert(
            &Nibbles::unpack(keccak256(addr)),
            alloy_rlp::encode(account).into(),
            &NoopTrieProvider,
        )
        .unwrap();
    }
    let header = Header {
        state_root: trie.blind(),
        number: 0,
        timestamp: 1000,
        gas_limit: 30_000_000,
        base_fee_per_gas: Some(0),
        withdrawals_root: Some(EMPTY_ROOT_HASH),
        extra_data: if sdm {
            hex!("01000000fa000000060000000000000000").into()
        } else {
            hex!("00000000fa00000006").into()
        },
        ..Default::default()
    };
    let root = output(&header).unwrap();
    let mut private_cfg = RollupConfig {
        block_time: 2,
        l2_chain_id: 12345.into(),
        hardforks: HardForkConfig {
            regolith_time: Some(0),
            canyon_time: Some(0),
            delta_time: Some(0),
            ecotone_time: Some(0),
            fjord_time: Some(0),
            granite_time: Some(0),
            holocene_time: Some(0),
            isthmus_time: Some(0),
            jovian_time: sdm.then_some(0),
            karst_time: sdm.then_some(0),
            lagoon_time: sdm.then_some(0),
            ..Default::default()
        },
        ..Default::default()
    };
    private_cfg.genesis.l2_time = 1000;
    private_cfg.genesis.l2 = BlockNumHash { number: 0, hash: header.hash_slow() };
    let mut cfg = private_cfg.clone();
    cfg.genesis.l2.hash = B256::repeat_byte(0x88);
    cfg.private_projection = Some(PrivateProjectionConfig {
        genesis_output_root: root,
        verifier: "insecure-stub-v1".into(),
        allow_events: false,
    });
    let mut preimages = BTreeMap::new();
    save_trie(&trie, &mut preimages);
    preimages.insert(keccak256(&code), code.into());
    preimages.insert(keccak256(&inbox_code), inbox_code.into());
    preimages.insert(EMPTY_ROOT_HASH, Bytes::from_static(&[0x80]));
    let mut witness =
        Witness { anchor_header: header.clone(), private_data: Bytes::new(), preimages };
    let mut public = PublicInputs {
        private_config: serde_json::to_vec(&private_cfg).unwrap().into(),
        projection_config: serde_json::to_vec(&cfg).unwrap().into(),
        dependency_set: Bytes::from_static(b"{}"),
        parent_hash: cfg.genesis.l2.hash,
        anchor: cfg.genesis.l2,
        anchor_output: root,
        recovery: vec![],
        attributes: vec![],
        blocks: vec![],
    };
    let mut exec = StatelessL2Builder::new(
        &private_cfg,
        PostExecEvmFactoryAdapter::new(ZkvmOpEvmFactory),
        OpAlloyReceiptBuilder::default(),
        Store(&witness.preimages),
        NoopTrieHinter,
        header.seal_slow(),
    );
    let mut private_parent = witness.anchor_header.hash_slow();
    let mut parent_output = root;
    let mut terminal = None;
    let mut private_span =
        SpanBatch { genesis_timestamp: 1000, chain_id: 12345, ..Default::default() };
    for i in 0..replacements + 2 {
        let n = i as u64 + 1;
        let base = L1BlockInfoIsthmus::new(
            100,
            999,
            0,
            B256::repeat_byte(5),
            n,
            Address::ZERO,
            0,
            17,
            19,
            23,
            29,
        );
        let info = if sdm {
            L1BlockInfoTx::Jovian(kona_protocol::L1BlockInfoJovian {
                base,
                da_footprint_gas_scalar: 400,
            })
        } else {
            L1BlockInfoTx::Isthmus(base)
        };
        let deposit = OpTxEnvelope::from(TxDeposit {
            source_hash: L1InfoDepositSource::new(info.block_hash(), n).source_hash(),
            from: address!("deaddeaddeaddeaddeaddeaddeaddeaddead0001"),
            to: address!("4200000000000000000000000000000000000015").into(),
            input: info.encode_calldata(),
            gas_limit: 1_000_000,
            ..Default::default()
        });
        let mut attrs = OpPayloadAttributes {
            payload_attributes: PayloadAttributes {
                timestamp: 1000 + n * 2,
                withdrawals: Some(vec![]),
                parent_beacon_block_root: Some(B256::ZERO),
                ..Default::default()
            },
            transactions: Some(vec![deposit.encoded_2718().into()]),
            gas_limit: Some(30_000_000),
            no_tx_pool: Some(true),
            eip_1559_params: Some(hex!("000000fa00000006").into()),
            min_base_fee: sdm.then_some(0),
        };
        // Recovery must execute real forced state changes, not merely empty L1-info blocks.
        if i < replacements {
            attrs.transactions.as_mut().unwrap().push(
                OpTxEnvelope::from(TxDeposit {
                    source_hash: op_alloy_consensus::UserDepositSource::new(info.block_hash(), n)
                        .source_hash(),
                    from: Address::with_last_byte(0x50),
                    to: Address::with_last_byte(0x51).into(),
                    mint: 100,
                    value: U256::from(100),
                    gas_limit: 100_000,
                    ..Default::default()
                })
                .encoded_2718()
                .into(),
            );
            if sdm {
                attrs.transactions.as_mut().unwrap().push(
                    OpTxEnvelope::from(op_alloy_consensus::build_post_exec_tx(n, vec![]))
                        .encoded_2718()
                        .into(),
                );
            }
        }
        public.attributes.push(attrs.clone());
        let mut executing = attrs.clone();
        let mut txs =
            if i == replacements { vec![private_tx.clone(), import_tx.clone()] } else { vec![] };
        if sdm && i >= replacements {
            txs.push(
                OpTxEnvelope::from(op_alloy_consensus::build_post_exec_tx(n, vec![]))
                    .encoded_2718()
                    .into(),
            );
        }
        executing.transactions.as_mut().unwrap().extend(txs.clone());
        if i >= replacements {
            private_span
                .append_singular_batch(
                    SingleBatch {
                        parent_hash: private_parent,
                        epoch_num: 100,
                        epoch_hash: info.block_hash(),
                        timestamp: 1000 + n * 2,
                        transactions: txs,
                    },
                    n,
                )
                .unwrap();
        }
        let result = exec.build_block(executing).unwrap();
        let computed = exec.compute_output_root().unwrap();
        if i < replacements {
            let public_base = L1BlockInfoIsthmus::new(
                100,
                999,
                0,
                B256::repeat_byte(5),
                n,
                Address::ZERO,
                0,
                0,
                0,
                0,
                0,
            );
            let public_info = if sdm {
                L1BlockInfoTx::Jovian(kona_protocol::L1BlockInfoJovian {
                    base: public_base,
                    da_footprint_gas_scalar: 400,
                })
            } else {
                L1BlockInfoTx::Isthmus(public_base)
            };
            let mut public_txs = attrs.transactions.clone().unwrap();
            let OpTxEnvelope::Deposit(ref tx) = deposit else { unreachable!() };
            let mut public_deposit = tx.inner().clone();
            public_deposit.input = public_info.encode_calldata();
            public_txs[0] = OpTxEnvelope::from(public_deposit).encoded_2718().into();
            assert_ne!(
                public_txs[0],
                attrs.transactions.as_ref().unwrap()[0],
                "private fee settings differ from projection"
            );
            let mut projection_header = result.header.clone().unseal();
            projection_header.gas_limit = i64::MAX as u64;
            projection_header.transactions_root =
                ordered_trie_with_encoder(&public_txs, |tx, out| out.put_slice(tx)).root();
            projection_header.parent_hash = public.parent_hash;
            public.parent_hash = projection_header.hash_slow();
            public.recovery.push(OpBlock {
                header: projection_header,
                body: alloy_consensus::BlockBody {
                    transactions: public_txs
                        .iter()
                        .map(|raw| OpTxEnvelope::decode_2718_exact(raw).unwrap())
                        .collect(),
                    ..Default::default()
                },
            });
            parent_output = computed;
        } else {
            let mut txs = vec![signed(
                address!("420000000000000000000000000000000000002e"),
                projection::recordOutputCall { outputRoot: computed }.abi_encode().into(),
                1,
            )];
            for (to, data) in
                render(result.execution_result.receipts.iter().flat_map(|r| r.logs())).unwrap()
            {
                txs.push(signed(to, data, txs.len() as u64 + 1));
            }
            public.blocks.push(ProjectionBlock {
                timestamp: 1000 + n * 2,
                epoch: 100,
                transactions: txs,
            });
        }
        private_parent = result.header.hash();
        terminal = Some(result.header);
    }
    let mut raw = Vec::new();
    Batch::Span(private_span).encode(&mut raw).unwrap();
    let compressed =
        miniz_oxide::deflate::compress_to_vec_zlib(&alloy_rlp::encode(Bytes::from(raw)), 6);
    let mut data = vec![0];
    data.extend(Frame::new([0; 16], 0, compressed, true).encode());
    witness.private_data = data.into();
    let terminal = terminal.unwrap();
    let claim = projection::RangeClaim {
        version: 2,
        firstBlock: replacements as u64 + 1,
        lastBlock: replacements as u64 + 2,
        privateTerminalBlockHash: terminal.hash(),
        privateTerminalParentHash: terminal.parent_hash,
        anchorBlock: 0,
        anchorOutputRoot: root,
        parentOutputRoot: parent_output,
        recoveryHash: public.recovery.iter().rev().fold(B256::ZERO, |h, b| {
            projection::recovery_step(
                h,
                b,
                &b.body.transactions.iter().map(|tx| tx.encoded_2718().into()).collect::<Vec<_>>(),
            )
        }),
        l1Head: B256::repeat_byte(5),
        rollupConfigHash: keccak256(&public.projection_config),
        depSetHash: keccak256(&public.dependency_set),
        privateDataHash: keccak256(&witness.private_data),
        proof: Bytes::new(),
    };
    public.blocks[0].transactions.insert(
        0,
        signed(
            address!("420000000000000000000000000000000000002e"),
            projection::postClaimCall { claim }.abi_encode().into(),
            0,
        ),
    );
    (public, witness)
}

#[test]
fn executes_private_range_and_recovery() {
    for replacements in [0, 1, 3] {
        let (inputs, witness) = fixture(replacements);
        let result = execute(&inputs, &witness).unwrap();
        assert_ne!(result.terminal_output, inputs.anchor_output);
        assert_eq!(result, execute(&inputs, &witness).unwrap());
        if let Ok(dir) = std::env::var("PRIVATE_PROJECTION_FIXTURE_DIR") {
            std::fs::create_dir_all(&dir).unwrap();
            std::fs::write(
                format!("{dir}/range-{replacements}.json"),
                serde_json::to_vec_pretty(&(inputs, witness, result)).unwrap(),
            )
            .unwrap();
        }
    }
}

#[test]
fn rejects_tampered_execution_and_publication() {
    for replacements in [0, 3] {
        let (inputs, witness) = fixture(replacements);
        let mut bad = witness.clone();
        bad.anchor_header.state_root = B256::ZERO;
        assert!(execute(&inputs, &bad).unwrap_err().to_string().contains("anchor output"));
        let mut bad = witness.clone();
        bad.private_data = Bytes::from_static(b"tampered");
        assert!(
            execute(&inputs, &bad).unwrap_err().to_string().contains("private data commitment")
        );
        let mut bad = witness.clone();
        for value in bad.preimages.values_mut() {
            *value = Bytes::from_static(b"wrong");
        }
        assert!(execute(&inputs, &bad).is_err());
        let mut bad = inputs.clone();
        bad.blocks[0].transactions.pop();
        assert!(execute(&bad, &witness).unwrap_err().to_string().contains("projection messages"));
        let mut bad = inputs.clone();
        bad.blocks[0].transactions.swap(2, 3);
        assert!(execute(&bad, &witness).unwrap_err().to_string().contains("projection messages"));
        let mut bad = inputs.clone();
        bad.blocks[1].transactions[0] = signed(
            address!("420000000000000000000000000000000000002e"),
            projection::recordOutputCall { outputRoot: B256::repeat_byte(3) }.abi_encode().into(),
            1,
        );
        assert!(execute(&bad, &witness).unwrap_err().to_string().contains("published checkpoint"));
        let mut bad = inputs.clone();
        bad.dependency_set = Bytes::from_static(b"{ }");
        assert!(
            execute(&bad, &witness).unwrap_err().to_string().contains("dependency-set commitment")
        );
        let mut bad = inputs.clone();
        bad.projection_config = [bad.projection_config.as_ref(), b" "].concat().into();
        assert!(
            execute(&bad, &witness)
                .unwrap_err()
                .to_string()
                .contains("projection configuration commitment")
        );
        let mut bad = inputs.clone();
        bad.parent_hash = B256::repeat_byte(2);
        assert!(
            execute(&bad, &witness).unwrap_err().to_string().contains("canonical public parent")
        );
        let mut bad = inputs.clone();
        bad.attributes[0].payload_attributes.timestamp += 1;
        assert!(execute(&bad, &witness).is_err());
        if replacements > 0 {
            let mut bad = inputs.clone();
            bad.recovery[0].body.transactions.clear();
            assert!(execute(&bad, &witness).unwrap_err().to_string().contains("transaction root"));
            let mut bad = inputs.clone();
            bad.recovery.reverse();
            assert!(execute(&bad, &witness).unwrap_err().to_string().contains("recovery ancestry"));
        }
    }
}

#[test]
fn executes_lagoon_recovery_post_exec() {
    let (inputs, witness) = fixture_with_sdm(3, true);
    let result = execute(&inputs, &witness).unwrap();
    assert_ne!(result.terminal_output, inputs.anchor_output);
    if let Ok(dir) = std::env::var("PRIVATE_PROJECTION_FIXTURE_DIR") {
        std::fs::create_dir_all(&dir).unwrap();
        std::fs::write(
            format!("{dir}/lagoon-recovery.json"),
            serde_json::to_vec_pretty(&(inputs, witness, result)).unwrap(),
        )
        .unwrap();
    }
}
