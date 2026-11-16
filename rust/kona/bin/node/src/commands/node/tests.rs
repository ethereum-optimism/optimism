use super::*;
use crate::flags::GlobalArgs;
use anyhow::anyhow;
use clap::Parser;
use kona_genesis::{HardForkConfig, RollupConfig};
use kona_node_service::NodeMode;
use kona_registry::{DEPENDENCY_SETS, ROLLUP_CONFIGS};

#[derive(Debug)]
struct MockError {
    message: String,
}

impl std::fmt::Display for MockError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.message)
    }
}

impl std::error::Error for MockError {}

const fn default_flags() -> &'static [&'static str] {
    &[
        "--l1-eth-rpc",
        "http://localhost:8545",
        "--l1-beacon",
        "http://localhost:5052",
        "--l2-engine-rpc",
        "http://localhost:8551",
    ]
}

#[test]
fn test_node_cli_defaults() {
    let args =
        NodeCommand::parse_from(std::iter::once(&"node").chain(default_flags().iter()).copied());
    assert_eq!(args.node_mode, NodeMode::Validator);
    assert_eq!(args.verifier_l1_confs, 0);
    assert_eq!(NodeCommand::default().verifier_l1_confs, 0);
}

#[test]
fn verifier_confirmation_depth_is_available_in_both_node_modes() {
    for mode in ["Validator", "Sequencer"] {
        let args = NodeCommand::parse_from(
            std::iter::once("node").chain(default_flags().iter().copied()).chain([
                "--mode",
                mode,
                "--verifier.l1-confs",
                "4",
            ]),
        );
        assert_eq!(args.verifier_l1_confs, 4);
    }
}

#[test]
fn verifier_confirmation_depth_has_the_expected_environment_variable() {
    use clap::CommandFactory;
    let command = NodeCommand::command();
    let arg = command.get_arguments().find(|arg| arg.get_id() == "verifier_l1_confs").unwrap();
    assert_eq!(arg.get_env().unwrap(), "KONA_NODE_VERIFIER_L1_CONFS");
}

#[test]
fn test_node_cli_missing_l1_eth_rpc() {
    let err = NodeCommand::try_parse_from(["node"]).unwrap_err();
    assert!(err.to_string().contains("--l1-eth-rpc"));
}

#[test]
fn test_node_cli_missing_l1_beacon() {
    let err =
        NodeCommand::try_parse_from(["node", "--l1-eth-rpc", "http://localhost:8545"]).unwrap_err();
    assert!(err.to_string().contains("--l1-beacon"));
}

#[test]
fn test_node_cli_missing_l2_engine_rpc() {
    let err = NodeCommand::try_parse_from([
        "node",
        "--l1-eth-rpc",
        "http://localhost:8545",
        "--l1-beacon",
        "http://localhost:5052",
    ])
    .unwrap_err();
    assert!(err.to_string().contains("--l2-engine-rpc"));
}

#[test]
fn test_get_l1_config_from_direct_and_genesis_files() {
    let documents = [
        serde_json::json!({"chainId": 123}),
        serde_json::json!({"config": {"chainId": 123}, "alloc": {}}),
    ];

    for document in documents {
        let mut file = tempfile::NamedTempFile::new().unwrap();
        serde_json::to_writer(&mut file, &document).unwrap();

        let command =
            NodeCommand { l1_config_file: Some(file.path().into()), ..Default::default() };
        assert_eq!(command.get_l1_config(123).unwrap().chain_id, 123);
    }
}

#[test]
fn test_is_jwt_signature_error() {
    let jwt_error = MockError { message: "signature invalid".to_string() };
    assert!(NodeCommand::is_jwt_signature_error(&jwt_error));

    let other_error = MockError { message: "network timeout".to_string() };
    assert!(!NodeCommand::is_jwt_signature_error(&other_error));
}

