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

use crate::addresses::{CONSOLE_ADDR, DEFAULT_SENDER, FORGE_DEPLOYER, SCRIPT_DEPLOYER, VM_ADDR};
use crate::allocs::{self, AllocAccount, ForgeAllocs};
use crate::artifacts::{ArtifactError, Artifacts};
use crate::cheatcodes::{Broadcast, CheatInspector};
use crate::precompiles::{HostPrecompile, OutputCapture};

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
    /// Block environment, mirroring `script.Context`'s BlockNum/Timestamp/PrevRandao.
    pub block_num: u64,
    pub timestamp: u64,
    pub prev_randao: B256,
}

impl Default for HostConfig {
    fn default() -> Self {
        Self {
            chain_id: 1337,
            no_max_code_size: false,
            use_create2_deployer: false,
            artifacts_dir: None,
            block_num: 0,
            timestamp: 0,
            prev_randao: B256::ZERO,
        }
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
pub type ScriptContext = Context<BlockEnv, TxEnv, CfgEnv, Db, revm::Journal<Db>>;

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
            number: U256::from(config.block_num),
            timestamp: U256::from(config.timestamp),
            gas_limit: FOUNDRY_GAS_LIMIT,
            basefee: 0,
            difficulty: U256::ZERO,
            prevrandao: Some(config.prev_randao),
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

    /// Imports a forge-allocs dump into the committed DB, mirroring `script.Host.ImportState`
    /// (balance + nonce + code + every storage slot, per account). Accounts are inserted via
    /// `insert_account_info` so they are visible to the EVM (avoids the CacheDB `NotExisting`
    /// cold-load trap).
    pub fn import_state(&mut self, allocs: ForgeAllocs) -> Result<(), HostError> {
        let parse = |what: &str, s: &str| -> Result<U256, HostError> {
            let digits = s.strip_prefix("0x").unwrap_or(s);
            U256::from_str_radix(digits, 16)
                .map_err(|e| HostError::Evm(format!("bad {what} {s:?}: {e}")))
        };
        for (addr_str, acct) in allocs {
            let addr: Address = addr_str
                .parse()
                .map_err(|e| HostError::Evm(format!("bad address {addr_str:?}: {e}")))?;
            let balance = parse("balance", &acct.balance)?;
            let nonce = parse("nonce", &acct.nonce)?
                .try_into()
                .map_err(|_| HostError::Evm(format!("nonce {} overflows u64", acct.nonce)))?;

            let code = match &acct.code {
                Some(c) if c != "0x" => {
                    let raw = alloy_primitives::hex::decode(c.strip_prefix("0x").unwrap_or(c))
                        .map_err(|e| HostError::Evm(format!("bad code for {addr_str}: {e}")))?;
                    Some(Bytecode::new_raw(Bytes::from(raw)))
                }
                _ => None,
            };
            let code_hash = code.as_ref().map(|b| b.hash_slow()).unwrap_or(KECCAK_EMPTY);

            let db = self.db_mut();
            if let Some(b) = &code {
                db.cache.contracts.insert(code_hash, b.clone());
            }
            db.insert_account_info(
                addr,
                AccountInfo { balance, nonce, code_hash, code, account_id: None },
            );

            if let Some(storage) = acct.storage {
                let acc =
                    self.db_mut().cache.accounts.get_mut(&addr).expect("account just inserted");
                for (slot, value) in storage {
                    let k = parse("storage slot", &slot)?;
                    let v = parse("storage value", &value)?;
                    acc.storage.insert(k, v);
                }
            }
        }
        Ok(())
    }

    /// Deployed code at `addr`, mirroring `script.Host.GetCode` (empty when absent).
    pub fn get_code(&self, addr: Address) -> Bytes {
        let db = self.db();
        let Some(acc) = db.cache.accounts.get(&addr) else {
            return Bytes::new();
        };
        if acc.info.code_hash == KECCAK_EMPTY {
            return Bytes::new();
        }
        if let Some(c) = &acc.info.code {
            return c.original_bytes();
        }
        db.cache.contracts.get(&acc.info.code_hash).map(|c| c.original_bytes()).unwrap_or_default()
    }

    /// Loads an artifact by `file`/`contract` and deploys it via CREATE from `from`.
    pub fn load_contract(
        &mut self,
        file: &str,
        contract: &str,
        from: Address,
    ) -> Result<Address, HostError> {
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
        let result =
            self.evm.inspect_tx_commit(tx).map_err(|e| HostError::Evm(format!("{e:?}")))?;
        match result {
            ExecutionResult::Success { output: Output::Create(_, Some(addr)), .. } => Ok(addr),
            ExecutionResult::Success { .. } => Err(HostError::NoCreateAddress),
            ExecutionResult::Revert { output, .. } => {
                Err(HostError::Reverted(alloy_primitives::hex::encode(output)))
            }
            ExecutionResult::Halt { reason, .. } => Err(HostError::Halted(format!("{reason:?}"))),
        }
    }

    /// Raw message call, matching `vm.EVM.Call`: the caller nonce is left untouched — including
    /// DURING execution (the inspector undoes revm's tx-level bump at first frame entry, so
    /// scripts that CREATE from the caller derive geth-identical addresses).
    pub fn call(&mut self, from: Address, to: Address, input: Bytes) -> Result<Bytes, HostError> {
        self.evm.inspector.reset_call_state();
        self.evm.inspector.pending_caller_nonce_undo = Some(from);
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
        let result =
            self.evm.inspect_tx_commit(tx).map_err(|e| HostError::Evm(format!("{e:?}")))?;
        // The tx-level caller-nonce bump was already undone at frame entry (see
        // pending_caller_nonce_undo), so in-run nonce increments on the caller (broadcast
        // bumps, CREATEs from the caller) persist exactly like geth's raw EVM.Call.
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

    /// Mint a fresh CREATE address off the script-deployer and bump its nonce, mirroring
    /// `script.Host.NewScriptAddress`. Used to place OPCM input/output precompiles so that
    /// (a) they never collide with the subsequent script deploy and (b) they fall inside the
    /// script-deployer nonce range that `state_dump` prunes.
    pub fn new_script_address(&mut self) -> Address {
        let nonce = self.get_nonce(SCRIPT_DEPLOYER);
        let addr = allocs::create_address(&SCRIPT_DEPLOYER, nonce);
        // Bump the script-deployer nonce via `insert_account_info` (not a raw field write): a bare
        // `cache.accounts.entry(..).or_default()` leaves `account_state = NotExisting`, so the
        // account stays invisible to the EVM and the next cold load panics (`ColdLoadSkipped`).
        // `insert_account_info` promotes `NotExisting -> None`, making the bumped nonce visible.
        let mut info = self
            .db()
            .cache
            .accounts
            .get(&SCRIPT_DEPLOYER)
            .map(|a| a.info.clone())
            .unwrap_or_default();
        info.nonce = nonce + 1;
        self.db_mut().insert_account_info(SCRIPT_DEPLOYER, info);
        addr
    }

    /// Insert the 1-byte placeholder code the Go host writes at every precompile-override address
    /// (`script.Host.SetPrecompile`), so Solidity's `EXTCODESIZE > 0` guard before an external call
    /// to the precompile passes. Uses `insert_account_info` so the account is EVM-visible (a raw
    /// `.or_default()` write would leave it `NotExisting` and the code invisible).
    fn insert_placeholder_code(&mut self, addr: Address) {
        let placeholder = Bytecode::new_raw(Bytes::from(vec![0u8]));
        let info = AccountInfo {
            balance: U256::ZERO,
            nonce: 0,
            code_hash: placeholder.hash_slow(),
            account_id: None,
            code: Some(placeholder),
        };
        self.db_mut().insert_account_info(addr, info);
    }

    /// Install an input getter-snapshot precompile (OPCM `RunScript*` input `I`) at a freshly
    /// minted script address and return that address, to be passed as an ABI arg to `run(...)`.
    pub fn install_input_precompile(
        &mut self,
        snapshot: alloy_primitives::map::HashMap<[u8; 4], Bytes>,
    ) -> Address {
        let addr = self.new_script_address();
        self.evm.inspector.precompiles.insert(addr, HostPrecompile::InputSnapshot(snapshot));
        self.insert_placeholder_code(addr);
        addr
    }

    /// Install an output setter-capture precompile (OPCM `RunScriptSingle` output `O`) at a freshly
    /// minted script address and return that address. `getters` are `O`'s valid field-getter
    /// selectors (a call to any other selector reverts, matching the Go host).
    pub fn install_output_precompile(
        &mut self,
        getters: alloy_primitives::map::HashSet<[u8; 4]>,
    ) -> Address {
        let addr = self.new_script_address();
        self.evm
            .inspector
            .precompiles
            .insert(addr, HostPrecompile::OutputCapture(OutputCapture::new(getters)));
        self.insert_placeholder_code(addr);
        addr
    }

    /// Drain the captured `set(...)` calldata from an output precompile, in call order, for the Go
    /// side to replay through `WithFieldSetter`.
    pub fn take_captured_sets(&mut self, addr: Address) -> Vec<Bytes> {
        match self.evm.inspector.precompiles.get_mut(&addr) {
            Some(HostPrecompile::OutputCapture(out)) => std::mem::take(&mut out.captured),
            _ => Vec::new(),
        }
    }

    /// Remove an installed precompile override (cleanup after a `RunScript*` invocation).
    pub fn remove_precompile(&mut self, addr: Address) {
        self.evm.inspector.precompiles.remove(&addr);
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
        // OPCM input/output precompile addresses (script.go prunes `h.precompiles`). These are
        // already in the pruned script-deployer nonce range, but prune explicitly for robustness.
        for a in self.evm.inspector.precompiles.keys() {
            allocs.remove(&allocs::fmt_addr(a));
        }

        allocs
    }

    pub fn default_sender() -> Address {
        DEFAULT_SENDER
    }
}
