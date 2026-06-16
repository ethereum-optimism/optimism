//! [`PrecompileProvider`] for FPVM-accelerated OP Stack precompiles.

use crate::fpvm_evm::precompiles::{
    ecrecover::ECRECOVER_ADDR, kzg_point_eval::KZG_POINT_EVAL_ADDR,
};
use alloc::{string::String, vec, vec::Vec};
use alloy_primitives::{Address, Bytes};
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use op_revm::{
    OpSpecId,
    precompiles::{fjord, granite, isthmus, jovian, karst},
};
use revm::{
    context::{Cfg, ContextTr},
    handler::{EthPrecompiles, PrecompileProvider},
    interpreter::{CallInputs, Gas, InstructionResult, InterpreterResult},
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
            OpSpecId::KARST | OpSpecId::INTEROP => karst(),
        };

        let accelerated_precompiles = match spec {
            OpSpecId::BEDROCK | OpSpecId::REGOLITH | OpSpecId::CANYON => {
                accelerated_bedrock::<H, O>()
            }
            OpSpecId::ECOTONE | OpSpecId::FJORD => accelerated_ecotone::<H, O>(),
            OpSpecId::GRANITE | OpSpecId::HOLOCENE => accelerated_granite::<H, O>(),
            OpSpecId::ISTHMUS => accelerated_isthmus::<H, O>(),
            OpSpecId::JOVIAN => accelerated_jovian::<H, O>(),
            OpSpecId::KARST | OpSpecId::INTEROP => accelerated_karst::<H, O>(),
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

    #[inline]
    fn run(
        &mut self,
        context: &mut CTX,
        inputs: &CallInputs,
    ) -> Result<Option<Self::Output>, String> {
        let mut result = InterpreterResult {
            result: InstructionResult::Return,
            gas: Gas::new(inputs.gas_limit),
            output: Bytes::new(),
        };

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

        if output.is_halt() {
            result.result = if output.halt_reason().is_some_and(|r| r.is_oog()) {
                InstructionResult::PrecompileOOG
            } else {
                InstructionResult::PrecompileError
            };
        } else {
            let underflow = result.gas.record_regular_cost(output.gas_used);
            assert!(underflow, "Gas underflow is not possible");
            result.result = InstructionResult::Return;
            result.output = output.bytes;
        }

        Ok(Some(result))
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

#[cfg(test)]
mod test {
    use super::*;
    use alloy_op_evm::{OpEvmContext, OpTx};
    use kona_preimage::{HintWriterClient, PreimageOracleClient};
    use op_revm::{L1BlockInfo, OpSpecId, OpTransaction};
    use revm::{
        Context, MainContext, database::EmptyDB, handler::PrecompileProvider,
        interpreter::CallInput,
    };

    type TestContext = OpEvmContext<EmptyDB>;

    fn create_call_inputs(address: Address, input: Bytes, gas_limit: u64) -> CallInputs {
        CallInputs {
            input: CallInput::Bytes(input),
            return_memory_offset: 0..0,
            gas_limit,
            reservoir: 0,
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
        let interop_provider = OpFpvmPrecompiles::new_with_spec(
            OpSpecId::INTEROP,
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
        let interop_addrs: Vec<_> = {
            let mut addrs: Vec<_> =
                interop_provider.accelerated_precompiles.keys().copied().collect();
            addrs.sort();
            addrs
        };
        assert_eq!(
            jovian_addrs, karst_addrs,
            "KARST should accelerate the same addresses as JOVIAN (functions may differ)"
        );
        assert_eq!(
            karst_addrs, interop_addrs,
            "INTEROP should accelerate the same addresses as KARST (functions may differ)"
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
            core::ptr::eq(interop_provider.inner.precompiles, karst()),
            "INTEROP should use karst() precompiles"
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

    #[tokio::test(flavor = "multi_thread")]
    async fn test_provider_dispatches_l1_precompile_gas() {
        use alloy_eips::eip4844::VERSIONED_HASH_VERSION_KZG;
        use alloy_primitives::hex;
        use revm::precompile::{
            bls12_381_const::{G1_ADD_INPUT_LENGTH, G2_ADD_INPUT_LENGTH},
            bn254::PAIR_ELEMENT_LEN,
        };
        use sha2::{Digest, Sha256};

        // For each of the four precompiles whose L1 cost diverges from the L2 charge, drive
        // `OpFpvmPrecompiles::run` at JOVIAN and assert the hint payload carries the L1 cost.
        // This locks the production call site (provider-level dispatch), not just the leaf
        // function — a future provider rewrite that swaps in a stale leaf would fail here even
        // if the per-leaf tests still pass. Current L1 values reflect EIP-7904.

        // KZG: 0x0a, valid input → expect 89_363
        let kzg_input: Bytes = {
            let commitment = hex!("8f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7").to_vec();
            let mut versioned_hash = Sha256::digest(&commitment).to_vec();
            versioned_hash[0] = VERSIONED_HASH_VERSION_KZG;
            let z =
                hex!("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000").to_vec();
            let y =
                hex!("1522a4a7f34e1ea350ae07c29c96c7e79655aa926122e95fe69fcbd932ca49e9").to_vec();
            let proof = hex!("a62ad71d14c5719385c0686f1871430475bf3a00f0aa3f7b8dd99a9abc2160744faf0070725e00b60ad9a026a15b1a8c").to_vec();
            [versioned_hash, z, y, commitment, proof].concat().into()
        };
        run_dispatch_and_assert_oracle_gas(
            super::super::kzg_point_eval::KZG_POINT_EVAL_ADDR,
            kzg_input,
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
