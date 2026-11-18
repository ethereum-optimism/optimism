//! [`PrecompileProvider`] for FPVM-accelerated OP Stack precompiles.

use crate::fpvm_evm::precompiles::{
    ecrecover::ECRECOVER_ADDR, kzg_point_eval::KZG_POINT_EVAL_ADDR,
};
use alloc::{string::String, vec, vec::Vec};
use alloy_primitives::{Address, Bytes};
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use op_revm::{
    OpSpecId,
    precompiles::{fjord, granite, isthmus, jovian, karst, lagoon},
};
use revm::{
    context::{Cfg, ContextTr},
    handler::{EthPrecompiles, PrecompileProvider, precompile_output_to_interpreter_result},
    interpreter::{CallInputs, InterpreterResult},
    precompile::{
        EthPrecompileResult, PrecompileError, PrecompileOutput, Precompiles, bls12_381_const, bn254,
    },
    primitives::{AddressSet, hardfork::SpecId, hash_map::HashMap},
};

/// The FPVM-accelerated precompiles.
#[derive(Debug)]
pub struct OpFpvmPrecompiles<H, O> {
    /// The default [`EthPrecompiles`] provider.
    inner: EthPrecompiles,
    /// The accelerated precompiles for the current [`OpSpecId`].
    accelerated_precompiles: HashMap<Address, AcceleratedPrecompileFn<H, O>>,
    /// The [`OpSpecId`] of the precompiles.
    spec: OpSpecId,
    /// The inner [`HintWriterClient`].
    hint_writer: H,
    /// The inner [`PreimageOracleClient`].
    oracle_reader: O,
}

impl<H, O> OpFpvmPrecompiles<H, O>
where
    H: HintWriterClient + Clone + Send + Sync + 'static,
    O: PreimageOracleClient + Clone + Send + Sync + 'static,
{
    /// Create a new precompile provider with the given [`OpSpecId`].
    #[inline]
    pub fn new_with_spec(spec: OpSpecId, hint_writer: H, oracle_reader: O) -> Self {
        let precompiles = match spec {
            spec @ (OpSpecId::BEDROCK |
            OpSpecId::REGOLITH |
            OpSpecId::CANYON |
            OpSpecId::ECOTONE) => Precompiles::new(spec.into_eth_spec().into()),
            OpSpecId::FJORD => fjord(),
            OpSpecId::GRANITE | OpSpecId::HOLOCENE => granite(),
            OpSpecId::ISTHMUS => isthmus(),
            OpSpecId::JOVIAN => jovian(),
            OpSpecId::KARST => karst(),
            OpSpecId::LAGOON => lagoon(),
        };

        let accelerated_precompiles = match spec {
            OpSpecId::BEDROCK | OpSpecId::REGOLITH | OpSpecId::CANYON => {
                accelerated_bedrock::<H, O>()
            }
            OpSpecId::ECOTONE | OpSpecId::FJORD => accelerated_ecotone::<H, O>(),
            OpSpecId::GRANITE | OpSpecId::HOLOCENE => accelerated_granite::<H, O>(),
            OpSpecId::ISTHMUS => accelerated_isthmus::<H, O>(),
            OpSpecId::JOVIAN => accelerated_jovian::<H, O>(),
            OpSpecId::KARST => accelerated_karst::<H, O>(),
            OpSpecId::LAGOON => accelerated_lagoon::<H, O>(),
        };

        Self {
            inner: EthPrecompiles { precompiles, spec: SpecId::default() },
            accelerated_precompiles: accelerated_precompiles
                .into_iter()
                .map(|p| (p.address, p.precompile))
                .collect(),
            spec,
            hint_writer,
            oracle_reader,
        }
    }
}

