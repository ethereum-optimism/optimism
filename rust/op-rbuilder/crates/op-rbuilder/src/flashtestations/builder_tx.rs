use alloy_eips::Encodable2718;
use alloy_evm::Database;
use alloy_op_evm::OpEvm;
use alloy_primitives::{Address, B256, Bytes, Signature, U256, keccak256};
use alloy_rpc_types_eth::TransactionInput;
use alloy_sol_types::{SolCall, SolEvent, SolValue};
use core::fmt::Debug;
use op_alloy_rpc_types::OpTransactionRequest;
use reth_evm::{ConfigureEvm, Evm, precompiles::PrecompilesMap};
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::StateProvider;
use reth_revm::{State, database::StateProviderDatabase};
use revm::{DatabaseCommit, DatabaseRef, inspector::NoOpInspector};
use std::sync::{Arc, atomic::AtomicBool};
use tracing::{debug, info, warn};

use crate::{
    builders::{
        BuilderTransactionCtx, BuilderTransactionError, BuilderTransactions, OpPayloadBuilderCtx,
        SimulationSuccessResult, get_nonce,
    },
    flashtestations::{
        BlockData,
        IBlockBuilderPolicy::{self, BlockBuilderProofVerified},
        IERC20Permit,
        IFlashtestationRegistry::{self, TEEServiceRegistered},
    },
    primitives::reth::ExecutionInfo,
    tx_signer::Signer,
};

pub struct FlashtestationsBuilderTxArgs {
    pub attestation: Vec<u8>,
    pub extra_registration_data: Bytes,
    pub tee_service_signer: Signer,
    pub registry_address: Address,
    pub builder_policy_address: Address,
    pub builder_proof_version: u8,
    pub enable_block_proofs: bool,
    pub registered: bool,
    pub builder_key: Signer,
}

#[derive(Debug, Clone)]
pub struct FlashtestationsBuilderTx<ExtraCtx = (), Extra = ()>
where
    ExtraCtx: Debug + Default,
    Extra: Debug + Default,
{
    // Attestation for the builder
    attestation: Vec<u8>,
    // Extra registration data for the builder
    extra_registration_data: Bytes,
    // TEE service generated key
    tee_service_signer: Signer,
    // Registry address for the attestation
    registry_address: Address,
    // Builder policy address for the block builder proof
    builder_policy_address: Address,
    // Builder proof version
    builder_proof_version: u8,
    // Whether the workload and address has been registered
    registered: Arc<AtomicBool>,
    // Whether block proofs are enabled
    enable_block_proofs: bool,
    // Builder key for the flashtestation permit tx
    builder_signer: Signer,
    // Extra context and data
    _marker: std::marker::PhantomData<(ExtraCtx, Extra)>,
}

