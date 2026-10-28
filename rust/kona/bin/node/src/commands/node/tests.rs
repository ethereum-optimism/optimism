use super::*;
use anyhow::anyhow;
use clap::Parser;
use kona_genesis::RollupConfig;
use kona_node_service::NodeMode;

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
fn a_safe_database_is_rejected_where_the_node_derives_nothing() {
    let public = RollupConfig { l2_chain_id: 901.into(), ..Default::default() };
    let private = RollupConfig { settlement_chain_id: 1337, ..public.clone() };
    let with_safe_db = NodeCommand { safedb_path: Some("safedb".into()), ..Default::default() };
    let mut following = with_safe_db.clone();
    following.derivation_delegate_args.l2_follow_source =
        Some("http://localhost:9545".parse().unwrap());

    assert!(with_safe_db.check_safe_db_supported(&public).is_ok());
    assert!(NodeCommand::default().check_safe_db_supported(&private).is_ok());
    let error = following.check_safe_db_supported(&public).unwrap_err().to_string();
    assert!(error.contains("derivation delegation"), "{error}");
    let error = with_safe_db.check_safe_db_supported(&private).unwrap_err().to_string();
    assert!(error.contains("OP Private chains"), "{error}");
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
