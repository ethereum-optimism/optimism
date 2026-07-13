//! The forge cheatcode + console inspector. This replaces op-geth's `PrecompileOverrides`
//! (cheatcode / console precompiles) and `CallerOverride` (prank / broadcast) hooks with a
//! single revm `Inspector`, mirroring the semantics of `op-chain-ops/script`.

use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_sol_types::{SolCall, sol};
use revm::{
    Inspector,
    context_interface::{ContextTr, JournalTr, journaled_state::account::JournaledAccountTr},
    interpreter::{
        CallInputs, CallOutcome, CallScheme, CreateInputs, CreateOutcome, CreateScheme, Gas,
        InstructionResult, InterpreterResult,
    },
    primitives::KECCAK_EMPTY,
    state::Bytecode,
};
use serde::Serialize;

use crate::ScriptContext;
use crate::addresses::{CONSOLE_ADDR, CREATE2_DEPLOYER, VM_ADDR};
use crate::artifacts::Artifacts;
use crate::precompiles::{HostPrecompile, PrecompileOutcome};

sol! {
    #[allow(missing_docs, clippy::too_many_arguments)]
    interface Vm {
        function getNonce(address account) external view returns (uint64);
        function etch(address who, bytes calldata code) external;
        function store(address account, bytes32 slot, bytes32 value) external;
        function load(address account, bytes32 slot) external view returns (bytes32);
        function deal(address who, uint256 newBalance) external;
        function setNonce(address account, uint64 newNonce) external;
        function resetNonce(address account) external;
        function chainId(uint256 newChainId) external;
        function label(address account, string calldata newLabel) external;
        function allowCheatcodes(address account) external;
        function getCode(string calldata artifactPath) external view returns (bytes memory);
        function getDeployedCode(string calldata artifactPath) external view returns (bytes memory);
        function computeCreate2Address(bytes32 salt, bytes32 initCodeHash) external pure returns (address);
        function addr(uint256 privateKey) external pure returns (address);
        function prank(address msgSender) external;
        function prank(address msgSender, address txOrigin) external;
        function startPrank(address msgSender) external;
        function startPrank(address msgSender, address txOrigin) external;
        function stopPrank() external;
        function broadcast() external;
        function broadcast(address signer) external;
        function startBroadcast() external;
        function startBroadcast(address signer) external;
        function stopBroadcast() external;
    }
}

/// A broadcast captured by `vm.broadcast()` / `vm.startBroadcast()`. Serializes to the same
/// JSON as `op-chain-ops/script.Broadcast`.
#[derive(Debug, Clone, Serialize)]
pub struct Broadcast {
    pub from: String,
    pub to: String,
    pub input: String,
    pub value: String,
    pub salt: String,
    #[serde(rename = "gasUsed")]
    pub gas_used: u64,
    #[serde(rename = "type")]
    pub kind: String,
    pub nonce: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum BroadcastKind {
    Call,
    Create,
    Create2,
}

impl BroadcastKind {
    fn as_str(self) -> &'static str {
        match self {
            BroadcastKind::Call => "call",
            BroadcastKind::Create => "create",
            BroadcastKind::Create2 => "create2",
        }
    }
}

/// Active prank/broadcast task, set on the executing frame, applied to its next sub-call.
#[derive(Debug, Clone)]
struct Prank {
    sender: Option<Address>,
    /// tx.origin override (2-arg prank forms). Not exercised by the L2Genesis path, which only
    /// uses 1-arg pranks; stored for completeness.
    origin: Option<Address>,
    repeat: bool,
    broadcast: bool,
}

/// A real (non-cheatcode) call/create frame on the stack.
#[derive(Debug)]
struct Frame {
    target: Address,
    prank: Option<Prank>,
    // Broadcast capture, filled when the sub-call is entered under a broadcast prank.
    captured: Option<Capture>,
}

#[derive(Debug, Clone)]
struct Capture {
    from: Address,
    to: Address, // for creates, filled from the outcome
    input: Bytes,
    value: U256,
    salt: B256,
    nonce: u64,
    kind: BroadcastKind,
}

