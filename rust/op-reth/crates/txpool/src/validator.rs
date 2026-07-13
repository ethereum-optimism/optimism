use crate::{InvalidCrossTx, OpPooledTx, interop_filter::InteropFilterClient};
use alloy_consensus::{BlockHeader, Transaction};
use alloy_primitives::U256;
use op_revm::L1BlockInfo;
use parking_lot::RwLock;
use reth_chainspec::ChainSpecProvider;
use reth_evm::ConfigureEvm;
use reth_optimism_evm::{RethL1BlockInfo, revm_spec_by_timestamp_after_bedrock};
use reth_optimism_forks::OpHardforks;
use reth_primitives_traits::{
    Block, BlockBody, BlockTy, GotExpected, SealedBlock,
    transaction::error::InvalidTransactionError,
};
use reth_storage_api::{AccountInfoReader, BlockReaderIdExt, StateProviderFactory};
use reth_transaction_pool::{
    EthPoolTransaction, EthTransactionValidator, TransactionOrigin, TransactionValidationOutcome,
    TransactionValidator, error::InvalidPoolTransactionError,
};
use std::sync::{
    Arc,
    atomic::{AtomicBool, AtomicU64, Ordering},
};

/// The timeout for cross-chain transaction validation against the interop filter.
pub(crate) const CHECK_ACCESS_LIST_TIMEOUT_SECS: u64 = 7200;

/// Tracks additional infos for the current block.
#[derive(Debug, Default)]
pub struct OpL1BlockInfo {
    /// The current L1 block info.
    l1_block_info: RwLock<L1BlockInfo>,
    /// Current block timestamp.
    timestamp: AtomicU64,
}

impl OpL1BlockInfo {
    /// Returns the most recent timestamp
    pub fn timestamp(&self) -> u64 {
        self.timestamp.load(Ordering::Relaxed)
    }
}

/// Validator for Optimism transactions.
#[derive(Debug, Clone)]
pub struct OpTransactionValidator<Client, Tx, Evm> {
    /// The type that performs the actual validation.
    inner: Arc<EthTransactionValidator<Client, Tx, Evm>>,
    /// Additional block info required for validation.
    block_info: Arc<OpL1BlockInfo>,
    /// If true, ensure that the transaction's sender has enough balance to cover the L1 gas fee
    /// derived from the tracked L1 block info that is extracted from the first transaction in the
    /// L2 block.
    require_l1_data_gas_fee: bool,
    /// Client used to check transaction validity with the interop filter.
    interop_client: Option<InteropFilterClient>,
    /// tracks activated forks relevant for transaction validation
    fork_tracker: Arc<OpForkTracker>,
}

impl<Client, Tx, Evm> OpTransactionValidator<Client, Tx, Evm> {
    /// Returns the configured chain spec
    pub fn chain_spec(&self) -> Arc<Client::ChainSpec>
    where
        Client: ChainSpecProvider,
    {
        self.inner.chain_spec()
    }

    /// Returns the configured client
    pub fn client(&self) -> &Client {
        self.inner.client()
    }

    /// Returns the current block timestamp.
    fn block_timestamp(&self) -> u64 {
        self.block_info.timestamp.load(Ordering::Relaxed)
    }

    /// Whether to ensure that the transaction's sender has enough balance to also cover the L1 gas
    /// fee.
    pub fn require_l1_data_gas_fee(self, require_l1_data_gas_fee: bool) -> Self {
        Self { require_l1_data_gas_fee, ..self }
    }

    /// Returns whether this validator also requires the transaction's sender to have enough balance
    /// to cover the L1 gas fee.
    pub const fn requires_l1_data_gas_fee(&self) -> bool {
        self.require_l1_data_gas_fee
    }
}

