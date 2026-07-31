use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::B256;
use reth_chainspec::MAINNET;
use reth_ethereum_primitives::Block;
use reth_evm_ethereum::EthEvmConfig;
use reth_optimism_trie::{
    MdbxProofsStorageV2, OpProofStoragePruner, OpProofsInitProvider, OpProofsProviderRO,
    OpProofsStore, engine::EngineHandle,
};
use reth_provider::test_utils::MockEthProvider;
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use std::{
    env, fs,
    path::Path,
    process::{Child, Command, Stdio},
    sync::Arc,
    thread,
    time::{Duration, Instant},
};
use tempfile::TempDir;

const CHILD_MODE_ENV: &str = "OP_RETH_UNCLEAN_SHUTDOWN_CHILD_MODE";
const CHILD_PROOFS_PATH_ENV: &str = "OP_RETH_UNCLEAN_SHUTDOWN_PROOFS_PATH";
const CHILD_SIGNAL_PATH_ENV: &str = "OP_RETH_UNCLEAN_SHUTDOWN_SIGNAL_PATH";
const BUFFER_MODE: &str = "buffer";
const QUERY_MODE: &str = "query";

fn genesis() -> BlockNumHash {
    BlockNumHash::new(0, MAINNET.genesis_hash())
}

const fn tail() -> NumHash {
    NumHash::new(1, B256::repeat_byte(0x01))
}

fn spawn_engine(storage: Arc<MdbxProofsStorageV2>) -> EngineHandle<Block> {
    let provider = MockEthProvider::default();
    let pruner = OpProofStoragePruner::new(storage.clone(), provider.clone(), 1_000);
    EngineHandle::<Block>::spawn(EthEvmConfig::ethereum(MAINNET.clone()), provider, storage, pruner)
}

fn run_buffer_child(proofs_path: &Path, ready_path: &Path) -> eyre::Result<()> {
    let storage = Arc::new(MdbxProofsStorageV2::new(proofs_path)?);
    let init = storage.initialization_provider()?;
    init.set_initial_state_anchor(genesis())?;
    init.commit_initial_state()?;
    OpProofsInitProvider::commit(init)?;

    let engine = spawn_engine(storage.clone());
    engine.index_block(
        BlockWithParent::new(genesis().hash, tail()),
        TrieUpdatesSorted::default(),
        HashedPostStateSorted::default(),
    )?;

    // Confirm the one-block tail remains below the persistence threshold.
    assert_eq!(storage.provider_ro()?.get_latest_block()?, genesis());
    fs::write(ready_path, b"ready")?;

    // Keep the engine alive until the parent kills this process without running destructors.
    loop {
        thread::park();
    }
}

fn run_query_child(proofs_path: &Path, result_path: &Path) -> eyre::Result<()> {
    let storage = Arc::new(MdbxProofsStorageV2::new(proofs_path)?);
    let _engine = spawn_engine(storage.clone());

    // This is the same proof-window query used by debug_proofsSyncStatus.
    let latest = storage.provider_ro()?.get_proof_window()?.latest.number;
    fs::write(result_path, latest.to_string())?;
    Ok(())
}

fn spawn_child(mode: &str, proofs_path: &Path, signal_path: &Path) -> eyre::Result<Child> {
    Ok(Command::new(env::current_exe()?)
        .args(["--exact", "shutdown::buffered_tail_is_persisted_after_unclean_shutdown"])
        .arg("--nocapture")
        .env(CHILD_MODE_ENV, mode)
        .env(CHILD_PROOFS_PATH_ENV, proofs_path)
        .env(CHILD_SIGNAL_PATH_ENV, signal_path)
        .stdout(Stdio::null())
        .spawn()?)
}

fn wait_for_child_ready(child: &mut Child, ready_path: &Path) -> eyre::Result<()> {
    let deadline = Instant::now() + Duration::from_secs(30);
    loop {
        if ready_path.exists() {
            return Ok(());
        }
        if let Some(status) = child.try_wait()? {
            eyre::bail!("child exited before buffering the proof tail: {status}");
        }
        if Instant::now() >= deadline {
            let _ = child.kill();
            let _ = child.wait();
            eyre::bail!("timed out waiting for child to buffer the proof tail");
        }
        thread::sleep(Duration::from_millis(10));
    }
}

#[test]
fn buffered_tail_is_persisted_after_unclean_shutdown() -> eyre::Result<()> {
    match (
        env::var_os(CHILD_MODE_ENV),
        env::var_os(CHILD_PROOFS_PATH_ENV),
        env::var_os(CHILD_SIGNAL_PATH_ENV),
    ) {
        (Some(mode), Some(proofs_path), Some(signal_path)) => {
            let proofs_path = Path::new(&proofs_path);
            let signal_path = Path::new(&signal_path);
            return match mode.to_str() {
                Some(BUFFER_MODE) => run_buffer_child(proofs_path, signal_path),
                Some(QUERY_MODE) => run_query_child(proofs_path, signal_path),
                _ => Err(eyre::eyre!("unknown unclean-shutdown child mode: {mode:?}")),
            };
        }
        (None, None, None) => {}
        _ => return Err(eyre::eyre!("unclean-shutdown child environment is incomplete")),
    }

    let temp_dir = TempDir::new()?;
    let proofs_path = temp_dir.path().join("proofs");
    let ready_path = temp_dir.path().join("ready");
    let mut child = spawn_child(BUFFER_MODE, &proofs_path, &ready_path)?;

    wait_for_child_ready(&mut child, &ready_path)?;
    child.kill()?;
    let status = child.wait()?;
    assert!(!status.success(), "uncleanly terminated child unexpectedly succeeded");

    // Restart the collector against the same proofs DB and query it from the restarted process.
    let result_path = temp_dir.path().join("latest");
    let mut restarted_child = spawn_child(QUERY_MODE, &proofs_path, &result_path)?;
    let status = restarted_child.wait()?;
    assert!(status.success(), "restarted child failed: {status}");
    let latest = fs::read_to_string(result_path)?.parse::<u64>()?;

    assert_eq!(latest, tail().number);
    Ok(())
}