#[test]
fn test_is_jwt_signature_error_from_anyhow() {
    let jwt_anyhow_error = anyhow!("signature invalid");
    assert!(NodeCommand::is_jwt_signature_error_from_anyhow(&jwt_anyhow_error));

    let other_anyhow_error = anyhow!("network timeout");
    assert!(!NodeCommand::is_jwt_signature_error_from_anyhow(&other_anyhow_error));
}

#[test]
fn an_unconfigured_safe_database_is_absent() {
    assert!(NodeCommand::default().open_safe_db().unwrap().is_none());
}

#[test]
fn a_configured_safe_database_is_opened() {
    let directory = tempfile::TempDir::new().unwrap();
    let command =
        NodeCommand { safedb_path: Some(directory.path().to_path_buf()), ..Default::default() };

    assert!(command.open_safe_db().unwrap().is_some());
}

#[test]
fn a_safe_database_open_failure_reports_the_path() {
    let directory = tempfile::TempDir::new().unwrap();
    let command =
        NodeCommand { safedb_path: Some(directory.path().to_path_buf()), ..Default::default() };
    let _database = command.open_safe_db().unwrap().unwrap();

    let error = command.open_safe_db().unwrap_err();

    assert!(error.to_string().contains(&directory.path().display().to_string()));
}

#[test]
fn a_registry_chain_without_the_flag_gets_its_registry_dependency_set() {
    // OP Sepolia is in no interop cluster, so the registry gives it a dependency set holding only
    // itself, the same set op-node loads. Scheduling Lagoon makes the set required.
    const OP_SEPOLIA: u64 = 11155420;
    let registry_cfg = &ROLLUP_CONFIGS[&OP_SEPOLIA];
    let cfg = RollupConfig {
        hardforks: HardForkConfig { lagoon_time: Some(0), ..registry_cfg.hardforks },
        ..registry_cfg.clone()
    };

    let dependency_set = NodeCommand::default().load_dependency_set(&cfg).unwrap().unwrap();

    assert_eq!(*dependency_set, DEPENDENCY_SETS[&OP_SEPOLIA]);
    assert_eq!(dependency_set.dependencies.keys().copied().collect::<Vec<_>>(), [OP_SEPOLIA]);
}

#[test]
fn the_flag_overrides_the_registry_dependency_set() {
    const OP_SEPOLIA: u64 = 11155420;
    let file = tempfile::NamedTempFile::new().unwrap();
    std::fs::write(file.path(), r#"{"dependencies":{"11155420":{},"1301":{}}}"#).unwrap();
    let command =
        NodeCommand { interop_dependency_set: Some(file.path().into()), ..Default::default() };

    let dependency_set =
        command.load_dependency_set(&ROLLUP_CONFIGS[&OP_SEPOLIA]).unwrap().unwrap();

    assert_eq!(dependency_set.dependencies.keys().copied().collect::<Vec<_>>(), [1301, OP_SEPOLIA]);
}

#[test]
fn an_unknown_chain_with_lagoon_scheduled_needs_the_flag() {
    let cfg = RollupConfig {
        l2_chain_id: alloy_chains::Chain::from_id(999_999),
        hardforks: HardForkConfig { lagoon_time: Some(0), ..Default::default() },
        ..Default::default()
    };

    let err = NodeCommand::default().load_dependency_set(&cfg).unwrap_err();

    assert!(err.to_string().contains("chain 999999"), "{err}");
    assert!(err.to_string().contains("--interop.dependency-set"), "{err}");

    let cfg = RollupConfig { hardforks: HardForkConfig::default(), ..cfg };
    assert!(NodeCommand::default().load_dependency_set(&cfg).unwrap().is_none());
}

#[tokio::test]
async fn alt_da_chains_are_refused() {
    // Redstone uses keccak commitments, Celo Sepolia generic commitments.
    for chain_id in [690, 11142220] {
        let args = GlobalArgs {
            l2_chain_id: alloy_chains::Chain::from_id(chain_id),
            ..Default::default()
        };
        let err = NodeCommand::default().run(&args).await.unwrap_err();
        assert!(err.to_string().contains("alt-DA"), "chain {chain_id}: {err}");
    }
}
