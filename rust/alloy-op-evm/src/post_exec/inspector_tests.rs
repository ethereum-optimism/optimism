use super::*;
use alloy_primitives::{Address, Bytes, U256, address};
use revm::{
    Context, MainContext,
    context_interface::{CreateScheme, JournalTr},
    database::EmptyDB,
    interpreter::CreateInputs,
};

const CALLER: Address = address!("00000000000000000000000000000000000000aa");

fn create_inputs(scheme: CreateScheme) -> CreateInputs {
    CreateInputs::new(CALLER, scheme, U256::ZERO, Bytes::new(), 100_000, 0)
}

#[test]
fn create_observation_does_not_seed_execution_address_cache() {
    let mut context = Context::mainnet().with_db(EmptyDB::default());
    context.journal_mut().load_account(CALLER).unwrap();
    let inputs = create_inputs(CreateScheme::Create);

    let observation = PostExecCreateObservation::from_evm(&context, &inputs);
    assert_eq!(observation.created_address(), Some(CALLER.create(0)));

    // The observer derived the current address independently. If it had called
    // CreateInputs::created_address(0), revm's OnceCell would ignore this later nonce.
    assert_eq!(inputs.created_address(7), CALLER.create(7));
}

#[test]
fn create_observation_reports_missing_caller_without_panicking() {
    let context = Context::mainnet().with_db(EmptyDB::default());
    let inputs = create_inputs(CreateScheme::Create);

    let observation = PostExecCreateObservation::from_evm(&context, &inputs);
    assert_eq!(observation.caller(), CALLER);
    assert_eq!(observation.scheme(), CreateScheme::Create);
    assert_eq!(observation.created_address(), None);
    assert_eq!(inputs.created_address(7), CALLER.create(7));
}
