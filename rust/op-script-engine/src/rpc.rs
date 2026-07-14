//! jsonrpsee `script` namespace over a Unix socket (served by reth-ipc), go-ethereum
//! `rpc.DialIPC`-compatible (the #20415 transport).
//!
//! revm's `Context` is `!Send` (it holds an `Rc<RefCell<Vec<u8>>>` shared-memory buffer), so the
//! `ScriptHost` is pinned to a dedicated worker thread — constructed there, never crossing a
//! thread boundary — and driven through a channel. The RPC context is just the (Send+Sync)
//! channel handle, and every jsonrpsee method is a synchronous request/response round-trip.

use std::str::FromStr;

use alloy_primitives::{Address, Bytes, U256};
use jsonrpsee::{RpcModule, types::ErrorObjectOwned};
use tokio::sync::mpsc::UnboundedSender;

use crate::host::{HostConfig, ScriptHost};

type JobResult = Result<serde_json::Value, String>;
type Job = Box<dyn FnOnce(&mut ScriptHost) -> JobResult + Send>;
type Reply = std::sync::mpsc::Sender<JobResult>;

/// Handle to the worker thread that owns the `ScriptHost`.
///
/// The revm `Context` is `!Send`, so the host is pinned to a dedicated worker thread and driven
/// through this `Send + Sync + Clone` channel handle — the jsonrpsee module's context.
#[derive(Debug, Clone)]
pub struct Engine {
    tx: UnboundedSender<(Job, Reply)>,
}

impl Engine {
    /// Spawns the worker thread and builds the host inside it (the host never crosses threads).
    pub fn spawn(config: HostConfig) -> Self {
        let (tx, mut rx) = tokio::sync::mpsc::unbounded_channel::<(Job, Reply)>();
        std::thread::spawn(move || {
            let mut host = ScriptHost::new(config);
            while let Some((job, reply)) = rx.blocking_recv() {
                // A panic inside a job (e.g. a revm invariant trip) must not tear down the worker
                // thread — that would close the reply channel and hang every subsequent request for
                // the engine's lifetime. Contain it and surface it as an RPC error instead.
                let result =
                    std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| job(&mut host)))
                        .unwrap_or_else(|_| {
                            Err("engine panicked while handling request".to_string())
                        });
                let _ = reply.send(result);
            }
        });
        Self { tx }
    }

    fn run<F>(&self, f: F) -> Result<serde_json::Value, ErrorObjectOwned>
    where
        F: FnOnce(&mut ScriptHost) -> JobResult + Send + 'static,
    {
        let (rtx, rrx) = std::sync::mpsc::channel();
        self.tx.send((Box::new(f), rtx)).map_err(|_| err("engine worker gone"))?;
        rrx.recv().map_err(|_| err("engine worker dropped reply"))?.map_err(err)
    }
}

fn err(msg: impl std::fmt::Display) -> ErrorObjectOwned {
    ErrorObjectOwned::owned(-32000, msg.to_string(), None::<()>)
}

fn addr(s: &str) -> Result<Address, String> {
    Address::from_str(s).map_err(|e| format!("bad address {s:?}: {e}"))
}

fn bytes(s: &str) -> Result<Bytes, String> {
    let s = s.strip_prefix("0x").unwrap_or(s);
    alloy_primitives::hex::decode(s).map(Bytes::from).map_err(|e| format!("bad bytes: {e}"))
}

fn u256(s: &str) -> Result<U256, String> {
    U256::from_str(s).map_err(|e| format!("bad u256 {s:?}: {e}"))
}

/// Parse a 4-byte function selector from hex (with or without `0x`).
fn sel4(s: &str) -> Result<[u8; 4], String> {
    let b = bytes(s)?;
    if b.len() != 4 {
        return Err(format!("selector must be 4 bytes, got {}", b.len()));
    }
    Ok([b[0], b[1], b[2], b[3]])
}

fn hexstr(b: &[u8]) -> serde_json::Value {
    serde_json::Value::String(format!("0x{}", alloy_primitives::hex::encode(b)))
}