impl<Client, Tx, Evm> OpTransactionValidator<Client, Tx, Evm>
where
    Client:
        ChainSpecProvider<ChainSpec: OpHardforks> + StateProviderFactory + BlockReaderIdExt + Sync,
    Tx: EthPoolTransaction + OpPooledTx,
    Evm: ConfigureEvm,
{
    /// Create a new [`OpTransactionValidator`].
    pub fn new(inner: EthTransactionValidator<Client, Tx, Evm>) -> Self {
        let this = Self::with_block_info(inner, OpL1BlockInfo::default());
        if let Ok(Some(block)) =
            this.inner.client().block_by_number_or_tag(alloy_eips::BlockNumberOrTag::Latest)
        {
            // genesis block has no txs, so we can't extract L1 info, we set the block info to empty
            // so that we will accept txs into the pool before the first block
            if block.header().number() == 0 {
                this.block_info.timestamp.store(block.header().timestamp(), Ordering::Relaxed);
            } else {
                this.update_l1_block_info(block.header(), block.body().transactions().first());
            }
        }

        this
    }

    /// Create a new [`OpTransactionValidator`] with the given [`OpL1BlockInfo`].
    pub fn with_block_info(
        inner: EthTransactionValidator<Client, Tx, Evm>,
        block_info: OpL1BlockInfo,
    ) -> Self {
        Self {
            inner: Arc::new(inner),
            block_info: Arc::new(block_info),
            require_l1_data_gas_fee: true,
            interop_client: None,
            fork_tracker: Arc::new(OpForkTracker { interop: AtomicBool::from(false) }),
        }
    }

    /// Sets the interop filter client and safety level.
    pub fn with_interop(mut self, interop_client: InteropFilterClient) -> Self {
        self.interop_client = Some(interop_client);
        self
    }

    /// Update the L1 block info for the given header and system transaction, if any.
    ///
    /// Note: this supports optional system transaction, in case this is used in a dev setup
    pub fn update_l1_block_info<H, T>(&self, header: &H, tx: Option<&T>)
    where
        H: BlockHeader,
        T: Transaction,
    {
        self.block_info.timestamp.store(header.timestamp(), Ordering::Relaxed);

        if let Some(Ok(l1_block_info)) = tx.map(reth_optimism_evm::extract_l1_info_from_tx) {
            *self.block_info.l1_block_info.write() = l1_block_info;
        }

        if self.chain_spec().is_interop_active_at_timestamp(header.timestamp()) {
            self.fork_tracker.interop.store(true, Ordering::Relaxed);
        }
    }

    /// Validates a single transaction.
    ///
    /// See also [`TransactionValidator::validate_transaction`]
    ///
    /// This behaves the same as [`OpTransactionValidator::validate_one_with_state`], but creates
    /// a new state provider internally.
    pub async fn validate_one(
        &self,
        origin: TransactionOrigin,
        transaction: Tx,
    ) -> TransactionValidationOutcome<Tx> {
        self.validate_one_with_state(origin, transaction, &mut None).await
    }

    /// Validates a single transaction with a provided state provider.
    ///
    /// This allows reusing the same state provider across multiple transaction validations.
    ///
    /// See also [`TransactionValidator::validate_transaction`]
    ///
    /// This behaves the same as [`EthTransactionValidator::validate_one_with_state`], but in
    /// addition applies OP validity checks:
    /// - ensures tx is not eip4844
    /// - ensures cross chain transactions are valid wrt locally configured safety level
    /// - ensures that the account has enough balance to cover the L1 gas cost
    pub async fn validate_one_with_state(
        &self,
        origin: TransactionOrigin,
        transaction: Tx,
        state: &mut Option<Box<dyn AccountInfoReader + Send>>,
    ) -> TransactionValidationOutcome<Tx> {
        if transaction.is_eip4844() {
            return TransactionValidationOutcome::Invalid(
                transaction,
                InvalidTransactionError::TxTypeNotSupported.into(),
            );
        }

        // Interop cross tx validation
        match self.is_valid_cross_tx(&transaction).await {
            Some(Err(err)) => {
                let err = match err {
                    InvalidCrossTx::CrossChainTxPreInterop => {
                        InvalidTransactionError::TxTypeNotSupported.into()
                    }
                    err => InvalidPoolTransactionError::Other(Box::new(err)),
                };
                return TransactionValidationOutcome::Invalid(transaction, err);
            }
            Some(Ok(_)) => {
                // valid interop tx
                transaction
                    .set_interop_deadline(self.block_timestamp() + CHECK_ACCESS_LIST_TIMEOUT_SECS);
            }
            _ => {}
        }

        let outcome = self.inner.validate_one_with_state(origin, transaction, state);

        self.apply_op_checks(outcome)
    }

    /// Performs the necessary opstack specific checks based on top of the regular eth outcome.
    fn apply_op_checks(
        &self,
        outcome: TransactionValidationOutcome<Tx>,
    ) -> TransactionValidationOutcome<Tx> {
        if !self.requires_l1_data_gas_fee() {
            // no need to check L1 gas fee
            return outcome;
        }
        // ensure that the account has enough balance to cover the L1 gas cost
        if let TransactionValidationOutcome::Valid {
            balance,
            state_nonce,
            transaction: valid_tx,
            propagate,
            bytecode_hash,
            authorities,
        } = outcome
        {
            let mut l1_block_info = self.block_info.l1_block_info.read().clone();

            let encoded = valid_tx.transaction().encoded_2718();

            let mut cost_addition = match l1_block_info.l1_tx_data_fee(
                self.chain_spec(),
                self.block_timestamp(),
                &encoded,
                false,
            ) {
                Ok(cost) => cost,
                Err(err) => {
                    return TransactionValidationOutcome::Error(*valid_tx.hash(), Box::new(err));
                }
            };

            // Post-Isthmus, execution charges the operator fee up front based on the tx gas
            // limit (see `L1BlockInfo::tx_cost`), so pool admission must reserve it as well or
            // under-funded transactions are admitted and later fail in the block. The operator
            // fee params are only populated once a post-Isthmus L1-info block has been tracked;
            // until then (`operator_fee_charge` panics on unset params) no fee is reserved.
            if self.chain_spec().is_isthmus_active_at_timestamp(self.block_timestamp()) &&
                l1_block_info.operator_fee_scalar.is_some() &&
                l1_block_info.operator_fee_constant.is_some()
            {
                let spec =
                    revm_spec_by_timestamp_after_bedrock(self.chain_spec(), self.block_timestamp());
                let operator_fee = l1_block_info.operator_fee_charge(
                    &encoded,
                    U256::from(valid_tx.transaction().gas_limit()),
                    spec,
                );
                cost_addition = cost_addition.saturating_add(operator_fee);
            }

            let cost = valid_tx.transaction().cost().saturating_add(cost_addition);

            // Checks for max cost
            if cost > balance {
                return TransactionValidationOutcome::Invalid(
                    valid_tx.into_transaction(),
                    InvalidTransactionError::InsufficientFunds(
                        GotExpected { got: balance, expected: cost }.into(),
                    )
                    .into(),
                );
            }

            return TransactionValidationOutcome::Valid {
                balance,
                state_nonce,
                transaction: valid_tx,
                propagate,
                bytecode_hash,
                authorities,
            };
        }
        outcome
    }

    /// Wrapper for is valid cross tx
    pub async fn is_valid_cross_tx(&self, tx: &Tx) -> Option<Result<(), InvalidCrossTx>> {
        // We don't need to check for deposit transaction in here, because they won't come from
        // txpool
        self.interop_client
            .as_ref()?
            .is_valid_cross_tx(
                tx.access_list(),
                tx.hash(),
                self.block_info.timestamp.load(Ordering::Relaxed),
                Some(CHECK_ACCESS_LIST_TIMEOUT_SECS),
                self.fork_tracker.is_interop_activated(),
            )
            .await
    }
}

