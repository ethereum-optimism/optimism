//! The forge cheatcode + console inspector. This replaces op-geth's `PrecompileOverrides`
//! (cheatcode / console precompiles) and `CallerOverride` (prank / broadcast) hooks with a
//! single revm `Inspector`, mirroring the semantics of `op-chain-ops/script`.

use alloy_primitives::{Address, B256, Bytes, U256};
use revm::{
    Inspector,
    context_interface::{ContextTr, JournalTr},
    interpreter::{
        CallInputs, CallOutcome, CallScheme, CreateInputs, CreateOutcome, CreateScheme, Gas,
        InstructionResult, InterpreterResult,
    },
};
use serde::Serialize;

use crate::ScriptContext;
use crate::addresses::{CONSOLE_ADDR, CREATE2_DEPLOYER, VM_ADDR};

// Cheatcode selectors (from the Vm artifact methodIdentifiers).
const SEL_GET_NONCE: [u8; 4] = [0x2d, 0x03, 0x35, 0xab];
const SEL_BROADCAST0: [u8; 4] = [0xaf, 0xc9, 0x80, 0x40]; // broadcast()
const SEL_BROADCAST1: [u8; 4] = [0xe6, 0x96, 0x2c, 0xdb]; // broadcast(address)
const SEL_START_BROADCAST0: [u8; 4] = [0x7f, 0xb5, 0x29, 0x7f]; // startBroadcast()
const SEL_START_BROADCAST1: [u8; 4] = [0x7f, 0xec, 0x2a, 0x8d]; // startBroadcast(address)
const SEL_STOP_BROADCAST: [u8; 4] = [0x76, 0xea, 0xdd, 0x36]; // stopBroadcast()

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
    frames: Vec<Frame>,
}

impl CheatInspector {
    pub fn reset_call_state(&mut self) {
        self.frames.clear();
    }

    fn is_cheat_target(addr: &Address) -> bool {
        *addr == VM_ADDR || *addr == CONSOLE_ADDR
    }
}

fn read_nonce(ctx: &mut ScriptContext, addr: Address) -> u64 {
    ctx.journal_mut()
        .load_account(addr)
        .map(|a| a.data.info.nonce)
        .unwrap_or(0)
}

fn bump_nonce(ctx: &mut ScriptContext, addr: Address) {
    if let Ok(acc) = ctx.journal_mut().load_account_mut(addr) {
        acc.data.info.nonce += 1;
    }
    ctx.journal_mut().touch_account(addr);
}

fn cheat_outcome(output: Bytes, gas_limit: u64, memory_offset: std::ops::Range<usize>) -> CallOutcome {
    CallOutcome {
        result: InterpreterResult {
            result: InstructionResult::Return,
            output,
            gas: Gas::new(gas_limit), // 0 spent -> only the CALL opcode overhead is charged
        },
        memory_offset,
    }
}

fn addr_from_word(word: &[u8]) -> Address {
    // 32-byte ABI word, address in the low 20 bytes.
    Address::from_slice(&word[12..32])
}

fn abi_u64(v: u64) -> Bytes {
    let mut out = [0u8; 32];
    out[24..32].copy_from_slice(&v.to_be_bytes());
    Bytes::from(out.to_vec())
}

impl CheatInspector {
    /// Handle a call to the VM cheatcode precompile. Returns the ABI-encoded return data.
    fn dispatch_cheatcode(&mut self, ctx: &mut ScriptContext, data: &[u8]) -> Bytes {
        if data.len() < 4 {
            return Bytes::new();
        }
        let sel: [u8; 4] = data[0..4].try_into().unwrap();
        let args = &data[4..];
        match sel {
            SEL_GET_NONCE if args.len() >= 32 => {
                let addr = addr_from_word(&args[0..32]);
                abi_u64(read_nonce(ctx, addr))
            }
            SEL_BROADCAST0 => {
                self.set_prank(None, false, true);
                Bytes::new()
            }
            SEL_BROADCAST1 if args.len() >= 32 => {
                let who = addr_from_word(&args[0..32]);
                self.set_prank(Some(who), false, true);
                Bytes::new()
            }
            SEL_START_BROADCAST0 => {
                self.set_prank(None, true, true);
                Bytes::new()
            }
            SEL_START_BROADCAST1 if args.len() >= 32 => {
                let who = addr_from_word(&args[0..32]);
                self.set_prank(Some(who), true, true);
                Bytes::new()
            }
            SEL_STOP_BROADCAST => {
                if let Some(f) = self.frames.last_mut() {
                    f.prank = None;
                }
                Bytes::new()
            }
            _ => Bytes::new(),
        }
    }

