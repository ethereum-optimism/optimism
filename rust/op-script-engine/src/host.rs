//! `ScriptHost`: a revm-backed executor that matches the semantics of
//! `op-chain-ops/script.Host` for the forge-script parity target.
//!
//! The Go host runs scripts through raw `vm.EVM.Call` / `vm.EVM.Create` (no intrinsic gas,
//! no fee, and — for top-level calls — no caller-nonce bump). revm only exposes the full
//! transaction pipeline, so we neutralize fees/limits via `CfgEnv` disable-flags and undo the
//! tx-level caller-nonce bump for top-level CALLs (CREATE keeps its bump, as in geth).

use std::collections::BTreeMap;

use alloy_primitives::{Address, B256, Bytes, U256};
use revm::{
    Context, ExecuteCommitEvm, InspectCommitEvm, MainBuilder, MainContext, MainnetEvm,
    context::{
        BlockEnv, CfgEnv, TxEnv,
        result::{ExecutionResult, Output},
    },
    database::{CacheDB, EmptyDB},
    primitives::{KECCAK_EMPTY, TxKind, hardfork::SpecId},
    state::{AccountInfo, Bytecode},
};

use crate::addresses::{
    CONSOLE_ADDR, DEFAULT_SENDER, FORGE_DEPLOYER, SCRIPT_DEPLOYER, VM_ADDR,
};
use crate::allocs::{self, AllocAccount, ForgeAllocs};
use crate::artifacts::{ArtifactError, Artifacts};
use crate::cheatcodes::{Broadcast, CheatInspector};

/// forge's `DefaultFoundryGasLimit` (int64.max).
pub const FOUNDRY_GAS_LIMIT: u64 = 9_223_372_036_854_775_807;

/// L1-only Cancun EVM, matching the Go host chain config.
pub const SPEC: SpecId = SpecId::CANCUN;

#[derive(Debug, Clone)]
pub struct HostConfig {
    pub chain_id: u64,
    pub no_max_code_size: bool,
    pub use_create2_deployer: bool,
    pub artifacts_dir: Option<std::path::PathBuf>,
}

impl Default for HostConfig {
    fn default() -> Self {
        Self { chain_id: 1337, no_max_code_size: false, use_create2_deployer: false, artifacts_dir: None }
    }
}

#[derive(Debug, thiserror::Error)]
pub enum HostError {
    #[error("evm error: {0}")]
    Evm(String),
    #[error("execution reverted: 0x{0}")]
    Reverted(String),
    #[error("execution halted: {0}")]
    Halted(String),
    #[error("create returned no address")]
    NoCreateAddress,
    #[error(transparent)]
    Artifact(#[from] ArtifactError),
}

type Db = CacheDB<EmptyDB>;

pub struct ScriptHost {
    evm: MainnetEvm<ScriptContext, CheatInspector>,
    artifacts: Option<Artifacts>,
    chain_id: u64,
}

/// The concrete revm context type used throughout the engine.
pub type ScriptContext =
    Context<BlockEnv, TxEnv, CfgEnv, Db, revm::Journal<Db>>;

impl ScriptHost {
    pub fn new(config: HostConfig) -> Self {
        let mut cache: Db = CacheDB::new(EmptyDB::default());

        // EnableCheats: VM_ADDR gets a 1-byte placeholder so solidity EXTCODESIZE guards pass.
        let placeholder = Bytecode::new_raw(Bytes::from(vec![0u8]));
        cache.insert_account_info(
            VM_ADDR,
            AccountInfo {
                balance: U256::ZERO,
                nonce: 0,
                code_hash: placeholder.hash_slow(),
                account_id: None,
                code: Some(placeholder),
            },
        );

        let mut cfg = CfgEnv::new_with_spec(SPEC);
        cfg.chain_id = config.chain_id;
        cfg.disable_nonce_check = true;
        cfg.disable_balance_check = true;
        cfg.disable_block_gas_limit = true;
        cfg.disable_base_fee = true;
        cfg.disable_eip3607 = true;
        if config.no_max_code_size {
            cfg.limit_contract_code_size = Some(usize::MAX);
        }

        let block = BlockEnv {
            number: U256::ZERO,
            timestamp: U256::ZERO,
            gas_limit: FOUNDRY_GAS_LIMIT,
            basefee: 0,
            difficulty: U256::ZERO,
            prevrandao: Some(B256::ZERO),
            beneficiary: Address::ZERO,
            ..Default::default()
        };

        let artifacts = config.artifacts_dir.clone().map(Artifacts::new);

        let mut inspector = CheatInspector::default();
        inspector.use_create2_deployer = config.use_create2_deployer;
        // The inspector resolves `vm.getCode` / `vm.getDeployedCode` by name from the same FS.
        inspector.artifacts = artifacts.clone();

        let ctx = Context::mainnet().with_db(cache).with_cfg(cfg).with_block(block);
        let evm = ctx.build_mainnet_with_inspector(inspector);

        Self { evm, artifacts, chain_id: config.chain_id }
    }