impl<ExtraCtx, Extra> FlashtestationsBuilderTx<ExtraCtx, Extra>
where
    ExtraCtx: Debug + Default,
    Extra: Debug + Default,
{
    pub fn new(args: FlashtestationsBuilderTxArgs) -> Self {
        Self {
            attestation: args.attestation,
            extra_registration_data: args.extra_registration_data,
            tee_service_signer: args.tee_service_signer,
            registry_address: args.registry_address,
            builder_policy_address: args.builder_policy_address,
            builder_proof_version: args.builder_proof_version,
            registered: Arc::new(AtomicBool::new(args.registered)),
            enable_block_proofs: args.enable_block_proofs,
            builder_signer: args.builder_key,
            _marker: std::marker::PhantomData,
        }
    }

    pub fn tee_signer(&self) -> &Signer {
        &self.tee_service_signer
    }

    /// Computes the block content hash according to the formula:
    /// keccak256(abi.encode(parentHash, blockNumber, timestamp, transactionHashes))
    /// https://github.com/flashbots/rollup-boost/blob/main/specs/flashtestations.md#block-building-process
    fn compute_block_content_hash(
        transactions: &[OpTransactionSigned],
        parent_hash: B256,
        block_number: u64,
        timestamp: u64,
    ) -> B256 {
        // Create ordered list of transaction hashes
        let transaction_hashes: Vec<B256> = transactions
            .iter()
            .map(|tx| {
                // RLP encode the transaction and hash it
                let mut encoded = Vec::new();
                tx.encode_2718(&mut encoded);
                keccak256(&encoded)
            })
            .collect();

        // Create struct and ABI encode
        let block_data = BlockData {
            parentHash: parent_hash,
            blockNumber: U256::from(block_number),
            timestamp: U256::from(timestamp),
            transactionHashes: transaction_hashes,
        };

        let encoded = block_data.abi_encode();
        keccak256(&encoded)
    }

    fn set_registered(
        &self,
        state_provider: impl StateProvider + Clone,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
    ) -> Result<(), BuilderTransactionError> {
        let state = StateProviderDatabase::new(state_provider.clone());
        let mut simulation_state = State::builder()
            .with_database(state)
            .with_bundle_update()
            .build();
        let mut evm = ctx
            .evm_config
            .evm_with_env(&mut simulation_state, ctx.evm_env.clone());
        evm.modify_cfg(|cfg| {
            cfg.disable_balance_check = true;
            cfg.disable_nonce_check = true;
        });
        let calldata = IFlashtestationRegistry::getRegistrationStatusCall {
            teeAddress: self.tee_service_signer.address,
        };
        let SimulationSuccessResult { output, .. } =
            self.flashtestations_contract_read(self.registry_address, calldata, ctx, &mut evm)?;
        if output.isValid {
            self.registered
                .store(true, std::sync::atomic::Ordering::SeqCst);
        }
        Ok(())
    }

    fn get_permit_nonce(
        &self,
        contract_address: Address,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<U256, BuilderTransactionError> {
        let calldata = IERC20Permit::noncesCall {
            owner: self.tee_service_signer.address,
        };
        let SimulationSuccessResult { output, .. } =
            self.flashtestations_contract_read(contract_address, calldata, ctx, evm)?;
        Ok(output)
    }

    fn registration_permit_signature(
        &self,
        permit_nonce: U256,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<Signature, BuilderTransactionError> {
        let struct_hash_calldata = IFlashtestationRegistry::computeStructHashCall {
            rawQuote: self.attestation.clone().into(),
            extendedRegistrationData: self.extra_registration_data.clone(),
            nonce: permit_nonce,
            deadline: U256::from(ctx.timestamp()),
        };
        let SimulationSuccessResult { output, .. } = self.flashtestations_contract_read(
            self.registry_address,
            struct_hash_calldata,
            ctx,
            evm,
        )?;
        let typed_data_hash_calldata =
            IFlashtestationRegistry::hashTypedDataV4Call { structHash: output };
        let SimulationSuccessResult { output, .. } = self.flashtestations_contract_read(
            self.registry_address,
            typed_data_hash_calldata,
            ctx,
            evm,
        )?;
        let signature = self.tee_service_signer.sign_message(output)?;
        Ok(signature)
    }

    fn signed_registration_permit_tx(
        &self,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<&mut State<impl Database + DatabaseRef>, NoOpInspector, PrecompilesMap>,
    ) -> Result<BuilderTransactionCtx, BuilderTransactionError> {
        let permit_nonce = self.get_permit_nonce(self.registry_address, ctx, evm)?;
        let signature = self.registration_permit_signature(permit_nonce, ctx, evm)?;
        let calldata = IFlashtestationRegistry::permitRegisterTEEServiceCall {
            rawQuote: self.attestation.clone().into(),
            extendedRegistrationData: self.extra_registration_data.clone(),
            nonce: permit_nonce,
            deadline: U256::from(ctx.timestamp()),
            signature: signature.as_bytes().into(),
        };
        let SimulationSuccessResult {
            gas_used,
            state_changes,
            ..
        } = self.flashtestations_call(
            self.registry_address,
            calldata.clone(),
            vec![TEEServiceRegistered::SIGNATURE_HASH],
            ctx,
            evm,
        )?;
        let signed_tx = self.sign_tx(
            self.registry_address,
            self.builder_signer,
            gas_used,
            calldata.abi_encode().into(),
            ctx,
            evm.db(),
        )?;
        let da_size =
            op_alloy_flz::tx_estimated_size_fjord_bytes(signed_tx.encoded_2718().as_slice());
        // commit the register transaction state so the block proof transaction can succeed
        evm.db_mut().commit(state_changes);
        Ok(BuilderTransactionCtx {
            gas_used,
            da_size,
            signed_tx,
            is_top_of_block: false,
        })
    }

    fn block_proof_permit_signature(
        &self,
        permit_nonce: U256,
        block_content_hash: B256,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<Signature, BuilderTransactionError> {
        let struct_hash_calldata = IBlockBuilderPolicy::computeStructHashCall {
            version: self.builder_proof_version,
            blockContentHash: block_content_hash,
            nonce: permit_nonce,
        };
        let SimulationSuccessResult { output, .. } = self.flashtestations_contract_read(
            self.builder_policy_address,
            struct_hash_calldata,
            ctx,
            evm,
        )?;
        let typed_data_hash_calldata =
            IBlockBuilderPolicy::getHashedTypeDataV4Call { structHash: output };
        let SimulationSuccessResult { output, .. } = self.flashtestations_contract_read(
            self.builder_policy_address,
            typed_data_hash_calldata,
            ctx,
            evm,
        )?;
        let signature = self.tee_service_signer.sign_message(output)?;
        Ok(signature)
    }

    fn signed_block_proof_permit_tx(
        &self,
        transactions: &[OpTransactionSigned],
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<BuilderTransactionCtx, BuilderTransactionError> {
        let permit_nonce = self.get_permit_nonce(self.builder_policy_address, ctx, evm)?;
        let block_content_hash = Self::compute_block_content_hash(
            transactions,
            ctx.parent_hash(),
            ctx.block_number(),
            ctx.timestamp(),
        );
        let signature =
            self.block_proof_permit_signature(permit_nonce, block_content_hash, ctx, evm)?;
        let calldata = IBlockBuilderPolicy::permitVerifyBlockBuilderProofCall {
            blockContentHash: block_content_hash,
            nonce: permit_nonce,
            version: self.builder_proof_version,
            eip712Sig: signature.as_bytes().into(),
        };
        let SimulationSuccessResult { gas_used, .. } = self.flashtestations_call(
            self.builder_policy_address,
            calldata.clone(),
            vec![BlockBuilderProofVerified::SIGNATURE_HASH],
            ctx,
            evm,
        )?;
        let signed_tx = self.sign_tx(
            self.builder_policy_address,
            self.builder_signer,
            gas_used,
            calldata.abi_encode().into(),
            ctx,
            evm.db(),
        )?;
        let da_size =
            op_alloy_flz::tx_estimated_size_fjord_bytes(signed_tx.encoded_2718().as_slice());
        Ok(BuilderTransactionCtx {
            gas_used,
            da_size,
            signed_tx,
            is_top_of_block: false,
        })
    }

    fn flashtestations_contract_read<T: SolCall>(
        &self,
        contract_address: Address,
        calldata: T,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<SimulationSuccessResult<T>, BuilderTransactionError> {
        self.flashtestations_call(contract_address, calldata, vec![], ctx, evm)
    }

    fn flashtestations_call<T: SolCall>(
        &self,
        contract_address: Address,
        calldata: T,
        expected_topics: Vec<B256>,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        evm: &mut OpEvm<impl Database + DatabaseRef, NoOpInspector, PrecompilesMap>,
    ) -> Result<SimulationSuccessResult<T>, BuilderTransactionError> {
        let tx_req = OpTransactionRequest::default()
            .gas_limit(ctx.block_gas_limit())
            .max_fee_per_gas(ctx.base_fee().into())
            .to(contract_address)
            .from(self.builder_signer.address)
            .nonce(get_nonce(evm.db(), self.builder_signer.address)?)
            .input(TransactionInput::new(calldata.abi_encode().into()));
        if contract_address == self.registry_address {
            self.simulate_call::<T, IFlashtestationRegistry::IFlashtestationRegistryErrors>(
                tx_req,
                expected_topics,
                evm,
            )
        } else if contract_address == self.builder_policy_address {
            self.simulate_call::<T, IBlockBuilderPolicy::IBlockBuilderPolicyErrors>(
                tx_req,
                expected_topics,
                evm,
            )
        } else {
            Err(BuilderTransactionError::msg(
                "invalid contract address for flashtestations",
            ))
        }
    }
}

impl<ExtraCtx, Extra> BuilderTransactions<ExtraCtx, Extra>
    for FlashtestationsBuilderTx<ExtraCtx, Extra>
where
    ExtraCtx: Debug + Default,
    Extra: Debug + Default,
{
    fn simulate_builder_txs(
        &self,
        state_provider: impl StateProvider + Clone,
        info: &mut ExecutionInfo<Extra>,
        ctx: &OpPayloadBuilderCtx<ExtraCtx>,
        db: &mut State<impl Database + DatabaseRef>,
        _top_of_block: bool,
    ) -> Result<Vec<BuilderTransactionCtx>, BuilderTransactionError> {
        // set registered by simulating against the committed state
        if !self.registered.load(std::sync::atomic::Ordering::SeqCst) {
            self.set_registered(state_provider, ctx)?;
        }

        let mut evm = ctx.evm_config.evm_with_env(&mut *db, ctx.evm_env.clone());
        evm.modify_cfg(|cfg| {
            cfg.disable_balance_check = true;
            cfg.disable_block_gas_limit = true;
        });

        let mut builder_txs = Vec::<BuilderTransactionCtx>::new();

        if !self.registered.load(std::sync::atomic::Ordering::SeqCst) {
            info!(target: "flashtestations", "tee service not registered yet, attempting to register");
            let register_tx = self.signed_registration_permit_tx(ctx, &mut evm)?;
            builder_txs.push(register_tx);
        }

        // don't return on error for block proof as previous txs in builder_txs will not be returned
        if self.enable_block_proofs {
            debug!(target: "flashtestations", "adding permit verify block proof tx");
            match self.signed_block_proof_permit_tx(&info.executed_transactions, ctx, &mut evm) {
                Ok(block_proof_tx) => builder_txs.push(block_proof_tx),
                Err(e) => {
                    warn!(target: "flashtestations", error = ?e, "failed to add permit block proof transaction")
                }
            }
        }
        Ok(builder_txs)
    }
}