impl<CTX, H, O> PrecompileProvider<CTX> for OpFpvmPrecompiles<H, O>
where
    H: HintWriterClient + Clone + Send + Sync + 'static,
    O: PreimageOracleClient + Clone + Send + Sync + 'static,
    CTX: ContextTr<Cfg: Cfg<Spec = OpSpecId>>,
{
    type Output = InterpreterResult;

    #[inline]
    fn set_spec(&mut self, spec: <CTX::Cfg as Cfg>::Spec) -> bool {
        if spec == self.spec {
            return false;
        }
        *self = Self::new_with_spec(spec, self.hint_writer.clone(), self.oracle_reader.clone());
        true
    }

    /// UPSTREAM-MIRROR(copy): revm-handler@43.0.3 `revm_handler::EthPrecompiles::run`
    ///
    /// FPVM acceleration replaces dispatch, while output conversion delegates to revm. Re-check
    /// input materialization, execution errors, and gas/reservoir propagation across both paths.
    /// Unlike the RPC-facing provider, this provider does not record precompile error context.
    #[inline]
    fn run(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
    ) -> Result<Option<Self::Output>, String> {
        use revm::context::LocalContextTr;
        let input = match &inputs.input {
            revm::interpreter::CallInput::Bytes(bytes) => bytes.clone(),
            revm::interpreter::CallInput::SharedBuffer(range) => context
                .local()
                .shared_memory_buffer_slice(range.clone())
                .map(|b| Bytes::from(b.to_vec()))
                .unwrap_or_default(),
        };

        // Priority:
        // 1. If the precompile has an accelerated version, use that.
        // 2. If the precompile is not accelerated, use the default version.
        // 3. If the precompile is not found, return None.
        let output =
            if let Some(accelerated) = self.accelerated_precompiles.get(&inputs.bytecode_address) {
                let eth_result =
                    (accelerated)(&input, inputs.gas_limit, &self.hint_writer, &self.oracle_reader);
                PrecompileOutput::from_eth_result(eth_result, inputs.reservoir)
            } else if let Some(precompile) = self.inner.precompiles.get(&inputs.bytecode_address) {
                match precompile.execute(&input, inputs.gas_limit, inputs.reservoir) {
                    Ok(output) => output,
                    Err(PrecompileError::Fatal(e)) => return Err(e),
                    Err(PrecompileError::FatalAny(e)) => return Err(alloc::format!("{e:?}")),
                }
            } else {
                return Ok(None);
            };

        // Turning the output into an interpreter result is revm's job, not ours: every field it
        // sets on the result gas has to reach the caller frame the same way it does under
        // `EthPrecompiles`, which op-reth uses.
        Ok(Some(precompile_output_to_interpreter_result(output, inputs.gas_limit)))
    }

    #[inline]
    fn warm_addresses(&self) -> &AddressSet {
        self.inner.warm_addresses()
    }

    #[inline]
    fn contains(&self, address: &Address) -> bool {
        self.inner.contains(address)
    }
}

/// A precompile function that can be accelerated by the FPVM.
type AcceleratedPrecompileFn<H, O> = fn(&[u8], u64, &H, &O) -> EthPrecompileResult;

/// A tuple type for accelerated precompiles with an associated [`Address`].
struct AcceleratedPrecompile<H, O> {
    /// The address of the precompile.
    address: Address,
    /// The precompile function.
    precompile: AcceleratedPrecompileFn<H, O>,
}

impl<H, O> AcceleratedPrecompile<H, O> {
    /// Create a new accelerated precompile.
    fn new(address: Address, precompile: AcceleratedPrecompileFn<H, O>) -> Self {
        Self { address, precompile }
    }
}

/// The accelerated precompiles for the bedrock spec.
fn accelerated_bedrock<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    vec![
        AcceleratedPrecompile::new(ECRECOVER_ADDR, super::ecrecover::fpvm_ec_recover::<H, O>),
        AcceleratedPrecompile::new(
            bn254::pair::ADDRESS,
            super::bn128_pair::fpvm_bn128_pair::<H, O>,
        ),
    ]
}

/// The accelerated precompiles for the ecotone spec.
fn accelerated_ecotone<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_bedrock::<H, O>();
    base.push(AcceleratedPrecompile::new(
        KZG_POINT_EVAL_ADDR,
        super::kzg_point_eval::fpvm_kzg_point_eval::<H, O>,
    ));
    base
}

/// The accelerated precompiles for the granite spec.
fn accelerated_granite<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_ecotone::<H, O>();
    base.push(AcceleratedPrecompile::new(
        bn254::pair::ADDRESS,
        super::bn128_pair::fpvm_bn128_pair_granite::<H, O>,
    ));
    base
}

/// The accelerated precompiles for the isthmus spec.
fn accelerated_isthmus<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_granite::<H, O>();
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G1_ADD_ADDRESS,
        super::bls12_g1_add::fpvm_bls12_g1_add::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G1_MSM_ADDRESS,
        super::bls12_g1_msm::fpvm_bls12_g1_msm::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G2_ADD_ADDRESS,
        super::bls12_g2_add::fpvm_bls12_g2_add::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G2_MSM_ADDRESS,
        super::bls12_g2_msm::fpvm_bls12_g2_msm::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::MAP_FP_TO_G1_ADDRESS,
        super::bls12_map_fp::fpvm_bls12_map_fp::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::MAP_FP2_TO_G2_ADDRESS,
        super::bls12_map_fp2::fpvm_bls12_map_fp2::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::PAIRING_ADDRESS,
        super::bls12_pair::fpvm_bls12_pairing::<H, O>,
    ));
    base
}

/// The accelerated precompiles for the jovian spec.
fn accelerated_jovian<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_isthmus::<H, O>();

    // Replace the 4 variable-input precompiles with Jovian versions (reduced limits)
    base.retain(|p| {
        p.address != bn254::pair::ADDRESS &&
            p.address != bls12_381_const::G1_MSM_ADDRESS &&
            p.address != bls12_381_const::G2_MSM_ADDRESS &&
            p.address != bls12_381_const::PAIRING_ADDRESS
    });

    base.push(AcceleratedPrecompile::new(
        bn254::pair::ADDRESS,
        super::bn128_pair::fpvm_bn128_pair_jovian::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G1_MSM_ADDRESS,
        super::bls12_g1_msm::fpvm_bls12_g1_msm_jovian::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G2_MSM_ADDRESS,
        super::bls12_g2_msm::fpvm_bls12_g2_msm_jovian::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::PAIRING_ADDRESS,
        super::bls12_pair::fpvm_bls12_pairing_jovian::<H, O>,
    ));

    base
}

