//! Execution parity and a serial two-chain continuation experiment.

use op_atomic_builder::{Candidate, Error, Limits, Progress, Reply, discover};
use op_revm::{DefaultOp, OpBuilder, OpContext, OpSpecId, OpTransaction};
use revm::{
    Context, ExecuteEvm,
    context::{CfgEnv, TxEnv},
    database::InMemoryDB,
    primitives::{Address, B256, Bytes, U256, bytes},
    state::{AccountInfo, Bytecode},
};

const CALLER: Address = Address::new([0x10; 20]);
const APP: Address = Address::new([0x20; 20]);
const ENDPOINT: Address = Address::new([0x30; 20]);
const LIMITS: Limits = Limits { calls: 8, bytes_per_call: 4096 };

fn context(code: Bytes, endpoint: Bytes, chain: u64) -> OpContext<InMemoryDB> {
    let mut db = InMemoryDB::default();
    db.insert_account_info(
        CALLER,
        AccountInfo { balance: U256::from(10u64.pow(18)), ..Default::default() },
    );
    for (address, code) in [(APP, code), (ENDPOINT, endpoint)] {
        let code = Bytecode::new_raw(code);
        db.insert_account_info(
            address,
            AccountInfo {
                nonce: 1,
                code_hash: code.hash_slow(),
                code: Some(code),
                ..Default::default()
            },
        );
    }
    let tx = OpTransaction::builder()
        .base(
            TxEnv::builder()
                .caller(CALLER)
                .to(APP)
                .gas_limit(1_000_000)
                .gas_price(1)
                .chain_id(Some(chain)),
        )
        .enveloped_tx(Some(bytes!("02c0")))
        .build_fill();
    Context::op()
        .with_db(db)
        .with_cfg(CfgEnv::new_with_spec(OpSpecId::ISTHMUS).with_chain_id(chain))
        .with_tx(tx)
        .modify_block_chained(|block| {
            block.basefee = 0;
            block.gas_limit = 30_000_000;
        })
}

fn append_call(code: &mut Vec<u8>, output_offset: u8) {
    // CALL(endpoint, gas=100_000, value=0, input=empty, output=32 bytes).
    code.extend([0x60, 32, 0x60, output_offset, 0x60, 0, 0x60, 0, 0x60, 0, 0x73]);
    code.extend(ENDPOINT.as_slice());
    code.extend([0x62, 0x01, 0x86, 0xa0, 0xf1, 0x50]);
}

fn app(revert: bool) -> Bytes {
    // Persist a value, set transient storage, and retain a live stack value.
    let mut code = bytes!("6007600055600960015d60ab").to_vec();
    append_call(&mut code, 0);
    // Warm SLOAD, GAS and TLOAD after the first pause.
    code.extend(bytes!("6000546040525a60605260015c608052"));
    append_call(&mut code, 32);
    // The pre-pause stack value and block gas limit must survive too.
    code.extend(bytes!("60a0524560c05260e06000"));
    code.push(if revert { 0xfd } else { 0xf3 });
    code.into()
}

const fn endpoint() -> Bytes {
    bytes!("602a60005260206000f3")
}

fn complete(progress: Progress<InMemoryDB>) -> Candidate {
    match progress {
        Progress::Complete(candidate) => candidate,
        Progress::Paused(_) => panic!("unexpected additional call"),
    }
}

fn resume_all(mut progress: Progress<InMemoryDB>) -> (Candidate, usize) {
    let mut count = 0;
    loop {
        match progress {
            Progress::Complete(candidate) => return (candidate, count),
            Progress::Paused(paused) => {
                count += 1;
                progress = paused.resume().unwrap();
            }
        }
    }
}

#[test]
fn preserves_stack_memory_gas_warm_storage_and_transient_storage() {
    let (candidate, calls) =
        resume_all(discover(context(app(false), endpoint(), 901), vec![ENDPOINT], LIMITS).unwrap());
    assert_eq!(calls, 2);
    let verified = candidate.verify(context(app(false), endpoint(), 901)).unwrap();
    let output = verified.result.output().unwrap();
    let word = |i: usize| U256::from_be_slice(&output[i * 32..(i + 1) * 32]);
    assert_eq!(word(0), U256::from(42));
    assert_eq!(word(1), U256::from(42));
    assert_eq!(word(2), U256::from(7));
    assert_eq!(word(4), U256::from(9));
    assert_eq!(word(5), U256::from(0xab));
    assert_eq!(word(6), U256::from(30_000_000));
    assert_eq!(verified.state[&APP].storage[&U256::ZERO].present_value(), U256::from(7));
}