impl<Client, Tx, Evm> TransactionValidator for OpTransactionValidator<Client, Tx, Evm>
where
    Client:
        ChainSpecProvider<ChainSpec: OpHardforks> + StateProviderFactory + BlockReaderIdExt + Sync,
    Tx: EthPoolTransaction + OpPooledTx,
    Evm: ConfigureEvm,
{
    type Transaction = Tx;
    type Block = BlockTy<Evm::Primitives>;

    async fn validate_transaction(
        &self,
        origin: TransactionOrigin,
        transaction: Self::Transaction,
    ) -> TransactionValidationOutcome<Self::Transaction> {
        self.validate_one(origin, transaction).await
    }

    fn on_new_head_block(&self, new_tip_block: &SealedBlock<Self::Block>) {
        self.inner.on_new_head_block(new_tip_block);
        self.update_l1_block_info(
            new_tip_block.header(),
            new_tip_block.body().transactions().first(),
        );
    }
}

/// Keeps track of whether certain forks are activated
#[derive(Debug)]
pub(crate) struct OpForkTracker {
    /// Tracks if interop is activated at the block's timestamp.
    interop: AtomicBool,
}

impl OpForkTracker {
    /// Returns `true` if Lagoon fork is activated.
    pub(crate) fn is_interop_activated(&self) -> bool {
        self.interop.load(Ordering::Relaxed)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpPooledTransaction;
    use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, Signature, TxKind, U256};
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_evm::OpEvmConfig;
    use reth_optimism_forks::OpHardfork;
    use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
    use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
    use reth_transaction_pool::{
        TransactionOrigin, TransactionValidationOutcome, blobstore::InMemoryBlobStore,
        validate::EthTransactionValidatorBuilder,
    };