    fn db(&self) -> &Db {
        &self.evm.ctx.journaled_state.database
    }

    fn db_mut(&mut self) -> &mut Db {
        &mut self.evm.ctx.journaled_state.database
    }

    pub fn allow_cheatcodes(&mut self, addr: Address) {
        self.evm.inspector.allowed.insert(addr);
    }

    pub fn set_env(&mut self, key: String, value: String) {
        self.evm.inspector.env_vars.insert(key, value);
    }

    pub fn get_nonce(&self, addr: Address) -> u64 {
        self.db().cache.accounts.get(&addr).map(|a| a.info.nonce).unwrap_or(0)
    }

    pub fn set_nonce(&mut self, addr: Address, nonce: u64) {
        let acc = self.db_mut().cache.accounts.entry(addr).or_default();
        acc.info.nonce = nonce;
    }

    pub fn set_balance(&mut self, addr: Address, balance: U256) {
        let acc = self.db_mut().cache.accounts.entry(addr).or_default();
        acc.info.balance = balance;
    }

    pub fn set_storage(&mut self, addr: Address, key: U256, value: U256) {
        let acc = self.db_mut().cache.accounts.entry(addr).or_default();
        acc.storage.insert(key, value);
    }

    pub fn set_code(&mut self, addr: Address, code: Bytes) {
        let bytecode = Bytecode::new_raw(code);
        let hash = bytecode.hash_slow();
        let db = self.db_mut();
        db.cache.contracts.insert(hash, bytecode.clone());
        let acc = db.cache.accounts.entry(addr).or_default();
        acc.info.code_hash = hash;
        acc.info.code = Some(bytecode);
    }

    /// Loads an artifact by `file`/`contract` and deploys it via CREATE from `from`.
    pub fn load_contract(&mut self, file: &str, contract: &str, from: Address) -> Result<Address, HostError> {
        let art = self
            .artifacts
            .as_ref()
            .ok_or_else(|| HostError::Evm("no artifacts dir configured".into()))?
            .read(file, contract)?;
        self.create(from, art.bytecode)
    }

    pub fn create(&mut self, from: Address, init_code: Bytes) -> Result<Address, HostError> {
        self.evm.inspector.reset_call_state();
        let nonce = self.get_nonce(from);
        let tx = TxEnv {
            caller: from,
            gas_limit: FOUNDRY_GAS_LIMIT,
            gas_price: 0,
            kind: TxKind::Create,
            value: U256::ZERO,
            data: init_code,
            nonce,
            chain_id: Some(self.chain_id),
            ..Default::default()
        };
        let result = self.evm.inspect_tx_commit(tx).map_err(|e| HostError::Evm(format!("{e:?}")))?;
        match result {
            ExecutionResult::Success { output: Output::Create(_, Some(addr)), .. } => Ok(addr),
            ExecutionResult::Success { .. } => Err(HostError::NoCreateAddress),
            ExecutionResult::Revert { output, .. } => {
                Err(HostError::Reverted(alloy_primitives::hex::encode(output)))
            }
            ExecutionResult::Halt { reason, .. } => Err(HostError::Halted(format!("{reason:?}"))),
        }
    }

    /// Raw message call, matching `vm.EVM.Call`: the caller nonce is left untouched.
    pub fn call(&mut self, from: Address, to: Address, input: Bytes) -> Result<Bytes, HostError> {
        self.evm.inspector.reset_call_state();
        let pre_nonce = self.get_nonce(from);
        let tx = TxEnv {
            caller: from,
            gas_limit: FOUNDRY_GAS_LIMIT,
            gas_price: 0,
            kind: TxKind::Call(to),
            value: U256::ZERO,
            data: input,
            nonce: pre_nonce,
            chain_id: Some(self.chain_id),
            ..Default::default()
        };
        let result = self.evm.inspect_tx_commit(tx).map_err(|e| HostError::Evm(format!("{e:?}")))?;
        // Undo the transaction-level caller-nonce bump (raw EVM.Call does not touch it).
        self.set_nonce(from, pre_nonce);
        match result {
            ExecutionResult::Success { output: Output::Call(bytes), .. } => Ok(bytes),
            ExecutionResult::Success { output: Output::Create(bytes, _), .. } => Ok(bytes),
            ExecutionResult::Revert { output, .. } => {
                Err(HostError::Reverted(alloy_primitives::hex::encode(output)))
            }
            ExecutionResult::Halt { reason, .. } => Err(HostError::Halted(format!("{reason:?}"))),
        }
    }

    pub fn take_broadcasts(&mut self) -> Vec<Broadcast> {
        std::mem::take(&mut self.evm.inspector.broadcasts)
    }

