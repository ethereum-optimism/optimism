use crate::{InvalidCrossTx, OpPooledTx, interop_filter::InteropFilterClient};
use alloy_consensus::{BlockHeader, Transaction};
use alloy_primitives::U256;
use op_revm::{L1BlockInfo, OpSpecId};
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
    TransactionValidator, error::InvalidPoolTransactionError, validate::ValidTransaction,
};
use std::sync::{
    Arc,
    atomic::{AtomicBool, AtomicU64, Ordering},
};

/// The timeout for cross-chain transaction validation against the interop filter.
pub(crate) const CHECK_ACCESS_LIST_TIMEOUT_SECS: u64 = 7200;

/// Gas reserved on every L2 block for the L1-info deposit transaction.
///
/// The L1-info deposit is injected as the first transaction of every OP block and always
/// consumes some gas, so a non-deposit transaction can never use the full block gas limit.
/// Mirrors op-geth's `l1InfoGasOverhead` (`core/txpool/validation.go`) so the tx-pool does not
/// admit transactions that could never be included.
pub(crate) const L1_INFO_GAS_OVERHEAD: u64 = 70_000;

/// Contains the Gas Limit configuration for the OP builder (`miner_setGasLimit`).
///
/// Cloning shares the value: the miner API writes it, and both the payload builder and the pool's
/// validator read it.
#[derive(Debug, Clone, Default)]
pub struct OpGasLimitConfig {
    /// The gas budget of the blocks this node builds. 0 means unset.
    gas_limit: Arc<AtomicU64>,
}