#[test]
fn reverts_all_local_changes_after_multiple_pauses() {
    let (candidate, _) =
        resume_all(discover(context(app(true), endpoint(), 901), vec![ENDPOINT], LIMITS).unwrap());
    let verified = candidate.verify(context(app(true), endpoint(), 901)).unwrap();
    assert!(!verified.result.is_success());
    assert!(verified.state[&APP].storage.values().all(|slot| !slot.is_changed()));
    assert_eq!(verified.state[&CALLER].info.nonce, 1);
}

#[test]
fn three_evms_can_remain_suspended_and_resume_in_serial() {
    let mut pending = (901..904)
        .map(|chain| {
            discover(context(app(false), endpoint(), chain), vec![ENDPOINT], LIMITS).unwrap()
        })
        .collect::<Vec<_>>();
    // All three chains have live frames before any endpoint has completed.
    assert!(pending.iter().all(|p| matches!(p, Progress::Paused(_))));
    for round in 0..2 {
        pending = pending
            .into_iter()
            .rev()
            .map(|p| {
                let Progress::Paused(paused) = p else { panic!("premature completion") };
                assert_eq!(paused.request().gas_limit, 100_000);
                paused.resume().unwrap()
            })
            .collect();
        if round == 0 {
            assert!(pending.iter().all(|p| matches!(p, Progress::Paused(_))));
        }
    }
    // Two reversals restore chain order. Each chain gets exactly one normal replay.
    for (chain, progress) in (901..904).zip(pending) {
        complete(progress).verify(context(app(false), endpoint(), chain)).unwrap();
    }
}

fn resolved(value: u8, gas_used: u64) -> Candidate {
    let mut progress =
        discover(context(app(false), endpoint(), 901), vec![ENDPOINT], LIMITS).unwrap();
    loop {
        match progress {
            Progress::Complete(candidate) => return candidate,
            Progress::Paused(paused) => {
                progress = paused
                    .resolve(Reply {
                        success: true,
                        output: U256::from(value).to_be_bytes::<32>().to_vec().into(),
                        gas_used,
                    })
                    .unwrap()
            }
        }
    }
}

#[test]
fn exact_reply_and_local_gas_pass_one_canonical_replay() {
    resolved(42, 18).verify(context(app(false), endpoint(), 901)).unwrap();
}

#[test]
fn wrong_return_value_is_rejected_without_retry() {
    assert!(matches!(
        resolved(43, 18).verify(context(app(false), endpoint(), 901)),
        Err(Error::Diverged)
    ));
}

#[test]
fn wrong_local_gas_is_rejected_even_when_return_value_matches() {
    assert!(matches!(
        resolved(42, 19).verify(context(app(false), endpoint(), 901)),
        Err(Error::Diverged)
    ));
}

#[test]
fn finite_call_budget_stops_ping_pong() {
    let p = discover(
        context(app(false), endpoint(), 901),
        vec![ENDPOINT],
        Limits { calls: 1, ..LIMITS },
    )
    .unwrap();
    let Progress::Paused(paused) = p else { panic!("expected pause") };
    assert!(matches!(paused.resume(), Err(Error::Limit)));
}

#[test]
fn reply_cannot_create_gas_or_unbounded_return_data() {
    for reply in [
        Reply { success: true, output: Bytes::new(), gas_used: 100_001 },
        Reply { success: true, output: vec![0; 4097].into(), gas_used: 0 },
    ] {
        let Progress::Paused(paused) =
            discover(context(app(false), endpoint(), 901), vec![ENDPOINT], LIMITS).unwrap()
        else {
            panic!("expected pause")
        };
        assert!(matches!(paused.resolve(reply), Err(Error::InvalidReply)));
    }
}

#[test]
fn immediate_top_level_completion_uses_normal_post_execution() {
    for code in [Bytes::new(), bytes!("00"), bytes!("60006000fd")] {
        let candidate = complete(
            discover(context(code.clone(), endpoint(), 901), vec![ENDPOINT], LIMITS).unwrap(),
        );
        candidate.verify(context(code, endpoint(), 901)).unwrap();
    }
}

#[test]
fn dropping_a_continuation_does_not_commit_nonce_or_storage() {
    let mut original = context(app(false), endpoint(), 901);
    let mut db = std::mem::take(&mut original.journaled_state.database);
    // Give discovery the actual mutable database, not an independent clone.
    drop(discover(original.with_db(&mut db), vec![ENDPOINT], LIMITS).unwrap());
    assert_eq!(db.cache.accounts[&CALLER].info.nonce, 0);
    // Reading a slot may populate CacheDB; its value must remain the original zero.
    assert_eq!(db.cache.accounts[&APP].storage[&U256::ZERO], U256::ZERO);
}