#[derive(Default)]
pub struct CheatInspector {
    pub allowed: alloy_primitives::map::HashSet<Address>,
    pub env_vars: std::collections::HashMap<String, String>,
    pub use_create2_deployer: bool,
    pub broadcasts: Vec<Broadcast>,
    /// Artifacts loader, used by `vm.getCode` / `vm.getDeployedCode` to resolve bytecode by name.
    pub artifacts: Option<Artifacts>,
    /// `vm.label` names, kept only so labels are accepted; they don't affect the state dump.
    pub labels: std::collections::HashMap<Address, String>,
    /// OPCM input/output precompiles installed at arbitrary addresses (`RunScript*` path). The
    /// revm-inspector replacement for op-geth's per-address `PrecompileOverrides`.
    pub precompiles: alloy_primitives::map::HashMap<Address, HostPrecompile>,
    /// Set by `ScriptHost::call` (top-level CALL) to the tx caller: revm's tx pipeline bumps the
    /// caller nonce pre-execution, but geth's raw `EVM.Call` does not, so scripts must observe
    /// the un-bumped nonce mid-run (e.g. a broadcast-pranked CREATE from the caller derives its
    /// address from that nonce). The first frame entry undoes the bump inside the journal.
    pub pending_caller_nonce_undo: Option<Address>,
    frames: Vec<Frame>,
}

impl CheatInspector {
    pub fn reset_call_state(&mut self) {
        self.frames.clear();
        self.pending_caller_nonce_undo = None;
    }

    fn is_cheat_target(addr: &Address) -> bool {
        *addr == VM_ADDR || *addr == CONSOLE_ADDR
    }
}

fn read_nonce(ctx: &mut ScriptContext, addr: Address) -> u64 {
    ctx.journal_mut().load_account(addr).map(|a| a.data.info.nonce).unwrap_or(0)
}

fn bump_nonce(ctx: &mut ScriptContext, addr: Address) {
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(addr) {
        acc.data.bump_nonce();
    }
}

fn decrement_nonce(ctx: &mut ScriptContext, addr: Address) {
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(addr) {
        let nonce = acc.data.nonce();
        acc.data.set_nonce(nonce.saturating_sub(1));
    }
}

fn cheat_outcome(
    output: Bytes,
    gas_limit: u64,
    memory_offset: std::ops::Range<usize>,
) -> CallOutcome {
    CallOutcome::new(
        InterpreterResult {
            result: InstructionResult::Return,
            output,
            gas: Gas::new(gas_limit), // 0 spent -> only the CALL opcode overhead is charged
        },
        memory_offset,
    )
}

/// A loud revert from the cheatcode precompile, carrying an ABI-encoded `Error(string)` reason so
/// the message is visible to Solidity `try/catch` and in traces. Mirrors the Go host's
/// `encodeRevert` (`op-chain-ops/script/precompile.go`): unimplemented / undispatched cheatcodes
/// revert rather than silently returning empty success.
fn cheat_revert(
    reason: String,
    gas_limit: u64,
    memory_offset: std::ops::Range<usize>,
) -> CallOutcome {
    CallOutcome::new(
        InterpreterResult {
            result: InstructionResult::Revert,
            output: encode_error_string(&reason),
            gas: Gas::new(gas_limit), // cheat precompile costs 0 gas, same as the success path
        },
        memory_offset,
    )
}

/// ABI-encode a Solidity `Error(string)` revert payload, byte-identical to the Go host's
/// `encodeRevert`: selector `0x08c379a0`, offset `0x20`, length, right-padded UTF-8 bytes.
fn encode_error_string(msg: &str) -> Bytes {
    const ERROR_SELECTOR: [u8; 4] = [0x08, 0xc3, 0x79, 0xa0]; // keccak256("Error(string)")[:4]
    let msg = msg.as_bytes();
    let padded_len = msg.len().div_ceil(32) * 32;
    let mut out = Vec::with_capacity(4 + 32 + 32 + padded_len);
    out.extend_from_slice(&ERROR_SELECTOR);
    out.extend_from_slice(&U256::from(0x20).to_be_bytes::<32>()); // offset to string
    out.extend_from_slice(&U256::from(msg.len()).to_be_bytes::<32>()); // string length
    out.extend_from_slice(msg);
    out.resize(4 + 32 + 32 + padded_len, 0); // right-pad to 32 bytes
    Bytes::from(out)
}