/// Registers every `script_*` JSON-RPC method on a jsonrpsee module backed by `engine`.
pub fn build_module(engine: Engine) -> RpcModule<Engine> {
    let mut m = RpcModule::new(engine);

    m.register_method("script_loadContract", |params, ctx, _| {
        let (file, contract, from): (String, String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            let a = h.load_contract(&file, &contract, addr(&from)?).map_err(|e| e.to_string())?;
            Ok(serde_json::Value::String(format!("0x{:x}", a)))
        })
    })
    .unwrap();

    m.register_method("script_create", |params, ctx, _| {
        let (from, code): (String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            let a = h.create(addr(&from)?, bytes(&code)?).map_err(|e| e.to_string())?;
            Ok(serde_json::Value::String(format!("0x{:x}", a)))
        })
    })
    .unwrap();

    m.register_method("script_call", |params, ctx, _| {
        let (from, to, input): (String, String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            let out =
                h.call(addr(&from)?, addr(&to)?, bytes(&input)?).map_err(|e| e.to_string())?;
            Ok(hexstr(&out))
        })
    })
    .unwrap();

    m.register_method("script_runScript", |params, ctx, _| {
        let (file, contract, calldata, deployer): (String, String, String, String) =
            params.parse().map_err(err)?;
        ctx.run(move |h| {
            let out = h
                .run_script(&file, &contract, bytes(&calldata)?, addr(&deployer)?)
                .map_err(|e| e.to_string())?;
            Ok(hexstr(&out))
        })
    })
    .unwrap();

    // --- OPCM RunScript* path: input/output precompiles (design §4) ---

    m.register_method("script_installInputPrecompile", |params, ctx, _| {
        let snapshot: std::collections::HashMap<String, String> = params.one().map_err(err)?;
        ctx.run(move |h| {
            let mut map = alloy_primitives::map::HashMap::default();
            for (sel, data) in &snapshot {
                map.insert(sel4(sel)?, bytes(data)?);
            }
            let addr = h.install_input_precompile(map);
            Ok(serde_json::Value::String(format!("0x{:x}", addr)))
        })
    })
    .unwrap();

    m.register_method("script_installOutputPrecompile", |params, ctx, _| {
        let getters: Vec<String> = params.one().map_err(err)?;
        ctx.run(move |h| {
            let mut set = alloy_primitives::map::HashSet::default();
            for g in &getters {
                set.insert(sel4(g)?);
            }
            let addr = h.install_output_precompile(set);
            Ok(serde_json::Value::String(format!("0x{:x}", addr)))
        })
    })
    .unwrap();

    m.register_method("script_takeCapturedSets", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| {
            let sets = h.take_captured_sets(addr(&a)?);
            let out: Vec<serde_json::Value> = sets.iter().map(|b| hexstr(b)).collect();
            Ok(serde_json::Value::Array(out))
        })
    })
    .unwrap();

    m.register_method("script_removePrecompile", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| {
            h.remove_precompile(addr(&a)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_wipe", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| {
            h.wipe(addr(&a)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_importState", |params, ctx, _| {
        let allocs: crate::allocs::ForgeAllocs = params.one().map_err(err)?;
        ctx.run(move |h| {
            h.import_state(allocs).map_err(|e| e.to_string())?;
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_getCode", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| Ok(hexstr(&h.get_code(addr(&a)?))))
    })
    .unwrap();

    m.register_method("script_allowCheatcodes", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| {
            h.allow_cheatcodes(addr(&a)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_setEnv", |params, ctx, _| {
        let (k, v): (String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            h.set_env(k, v);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_getNonce", |params, ctx, _| {
        let a: String = params.one().map_err(err)?;
        ctx.run(move |h| Ok(serde_json::Value::from(h.get_nonce(addr(&a)?))))
    })
    .unwrap();

    m.register_method("script_setBalance", |params, ctx, _| {
        let (a, bal): (String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            h.set_balance(addr(&a)?, u256(&bal)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_setNonce", |params, ctx, _| {
        let (a, nonce): (String, u64) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            h.set_nonce(addr(&a)?, nonce);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_setStorage", |params, ctx, _| {
        let (a, key, val): (String, String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            h.set_storage(addr(&a)?, u256(&key)?, u256(&val)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_setCode", |params, ctx, _| {
        let (a, code): (String, String) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            h.set_code(addr(&a)?, bytes(&code)?);
            Ok(serde_json::Value::Bool(true))
        })
    })
    .unwrap();

    m.register_method("script_stateDump", |_params, ctx, _| {
        ctx.run(|h| {
            // Mirror script.go:801-803: a state dump while a fork is active is an error (the forked
            // callers extract broadcasts, never allocs).
            if h.is_forked() {
                return Err("cannot state-dump while a fork is active".to_string());
            }
            serde_json::to_value(h.state_dump()).map_err(|e| e.to_string())
        })
    })
    .unwrap();

    // --- Fork mode (design §1): install an RPC-backed base state + export the overlay diff ---

    m.register_method("script_createSelectFork", |params, ctx, _| {
        // (url, blockNumber?) — blockNumber null/absent means "latest" (mirrors
        // DefaultForkedScriptHost's HeaderByNumber(nil)).
        let (url, block): (String, Option<u64>) = params.parse().map_err(err)?;
        ctx.run(move |h| {
            let meta = h.create_select_fork(&url, block).map_err(|e| e.to_string())?;
            serde_json::to_value(meta).map_err(|e| e.to_string())
        })
    })
    .unwrap();

    m.register_method("script_forkDiff", |_params, ctx, _| {
        ctx.run(|h| h.fork_diff().map_err(|e| e.to_string()))
    })
    .unwrap();

    m.register_method("script_takeBroadcasts", |_params, ctx, _| {
        ctx.run(|h| serde_json::to_value(h.take_broadcasts()).map_err(|e| e.to_string()))
    })
    .unwrap();

    m
}