    const GAS_LIMIT: u64 = 100_000;
    const MAX_FEE_PER_GAS: u128 = 1_000_000_000;
    const OPERATOR_FEE_SCALAR: u64 = 1_000_000;
    const OPERATOR_FEE_CONSTANT: u64 = 500;

    /// Builds a validator tracking a post-Isthmus timestamp, an account for `signer` with
    /// `balance`, and (optionally) operator fee params, mirroring the state after a
    /// post-Isthmus L1-info block has been processed.
    fn isthmus_validator(
        signer: Address,
        balance: U256,
        operator_fee_params_set: bool,
    ) -> OpTransactionValidator<
        MockEthProvider<OpPrimitives, Arc<reth_optimism_chainspec::OpChainSpec>>,
        OpPooledTransaction,
        OpEvmConfig,
    > {
        let client = MockEthProvider::<OpPrimitives>::new()
            .with_chain_spec(OP_MAINNET.clone())
            .with_genesis_block();
        client.add_account(signer, ExtendedAccount::new(0, balance));
        let evm_config = OpEvmConfig::optimism(OP_MAINNET.clone());
        let validator = EthTransactionValidatorBuilder::new(client, evm_config)
            .no_shanghai()
            .no_cancun()
            .build(InMemoryBlobStore::default());
        let validator = OpTransactionValidator::new(validator);

        let isthmus_ts = OP_MAINNET
            .op_fork_activation(OpHardfork::Isthmus)
            .as_timestamp()
            .expect("OP mainnet activates Isthmus by timestamp");
        validator.block_info.timestamp.store(isthmus_ts, Ordering::Relaxed);
        if operator_fee_params_set {
            let mut info = validator.block_info.l1_block_info.write();
            info.operator_fee_scalar = Some(U256::from(OPERATOR_FEE_SCALAR));
            info.operator_fee_constant = Some(U256::from(OPERATOR_FEE_CONSTANT));
        }
        validator
    }