/// Lowercase, unprefixed hex, matching Go's `%x` used in the host's revert messages.
pub(crate) fn hex(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push_str(&format!("{b:02x}"));
    }
    s
}

/// Decode a cheatcode's ABI arguments (selector already stripped) into its `sol!` call struct.
fn decode<T: SolCall>(args: &[u8]) -> Result<T, String> {
    T::abi_decode_raw(args).map_err(|e| format!("failed to decode cheatcode args: {e}"))
}

fn abi_u64(v: u64) -> Bytes {
    let mut out = [0u8; 32];
    out[24..32].copy_from_slice(&v.to_be_bytes());
    Bytes::from(out.to_vec())
}

/// ABI-encode a single `address` return: 32-byte left-padded.
fn abi_address(a: &Address) -> Bytes {
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(a.as_slice());
    Bytes::from(out.to_vec())
}

/// ABI-encode a single dynamic `bytes` return: head offset `0x20`, length, right-padded data.
/// Matches geth `abi.Arguments.PackValues` for a lone `bytes` output.
fn abi_bytes(data: &[u8]) -> Bytes {
    let padded = data.len().div_ceil(32) * 32;
    let mut out = Vec::with_capacity(32 + 32 + padded);
    out.extend_from_slice(&U256::from(0x20).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(data.len()).to_be_bytes::<32>());
    out.extend_from_slice(data);
    out.resize(32 + 32 + padded, 0);
    Bytes::from(out)
}

/// `vm.addr`: derive the address of a secp256k1 private key, matching the Go host's
/// `crypto.PubkeyToAddress(crypto.ToECDSA(privKey))` = keccak256(uncompressed_pubkey[1..])[12..].
fn cheat_addr(private_key: U256) -> Result<Address, String> {
    use k256::elliptic_curve::sec1::ToEncodedPoint;
    let bytes: [u8; 32] = private_key.to_be_bytes();
    let sk =
        k256::SecretKey::from_slice(&bytes).map_err(|e| format!("invalid private key: {e}"))?;
    let point = sk.public_key().to_encoded_point(false); // 0x04 || X(32) || Y(32)
    let hash = alloy_primitives::keccak256(&point.as_bytes()[1..]);
    Ok(Address::from_slice(&hash[12..]))
}

/// `vm.etch`: set the account code (mirrors `state.SetCode`). Empty code clears it to
/// `KECCAK_EMPTY`, which — combined with a later `resetNonce` — makes the account EIP-161 empty.
fn cheat_etch(ctx: &mut ScriptContext, who: Address, code: Bytes) {
    let bytecode = Bytecode::new_raw(code);
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(who) {
        acc.data.set_code_and_hash_slow(bytecode);
        acc.data.touch();
    }
}

/// `vm.deal`: set an account's balance (mirrors `state.SetBalance`).
fn cheat_deal(ctx: &mut ScriptContext, who: Address, balance: U256) {
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(who) {
        acc.data.set_balance(balance);
        acc.data.touch();
    }
}

/// `vm.setNonce`: set an account's nonce (mirrors `state.SetNonce`).
fn cheat_set_nonce(ctx: &mut ScriptContext, who: Address, nonce: u64) {
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(who) {
        acc.data.set_nonce(nonce);
        acc.data.touch();
    }
}

/// `vm.resetNonce`: 0 for an EOA (empty code), 1 for a contract. Mirrors the Go host's
/// undocumented `resetNonce`.
fn cheat_reset_nonce(ctx: &mut ScriptContext, who: Address) {
    if let Ok(mut acc) = ctx.journal_mut().load_account_mut(who) {
        let n = if *acc.data.code_hash() == KECCAK_EMPTY { 0 } else { 1 };
        acc.data.set_nonce(n);
        acc.data.touch();
    }
}