    /// Wipe an account: clear code and zero the nonce/balance, making it EIP-161 empty (so it is
    /// dropped from the dump). Storage is retained, mirroring `script.Host.Wipe`. Operates on the
    /// committed DB state, so it is called between execution and `state_dump`.
    pub fn wipe(&mut self, addr: Address) {
        if let Some(acc) = self.db_mut().cache.accounts.get_mut(&addr) {
            acc.info.code = Some(Bytecode::default());
            acc.info.code_hash = KECCAK_EMPTY;
            acc.info.nonce = 0;
            acc.info.balance = U256::ZERO;
        }
    }

    /// Deploy a forge script from the script-deployer, run its `run(input)` entrypoint from
    /// `deployer` (tx.origin), then wipe the script account. Mirrors `forgeScriptImpl.Call` +
    /// `forgeScriptBackendImpl.{Deploy,Call,Destroy}` (`op-chain-ops/script/deploy.go`): the script
    /// is deployed with the code-size checks disabled and granted cheatcode access.
    pub fn run_script(
        &mut self,
        file: &str,
        contract: &str,
        calldata: Bytes,
        deployer: Address,
    ) -> Result<Bytes, HostError> {
        let bytecode = self
            .artifacts
            .as_ref()
            .ok_or_else(|| HostError::Evm("no artifacts dir configured".into()))?
            .read(file, contract)?
            .bytecode;

        let deploy_nonce = self.get_nonce(SCRIPT_DEPLOYER);
        let expected = allocs::create_address(&SCRIPT_DEPLOYER, deploy_nonce);
        self.allow_cheatcodes(expected);
        self.evm.inspector.labels.insert(expected, contract.to_string());

        // Scripts exceed the EIP-170/EIP-3860 limits; deploy with the size checks lifted, exactly
        // as forge's Deploy does via EnforceMaxCodeSize(false). Restored right after the CREATE so
        // script-internal deployments are still size-checked (matching the Go host).
        let prev_limit = self.evm.ctx.cfg.limit_contract_code_size;
        self.evm.ctx.cfg.limit_contract_code_size = Some(usize::MAX);
        let deployed = self.create(SCRIPT_DEPLOYER, bytecode);
        self.evm.ctx.cfg.limit_contract_code_size = prev_limit;
        let deployed = deployed?;
        if deployed != expected {
            return Err(HostError::Evm(format!(
                "script deployed to {deployed:?}, expected {expected:?}"
            )));
        }

        let out = self.call(deployer, deployed, calldata)?;
        self.wipe(deployed);
        Ok(out)
    }

    /// Dumps state into forge-allocs form, applying the same pruning as
    /// `script.Host.StateDump` + `foundry.ForgeAllocs.FromState`.
    pub fn state_dump(&self) -> ForgeAllocs {
        let db = self.db();
        let mut allocs: ForgeAllocs = BTreeMap::new();

        for (addr, acc) in db.cache.accounts.iter() {
            let info = &acc.info;
            // Resolve code (CacheDB keeps code in `contracts`, not on the account info).
            let code: Vec<u8> = if info.code_hash == KECCAK_EMPTY {
                Vec::new()
            } else if let Some(c) = &info.code {
                c.original_bytes().to_vec()
            } else if let Some(c) = db.cache.contracts.get(&info.code_hash) {
                c.original_bytes().to_vec()
            } else {
                Vec::new()
            };

            // EIP-161 empty accounts are absent from geth's committed trie.
            if info.nonce == 0 && info.balance.is_zero() && code.is_empty() {
                continue;
            }

            let mut storage = BTreeMap::new();
            for (slot, value) in acc.storage.iter() {
                if value.is_zero() {
                    continue; // matches script.go zero-slot pruning
                }
                storage.insert(
                    allocs::fmt_hash(&B256::from(slot.to_be_bytes::<32>())),
                    allocs::fmt_hash(&B256::from(value.to_be_bytes::<32>())),
                );
            }

            let account = AllocAccount {
                balance: allocs::fmt_u256(&info.balance),
                nonce: allocs::fmt_u64(info.nonce),
                code: if code.is_empty() { None } else { Some(allocs::fmt_code(&code)) },
                storage: if storage.is_empty() { None } else { Some(storage) },
            };
            allocs.insert(allocs::fmt_addr(addr), account);
        }

        // Prune script-infrastructure accounts (script.go:828-861).
        let script_deployer_nonce = self.get_nonce(SCRIPT_DEPLOYER);
        for i in 0..=script_deployer_nonce {
            allocs.remove(&allocs::fmt_addr(&allocs::create_address(&SCRIPT_DEPLOYER, i)));
        }
        for a in [SCRIPT_DEPLOYER, FORGE_DEPLOYER, VM_ADDR, CONSOLE_ADDR] {
            allocs.remove(&allocs::fmt_addr(&a));
        }

        allocs
    }

    pub fn default_sender() -> Address {
        DEFAULT_SENDER
    }
}
