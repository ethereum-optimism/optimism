//! Real-contract integration. Build contracts first; run with `nextest --run-ignored all`.
use alloy_eips::eip2930::AccessList;
use alloy_sol_types::{SolCall, SolValue, sol};
use op_atomic_builder::router::{Builder, Chain, INBOX};
use op_revm::{DefaultOp, OpBuilder, OpContext, OpSpecId, OpTransaction};
use revm::{
    Context, DatabaseCommit, ExecuteEvm,
    context::{CfgEnv, TxEnv},
    database::InMemoryDB,
    primitives::{Address, Bytes, U256, address, bytes},
    state::{AccountInfo, Bytecode},
};
use std::{collections::BTreeMap, path::PathBuf};
const USER: Address = address!("1000000000000000000000000000000000000000");
const ROUTER: Address = address!("2000000000000000000000000000000000000000");
const APP: Address = address!("3000000000000000000000000000000000000000");
const COUNTER: Address = address!("4000000000000000000000000000000000000000");
sol! {
    function proxyFor(uint256 chainId, address target) returns(address);
    function run(address proxy, uint256 amount, uint256 limit) returns(uint256);
    function runAcross(address first, address second, uint256 amount) returns(uint256);
}
fn artifact(name: &str) -> serde_json::Value {
    let path = PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("../../packages/contracts-bedrock/forge-artifacts")
        .join(format!(
            "{}.sol/{name}.json",
            if name.starts_with("AtomicCallRouter_Nested") {
                "AtomicNestedFixture.s"
            } else if name.starts_with("AtomicCallRouter_") {
                "AtomicCallRouter.t"
            } else {
                name
            }
        ));
    serde_json::from_str(
        &std::fs::read_to_string(path)
            .expect("build contracts with mise exec -- just build-dev first"),
    )
    .unwrap()
}
fn code(name: &str) -> Bytes {
    artifact(name)["deployedBytecode"]["object"].as_str().unwrap().parse().unwrap()
}
fn account(db: &mut InMemoryDB, address: Address, code: Bytes) {
    let code = Bytecode::new_raw(code);
    db.insert_account_info(
        address,
        AccountInfo {
            balance: U256::ZERO,
            nonce: 1,
            code_hash: code.hash_slow(),
            code: Some(code),
            ..Default::default()
        },
    );
}
fn context(chain: u64) -> OpContext<InMemoryDB> {
    let mut db = InMemoryDB::default();
    db.insert_account_info(
        USER,
        AccountInfo { balance: U256::from(10).pow(U256::from(24)), ..Default::default() },
    );
    account(&mut db, ROUTER, code("AtomicCallRouter"));
    account(&mut db, INBOX, code("CrossL2Inbox"));
    account(&mut db, APP, code("AtomicCallExample"));
    account(&mut db, COUNTER, code("AtomicCounter"));
    Context::op()
        .with_db(db)
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS).with_chain_id(chain))
        .modify_block_chained(|block| {
            block.number = U256::from(10);
            block.timestamp = U256::from(100);
            block.basefee = 0;
            block.gas_limit = 100_000_000;
        })
}
fn transaction(
    chain: u64,
    nonce: u64,
    to: Address,
    data: Bytes,
    accesses: AccessList,
) -> OpTransaction<TxEnv> {
    OpTransaction::builder()
        .base(
            TxEnv::builder()
                .caller(USER)
                .to(to)
                .data(data)
                .nonce(nonce)
                .gas_limit(20_000_000)
                .gas_price(1)
                .chain_id(Some(chain))
                .access_list(accesses),
        )
        .enveloped_tx(Some(bytes!("02c0")))
        .build_fill()
}
fn proxy(context: &mut OpContext<InMemoryDB>, chain: u64, nonce: u64, destination: u64) -> Address {
    let tx = transaction(
        chain,
        nonce,
        ROUTER,
        proxyForCall { chainId: U256::from(destination), target: COUNTER }.abi_encode().into(),
        AccessList::default(),
    );
    let result = context.clone().with_tx(tx).build_op().replay().unwrap();
    assert!(result.result.is_success(), "{:?}", result.result);
    let proxy = Address::abi_decode(result.result.output().unwrap()).unwrap();
    context.journaled_state.database.commit(result.state);
    proxy
}
fn build(fail: bool, three: bool, gas_probe: bool, tamper: u8) {
    let layout = artifact("AtomicCallRouter");
    let slots = layout["storageLayout"]["storage"].as_array().unwrap();
    for (name, slot, offset) in [
        ("witnesses", "8", 0),
        ("remoteCalls", "9", 0),
        ("completion", "10", 0),
        ("callbacks", "15", 0),
        ("callbackFailure", "16", 0),
        ("nested", "16", 1),
        ("callbacksUsed", "17", 0),
        ("callbacksTotal", "18", 0),
    ] {
        let item = slots.iter().find(|s| s["label"] == name).unwrap();
        assert_eq!(item["slot"], slot);
        assert_eq!(item["offset"], offset);
    }
    let mut a = context(901);
    let b = context(902);
    let first = proxy(&mut a, 901, 0, 902);
    let second = if three { proxy(&mut a, 901, 1, 903) } else { first };
    let root_nonce = if three { 2 } else { 1 };
    if gas_probe {
        account(&mut a.journaled_state.database, APP, code("AtomicCallRouter_GasProbe_Harness"));
    }
    let mut chains = BTreeMap::new();
    for (id, context) in [(901, a), (902, b), (903, context(903))] {
        chains.insert(
            id,
            Chain {
                context,
                router: ROUTER,
                sender: USER,
                prefix_logs: 7,
                application_gas: 1_000_000,
            },
        );
    }
    let mut preparations = BTreeMap::<u64, usize>::new();
    let mut builder = Builder {
        chains,
        max_calls: 8,
        max_bytes: 65536,
        envelope: |chain, mut data: Bytes, mut accesses: AccessList| {
            let count = preparations.entry(chain).or_default();
            *count += 1;
            if chain == 901 && *count == 2 && tamper == 1 {
                use op_atomic_builder::router::abi::executeRootNestedCall;
                let mut call = executeRootNestedCall::abi_decode(&data).unwrap();
                call.applicationGas += 1;
                data = call.abi_encode().into();
            }
            if tamper == 2 {
                sol! { function add(uint256 amount) returns(uint256); }
                return Ok(transaction(
                    chain,
                    if chain == 901 { root_nonce } else { 0 },
                    COUNTER,
                    addCall { amount: U256::from(1) }.abi_encode().into(),
                    accesses,
                ));
            }
            if chain == 901 && *count == 2 && tamper == 3 {
                accesses.0[0].storage_keys.clear();
            }
            Ok(transaction(
                chain,
                if chain == 901 { root_nonce } else { 0 },
                ROUTER,
                data,
                accesses,
            ))
        },
    };
    sol! { function run(address proxy, uint256 amount) returns(uint256); }
    let input = if gas_probe {
        runCall { proxy: first, amount: U256::from(6) }.abi_encode()
    } else if three {
        runAcrossCall { first, second, amount: U256::from(6) }.abi_encode()
    } else {
        self::runCall {
            proxy: first,
            amount: U256::from(6),
            limit: U256::from(if fail { 10 } else { 20 }),
        }
        .abi_encode()
    };
    let built = builder.build(901, U256::ZERO, APP, input.into());
    if tamper != 0 {
        assert!(built.is_err(), "a substituted envelope or missing witness must fail");
        assert!(
            preparations.values().all(|count| *count <= 2),
            "never retry discovery/final replay"
        );
        return;
    }
    let bundle = built.unwrap();
    assert_eq!(bundle.reverted, fail);
    assert_eq!(
        bundle.chains.len(),
        if fail {
            1
        } else if three {
            3
        } else {
            2
        }
    );
    for id in bundle.chains.keys() {
        assert_eq!(preparations[id], 2, "one discovery plus one final envelope per included chain");
    }
    if fail {
        assert!(!bundle.chains[&901].execution.result.is_success());
        assert!(
            bundle.chains[&901].execution.state[&APP].storage.values().all(|s| !s.is_changed())
        );
    } else {
        assert_eq!(
            bundle.chains[&902].execution.state[&COUNTER].storage[&U256::ZERO].present_value(),
            U256::from(if three { 6 } else { 12 })
        );
        assert_eq!(
            bundle.chains[&901].execution.state[&APP].storage
                [&U256::from(if gas_probe { 4 } else { 0 })]
                .present_value(),
            U256::from(if three { 6 } else { 12 })
        );
    }
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_repeated_destination() {
    build(false, false, false, 0);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_root_revert_discards_remote_work() {
    build(true, false, false, 0);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_three_chains() {
    build(false, true, false, 0);
}

use alloy_signer::SignerSync;
use alloy_signer_local::PrivateKeySigner;
use revm::primitives::{B256, TxKind, keccak256};
const ENTRY: Address = address!("0000000071727de22e5e9d8baf0edac6f37da032");
sol! {
    #[derive(Debug, Default)]
    struct PackedUserOperation { address sender; uint256 nonce; bytes initCode; bytes callData; bytes32 accountGasLimits; uint256 preVerificationGas; bytes32 gasFees; bytes paymasterAndData; bytes signature; }
    function createAccount(address owner, uint256 salt) returns(address);
    function setAccountAllowed(address account, bool allowed);
    function deposit() payable;
    function execute(address dest, uint256 value, bytes func);
    function handleOps(PackedUserOperation[] ops, address beneficiary);
    event UserOperationEvent(bytes32 indexed userOpHash, address indexed sender, address indexed paymaster, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed);
}
fn execute_setup(
    context: &mut OpContext<InMemoryDB>,
    chain: u64,
    nonce: &mut u64,
    to: TxKind,
    data: Bytes,
    value: U256,
) -> revm::context_interface::result::ExecutionResult<op_revm::OpHaltReason> {
    let mut tx = transaction(chain, *nonce, ROUTER, data, AccessList::default());
    tx.base.kind = to;
    tx.base.value = value;
    let result = context.clone().with_tx(tx).build_op().replay().unwrap();
    assert!(result.result.is_success(), "setup failed: {:?}", result.result);
    context.journaled_state.database.commit(result.state);
    *nonce += 1;
    result.result
}
fn deploy(
    context: &mut OpContext<InMemoryDB>,
    chain: u64,
    nonce: &mut u64,
    name: &str,
    args: Vec<u8>,
) -> Address {
    let init: Bytes = artifact(name)["bytecode"]["object"].as_str().unwrap().parse().unwrap();
    let result = execute_setup(
        context,
        chain,
        nonce,
        TxKind::Create,
        [init.as_ref(), &args].concat().into(),
        U256::ZERO,
    );
    match result {
        revm::context_interface::result::ExecutionResult::Success {
            output: revm::context_interface::result::Output::Create(_, Some(address)),
            ..
        } => address,
        _ => panic!("creation failed"),
    }
}
fn sponsor(
    context: &mut OpContext<InMemoryDB>,
    chain: u64,
    nonce: &mut u64,
    owner: Address,
) -> (Address, Address) {
    // Execute the exact OP EntryPoint v0.7 preinstall runtime.
    let source = std::fs::read_to_string(
        PathBuf::from(env!("CARGO_MANIFEST_DIR"))
            .join("../../packages/contracts-bedrock/src/libraries/Preinstalls.sol"),
    )
    .unwrap();
    let runtime = source
        .split("bytes internal constant EntryPoint_v070Code =")
        .nth(1)
        .unwrap()
        .split("hex\"")
        .nth(1)
        .unwrap()
        .split('"')
        .next()
        .unwrap();
    account(&mut context.journaled_state.database, ENTRY, format!("0x{runtime}").parse().unwrap());
    let factory = deploy(context, chain, nonce, "AtomicAccountFactory", Vec::new());
    let result = execute_setup(
        context,
        chain,
        nonce,
        TxKind::Call(factory),
        createAccountCall { owner, salt: U256::ZERO }.abi_encode().into(),
        U256::ZERO,
    );
    let account = Address::abi_decode(result.output().unwrap()).unwrap();
    let paymaster = deploy(
        context,
        chain,
        nonce,
        "AtomicPaymaster",
        (USER, ROUTER, U256::from(100_000_000)).abi_encode(),
    );
    execute_setup(
        context,
        chain,
        nonce,
        TxKind::Call(paymaster),
        setAccountAllowedCall { account, allowed: true }.abi_encode().into(),
        U256::ZERO,
    );
    execute_setup(
        context,
        chain,
        nonce,
        TxKind::Call(paymaster),
        depositCall {}.abi_encode().into(),
        U256::from(10).pow(U256::from(20)),
    );
    (account, paymaster)
}
fn pack(high: u128, low: u128) -> B256 {
    B256::from_slice(&[high.to_be_bytes(), low.to_be_bytes()].concat())
}
fn sponsored(
    chain: u64,
    nonce: u64,
    account: Address,
    paymaster: Address,
    data: Bytes,
    accesses: AccessList,
    signer: &PrivateKeySigner,
) -> OpTransaction<TxEnv> {
    let mut op = PackedUserOperation {
        sender: account,
        callData: executeCall { dest: ROUTER, value: U256::ZERO, func: data }.abi_encode().into(),
        accountGasLimits: pack(1_000_000, 12_000_000),
        preVerificationGas: U256::from(100_000),
        gasFees: pack(1, 1),
        paymasterAndData: [paymaster.as_slice(), pack(500_000, 0).as_slice()].concat().into(),
        ..Default::default()
    };
    let packed = keccak256(
        (
            op.sender,
            op.nonce,
            keccak256(&op.initCode),
            keccak256(&op.callData),
            op.accountGasLimits,
            op.preVerificationGas,
            op.gasFees,
            keccak256(&op.paymasterAndData),
        )
            .abi_encode(),
    );
    let hash = keccak256((packed, ENTRY, U256::from(chain)).abi_encode());
    op.signature = signer.sign_message_sync(hash.as_slice()).unwrap().as_bytes().to_vec().into();
    transaction(
        chain,
        nonce,
        ENTRY,
        handleOpsCall { ops: vec![op], beneficiary: USER }.abi_encode().into(),
        accesses,
    )
}
fn build_sponsored(remote_failure: bool, catch: bool) {
    let signer: PrivateKeySigner = B256::with_last_byte(1).to_string().parse().unwrap();
    let mut a = context(901);
    let mut b = context(902);
    let first = proxy(&mut a, 901, 0, 902);
    let mut a_nonce = 1;
    let mut b_nonce = 0;
    let (a_account, a_paymaster) = sponsor(&mut a, 901, &mut a_nonce, signer.address());
    let (b_account, b_paymaster) = sponsor(&mut b, 902, &mut b_nonce, signer.address());
    if remote_failure {
        b.journaled_state
            .database
            .insert_account_storage(COUNTER, U256::from(1), U256::from(if catch { 1 } else { 10 }))
            .unwrap();
    }
    let mut chains = BTreeMap::new();
    for (id, context, sender) in [(901, a, a_account), (902, b, b_account)] {
        chains.insert(
            id,
            Chain { context, router: ROUTER, sender, prefix_logs: 11, application_gas: 1_000_000 },
        );
    }
    let mut preparations = BTreeMap::<u64, usize>::new();
    let mut builder = Builder {
        chains,
        max_calls: 8,
        max_bytes: 65536,
        envelope: |chain, data, accesses| {
            *preparations.entry(chain).or_default() += 1;
            let (nonce, account, paymaster) = if chain == 901 {
                (a_nonce, a_account, a_paymaster)
            } else {
                (b_nonce, b_account, b_paymaster)
            };
            Ok(sponsored(chain, nonce, account, paymaster, data, accesses, &signer))
        },
    };
    sol! { function catchRemoteFailure(address proxy); }
    let input = if catch {
        catchRemoteFailureCall { proxy: first }.abi_encode()
    } else {
        runCall { proxy: first, amount: U256::from(6), limit: U256::from(100) }.abi_encode()
    };
    let bundle = builder.build(901, U256::ZERO, APP, input.into()).unwrap();
    assert_eq!(bundle.reverted, remote_failure);
    assert_eq!(bundle.chains.len(), 2);
    use alloy_sol_types::SolEvent;
    for (id, included) in &bundle.chains {
        assert_eq!(preparations[id], 2);
        assert!(included.execution.result.is_success(), "EntryPoint catches the operation revert");
        let event = included
            .execution
            .result
            .logs()
            .iter()
            .find_map(|log| UserOperationEvent::decode_log(log).ok())
            .expect("real EntryPoint settlement");
        assert_eq!(event.data.success, !remote_failure);
        assert!(event.data.actualGasCost > U256::ZERO);
        assert!(event.data.actualGasUsed > U256::ZERO);
        if remote_failure {
            let app = if *id == 901 { APP } else { COUNTER };
            assert!(included.execution.state[&app].storage.values().all(|s| !s.is_changed()));
        }
    }
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn signed_4337_real_paymaster_repeated_destination() {
    build_sponsored(false, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn signed_4337_later_remote_failure_reverts_both_operations() {
    build_sponsored(true, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn signed_4337_caught_remote_failure_still_aborts() {
    build_sponsored(true, true);
}

#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_gas_and_transient_storage_match_replay() {
    build(false, false, true, 0);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_changed_final_gas_is_rejected_once() {
    build(false, false, false, 1);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn unrelated_successful_envelope_cannot_masquerade_as_abort() {
    build(false, false, false, 2);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn real_router_missing_final_access_list_is_rejected_once() {
    build(false, false, false, 3);
}

fn build_nested(failure: u64, limit: u16, distinct: bool) {
    let mut a = context(901);
    let mut b = context(902);
    account(&mut a.journaled_state.database, APP, code("AtomicCallRouter_Nested_Harness"));
    account(&mut b.journaled_state.database, COUNTER, code("AtomicCallRouter_Nested_Harness"));
    let callback_contract = if distinct {
        let address = address!("5000000000000000000000000000000000000000");
        account(
            &mut a.journaled_state.database,
            address,
            code("AtomicCallRouter_NestedCallback_Harness"),
        );
        a.journaled_state
            .database
            .insert_account_storage(address, U256::ZERO, U256::from_be_slice(APP.as_slice()))
            .unwrap();
        address
    } else {
        APP
    };
    let peer = proxy(&mut a, 901, 0, 902);
    let tx = transaction(
        902,
        0,
        ROUTER,
        proxyForCall { chainId: U256::from(901), target: callback_contract }.abi_encode().into(),
        AccessList::default(),
    );
    let deployed = b.clone().with_tx(tx).build_op().replay().unwrap();
    assert!(deployed.result.is_success());
    let callback = Address::abi_decode(deployed.result.output().unwrap()).unwrap();
    b.journaled_state.database.commit(deployed.state);
    let chains = [(901, a), (902, b)]
        .into_iter()
        .map(|(id, context)| {
            (
                id,
                Chain {
                    context,
                    router: ROUTER,
                    sender: USER,
                    prefix_logs: 7,
                    application_gas: 2_000_000,
                },
            )
        })
        .collect();
    let mut preparations = BTreeMap::<u64, usize>::new();
    let mut builder = Builder {
        chains,
        max_calls: limit,
        max_bytes: 65536,
        envelope: |chain, data, accesses| {
            *preparations.entry(chain).or_default() += 1;
            Ok(transaction(chain, 1, ROUTER, data, accesses))
        },
    };
    sol! { function run(address peer, address callback, uint256 failure) returns(uint256); }
    let built = builder.build(
        901,
        U256::ZERO,
        APP,
        runCall { peer, callback, failure: U256::from(failure) }.abi_encode().into(),
    );
    if limit < 3 {
        assert!(built.is_err());
        assert!(preparations.values().all(|count| *count == 1));
        return;
    }
    let bundle = built.unwrap();
    let reverted = (1..=4).contains(&failure) || failure == 6;
    assert_eq!(bundle.reverted, reverted);
    assert_eq!(bundle.chains.len(), if failure == 3 { 1 } else { 2 });
    for (id, tx) in &bundle.chains {
        if !reverted && failure != 5 {
            use alloy_sol_types::SolEvent;
            use op_atomic_builder::router::abi::ExecutingMessage;
            let references: Vec<_> = tx
                .execution
                .result
                .logs()
                .iter()
                .enumerate()
                .filter_map(|(index, log)| {
                    ExecutingMessage::decode_log(log)
                        .ok()
                        .map(|event| (index, event.data.id.logIndex))
                })
                .collect();
            let positions = if *id == 901 { vec![1, 3, 5] } else { vec![0, 2, 4, 6] };
            assert_eq!(
                references,
                positions.into_iter().map(|i| (i, U256::from(7 + i))).collect::<Vec<_>>(),
                "actual nested logs match the Go/Kona cycle-regression fixture plus prefix"
            );
        }
        assert_eq!(preparations[id], 2, "one original transaction and one canonical replay");
        assert_eq!(tx.execution.result.is_success(), !reverted);
        if distinct && *id == 901 {
            let writes = &tx.execution.state[&callback_contract].storage;
            if reverted {
                assert!(writes.values().all(|slot| !slot.is_changed()));
            } else {
                assert_eq!(
                    writes[&U256::from(1)].present_value(),
                    U256::from(if failure == 5 { 2 } else { 1 })
                );
            }
        }
        let address = if *id == 901 { APP } else { COUNTER };
        if reverted {
            assert!(tx.execution.state[&address].storage.values().all(|slot| !slot.is_changed()));
        } else {
            assert_eq!(
                tx.execution.state[&address].storage[&U256::ZERO].present_value(),
                U256::from(if *id == 901 {
                    if failure == 5 { 51 } else { 34 }
                } else if failure == 5 {
                    17
                } else {
                    10
                })
            );
        }
    }
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_callbacks_share_pending_storage_transient_storage_and_origin() {
    build_nested(0, 8, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_inner_callback_failure_rolls_back_both_chains() {
    build_nested(1, 8, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_outer_callback_failure_rolls_back_both_chains() {
    build_nested(2, 8, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_root_failure_discards_successful_destination() {
    build_nested(3, 8, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_caught_callback_failure_still_aborts() {
    build_nested(4, 8, false);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_calls_obey_the_global_discovery_limit() {
    build_nested(0, 2, false);
}

#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_distinct_contract_c_reads_pending_a_state() {
    build_nested(0, 8, true);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_distinct_contract_c_caught_failure_rolls_back() {
    build_nested(4, 8, true);
}
#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_multiple_callbacks_during_one_outbound_call() {
    build_nested(5, 8, true);
}

#[test]
#[ignore = "requires compiled Solidity artifacts"]
fn nested_later_callback_failure_rolls_back_earlier_callbacks() {
    build_nested(6, 8, true);
}