impl CheatInspector {
    /// Handle a call to the VM cheatcode precompile. `Ok` carries the ABI-encoded return data;
    /// `Err` carries a revert reason for an unimplemented / undispatched cheatcode. Mirrors the
    /// Go host: an unrecognized selector reverts loudly rather than returning empty success.
    fn dispatch_cheatcode(
        &mut self,
        ctx: &mut ScriptContext,
        data: &[u8],
    ) -> Result<Bytes, String> {
        if data.len() < 4 {
            return Err(format!("expected at least 4 bytes, but got '{}'", hex(data)));
        }
        let sel: [u8; 4] = data[0..4].try_into().unwrap();
        let args = &data[4..];

        // --- state readers / writers ---
        if sel == Vm::getNonceCall::SELECTOR {
            let c = decode::<Vm::getNonceCall>(args)?;
            return Ok(abi_u64(read_nonce(ctx, c.account)));
        }
        if sel == Vm::etchCall::SELECTOR {
            let c = decode::<Vm::etchCall>(args)?;
            cheat_etch(ctx, c.who, c.code);
            return Ok(Bytes::new());
        }
        if sel == Vm::storeCall::SELECTOR {
            let c = decode::<Vm::storeCall>(args)?;
            let slot = U256::from_be_bytes(c.slot.0);
            let value = U256::from_be_bytes(c.value.0);
            // `sstore` assumes the account is already loaded (warm); warm it first so a `vm.store`
            // to an untouched account (e.g. the first write in a script) does not cold-load-panic.
            let _ = ctx.journal_mut().load_account(c.account);
            let _ = ctx.journal_mut().sstore(c.account, slot, value);
            return Ok(Bytes::new());
        }
        if sel == Vm::loadCall::SELECTOR {
            let c = decode::<Vm::loadCall>(args)?;
            let slot = U256::from_be_bytes(c.slot.0);
            let _ = ctx.journal_mut().load_account(c.account);
            let val = ctx.journal_mut().sload(c.account, slot).map(|s| s.data).unwrap_or_default();
            return Ok(Bytes::from(val.to_be_bytes::<32>().to_vec()));
        }
        if sel == Vm::dealCall::SELECTOR {
            let c = decode::<Vm::dealCall>(args)?;
            cheat_deal(ctx, c.who, c.newBalance);
            return Ok(Bytes::new());
        }
        if sel == Vm::setNonceCall::SELECTOR {
            let c = decode::<Vm::setNonceCall>(args)?;
            cheat_set_nonce(ctx, c.account, c.newNonce);
            return Ok(Bytes::new());
        }
        if sel == Vm::resetNonceCall::SELECTOR {
            let c = decode::<Vm::resetNonceCall>(args)?;
            cheat_reset_nonce(ctx, c.account);
            return Ok(Bytes::new());
        }
        if sel == Vm::chainIdCall::SELECTOR {
            let c = decode::<Vm::chainIdCall>(args)?;
            ctx.cfg.chain_id = c.newChainId.saturating_to::<u64>();
            return Ok(Bytes::new());
        }

        // --- artifact / address utilities ---
        if sel == Vm::getCodeCall::SELECTOR {
            let c = decode::<Vm::getCodeCall>(args)?;
            let art = self.read_artifact(&c.artifactPath)?;
            return Ok(abi_bytes(&art.bytecode));
        }
        if sel == Vm::getDeployedCodeCall::SELECTOR {
            let c = decode::<Vm::getDeployedCodeCall>(args)?;
            let art = self.read_artifact(&c.artifactPath)?;
            return Ok(abi_bytes(&art.deployed_bytecode));
        }
        if sel == Vm::computeCreate2AddressCall::SELECTOR {
            let c = decode::<Vm::computeCreate2AddressCall>(args)?;
            let addr = crate::allocs::create2_address_from_hash(
                &CREATE2_DEPLOYER,
                &c.salt,
                &c.initCodeHash,
            );
            return Ok(abi_address(&addr));
        }
        if sel == Vm::addrCall::SELECTOR {
            let c = decode::<Vm::addrCall>(args)?;
            return Ok(abi_address(&cheat_addr(c.privateKey)?));
        }

        // --- label / access control (no state-dump effect; accept and continue) ---
        if sel == Vm::labelCall::SELECTOR {
            let c = decode::<Vm::labelCall>(args)?;
            self.labels.insert(c.account, c.newLabel);
            return Ok(Bytes::new());
        }
        if sel == Vm::allowCheatcodesCall::SELECTOR {
            let c = decode::<Vm::allowCheatcodesCall>(args)?;
            self.allowed.insert(c.account);
            return Ok(Bytes::new());
        }

        // --- prank family (non-broadcast caller override) ---
        if sel == Vm::prank_0Call::SELECTOR {
            let c = decode::<Vm::prank_0Call>(args)?;
            self.set_prank(Some(c.msgSender), None, false, false);
            return Ok(Bytes::new());
        }
        if sel == Vm::prank_1Call::SELECTOR {
            let c = decode::<Vm::prank_1Call>(args)?;
            self.set_prank(Some(c.msgSender), Some(c.txOrigin), false, false);
            return Ok(Bytes::new());
        }
        if sel == Vm::startPrank_0Call::SELECTOR {
            let c = decode::<Vm::startPrank_0Call>(args)?;
            self.set_prank(Some(c.msgSender), None, true, false);
            return Ok(Bytes::new());
        }
        if sel == Vm::startPrank_1Call::SELECTOR {
            let c = decode::<Vm::startPrank_1Call>(args)?;
            self.set_prank(Some(c.msgSender), Some(c.txOrigin), true, false);
            return Ok(Bytes::new());
        }
        if sel == Vm::stopPrankCall::SELECTOR {
            self.stop_prank(false);
            return Ok(Bytes::new());
        }

        // --- broadcast family ---
        if sel == Vm::broadcast_0Call::SELECTOR {
            self.set_prank(None, None, false, true);
            return Ok(Bytes::new());
        }
        if sel == Vm::broadcast_1Call::SELECTOR {
            let c = decode::<Vm::broadcast_1Call>(args)?;
            self.set_prank(Some(c.signer), None, false, true);
            return Ok(Bytes::new());
        }
        if sel == Vm::startBroadcast_0Call::SELECTOR {
            self.set_prank(None, None, true, true);
            return Ok(Bytes::new());
        }
        if sel == Vm::startBroadcast_1Call::SELECTOR {
            let c = decode::<Vm::startBroadcast_1Call>(args)?;
            self.set_prank(Some(c.signer), None, true, true);
            return Ok(Bytes::new());
        }
        if sel == Vm::stopBroadcastCall::SELECTOR {
            self.stop_prank(true);
            return Ok(Bytes::new());
        }

        Err(format!("unrecognized 4 byte signature: {}", hex(&sel)))
    }