#[test]
fn pauses_inside_nested_frames_preserve_both_callers() {
    let nested = || {
        let mut ctx = context(app(false), endpoint(), 901);
        let wrapper = Address::new([0x40; 20]);
        let mut code = vec![0x60, 224, 0x60, 0, 0x60, 0, 0x60, 0, 0x60, 0, 0x73];
        code.extend(APP.as_slice());
        code.extend(bytes!("5af15060e06000f3"));
        let code = Bytecode::new_raw(code.into());
        ctx.journaled_state.database.insert_account_info(
            wrapper,
            AccountInfo {
                nonce: 1,
                code_hash: code.hash_slow(),
                code: Some(code),
                ..Default::default()
            },
        );
        ctx.tx.base.kind = wrapper.into();
        ctx
    };
    let mut progress = discover(nested(), vec![ENDPOINT], LIMITS).unwrap();
    for _ in 0..2 {
        let paused = pause(progress);
        assert_eq!(paused.request().caller, APP);
        progress = paused.resolve(reply(42, 18, true)).unwrap();
    }
    complete(progress).verify(nested()).unwrap();
}

#[test]
fn skipped_endpoint_message_logs_are_rejected() {
    let logging_endpoint = bytes!("602a60005260206000a060206000f3");
    let mut progress =
        discover(context(app(false), logging_endpoint.clone(), 901), vec![ENDPOINT], LIMITS)
            .unwrap();
    for _ in 0..2 {
        progress = pause(progress).resolve(reply(42, 655, true)).unwrap();
    }
    let candidate = complete(progress);
    let canonical = context(app(false), logging_endpoint, 901).build_op().replay().unwrap();
    // The synthetic local gas charge is exact: only the missing logs differ.
    assert_eq!(candidate.result().gas(), canonical.result.gas());
    assert_eq!(candidate.result().output(), canonical.result.output());
    assert!(candidate.result().logs().is_empty());
    assert_eq!(canonical.result.logs().len(), 2);
    assert!(matches!(
        candidate.verify(context(app(false), bytes!("602a60005260206000a060206000f3"), 901)),
        Err(Error::Diverged)
    ));
}

#[test]
fn ordinary_replay_still_enforces_transaction_validation() {
    let mut canonical = context(app(false), endpoint(), 901);
    canonical.tx.base.gas_limit = 100;
    assert!(matches!(resolved(42, 18).verify(canonical), Err(Error::Evm(_))));
    assert!(context(app(false), endpoint(), 901).build_op().replay().unwrap().result.is_success());
}

#[test]
fn special_envelopes_cannot_enter_either_execution_path() {
    for kind in [3, 0x7d, 0x7e] {
        let special = || {
            let mut ctx = context(app(false), endpoint(), 901);
            ctx.tx.base.tx_type = kind;
            if kind == 0x7e {
                ctx.tx.deposit.source_hash = B256::with_last_byte(1);
            }
            ctx
        };
        assert!(matches!(
            discover(special(), vec![ENDPOINT], LIMITS),
            Err(Error::UnsupportedTransaction)
        ));
        assert!(matches!(resolved(42, 18).verify(special()), Err(Error::UnsupportedTransaction)));
    }
}

#[test]
fn prior_transaction_gas_does_not_shrink_the_application_budget() {
    let run = |padding_instructions| {
        let mut ctx = context(app(false), endpoint(), 901);
        let padding = Address::new([0x40; 20]);
        let mut code = [0x5a, 0x50].repeat(padding_instructions); // GAS, POP
        code.push(0x00);
        let code = Bytecode::new_raw(code.into());
        ctx.journaled_state.database.insert_account_info(
            padding,
            AccountInfo {
                nonce: 1,
                code_hash: code.hash_slow(),
                code: Some(code),
                ..Default::default()
            },
        );
        let mut application = ctx.tx.clone();
        application.base.nonce = 1;
        let mut prior = ctx.tx.clone();
        prior.base.kind = padding.into();
        let mut evm = ctx.build_op();
        let prefix = evm.transact_one(prior).unwrap();
        assert!(prefix.is_success());
        // A payload builder must still ensure this transaction fits the block.
        assert!(30_000_000 - prefix.tx_gas_used() >= application.base.gas_limit);
        let result = evm.transact_one(application).unwrap();
        (prefix.tx_gas_used(), result)
    };
    let (small, first) = run(0);
    let (large, second) = run(1000);
    assert!(large > small);
    // Includes GAS and GASLIMIT values read by the application.
    assert_eq!(first, second);
}