impl OpGasLimitConfig {
    /// Creates a new Gas Limit configuration with the given gas budget.
    pub fn new(max_gas_limit: u64) -> Self {
        let this = Self::default();
        this.set_gas_limit(max_gas_limit);
        this
    }
    /// Returns the gas budget, if any.
    pub fn gas_limit(&self) -> Option<u64> {
        let val = self.gas_limit.load(Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }
    /// Sets the gas budget. 0 means unset.
    pub fn set_gas_limit(&self, gas_limit: u64) {
        self.gas_limit.store(gas_limit, Ordering::Relaxed);
    }
    /// Caps `block_gas_limit` at the gas budget, if one is set.
    pub fn cap(&self, block_gas_limit: u64) -> u64 {
        self.gas_limit().map_or(block_gas_limit, |limit| limit.min(block_gas_limit))
    }
}

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
    /// The operator's gas limit, shared with the payload builder.
    gas_limit_config: OpGasLimitConfig,
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
    /// Returns the most recent block gas limit seen by the validator.
    pub fn block_gas_limit(&self) -> u64 {
        self.inner.block_gas_limit()
    }

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
            gas_limit_config: OpGasLimitConfig::default(),
        }
    }

    /// Shares the operator's gas limit, written by `miner_setGasLimit`, with this validator.
    pub fn with_gas_limit_config(mut self, gas_limit_config: OpGasLimitConfig) -> Self {
        self.gas_limit_config = gas_limit_config;
        self
    }

    /// The gas budget of the blocks this node builds: the block gas limit, or the operator's limit
    /// if that is lower. The payload builder applies this to both gas and DA footprint.
    fn build_gas_limit(&self) -> u64 {
        self.gas_limit_config.cap(self.inner.block_gas_limit())
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

        // Reserve gas for the L1-info deposit that is present in every OP block, mirroring
        // op-geth's `EffectiveGasLimit` (`core/txpool/validation.go`). The L1-info deposit is
        // injected as the first transaction of every block and always consumes some gas, so a
        // non-deposit transaction can never use the full block gas limit. The inner validator
        // caps at the full block gas limit, which would admit a transaction in
        // `(block_gas_limit - L1_INFO_GAS_OVERHEAD, block_gas_limit]` that can never be included
        // (it would sit in the pool until it expires). Reject it up front, as op-geth does.
        //
        // This is done before the interop `is_valid_cross_tx` check below: the gas check is cheap,
        // stateless and deterministic, whereas cross-tx validation is `async` and potentially
        // network-bound (access-list filtering, up to `CHECK_ACCESS_LIST_TIMEOUT_SECS`). Rejecting
        // an over-gas-limit transaction here avoids that work and is better admission-DoS hygiene.
        let gas_limit = transaction.gas_limit();
        let effective_gas_limit = self.inner.block_gas_limit().saturating_sub(L1_INFO_GAS_OVERHEAD);
        if gas_limit > effective_gas_limit {
            return TransactionValidationOutcome::Invalid(
                transaction,
                InvalidPoolTransactionError::ExceedsGasLimit(gas_limit, effective_gas_limit),
            );
        }

        // Apply the operator's limit too, if that is lower, as the payload builder does. This limit
        // is local to this node, so unlike the check above it isn't reported as a bad transaction.
        let build_gas_limit = self.build_gas_limit().saturating_sub(L1_INFO_GAS_OVERHEAD);
        if gas_limit > build_gas_limit {
            return TransactionValidationOutcome::Invalid(
                transaction,
                InvalidPoolTransactionError::MaxTxGasLimitExceeded(gas_limit, build_gas_limit),
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
        let outcome = self.apply_da_footprint_check(outcome);

        self.apply_op_checks(outcome)
    }

    /// Since Jovian a block's total DA footprint must not exceed its gas limit
    /// (<https://specs.optimism.io/protocol/jovian/exec-engine.html#da-footprint-block-limit>).
    /// Apply that limit to a single transaction at admission, as the payload builder does
    /// (`is_tx_over_limits`). Deposits don't count towards the footprint, so the full block gas
    /// limit applies. Pre-Jovian L1 info carries no scalar and has no such limit.
    ///
    /// The operator's gas limit (`miner_setGasLimit`) applies if it is lower, because the payload
    /// builder checks the footprint against it too. Batcher throttling (`miner_setMaxDASize`) is
    /// not enforced here: those limits are dynamic, and transactions over them become includable
    /// again once throttling eases.
    ///
    /// Runs on a `Valid` outcome because the footprint takes a `FastLZ` pass over the encoded
    /// transaction, and only the inner validator enforces the input size limit. It stays outside
    /// [`Self::apply_op_checks`], whose early return for `require_l1_data_gas_fee == false` would
    /// skip it.
    fn apply_da_footprint_check(
        &self,
        outcome: TransactionValidationOutcome<Tx>,
    ) -> TransactionValidationOutcome<Tx> {
        if let Some(valid_tx) = outcome.as_valid_transaction() &&
            let Err(err) = self.check_da_footprint(valid_tx.transaction())
        {
            return TransactionValidationOutcome::Invalid(valid_tx.transaction().clone(), err);
        }
        outcome
    }

    /// Rejects `tx` if its DA footprint alone exceeds the gas budget of the blocks this node
    /// builds.
    ///
    /// The verdict depends on this node's view of the limits, so it uses an error that isn't
    /// reported as a bad transaction.
    fn check_da_footprint(&self, tx: &Tx) -> Result<(), InvalidPoolTransactionError> {
        let Some(scalar) = self.block_info.l1_block_info.read().da_footprint_gas_scalar else {
            return Ok(());
        };
        let footprint = tx.estimated_da_size().saturating_mul(u64::from(scalar));
        let gas_limit = self.build_gas_limit();
        if footprint > gas_limit {
            return Err(InvalidPoolTransactionError::MaxTxGasLimitExceeded(footprint, gas_limit));
        }
        Ok(())
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
            transaction: mut valid_tx,
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

            // Also reserve the operator fee (introduced in Isthmus). The Isthmus exec-engine spec
            // mandates this pool behavior: "the transaction pool must reject transactions that do
            // not have enough balance to cover the worst-case cost of the transaction fee. This
            // worst-case cost of a transaction now includes the worst-case operator fee."
            // <https://specs.optimism.io/protocol/isthmus/exec-engine.html#operator-fee>
            // (Jovian changes the formula: <https://specs.optimism.io/protocol/jovian/exec-engine.html#operator-fee>.)
            // Reserving only the L1 data fee above lets a sender covering L2 gas + value + L1 data
            // fee — but not the operator fee — be admitted to the pool and then fail at execution.
            let spec_id =
                revm_spec_by_timestamp_after_bedrock(self.chain_spec(), self.block_timestamp());
            cost_addition = cost_addition.saturating_add(operator_fee_addition(
                &l1_block_info,
                spec_id,
                &encoded,
                valid_tx.transaction().gas_limit(),
            ));

            // Fold the OP fees into `cost()` so the pool's cumulative per-sender balance check
            // accounts for them too. This check only sees one transaction against the on-chain
            // balance; a sender's later nonces must also cover the OP fees of the earlier ones, or
            // the pool marks them pending although execution rejects them for insufficient funds.
            // The fees are priced at admission.
            match &mut valid_tx {
                ValidTransaction::Valid(tx) |
                ValidTransaction::ValidWithSidecar { transaction: tx, .. } => {
                    tx.set_op_fee_reservation(cost_addition)
                }
            }
            let cost = *valid_tx.transaction().cost();

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

/// UPSTREAM-MIRROR(delegate): reth@rev:e1bf972
/// `reth_transaction_pool::validate::EthTransactionValidator`
///
/// Wraps the Ethereum validator. Re-check newly defaulted batch methods and upstream overrides;
/// inheriting a generic default can lose the inner validator's shared-state batching.
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

/// Operator fee to reserve for a non-deposit transaction in the tx-pool balance check.
///
/// Implements the pool-admission rule from the Isthmus exec-engine spec (the pool must reserve the
/// worst-case operator fee): <https://specs.optimism.io/protocol/isthmus/exec-engine.html#operator-fee>.
/// The charge delegates to [`L1BlockInfo::operator_fee_charge`], so the fork-dependent formula
/// (Isthmus vs the Jovian 100× multiplier,
/// <https://specs.optimism.io/protocol/jovian/exec-engine.html#operator-fee>) stays in one place.
/// `apply_op_checks` otherwise reserves only the L1 data fee, so a sender covering L2 gas + value +
/// L1 data fee — but not the operator fee — would be admitted to the pool and then fail at
/// execution.
///
/// Returns zero before Isthmus or when the operator fee params are unset:
/// [`L1BlockInfo::operator_fee_charge`] panics on a missing scalar/constant, so both must be
/// present before the charge is computed. Uses the transaction's `gas_limit` (worst-case spend),
/// matching the spec's worst-case reservation.
fn operator_fee_addition(
    l1_block_info: &L1BlockInfo,
    spec_id: OpSpecId,
    encoded: &[u8],
    gas_limit: u64,
) -> U256 {
    if spec_id.is_enabled_in(OpSpecId::ISTHMUS) &&
        l1_block_info.operator_fee_scalar.is_some() &&
        l1_block_info.operator_fee_constant.is_some()
    {
        l1_block_info.operator_fee_charge(encoded, U256::from(gas_limit), spec_id)
    } else {
        U256::ZERO
    }
}

#[cfg(test)]
mod tests {
    use super::operator_fee_addition;
    use alloy_primitives::U256;
    use op_revm::{L1BlockInfo, OpSpecId};

    /// A non-deposit EIP-2718 encoding: first byte is a tx-type (0x02 = EIP-1559), not the `0x7E`
    /// deposit marker, and non-empty — so `operator_fee_charge` does not short-circuit.
    const NON_DEPOSIT_ENCODED: &[u8] = &[0x02, 0xAB, 0xCD];

    fn l1_info_with_operator_fee(scalar: u64, constant: u64) -> L1BlockInfo {
        L1BlockInfo {
            operator_fee_scalar: Some(U256::from(scalar)),
            operator_fee_constant: Some(U256::from(constant)),
            ..Default::default()
        }
    }

    #[test]
    fn operator_fee_added_when_isthmus_and_params_present() {
        let l1 = l1_info_with_operator_fee(1, 2);
        let gas_limit = 21_000u64;
        let expected =
            l1.operator_fee_charge(NON_DEPOSIT_ENCODED, U256::from(gas_limit), OpSpecId::ISTHMUS);
        assert!(expected > U256::ZERO);
        assert_eq!(
            operator_fee_addition(&l1, OpSpecId::ISTHMUS, NON_DEPOSIT_ENCODED, gas_limit),
            expected
        );
    }

    #[test]
    fn operator_fee_uses_jovian_formula_in_jovian() {
        // Jovian changes the operator fee formula (`gas × scalar × 100 + constant`) from the
        // Isthmus one (`gas × scalar ÷ 1e6 + constant`) — a ~100× difference for the same params.
        // The reservation passes `spec_id` straight through to `operator_fee_charge`, so this pins
        // it to the fork: if the delegation ever drifts, the Jovian path here breaks.
        // <https://specs.optimism.io/protocol/jovian/exec-engine.html#operator-fee>
        let l1 = l1_info_with_operator_fee(1, 2);
        let gas_limit = 21_000u64;

        let expected =
            l1.operator_fee_charge(NON_DEPOSIT_ENCODED, U256::from(gas_limit), OpSpecId::JOVIAN);
        assert_eq!(
            operator_fee_addition(&l1, OpSpecId::JOVIAN, NON_DEPOSIT_ENCODED, gas_limit),
            expected
        );

        // Same params, higher fork: Jovian's multiplier makes the reservation strictly larger than
        // Isthmus — so a fork mix-up can't go unnoticed.
        let isthmus = operator_fee_addition(&l1, OpSpecId::ISTHMUS, NON_DEPOSIT_ENCODED, gas_limit);
        assert!(
            expected > isthmus,
            "Jovian reservation ({expected}) must exceed Isthmus ({isthmus}) for the same params"
        );
    }

    #[test]
    fn operator_fee_zero_before_isthmus() {
        // Params present but spec pre-Isthmus → op-geth charges no operator fee.
        let l1 = l1_info_with_operator_fee(1, 2);
        assert_eq!(
            operator_fee_addition(&l1, OpSpecId::BEDROCK, NON_DEPOSIT_ENCODED, 21_000),
            U256::ZERO
        );
    }

    #[test]
    fn operator_fee_zero_and_no_panic_when_params_unset() {
        // Pre-Isthmus L1 info has no operator fee params; `operator_fee_charge` would panic on the
        // missing scalar/constant, so the guard must return zero without calling it.
        let l1 = L1BlockInfo::default();
        assert!(l1.operator_fee_scalar.is_none() && l1.operator_fee_constant.is_none());
        assert_eq!(
            operator_fee_addition(&l1, OpSpecId::ISTHMUS, NON_DEPOSIT_ENCODED, 21_000),
            U256::ZERO
        );
    }

    #[test]
    fn operator_fee_zero_for_deposit_input() {
        // A deposit-typed (0x7E) encoding is exempt from the operator fee.
        let l1 = l1_info_with_operator_fee(1, 2);
        assert_eq!(
            operator_fee_addition(&l1, OpSpecId::ISTHMUS, &[0x7E, 0x00], 21_000),
            U256::ZERO
        );
    }

    /// The pool must classify a sender's consecutive nonces against the full OP cost, not just the
    /// L2 `cost()`. Each tx below is affordable on its own (so both pass admission against the same
    /// on-chain balance), but the sender cannot pay the L1 data fee of both. Without the L1 fee in
    /// the pool's cumulative accounting, nonce 1 is marked pending and the payload builder retries
    /// it every block, failing with insufficient funds.
    #[tokio::test]
    async fn pool_parks_descendant_unaffordable_with_l1_fee() {
        use crate::{OpL1BlockInfo, OpPooledTransaction, OpTransactionValidator};
        use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
        use alloy_eips::eip2718::Encodable2718;
        use alloy_primitives::{Address, Signature, TxKind};
        use parking_lot::RwLock;
        use reth_optimism_chainspec::OP_MAINNET;
        use reth_optimism_evm::{OpEvmConfig, RethL1BlockInfo};
        use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
        use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
        use reth_transaction_pool::{
            CoinbaseTipOrdering, Pool, PoolConfig, PoolTransaction, TransactionOrigin,
            TransactionPool, blobstore::InMemoryBlobStore,
            validate::EthTransactionValidatorBuilder,
        };
        use std::sync::atomic::AtomicU64;

        let signer = Address::with_last_byte(1);
        let make_tx = |nonce: u64| -> OpPooledTransaction {
            let tx: OpTransactionSigned = TxEip1559 {
                chain_id: 10,
                nonce,
                gas_limit: 21_000,
                max_fee_per_gas: 1_000_000_000,
                to: TxKind::Call(Address::with_last_byte(0x42)),
                ..Default::default()
            }
            .into_signed(Signature::test_signature())
            .into();
            let recovered = Recovered::new_unchecked(tx, signer);
            let len = recovered.encode_2718_len();
            OpPooledTransaction::new(recovered, len)
        };
        let (tx0, tx1) = (make_tx(0), make_tx(1));

        let mut l1_block_info = L1BlockInfo {
            l1_base_fee: U256::from(1_000_000_000_000u64),
            l1_base_fee_scalar: U256::from(1_000_000),
            ..Default::default()
        };
        let l1_fee =
            l1_block_info.l1_tx_data_fee(OP_MAINNET.clone(), 0, tx0.encoded_2718(), false).unwrap();
        let l2_cost = *tx0.cost();
        assert!(l1_fee > l2_cost, "test needs the L1 fee to dominate, as on ink-mainnet");

        // Covers either tx alone (and both txs' L2 cost), but not both txs' full OP cost.
        let balance = l2_cost * U256::from(2) + l1_fee + l1_fee / U256::from(2);

        let client = MockEthProvider::<OpPrimitives>::new()
            .with_chain_spec(OP_MAINNET.clone())
            .with_genesis_block();
        client.add_account(signer, ExtendedAccount::new(0, balance));
        let inner =
            EthTransactionValidatorBuilder::new(client, OpEvmConfig::optimism(OP_MAINNET.clone()))
                .build(InMemoryBlobStore::default());
        let validator = OpTransactionValidator::with_block_info(
            inner,
            OpL1BlockInfo {
                l1_block_info: RwLock::new(l1_block_info),
                timestamp: AtomicU64::new(0),
            },
        );
        let pool = Pool::new(
            validator,
            CoinbaseTipOrdering::default(),
            InMemoryBlobStore::default(),
            PoolConfig::default(),
        );

        pool.add_transaction(TransactionOrigin::External, tx0).await.unwrap();
        pool.add_transaction(TransactionOrigin::External, tx1).await.unwrap();

        let pending: Vec<_> =
            pool.get_pending_transactions_by_sender(signer).iter().map(|tx| tx.nonce()).collect();
        let queued: Vec<_> =
            pool.get_queued_transactions_by_sender(signer).iter().map(|tx| tx.nonce()).collect();
        assert_eq!(pending, vec![0], "only nonce 0 is executable");
        assert_eq!(queued, vec![1], "nonce 1 cannot cover its L1 data fee after nonce 0");
    }

    mod da_footprint {
        use crate::{
            OpGasLimitConfig, OpL1BlockInfo, OpPooledTransaction, OpTransactionValidator,
            estimated_da_size::DataAvailabilitySized,
        };
        use alloy_consensus::{
            SignableTransaction, Transaction, TxEip1559, transaction::Recovered,
        };
        use alloy_eips::eip2718::Encodable2718;
        use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256};
        use op_revm::L1BlockInfo;
        use parking_lot::RwLock;
        use reth_optimism_chainspec::{OP_MAINNET, OpChainSpec};
        use reth_optimism_evm::OpEvmConfig;
        use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
        use reth_primitives_traits::transaction::error::InvalidTransactionError;
        use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
        use reth_transaction_pool::{
            TransactionOrigin, TransactionValidationOutcome,
            blobstore::InMemoryBlobStore,
            error::{InvalidPoolTransactionError, PoolError, PoolErrorKind},
            validate::EthTransactionValidatorBuilder,
        };
        use std::sync::{Arc, atomic::AtomicU64};

        /// OP Mainnet's block gas limit.
        const BLOCK_GAS_LIMIT: u64 = 40_000_000;
        /// The spec default `daFootprintGasScalar` (op-node substitutes it for a `SystemConfig` 0).
        const DEFAULT_SCALAR: u16 = 400;
        /// Under reth's 128 KiB input cap, yet ~100 KB DA-estimated once compressed, so the
        /// default scalar puts its footprint above [`BLOCK_GAS_LIMIT`].
        const CALLDATA_LEN: usize = 120_000;
        /// reth's default `--txpool.max-tx-input-bytes`.
        const DEFAULT_INPUT_LIMIT: usize = 128 * 1024;

        /// Calldata that doesn't compress, so the DA size estimate tracks its length.
        fn incompressible(len: usize) -> Bytes {
            let mut state = 0x9E37_79B9_7F4A_7C15_u64;
            (0..len)
                .map(|_| {
                    state ^= state << 13;
                    state ^= state >> 7;
                    state ^= state << 17;
                    state as u8
                })
                .collect()
        }

        fn large_tx(signer: Address) -> OpPooledTransaction {
            large_tx_with_input_len(signer, CALLDATA_LEN)
        }

        fn large_tx_with_input_len(signer: Address, input_len: usize) -> OpPooledTransaction {
            let tx: OpTransactionSigned = TxEip1559 {
                chain_id: 10,
                gas_limit: 5_000_000,
                max_fee_per_gas: 1_000_000_000,
                to: TxKind::Call(Address::with_last_byte(0x42)),
                input: incompressible(input_len),
                ..Default::default()
            }
            .into_signed(Signature::test_signature())
            .into();
            let recovered = Recovered::new_unchecked(tx, signer);
            let len = recovered.encode_2718_len();
            OpPooledTransaction::new(recovered, len)
        }

        async fn validate(
            tx: OpPooledTransaction,
            da_footprint_gas_scalar: Option<u16>,
            block_gas_limit: u64,
        ) -> TransactionValidationOutcome<OpPooledTransaction> {
            validator(&tx, U256::MAX, da_footprint_gas_scalar, block_gas_limit)
                .validate_one(TransactionOrigin::External, tx)
                .await
        }

        fn validator(
            tx: &OpPooledTransaction,
            balance: U256,
            da_footprint_gas_scalar: Option<u16>,
            block_gas_limit: u64,
        ) -> OpTransactionValidator<
            MockEthProvider<OpPrimitives, Arc<OpChainSpec>>,
            OpPooledTransaction,
            OpEvmConfig,
        > {
            let signer = tx.transaction().signer();
            let client = MockEthProvider::<OpPrimitives>::new()
                .with_chain_spec(OP_MAINNET.clone())
                .with_genesis_block();
            client.add_account(signer, ExtendedAccount::new(0, balance));
            let inner = EthTransactionValidatorBuilder::new(
                client,
                OpEvmConfig::optimism(OP_MAINNET.clone()),
            )
            .set_block_gas_limit(block_gas_limit)
            .build(InMemoryBlobStore::default());
            OpTransactionValidator::with_block_info(
                inner,
                OpL1BlockInfo {
                    l1_block_info: RwLock::new(L1BlockInfo {
                        da_footprint_gas_scalar,
                        ..Default::default()
                    }),
                    timestamp: AtomicU64::new(0),
                },
            )
        }

        /// The footprint and limit the transaction was rejected with, if any.
        fn footprint_error(
            outcome: &TransactionValidationOutcome<OpPooledTransaction>,
        ) -> Option<(u64, u64)> {
            match outcome {
                TransactionValidationOutcome::Invalid(
                    _,
                    InvalidPoolTransactionError::MaxTxGasLimitExceeded(footprint, limit),
                ) => Some((*footprint, *limit)),
                _ => None,
            }
        }

        /// A transaction whose footprint alone exceeds the block gas limit is rejected.
        #[tokio::test]
        async fn rejects_footprint_above_block_gas_limit() {
            let tx = large_tx(Address::with_last_byte(1));
            let footprint = tx.estimated_da_size() * u64::from(DEFAULT_SCALAR);
            assert!(footprint > BLOCK_GAS_LIMIT, "fixture must exceed the limit: {footprint}");

            let outcome = validate(tx, Some(DEFAULT_SCALAR), BLOCK_GAS_LIMIT).await;

            assert_eq!(
                footprint_error(&outcome),
                Some((footprint, BLOCK_GAS_LIMIT)),
                "got {outcome:?}"
            );
        }

        /// The limit is inclusive and uses the full block gas limit, matching the executor: a
        /// footprint exactly equal to the block gas limit fits in an otherwise empty block, and
        /// deposits (including the L1-info deposit) don't count towards it.
        #[tokio::test]
        async fn footprint_limit_is_inclusive() {
            let tx = large_tx(Address::with_last_byte(1));
            let footprint = tx.estimated_da_size() * u64::from(DEFAULT_SCALAR);

            let at_limit = validate(tx.clone(), Some(DEFAULT_SCALAR), footprint).await;
            assert!(
                at_limit.is_valid(),
                "footprint equal to the limit must be admitted: {at_limit:?}"
            );

            let over_limit = validate(tx, Some(DEFAULT_SCALAR), footprint - 1).await;
            assert_eq!(
                footprint_error(&over_limit),
                Some((footprint, footprint - 1)),
                "got {over_limit:?}"
            );
        }

        /// Before Jovian the L1-info deposit carries no scalar and there is no DA footprint limit.
        #[tokio::test]
        async fn no_footprint_check_before_jovian() {
            let outcome =
                validate(large_tx(Address::with_last_byte(1)), None, BLOCK_GAS_LIMIT).await;
            assert!(outcome.is_valid(), "got {outcome:?}");
        }

        /// The footprint takes a `FastLZ` pass, so it runs after the inner validator: a sender who
        /// can't pay is turned away by the cheap balance lookup, whatever the footprint.
        #[tokio::test]
        async fn unfunded_sender_is_rejected_before_footprint_check() {
            let tx = large_tx(Address::with_last_byte(1));
            let footprint = tx.estimated_da_size() * u64::from(DEFAULT_SCALAR);
            assert!(footprint > BLOCK_GAS_LIMIT, "fixture must exceed the limit: {footprint}");

            let outcome = validator(&tx, U256::ZERO, Some(DEFAULT_SCALAR), BLOCK_GAS_LIMIT)
                .validate_one(TransactionOrigin::External, tx)
                .await;

            assert!(
                matches!(
                    outcome,
                    TransactionValidationOutcome::Invalid(
                        _,
                        InvalidPoolTransactionError::Consensus(
                            InvalidTransactionError::InsufficientFunds(_)
                        )
                    )
                ),
                "got {outcome:?}"
            );
        }

        /// Only the inner validator enforces the input size limit, so the check must not run a
        /// `FastLZ` pass over input of any size. Whether the footprint fits is irrelevant here.
        #[tokio::test]
        async fn oversized_tx_is_rejected_before_footprint_check() {
            let tx = large_tx_with_input_len(Address::with_last_byte(1), 2 * DEFAULT_INPUT_LIMIT);

            let outcome = validate(tx, Some(DEFAULT_SCALAR), BLOCK_GAS_LIMIT).await;

            assert!(
                matches!(
                    outcome,
                    TransactionValidationOutcome::Invalid(
                        _,
                        InvalidPoolTransactionError::OversizedData { .. }
                    )
                ),
                "got {outcome:?}"
            );
        }

        /// Whether network gossip would take reputation from a peer that sent this transaction.
        fn penalizes_peer(err: InvalidPoolTransactionError) -> bool {
            PoolError::new(B256::ZERO, PoolErrorKind::InvalidTransaction(err)).is_bad_transaction()
        }

        /// `miner_setGasLimit` only exists on this node, so a peer relaying a transaction above it
        /// isn't misbehaving and mustn't lose reputation for it.
        #[tokio::test]
        async fn gas_above_operator_limit_does_not_penalize_peers() {
            let tx = large_tx(Address::with_last_byte(1));
            let gas = tx.gas_limit();
            let operator_limit = gas - 1;

            let outcome = validator(&tx, U256::MAX, None, BLOCK_GAS_LIMIT)
                .with_gas_limit_config(OpGasLimitConfig::new(operator_limit))
                .validate_one(TransactionOrigin::External, tx)
                .await;

            let TransactionValidationOutcome::Invalid(_, err) = outcome else {
                panic!("got {outcome:?}");
            };
            assert!(!penalizes_peer(err));
        }

        /// Above the block gas limit every node agrees, so the peer should have known better.
        #[tokio::test]
        async fn gas_above_block_limit_still_penalizes_peers() {
            let tx = large_tx(Address::with_last_byte(1));
            let block_gas_limit = tx.gas_limit();

            let outcome = validator(&tx, U256::MAX, None, block_gas_limit)
                .with_gas_limit_config(OpGasLimitConfig::new(1))
                .validate_one(TransactionOrigin::External, tx)
                .await;

            let TransactionValidationOutcome::Invalid(_, err) = outcome else {
                panic!("got {outcome:?}");
            };
            assert!(penalizes_peer(err));
        }

        /// `--dev` turns off the L1 fee requirement. The footprint check must still apply, which
        /// is why it isn't part of `apply_op_checks`.
        #[tokio::test]
        async fn footprint_check_applies_without_l1_fee_requirement() {
            let tx = large_tx(Address::with_last_byte(1));
            let footprint = tx.estimated_da_size() * u64::from(DEFAULT_SCALAR);

            let outcome = validator(&tx, U256::MAX, Some(DEFAULT_SCALAR), BLOCK_GAS_LIMIT)
                .require_l1_data_gas_fee(false)
                .validate_one(TransactionOrigin::External, tx)
                .await;

            assert_eq!(
                footprint_error(&outcome),
                Some((footprint, BLOCK_GAS_LIMIT)),
                "got {outcome:?}"
            );
        }
    }
}