    fn read_artifact(&self, spec: &str) -> Result<crate::artifacts::Artifact, String> {
        self.artifacts
            .as_ref()
            .ok_or_else(|| "no artifacts dir configured".to_string())?
            .read_spec(spec)
            .map_err(|e| e.to_string())
    }

    fn set_prank(
        &mut self,
        sender: Option<Address>,
        origin: Option<Address>,
        repeat: bool,
        broadcast: bool,
    ) {
        if let Some(f) = self.frames.last_mut() {
            f.prank = Some(Prank { sender, origin, repeat, broadcast });
        }
    }

    fn stop_prank(&mut self, _broadcast: bool) {
        if let Some(f) = self.frames.last_mut() {
            f.prank = None;
        }
    }
}

impl Inspector<ScriptContext> for CheatInspector {
    fn call(&mut self, ctx: &mut ScriptContext, inputs: &mut CallInputs) -> Option<CallOutcome> {
        // Top-level CALL: undo the tx-pipeline caller-nonce bump before any script code runs,
        // matching geth's raw `EVM.Call` (which never bumps the caller nonce).
        if let Some(caller) = self.pending_caller_nonce_undo.take() {
            decrement_nonce(ctx, caller);
        }

        let target = inputs.target_address;

        // console.log sink: swallow, 0 gas (matches Go console precompile RequiredGas=0).
        if target == CONSOLE_ADDR {
            return Some(cheat_outcome(
                Bytes::new(),
                inputs.gas_limit,
                inputs.return_memory_offset.clone(),
            ));
        }
        // cheatcode precompile.
        if target == VM_ADDR {
            let data = inputs.input.bytes(ctx);
            let memory_offset = inputs.return_memory_offset.clone();
            let outcome = match self.dispatch_cheatcode(ctx, &data) {
                Ok(out) => cheat_outcome(out, inputs.gas_limit, memory_offset),
                Err(reason) => cheat_revert(reason, inputs.gas_limit, memory_offset),
            };
            return Some(outcome);
        }

        // OPCM input/output precompile (RunScript* path). 0 gas, like the Go precompile.
        if let Some(pc) = self.precompiles.get_mut(&target) {
            let data = inputs.input.bytes(ctx);
            let memory_offset = inputs.return_memory_offset.clone();
            let outcome = match pc.run(&data) {
                PrecompileOutcome::Return(out) => {
                    cheat_outcome(out, inputs.gas_limit, memory_offset)
                }
                PrecompileOutcome::Revert(reason) => {
                    cheat_revert(reason, inputs.gas_limit, memory_offset)
                }
            };
            return Some(outcome);
        }

        // Real sub-call: apply an active broadcast prank from the parent frame.
        let mut captured = None;
        if let Some(parent) = self.frames.last_mut() {
            if let Some(prank) = parent.prank.clone() {
                let parent_addr = parent.target;
                if !prank.repeat {
                    parent.prank = None; // one-shot consumed
                }
                if prank.broadcast
                    && inputs.scheme != CallScheme::StaticCall
                    && inputs.scheme != CallScheme::DelegateCall
                {
                    let from = prank.sender.unwrap_or(parent_addr);
                    inputs.caller = from;
                    let pre = read_nonce(ctx, from);
                    bump_nonce(ctx, from); // onEnter-style bump so the call looks like a tx
                    captured = Some(Capture {
                        from,
                        to: target,
                        input: inputs.input.bytes(ctx),
                        value: inputs.value.get(),
                        salt: B256::ZERO,
                        nonce: pre,
                        kind: BroadcastKind::Call,
                    });
                } else if let Some(sender) = prank.sender {
                    // plain prank (non-broadcast): caller override only.
                    inputs.caller = sender;
                }
            }
        }

        self.frames.push(Frame { target, prank: None, captured });
        None
    }