    /// A signed EIP-1559 transaction recovered to `signer`, plus the operator fee execution
    /// will charge for it at `timestamp` (gas-limit based, like `L1BlockInfo::tx_cost`).
    fn pooled_tx_and_operator_fee(
        signer: Address,
        timestamp: u64,
        operator_fee_params_set: bool,
    ) -> (OpPooledTransaction, U256) {
        let tx = TxEip1559 {
            chain_id: OP_MAINNET.chain().id(),
            nonce: 0,
            gas_limit: GAS_LIMIT,
            max_fee_per_gas: MAX_FEE_PER_GAS,
            max_priority_fee_per_gas: 0,
            to: TxKind::Call(Address::ZERO),
            ..Default::default()
        };
        let signed: OpTransactionSigned = tx.into_signed(Signature::test_signature()).into();
        let recovered = Recovered::new_unchecked(signed, signer);
        let encoded = recovered.encoded_2718();
        let len = encoded.len();

        let operator_fee = if operator_fee_params_set {
            let info = L1BlockInfo {
                operator_fee_scalar: Some(U256::from(OPERATOR_FEE_SCALAR)),
                operator_fee_constant: Some(U256::from(OPERATOR_FEE_CONSTANT)),
                ..Default::default()
            };
            let spec = revm_spec_by_timestamp_after_bedrock(OP_MAINNET.clone(), timestamp);
            info.operator_fee_charge(&encoded, U256::from(GAS_LIMIT), spec)
        } else {
            U256::ZERO
        };

        (OpPooledTransaction::new(recovered, len), operator_fee)
    }

    fn isthmus_timestamp() -> u64 {
        OP_MAINNET
            .op_fork_activation(OpHardfork::Isthmus)
            .as_timestamp()
            .expect("OP mainnet activates Isthmus by timestamp")
    }

    /// The intrinsic pool cost of the test transaction (gas * max fee + value); the L1 data
    /// fee is zero because the tracked `L1BlockInfo` has zeroed L1 fee params.
    fn intrinsic_cost() -> U256 {
        U256::from(GAS_LIMIT as u128 * MAX_FEE_PER_GAS)
    }

    #[tokio::test]
    async fn operator_fee_short_balance_is_rejected() {
        let signer = Address::random();
        let (_, operator_fee) = pooled_tx_and_operator_fee(signer, isthmus_timestamp(), true);
        assert!(operator_fee > U256::ZERO, "test requires a non-zero operator fee");

        // Covers gas * max fee + value + L1 data fee, but is one wei short of the operator fee.
        let balance = intrinsic_cost() + operator_fee - U256::from(1);
        let validator = isthmus_validator(signer, balance, true);
        let (tx, _) = pooled_tx_and_operator_fee(signer, isthmus_timestamp(), true);

        let outcome = validator.validate_one(TransactionOrigin::External, tx).await;
        let TransactionValidationOutcome::Invalid(_, err) = outcome else {
            panic!(
                "expected invalid outcome for balance short of the operator fee, got {outcome:?}"
            );
        };
        assert!(err.to_string().to_lowercase().contains("enough funds"), "unexpected error: {err}");
    }

    #[tokio::test]
    async fn operator_fee_covered_balance_is_accepted() {
        let signer = Address::random();
        let (_, operator_fee) = pooled_tx_and_operator_fee(signer, isthmus_timestamp(), true);

        let balance = intrinsic_cost() + operator_fee;
        let validator = isthmus_validator(signer, balance, true);
        let (tx, _) = pooled_tx_and_operator_fee(signer, isthmus_timestamp(), true);

        let outcome = validator.validate_one(TransactionOrigin::External, tx).await;
        assert!(
            matches!(outcome, TransactionValidationOutcome::Valid { .. }),
            "expected valid outcome, got {outcome:?}"
        );
    }

    #[tokio::test]
    async fn missing_operator_fee_params_do_not_panic_or_reserve() {
        let signer = Address::random();

        // Post-Isthmus timestamp but operator fee params never tracked (e.g. validator booted
        // before the first block): no reservation is possible, but validation must not panic.
        let balance = intrinsic_cost();
        let validator = isthmus_validator(signer, balance, false);
        let (tx, _) = pooled_tx_and_operator_fee(signer, isthmus_timestamp(), false);

        let outcome = validator.validate_one(TransactionOrigin::External, tx).await;
        assert!(
            matches!(outcome, TransactionValidationOutcome::Valid { .. }),
            "expected valid outcome, got {outcome:?}"
        );
    }
}