/// The accelerated precompiles for the karst spec.
fn accelerated_karst<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_jovian::<H, O>();

    // Replace the bn254 pair precompile with the Karst version (reduced input size limit).
    base.retain(|p| p.address != bn254::pair::ADDRESS);
    base.push(AcceleratedPrecompile::new(
        bn254::pair::ADDRESS,
        super::bn128_pair::fpvm_bn128_pair_karst::<H, O>,
    ));

    base
}

/// The accelerated precompiles for the lagoon spec.
fn accelerated_lagoon<H, O>() -> Vec<AcceleratedPrecompile<H, O>>
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let mut base = accelerated_karst::<H, O>();

    base.retain(|p| {
        p.address != bls12_381_const::G1_MSM_ADDRESS &&
            p.address != bls12_381_const::G2_MSM_ADDRESS &&
            p.address != bls12_381_const::PAIRING_ADDRESS
    });

    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G1_MSM_ADDRESS,
        super::bls12_g1_msm::fpvm_bls12_g1_msm_lagoon::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::G2_MSM_ADDRESS,
        super::bls12_g2_msm::fpvm_bls12_g2_msm_lagoon::<H, O>,
    ));
    base.push(AcceleratedPrecompile::new(
        bls12_381_const::PAIRING_ADDRESS,
        super::bls12_pair::fpvm_bls12_pairing_lagoon::<H, O>,
    ));

    base
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{VALID_KZG_EVALUATION, kzg_point_eval_input};
    use alloy_op_evm::{OpEvmContext, OpTx};
    use alloy_primitives::hex;
    use kona_preimage::{HintWriterClient, PreimageOracleClient};
    use op_revm::{L1BlockInfo, OpSpecId, OpTransaction};
    use revm::{
        Context, MainContext,
        database::EmptyDB,
        handler::PrecompileProvider,
        interpreter::{CallInput, InstructionResult},
    };

    type TestContext = OpEvmContext<EmptyDB>;

    fn create_call_inputs(address: Address, input: Bytes, gas_limit: u64) -> CallInputs {
        create_call_inputs_with_reservoir(address, input, gas_limit, 0)
    }

    fn create_call_inputs_with_reservoir(
        address: Address,
        input: Bytes,
        gas_limit: u64,
        reservoir: u64,
    ) -> CallInputs {
        CallInputs {
            input: CallInput::Bytes(input),
            return_memory_offset: 0..0,
            gas_limit,
            reservoir,
            bytecode_address: address,
            known_bytecode: (revm::primitives::KECCAK_EMPTY, revm::bytecode::Bytecode::new()),
            target_address: Address::ZERO,
            caller: Address::ZERO,
            value: revm::interpreter::CallValue::Transfer(alloy_primitives::U256::ZERO),
            scheme: revm::interpreter::CallScheme::Call,
            is_static: false,
            charged_new_account_state_gas: false,
        }
    }

    fn create_test_context() -> TestContext {
        Context::mainnet()
            .with_tx(OpTx(OpTransaction::builder().build_fill()))
            .with_cfg(revm::context::CfgEnv::new_with_spec(OpSpecId::BEDROCK))
            .with_chain(L1BlockInfo::default())
            .with_db(EmptyDB::new())
    }

    /// A mock accelerated precompile function that returns a fixed output.
    fn mock_accelerated_precompile<H, O>(
        _input: &[u8],
        gas_limit: u64,
        _hint_writer: &H,
        _oracle_reader: &O,
    ) -> EthPrecompileResult
    where
        H: HintWriterClient + Send + Sync,
        O: PreimageOracleClient + Send + Sync,
    {
        Ok(revm::precompile::EthPrecompileOutput::new(gas_limit / 2, Bytes::from_static(b"mock")))
    }

    #[test]
    fn test_run_accelerated_precompile() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // Override the ecrecover accelerated precompile with our mock
        precompiles.accelerated_precompiles.insert(ECRECOVER_ADDR, mock_accelerated_precompile);

        let call_inputs = create_call_inputs(ECRECOVER_ADDR, Bytes::from_static(b"test"), 1000);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap();
        assert!(result.is_some());

        let interpreter_result = result.unwrap();
        assert_eq!(interpreter_result.result, InstructionResult::Return);
        assert_eq!(interpreter_result.output.as_ref(), b"mock");
    }

    /// A mock accelerated precompile that reports spending more gas than the call limit.
    fn overspending_accelerated_precompile<H, O>(
        _input: &[u8],
        gas_limit: u64,
        _hint_writer: &H,
        _oracle_reader: &O,
    ) -> EthPrecompileResult
    where
        H: HintWriterClient + Send + Sync,
        O: PreimageOracleClient + Send + Sync,
    {
        Ok(revm::precompile::EthPrecompileOutput::new(
            gas_limit.saturating_add(1),
            Bytes::from_static(b"overspend"),
        ))
    }

    #[test]
    fn test_run_overspending_precompile_oogs_without_panicking() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // A precompile that reports gas_used > gas_limit must gracefully OOG, matching
        // revm's `EthPrecompiles::run`, rather than panicking the fault-proof program.
        precompiles
            .accelerated_precompiles
            .insert(ECRECOVER_ADDR, overspending_accelerated_precompile);

        let call_inputs = create_call_inputs(ECRECOVER_ADDR, Bytes::from_static(b"test"), 1000);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap().unwrap();
        assert_eq!(result.result, InstructionResult::PrecompileOOG);
        assert!(result.output.is_empty());
        assert_eq!(result.gas.remaining(), 0);
    }

    /// A mock accelerated precompile that halts.
    fn halting_accelerated_precompile<H, O>(
        _input: &[u8],
        _gas_limit: u64,
        _hint_writer: &H,
        _oracle_reader: &O,
    ) -> EthPrecompileResult
    where
        H: HintWriterClient + Send + Sync,
        O: PreimageOracleClient + Send + Sync,
    {
        Err(revm::precompile::PrecompileHalt::OutOfGas)
    }

    #[test]
    fn test_run_halting_precompile_spends_all_gas() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // A halted precompile must consume all gas, matching revm's
        // `precompile_output_to_interpreter_result`, rather than leaving gas refundable.
        precompiles.accelerated_precompiles.insert(ECRECOVER_ADDR, halting_accelerated_precompile);

        let call_inputs = create_call_inputs(ECRECOVER_ADDR, Bytes::from_static(b"test"), 1000);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap().unwrap();
        assert_eq!(result.result, InstructionResult::PrecompileOOG);
        assert!(result.output.is_empty());
        assert_eq!(result.gas.remaining(), 0);
    }

    #[test]
    fn test_run_default_precompile_sha256() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // SHA256 precompile address (0x02) - not accelerated, uses default
        let sha256_addr = revm::precompile::u64_to_address(2);
        let input = b"hello world";
        let call_inputs = create_call_inputs(sha256_addr, input.to_vec().into(), u64::MAX);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap();
        assert!(result.is_some());

        let interpreter_result = result.unwrap();
        assert_eq!(interpreter_result.result, InstructionResult::Return);
        assert!(!interpreter_result.output.is_empty());
    }

    /// At every address it does not accelerate, this provider is a pass-through to the
    /// [`EthPrecompiles`] it wraps — the one op-reth runs — so it has to hand back exactly what
    /// that provider would. Compares the whole [`InterpreterResult`], gas tracker included,
    /// rather than the few fields any one test happens to look at: the conversion from
    /// [`PrecompileOutput`] carries more state than a returned status and gas cost.
    #[test]
    fn test_non_accelerated_result_matches_wrapped_provider() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::KARST, hint_writer, oracle_reader);
        let mut wrapped = precompiles.inner.clone();

        // Neither address is accelerated at KARST. SHA-256 (0x02) succeeds, or runs out of gas
        // when the call forwards none; blake2f (0x09) rejects any input that is not exactly 213
        // bytes, halting for a reason that is not out-of-gas. Each is driven with and without a
        // reservoir on the call.
        let sha256 = revm::precompile::u64_to_address(2);
        let blake2f = revm::precompile::u64_to_address(9);
        let cases: [(Address, &[u8], u64, u64); 6] = [
            (sha256, b"hello world", 100_000, 0),
            (sha256, b"hello world", 100_000, 4_096),
            (sha256, b"hello world", 0, 0),
            (sha256, b"hello world", 0, 4_096),
            (blake2f, b"too short", 100_000, 0),
            (blake2f, b"too short", 100_000, 4_096),
        ];

        for (address, input, gas_limit, reservoir) in cases {
            let call_inputs = create_call_inputs_with_reservoir(
                address,
                input.to_vec().into(),
                gas_limit,
                reservoir,
            );

            let ours = precompiles.run(&mut ctx, &call_inputs).unwrap();
            let theirs =
                PrecompileProvider::<TestContext>::run(&mut wrapped, &mut ctx, &call_inputs)
                    .unwrap();

            assert_eq!(
                ours, theirs,
                "diverged from the wrapped provider at {address} with gas limit {gas_limit} \
                 and reservoir {reservoir}",
            );
        }
    }

    /// A call that passes every check an accelerated precompile makes itself, gas and length, but
    /// whose input the host rejects. The client learns of the failure only from the host's answer.
    struct HostRejectedCall {
        name: &'static str,
        address: Address,
        /// The first spec that accelerates `address`.
        first_spec: OpSpecId,
        input: Vec<u8>,
    }

    /// One host-rejected call for each accelerated precompile whose host can reject an input.
    /// ecrecover is not one: revm returns no output for a bad signature instead of failing, and
    /// the client maps any host failure to the same empty output.
    fn host_rejected_calls() -> Vec<HostRejectedCall> {
        let mut kzg_wrong_evaluation = VALID_KZG_EVALUATION;
        kzg_wrong_evaluation[31] ^= 1;

        // G1 point (1, 3) is not on y^2 = x^3 + 3; the G2 point is the point at infinity.
        let mut bn254_off_curve = vec![0u8; bn254::PAIR_ELEMENT_LEN];
        bn254_off_curve[31] = 1;
        bn254_off_curve[63] = 3;

        vec![
            HostRejectedCall {
                name: "bn254 pairing: G1 point not on the curve",
                address: bn254::pair::ADDRESS,
                first_spec: OpSpecId::BEDROCK,
                input: bn254_off_curve,
            },
            HostRejectedCall {
                name: "kzg point evaluation: proof does not verify",
                address: KZG_POINT_EVAL_ADDR,
                first_spec: OpSpecId::ECOTONE,
                input: kzg_point_eval_input(kzg_wrong_evaluation),
            },
            // One failure vector per BLS12-381 precompile from the EIP-2537 test suite, named as
            // in kona/sp1/crates/bls12-381-check/testdata/eip2537.
            HostRejectedCall {
                name: "bls_g1add_point_not_on_curve",
                address: bls12_381_const::G1_ADD_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a2100000000000000000000000000000000112b98340eee2777cc3c14163dea3ec97977ac3dc5c70da32e6e87578f44912e902ccef9efe28d4a78b8999dfbca942600000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a21").to_vec(),
            },
            HostRejectedCall {
                name: "bls_g2add_invalid_field_element",
                address: bls12_381_const::G2_ADD_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("000000000000000000000000000000001c4bb49d2a0ef12b7123acdd7110bd292b5bc659edc54dc21b81de057194c79b2a5803255959bbef8e7f56c8c12168630000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be00000000000000000000000000000000103121a2ceaae586d240843a398967325f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f2700000000000000000000000000000000086b990f3da2aeac0a36143b7d7c824428215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d878451").to_vec(),
            },
            HostRejectedCall {
                name: "bls_g1msm_point_not_on_curve",
                address: bls12_381_const::G1_MSM_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb00000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a21000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000112b98340eee2777cc3c14163dea3ec97977ac3dc5c70da32e6e87578f44912e902ccef9efe28d4a78b8999dfbca942600000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbdede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a210000000000000000000000000000000000000000000000000000000000000002").to_vec(),
            },
            HostRejectedCall {
                name: "bls_g2msm_invalid_field_element",
                address: bls12_381_const::G2_MSM_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("000000000000000000000000000000001c4bb49d2a0ef12b7123acdd7110bd292b5bc659edc54dc21b81de057194c79b2a5803255959bbef8e7f56c8c12168630000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000103121a2ceaae586d240843a398967325f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f2700000000000000000000000000000000086b990f3da2aeac0a36143b7d7c824428215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d8784510000000000000000000000000000000000000000000000000000000000000002").to_vec(),
            },
            HostRejectedCall {
                name: "bls_invalid_fq_element",
                address: bls12_381_const::MAP_FP_TO_G1_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("000000000000000000000000000000002f6d9c5465982c0421b61e74579709b3b5b91e57bdd4f6015742b4ff301abb7ef895b9cce00c33c7d48f8e5fa4ac09ae").to_vec(),
            },
            HostRejectedCall {
                name: "bls_mapg2_invalid_fq_element",
                address: bls12_381_const::MAP_FP2_TO_G2_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("0000000000000000000000000000000021366f100476ce8d3be6cfc90d59fe13349e388ed12b6dd6dc31ccd267ff000e2c993a063ca66beced06f804d4b8e5af0000000000000000000000000000000002829ce3c021339ccb5caf3e187f6370e1e2a311dec9b75363117063ab2015603ff52c3d3b98f19c2f65575e99e8b78c").to_vec(),
            },
            HostRejectedCall {
                name: "bls_pairing_e(G1_not_in_correct_subgroup,-G2)=e(-G1,G2)",
                address: bls12_381_const::PAIRING_ADDRESS,
                first_spec: OpSpecId::ISTHMUS,
                input: hex!("000000000000000000000000000000000123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00000000000000000000000000000000193fb7cedb32b2c3adc06ec11a96bc0d661869316f5e4a577a9f7c179593987beb4fb2ee424dbb2f5dd891e228b46c4a00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb80000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0fc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4fcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e100000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb80000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000d1b3cc2c7027888be51d9ef691d77bcb679afda66c73f17f9ee3837a55024f78c71363275a75d75d86bab79f74782aa0000000000000000000000000000000013fa4d4a0ad8b1ce186ed5061789213d993923066dddaf1040bc3ff59f825c78df74f2d75467e25e0f55f8a00fa030ed").to_vec(),
            },
        ]
    }

    /// For an input the host rejects, an accelerated precompile has to return what the
    /// [`EthPrecompiles`] it wraps, the one op-reth runs, returns for the same call: a halt that
    /// spends all of the call's gas. Any other status or gas makes kona and op-reth compute
    /// different receipts for the same block. Each case runs at every spec from the one that
    /// first accelerates it, because a fork can swap in its own accelerated function, and with
    /// and without a reservoir on the call.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_host_rejected_input_matches_wrapped_provider() {
        use crate::fpvm_evm::precompiles::test_utils::test_accelerated_precompile;

        const SPECS: [OpSpecId; 11] = [
            OpSpecId::BEDROCK,
            OpSpecId::REGOLITH,
            OpSpecId::CANYON,
            OpSpecId::ECOTONE,
            OpSpecId::FJORD,
            OpSpecId::GRANITE,
            OpSpecId::HOLOCENE,
            OpSpecId::ISTHMUS,
            OpSpecId::JOVIAN,
            OpSpecId::KARST,
            OpSpecId::LAGOON,
        ];
        const GAS_LIMIT: u64 = 1_000_000;

        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let mut ctx = create_test_context();
            for spec in SPECS {
                let mut precompiles = OpFpvmPrecompiles::new_with_spec(
                    spec,
                    hint_writer.clone(),
                    oracle_reader.clone(),
                );
                let mut wrapped = precompiles.inner.clone();

                for HostRejectedCall { name, address, first_spec, input } in host_rejected_calls() {
                    if !spec.is_enabled_in(first_spec) {
                        continue;
                    }

                    // Without a host rejection the case would test a check the client makes
                    // itself, not the host-failure path.
                    let accelerated = precompiles
                        .accelerated_precompiles
                        .get(&address)
                        .copied()
                        .unwrap_or_else(|| panic!("{name}: not accelerated at {spec:?}"));
                    let res = accelerated(&input, GAS_LIMIT, hint_writer, oracle_reader);
                    assert!(
                        matches!(&res, Err(revm::precompile::PrecompileHalt::Other(msg)) if msg.contains("Error executing precompile in host")),
                        "{name} at {spec:?}: expected the host to reject the input, got {res:?}"
                    );

                    for reservoir in [0, 4_096] {
                        let call_inputs = create_call_inputs_with_reservoir(
                            address,
                            input.clone().into(),
                            GAS_LIMIT,
                            reservoir,
                        );
                        let ours = precompiles.run(&mut ctx, &call_inputs).unwrap();
                        let theirs = PrecompileProvider::<TestContext>::run(
                            &mut wrapped,
                            &mut ctx,
                            &call_inputs,
                        )
                        .unwrap();

                        assert!(
                            theirs.as_ref().is_some_and(|r| r.result == InstructionResult::PrecompileError),
                            "{name} at {spec:?}: the wrapped provider must reject the input, got {theirs:?}"
                        );
                        assert_eq!(
                            ours, theirs,
                            "{name} at {spec:?} with reservoir {reservoir}: diverged from the wrapped provider"
                        );
                    }
                }
            }

            // Every accelerated precompile but ecrecover has a case, so a newly accelerated one
            // fails here until it gets one.
            let lagoon = OpFpvmPrecompiles::new_with_spec(
                OpSpecId::LAGOON,
                hint_writer.clone(),
                oracle_reader.clone(),
            );
            let mut accelerated: Vec<_> = lagoon
                .accelerated_precompiles
                .keys()
                .copied()
                .filter(|address| *address != ECRECOVER_ADDR)
                .collect();
            accelerated.sort();
            let mut covered: Vec<_> = host_rejected_calls().iter().map(|c| c.address).collect();
            covered.sort();
            assert_eq!(accelerated, covered, "accelerated precompiles without a host-rejected call");
        })
        .await;
    }

    #[test]
    fn test_run_nonexistent_precompile() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // Non-existent precompile address
        let fake_addr = Address::from_slice(&[0xFFu8; 20]);
        let call_inputs = create_call_inputs(fake_addr, Bytes::new(), u64::MAX);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap();
        assert!(result.is_none());
    }

    #[test]
    fn test_run_out_of_gas() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // SHA256 with 0 gas to trigger OOG
        let sha256_addr = revm::precompile::u64_to_address(2);
        let input = b"hello world";
        let call_inputs = create_call_inputs(sha256_addr, input.to_vec().into(), 0);

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap();
        assert!(result.is_some());

        let interpreter_result = result.unwrap();
        assert_eq!(interpreter_result.result, InstructionResult::PrecompileOOG);
    }

    #[test]
    fn test_post_jovian_specs_use_jovian_precompiles() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let jovian_provider = OpFpvmPrecompiles::new_with_spec(
            OpSpecId::JOVIAN,
            hint_writer.clone(),
            oracle_reader.clone(),
        );
        let lagoon_provider = OpFpvmPrecompiles::new_with_spec(
            OpSpecId::LAGOON,
            hint_writer.clone(),
            oracle_reader.clone(),
        );
        let karst_provider = OpFpvmPrecompiles::new_with_spec(
            OpSpecId::KARST,
            hint_writer.clone(),
            oracle_reader.clone(),
        );
        let isthmus_provider =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::ISTHMUS, hint_writer, oracle_reader);

        // Each post-Jovian spec accelerates the same set of addresses as the previous fork.
        // The dispatched function at a given address may still change (e.g. KARST swaps the
        // bn254 pair function at 0x08 for a stricter input-size check).
        let jovian_addrs: Vec<_> = {
            let mut addrs: Vec<_> =
                jovian_provider.accelerated_precompiles.keys().copied().collect();
            addrs.sort();
            addrs
        };
        let karst_addrs: Vec<_> = {
            let mut addrs: Vec<_> =
                karst_provider.accelerated_precompiles.keys().copied().collect();
            addrs.sort();
            addrs
        };
        let lagoon_addrs: Vec<_> = {
            let mut addrs: Vec<_> =
                lagoon_provider.accelerated_precompiles.keys().copied().collect();
            addrs.sort();
            addrs
        };
        assert_eq!(
            jovian_addrs, karst_addrs,
            "KARST should accelerate the same addresses as JOVIAN (functions may differ)"
        );
        assert_eq!(
            karst_addrs, lagoon_addrs,
            "LAGOON should accelerate the same addresses as KARST (functions may differ)"
        );

        // Verify the non-accelerated precompile sets point to the correct static instances.
        assert!(
            core::ptr::eq(jovian_provider.inner.precompiles, jovian()),
            "JOVIAN should use jovian() precompiles"
        );
        assert!(
            core::ptr::eq(isthmus_provider.inner.precompiles, isthmus()),
            "ISTHMUS should use isthmus() precompiles"
        );
        assert!(
            core::ptr::eq(karst_provider.inner.precompiles, karst()),
            "KARST should use karst() precompiles"
        );
        assert!(
            core::ptr::eq(lagoon_provider.inner.precompiles, lagoon()),
            "LAGOON should use lagoon() precompiles"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_karst_bn128_pair_enforces_karst_limit() {
        use crate::fpvm_evm::precompiles::test_utils::test_accelerated_precompile;

        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // 301 pairs × 192 bytes = 57_792 — aligned to PAIR_ELEMENT_LEN and one pair
            // above BN256_MAX_PAIRING_SIZE_KARST (57_600).
            const OVER_KARST_LIMIT: usize = 57_792;
            let input = vec![0u8; OVER_KARST_LIMIT];

            let karst_provider = OpFpvmPrecompiles::new_with_spec(
                OpSpecId::KARST,
                hint_writer.clone(),
                oracle_reader.clone(),
            );
            let karst_fn = karst_provider
                .accelerated_precompiles
                .get(&bn254::pair::ADDRESS)
                .copied()
                .expect("KARST must have bn254 pair accelerated precompile");
            let karst_res = karst_fn(&input, u64::MAX, hint_writer, oracle_reader);
            assert!(
                matches!(karst_res, Err(revm::precompile::PrecompileHalt::Bn254PairLength)),
                "KARST should reject input > 57_600 bytes with Bn254PairLength"
            );
        })
        .await;
    }

    /// The accelerated BLS12-381 MSM and pairing precompiles at LAGOON accept inputs up to
    /// op-revm's Lagoon limits and reject inputs one element above them, as op-reth does.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_lagoon_bls12_381_enforces_lagoon_limits() {
        use crate::fpvm_evm::precompiles::test_utils::test_accelerated_precompile;
        use op_revm::precompiles::bls12_381::{
            LAGOON_G1_MSM_MAX_INPUT_SIZE, LAGOON_G2_MSM_MAX_INPUT_SIZE,
            LAGOON_PAIRING_MAX_INPUT_SIZE,
        };
        use revm::precompile::bls12_381_const::{
            G1_MSM_INPUT_LENGTH, G2_MSM_INPUT_LENGTH, PAIRING_INPUT_LENGTH,
        };

        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let lagoon_provider = OpFpvmPrecompiles::new_with_spec(
                OpSpecId::LAGOON,
                hint_writer.clone(),
                oracle_reader.clone(),
            );

            for (address, max_input_size, element_len) in [
                (bls12_381_const::G1_MSM_ADDRESS, LAGOON_G1_MSM_MAX_INPUT_SIZE, G1_MSM_INPUT_LENGTH),
                (bls12_381_const::G2_MSM_ADDRESS, LAGOON_G2_MSM_MAX_INPUT_SIZE, G2_MSM_INPUT_LENGTH),
                (
                    bls12_381_const::PAIRING_ADDRESS,
                    LAGOON_PAIRING_MAX_INPUT_SIZE,
                    PAIRING_INPUT_LENGTH,
                ),
            ] {
                let lagoon_fn = lagoon_provider
                    .accelerated_precompiles
                    .get(&address)
                    .copied()
                    .expect("LAGOON must accelerate the BLS12-381 MSM and pairing precompiles");

                let at_limit = vec![0u8; max_input_size];
                let res = lagoon_fn(&at_limit, u64::MAX, hint_writer, oracle_reader);
                assert!(res.is_ok(), "{address}: input at the limit: {res:?}");

                let over_limit = vec![0u8; max_input_size + element_len];
                let res = lagoon_fn(&over_limit, u64::MAX, hint_writer, oracle_reader);
                assert!(
                    matches!(&res, Err(revm::precompile::PrecompileHalt::Other(msg)) if msg.contains("must be at most")),
                    "{address}: input one element over the limit: {res:?}"
                );
            }
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_provider_dispatches_l1_precompile_gas() {
        use revm::precompile::{
            bls12_381_const::{G1_ADD_INPUT_LENGTH, G2_ADD_INPUT_LENGTH},
            bn254::PAIR_ELEMENT_LEN,
        };

        // For each of the four precompiles whose L1 cost diverges from the L2 charge, drive
        // `OpFpvmPrecompiles::run` at JOVIAN and assert the hint payload carries the L1 cost.
        // This locks the production call site (provider-level dispatch), not just the leaf
        // function — a future provider rewrite that swaps in a stale leaf would fail here even
        // if the per-leaf tests still pass. Current L1 values reflect EIP-7904.

        // KZG: 0x0a, valid input → expect 89_363
        run_dispatch_and_assert_oracle_gas(
            super::super::kzg_point_eval::KZG_POINT_EVAL_ADDR,
            kzg_point_eval_input(VALID_KZG_EVALUATION).into(),
            89_363,
        )
        .await;

        // bn254 pair: 0x08, 2 valid pairs of zero bytes → expect 113_206
        let bn128_input: Bytes = vec![0u8; 2 * PAIR_ELEMENT_LEN].into();
        run_dispatch_and_assert_oracle_gas(bn254::pair::ADDRESS, bn128_input, 113_206).await;

        // BLS12-381 G1Add: 0x0b, 256-byte zero input → expect 643
        let g1_input: Bytes = vec![0u8; G1_ADD_INPUT_LENGTH].into();
        run_dispatch_and_assert_oracle_gas(bls12_381_const::G1_ADD_ADDRESS, g1_input, 643).await;

        // BLS12-381 G2Add: 0x0d, 512-byte zero input → expect 765
        let g2_input: Bytes = vec![0u8; G2_ADD_INPUT_LENGTH].into();
        run_dispatch_and_assert_oracle_gas(bls12_381_const::G2_ADD_ADDRESS, g2_input, 765).await;
    }

    /// Helper: drive `OpFpvmPrecompiles::run` at JOVIAN for the given precompile address,
    /// capture the L1Precompile hint, and assert its 8-byte gas word equals `expected_gas`.
    async fn run_dispatch_and_assert_oracle_gas(address: Address, input: Bytes, expected_gas: u64) {
        use crate::fpvm_evm::precompiles::test_utils::test_accelerated_precompile_capture_hint;

        let captured =
            test_accelerated_precompile_capture_hint(move |hint_writer, oracle_reader| {
                let mut ctx = create_test_context();
                let mut precompiles = OpFpvmPrecompiles::new_with_spec(
                    OpSpecId::JOVIAN,
                    hint_writer.clone(),
                    oracle_reader.clone(),
                );
                let call_inputs = create_call_inputs(address, input.clone(), u64::MAX);
                let result = precompiles
                    .run(&mut ctx, &call_inputs)
                    .expect("provider run errored")
                    .expect("provider returned None for accelerated address");
                assert_eq!(
                    result.result,
                    InstructionResult::Return,
                    "expected successful precompile result, got {:?}",
                    result.result
                );
            })
            .await;

        assert_eq!(
            captured.oracle_gas(),
            expected_gas,
            "oracle gas mismatch at address {address:?}",
        );
    }

    #[test]
    fn test_run_with_shared_buffer_empty() {
        let (hint_chan, preimage_chan) = (
            kona_preimage::BidirectionalChannel::new().unwrap(),
            kona_preimage::BidirectionalChannel::new().unwrap(),
        );
        let hint_writer = kona_preimage::HintWriter::new(hint_chan.client);
        let oracle_reader = kona_preimage::OracleReader::new(preimage_chan.client);

        let mut ctx = create_test_context();

        let mut precompiles =
            OpFpvmPrecompiles::new_with_spec(OpSpecId::BEDROCK, hint_writer, oracle_reader);

        // Test SharedBuffer path with empty buffer
        let sha256_addr = revm::precompile::u64_to_address(2);
        let call_inputs = CallInputs {
            input: CallInput::SharedBuffer(0..0),
            return_memory_offset: 0..0,
            gas_limit: u64::MAX,
            reservoir: 0,
            bytecode_address: sha256_addr,
            known_bytecode: (revm::primitives::KECCAK_EMPTY, revm::bytecode::Bytecode::new()),
            target_address: Address::ZERO,
            caller: Address::ZERO,
            value: revm::interpreter::CallValue::Transfer(alloy_primitives::U256::ZERO),
            scheme: revm::interpreter::CallScheme::Call,
            is_static: false,
            charged_new_account_state_gas: false,
        };

        let result = precompiles.run(&mut ctx, &call_inputs).unwrap();
        assert!(result.is_some());
    }
}