    fn call_end(
        &mut self,
        _ctx: &mut ScriptContext,
        inputs: &CallInputs,
        outcome: &mut CallOutcome,
    ) {
        // Frame push/pop must stay symmetric with `call`: cheat targets AND installed host
        // precompiles (OPCM input/output) short-circuit in `call` without pushing a frame, so
        // popping here would drop the caller's frame — losing an active startBroadcast/startPrank.
        if Self::is_cheat_target(&inputs.target_address)
            || self.precompiles.contains_key(&inputs.target_address)
        {
            return;
        }
        if let Some(frame) = self.frames.pop() {
            if let Some(cap) = frame.captured {
                #[allow(deprecated)]
                let gas_used = outcome.result.gas.spent();
                self.emit_broadcast(cap, gas_used);
            }
        }
    }

    fn create(
        &mut self,
        ctx: &mut ScriptContext,
        inputs: &mut CreateInputs,
    ) -> Option<CreateOutcome> {
        // `CreateInputs.set_call` overrides the creator, the CREATE equivalent of op-geth's
        // `CallerOverride`, so revm's *state* deploys at the pranked address too.
        let mut captured = None;
        if let Some(parent) = self.frames.last_mut() {
            if let Some(prank) = parent.prank.clone() {
                let parent_addr = parent.target;
                if !prank.repeat {
                    parent.prank = None;
                }
                if prank.broadcast {
                    let init_code = inputs.init_code().clone();
                    let value = inputs.value();
                    match inputs.scheme() {
                        CreateScheme::Create2 { salt } => {
                            let salt = B256::from(salt);
                            let sender = prank.sender.unwrap_or(parent_addr);
                            bump_nonce(ctx, sender); // "tx" nonce bump on the pranked sender
                            let from =
                                if self.use_create2_deployer { CREATE2_DEPLOYER } else { sender };
                            inputs.set_call(from);
                            let to = crate::allocs::create2_address(&from, &salt, &init_code);
                            captured = Some(Capture {
                                from,
                                to,
                                input: init_code,
                                value,
                                salt,
                                nonce: 0,
                                kind: BroadcastKind::Create2,
                            });
                        }
                        _ => {
                            let from = prank.sender.unwrap_or(parent_addr);
                            inputs.set_call(from);
                            let pre = read_nonce(ctx, from); // prestate; the CREATE bumps it
                            let to = crate::allocs::create_address(&from, pre);
                            captured = Some(Capture {
                                from,
                                to,
                                input: init_code,
                                value,
                                salt: B256::ZERO,
                                nonce: pre,
                                kind: BroadcastKind::Create,
                            });
                        }
                    }
                } else if let Some(sender) = prank.sender {
                    inputs.set_call(sender);
                }
            }
        }

        self.frames.push(Frame { target: Address::ZERO, prank: None, captured });
        None
    }