    fn set_prank(&mut self, sender: Option<Address>, repeat: bool, broadcast: bool) {
        if let Some(f) = self.frames.last_mut() {
            f.prank = Some(Prank { sender, repeat, broadcast });
        }
    }
}

impl Inspector<ScriptContext> for CheatInspector {
    fn call(&mut self, ctx: &mut ScriptContext, inputs: &mut CallInputs) -> Option<CallOutcome> {
        let target = inputs.target_address;

        // console.log sink: swallow, 0 gas (matches Go console precompile RequiredGas=0).
        if target == CONSOLE_ADDR {
            return Some(cheat_outcome(Bytes::new(), inputs.gas_limit, inputs.return_memory_offset.clone()));
        }
        // cheatcode precompile.
        if target == VM_ADDR {
            let data = inputs.input.bytes(ctx);
            let out = self.dispatch_cheatcode(ctx, &data);
            return Some(cheat_outcome(out, inputs.gas_limit, inputs.return_memory_offset.clone()));
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

    fn call_end(&mut self, _ctx: &mut ScriptContext, inputs: &CallInputs, outcome: &mut CallOutcome) {
        if Self::is_cheat_target(&inputs.target_address) {
            return;
        }
        if let Some(frame) = self.frames.pop() {
            if let Some(cap) = frame.captured {
                self.emit_broadcast(cap, outcome.result.gas.spent());
            }
        }
    }

    fn create(&mut self, ctx: &mut ScriptContext, inputs: &mut CreateInputs) -> Option<CreateOutcome> {
        let mut captured = None;
        if let Some(parent) = self.frames.last_mut() {
            if let Some(prank) = parent.prank.clone() {
                let parent_addr = parent.target;
                if !prank.repeat {
                    parent.prank = None;
                }
                if prank.broadcast {
                    let (salt, kind) = match inputs.scheme {
                        CreateScheme::Create2 { salt } => (B256::from(salt), BroadcastKind::Create2),
                        _ => (B256::ZERO, BroadcastKind::Create),
                    };
                    match kind {
                        BroadcastKind::Create2 => {
                            // The prank sender's nonce is bumped (the "tx" nonce), but the
                            // creation is attributed to the deterministic deployer.
                            let sender = prank.sender.unwrap_or(parent_addr);
                            bump_nonce(ctx, sender);
                            let from = if self.use_create2_deployer {
                                CREATE2_DEPLOYER
                            } else {
                                sender
                            };
                            inputs.caller = from;
                            captured = Some(Capture {
                                from,
                                to: Address::ZERO,
                                input: inputs.init_code.clone(),
                                value: inputs.value,
                                salt,
                                nonce: 0,
                                kind: BroadcastKind::Create2,
                            });
                        }
                        _ => {
                            let from = prank.sender.unwrap_or(parent_addr);
                            inputs.caller = from;
                            let pre = read_nonce(ctx, from); // prestate; CREATE itself bumps it
                            captured = Some(Capture {
                                from,
                                to: Address::ZERO,
                                input: inputs.init_code.clone(),
                                value: inputs.value,
                                salt,
                                nonce: pre,
                                kind: BroadcastKind::Create,
                            });
                        }
                    }
                } else if let Some(sender) = prank.sender {
                    inputs.caller = sender;
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
            if let Some(mut cap) = frame.captured {
                cap.to = outcome.address.unwrap_or(Address::ZERO);
                self.emit_broadcast(cap, outcome.result.gas.spent());
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
