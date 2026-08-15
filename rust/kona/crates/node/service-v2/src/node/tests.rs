use std::{fs, path::Path};

#[test]
fn v2_binary_cannot_alias_the_v1_service_runtime() {
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../bin/node-v2/Cargo.toml");
    let manifest = fs::read_to_string(manifest).expect("read kona-node-v2 manifest");
    assert!(manifest.contains("kona-node-service-v2 ="));
    assert!(!manifest.contains("kona-node-service ="));
    assert!(!manifest.contains("package = \"kona-node-service-v2\""));

    let cli = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../bin/node-v2/src/cli.rs");
    let cli = fs::read_to_string(cli).expect("read kona-node-v2 CLI");
    assert!(cli.contains("Commands::Node(node) => Self::run_to_completion"));
    assert!(
        !cli.contains("run_until_ctrl_c"),
        "the CLI must not cancel and drop the Node-owned shutdown future"
    );
}

#[test]
fn v2_source_cannot_reintroduce_the_actor_runtime() {
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let forbidden = [
        concat!("Node", "Actor"),
        concat!("Engine", "Actor"),
        concat!("Derivation", "Actor"),
        concat!("Sequencer", "Actor"),
        concat!("Engine", "Driver"),
        concat!("mod ", "actors"),
    ];
    let raw_engine_operations = [
        concat!("fork_choice_", "updated"),
        concat!("new_payload_", "v"),
        concat!("get_payload_", "v"),
        concat!("Build", "Task"),
        concat!("Insert", "Task"),
        concat!("Consolidate", "Task"),
        concat!("Finalize", "Task"),
        concat!("Synchronize", "Task"),
        concat!("Raw", "EngineClient"),
    ];

    fn inspect(directory: &Path, forbidden: &[&str], raw_engine_operations: &[&str]) {
        for entry in fs::read_dir(directory).expect("read V2 source directory") {
            let path = entry.expect("read V2 source entry").path();
            if path.is_dir() {
                assert_ne!(path.file_name().and_then(|name| name.to_str()), Some("actors"));
                inspect(&path, forbidden, raw_engine_operations);
            } else if path.extension().and_then(|extension| extension.to_str()) == Some("rs") {
                let source = fs::read_to_string(&path).expect("read V2 Rust source");
                for symbol in forbidden {
                    assert!(
                        !source.contains(symbol),
                        "{} reintroduced forbidden V1 symbol {symbol}",
                        path.display()
                    );
                }
                let belongs_to_engine = path.components().any(|part| part.as_os_str() == "engine");
                if !belongs_to_engine {
                    for operation in raw_engine_operations {
                        assert!(
                            !source.contains(operation),
                            "{} issues raw Engine API operation {operation} outside engine service",
                            path.display()
                        );
                    }
                }
            }
        }
    }

    inspect(&source, &forbidden, &raw_engine_operations);
}

#[test]
fn v2_preserves_the_three_domain_boundary_and_explicit_lifecycle() {
    let source = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let lib = fs::read_to_string(source.join("lib.rs")).expect("read V2 crate root");
    assert!(lib.contains("pub mod derivation;"));
    assert!(lib.contains("pub mod engine;"));
    assert!(lib.contains("pub mod rpc;"));
    assert!(!lib.contains("pub mod safe_chain;"));
    assert!(!lib.contains("pub mod unsafe_chain;"));

    for removed in [
        concat!("Cancellation", "Token"),
        concat!("Interop", "Mode"),
        concat!("Indexed", "Pipeline"),
        concat!("new_", "indexed"),
    ] {
        for entry in walk_rust_files(&source) {
            let contents = fs::read_to_string(&entry).expect("read V2 Rust source");
            assert!(
                !contents.contains(removed),
                "{} reintroduced removed lifecycle or pipeline scaffolding {removed}",
                entry.display()
            );
        }
    }

    let api = fs::read_to_string(source.join("engine/api.rs")).expect("read Engine API");
    let handle_impl = api
        .split("impl EngineHandle {")
        .nth(1)
        .and_then(|suffix| suffix.split("/// Read-only Engine capability").next())
        .expect("locate EngineHandle implementation");
    let public_async_methods = handle_impl.matches("pub async fn ").count();
    assert_eq!(public_async_methods, 2, "EngineHandle must expose exactly two async operations");
    assert!(handle_impl.contains("pub async fn update_safe"));
    assert!(handle_impl.contains("pub async fn update_finalized"));
}

fn walk_rust_files(directory: &Path) -> Vec<std::path::PathBuf> {
    let mut files = Vec::new();
    for entry in fs::read_dir(directory).expect("read source directory") {
        let path = entry.expect("read source entry").path();
        if path.is_dir() {
            files.extend(walk_rust_files(&path));
        } else if path.extension().and_then(|extension| extension.to_str()) == Some("rs") {
            files.push(path);
        }
    }
    files
}