    fn create_end(
        &mut self,
        _ctx: &mut ScriptContext,
        _inputs: &CreateInputs,
        outcome: &mut CreateOutcome,
    ) {
        if let Some(frame) = self.frames.pop() {
            if let Some(cap) = frame.captured {
                #[allow(deprecated)]
                let gas_used = outcome.result.gas.spent();
                self.emit_broadcast(cap, gas_used);
            }
        }
    }
}

impl CheatInspector {
    fn emit_broadcast(&mut self, cap: Capture, gas_used: u64) {
        self.broadcasts.push(Broadcast {
            from: crate::allocs::fmt_addr(&cap.from),
            to: crate::allocs::fmt_addr(&cap.to),
            input: crate::allocs::fmt_code(&cap.input),
            value: crate::allocs::fmt_u256(&cap.value),
            salt: crate::allocs::fmt_hash(&cap.salt),
            gas_used,
            kind: cap.kind.as_str().to_string(),
            nonce: cap.nonce,
        });
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Byte-for-byte parity with the Go host's `encodeRevert` (op-chain-ops/script/precompile.go):
    // selector 0x08c379a0, 0x20 offset, length, right-padded UTF-8 payload.
    #[test]
    fn error_string_encoding_matches_go() {
        let out = encode_error_string("unrecognized 4 byte signature: deadbeef");
        assert_eq!(&out[0..4], &[0x08, 0xc3, 0x79, 0xa0]);
        assert_eq!(U256::from_be_slice(&out[4..36]), U256::from(0x20));
        let len = "unrecognized 4 byte signature: deadbeef".len();
        assert_eq!(U256::from_be_slice(&out[36..68]), U256::from(len));
        assert_eq!(&out[68..68 + len], "unrecognized 4 byte signature: deadbeef".as_bytes());
        // right-padded to a 32-byte boundary, zero padding.
        assert_eq!(out.len(), 4 + 32 + 32 + len.div_ceil(32) * 32);
        assert!(out[68 + len..].iter().all(|&b| b == 0));
    }

    #[test]
    fn hex_is_lowercase_unprefixed() {
        assert_eq!(hex(&[0xde, 0xad, 0xbe, 0xef]), "deadbeef");
        assert_eq!(hex(&[0x00, 0x0a]), "000a");
    }
}