// This endpoint is a test fixture for an already-materialized response tape.
// Empty input returns `first` (37 gas); nonempty input returns/reverts `second`
// (36 gas). Real inbox witness materialization is a separate adapter concern.
fn response_tape(first: u8, second: u8, fail_second: bool) -> Bytes {
    let mut code = bytes!("3615600f57").to_vec();
    code.extend([
        0x60,
        second,
        0x60,
        0,
        0x52,
        0x60,
        32,
        0x60,
        0,
        if fail_second { 0xfd } else { 0xf3 },
    ]);
    code.extend([0x5b, 0x60, first, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3]);
    code.into()
}

fn call_with_input(code: &mut Vec<u8>, size: u8) {
    code.extend([0x60, 32, 0x60, 0, 0x60, size, 0x60, 0, 0x60, 0, 0x73]);
    code.extend(ENDPOINT.as_slice());
    code.extend([0x62, 0x01, 0x86, 0xa0, 0xf1]);
}

fn root_code() -> Bytes {
    let mut code = Vec::new();
    call_with_input(&mut code, 0);
    code.push(0x50);
    call_with_input(&mut code, 32);
    code.extend([0x15, 0x60, 0, 0x57]);
    let failure_jump = code.len() - 2;
    code.extend(bytes!("60005160005560206000f3"));
    code[failure_jump] = u8::try_from(code.len()).unwrap();
    code.extend(bytes!("5b60206000fd"));
    code.into()
}

fn remote_code(revert: bool) -> Bytes {
    let mut code = Vec::new();
    call_with_input(&mut code, 0);
    code.extend(bytes!("50600051600055")); // counter = command one
    call_with_input(&mut code, 32); // yield result one; wait for command two
    code.extend(bytes!("50600051600054018060005560005260206000"));
    code.push(if revert { 0xfd } else { 0xf3 });
    code.into()
}

fn pause(progress: Progress<InMemoryDB>) -> Box<op_atomic_builder::Paused<InMemoryDB>> {
    let Progress::Paused(paused) = progress else { panic!("expected remote boundary") };
    paused
}

fn reply(value: u8, gas_used: u64, success: bool) -> Reply {
    Reply { success, output: U256::from(value).to_be_bytes::<32>().to_vec().into(), gas_used }
}

#[test]
fn a_to_b_to_a_to_b_uses_live_frames_and_one_final_replay_per_chain() {
    for remote_reverts in [false, true] {
        let a = || context(root_code(), response_tape(6, 13, remote_reverts), 901);
        let b = || context(remote_code(remote_reverts), response_tape(6, 7, false), 902);
        let a_waiting = pause(discover(a(), vec![ENDPOINT], LIMITS).unwrap());
        let b_waiting = pause(discover(b(), vec![ENDPOINT], LIMITS).unwrap());

        // A's first request schedules B's first leg. B stays alive after producing 6.
        let b_waiting = pause(b_waiting.resolve(reply(6, 37, true)).unwrap());
        assert_eq!(U256::from_be_slice(&b_waiting.request().input), U256::from(6));
        let a_waiting = pause(
            a_waiting
                .resolve(Reply {
                    success: true,
                    output: b_waiting.request().input.clone(),
                    gas_used: 37,
                })
                .unwrap(),
        );

        // A resumes and schedules B again. B continues with the SAME warm journal
        // and updates its earlier value from 6 to 13, without replaying its prefix.
        assert_eq!(U256::from_be_slice(&a_waiting.request().input), U256::from(6));
        let b_candidate = complete(b_waiting.resolve(reply(7, 36, true)).unwrap());
        assert_eq!(b_candidate.result().is_success(), !remote_reverts);
        let a_candidate = complete(
            a_waiting
                .resolve(Reply {
                    success: b_candidate.result().is_success(),
                    output: b_candidate.result().output().unwrap().clone(),
                    gas_used: 36,
                })
                .unwrap(),
        );

        let a_verified = a_candidate.verify(a()).unwrap();
        let b_verified = b_candidate.verify(b()).unwrap();
        assert_eq!(a_verified.result.is_success(), !remote_reverts);
        assert_eq!(b_verified.result.is_success(), !remote_reverts);
        for state in [a_verified.state, b_verified.state] {
            if remote_reverts {
                assert!(state[&APP].storage.values().all(|slot| !slot.is_changed()));
            } else {
                assert_eq!(state[&APP].storage[&U256::ZERO].present_value(), U256::from(13));
            }
        }
    }
}
