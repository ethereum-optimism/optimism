use std::{fs, path::Path};

#[test]
fn v2_binary_cannot_alias_the_v1_service_runtime() {
    let manifest = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../bin/node-v2/Cargo.toml");
    let manifest = fs::read_to_string(manifest).expect("read kona-node-v2 manifest");
    assert!(manifest.contains("kona-node-service-v2 ="));
    assert!(!manifest.contains("kona-node-service ="));
    assert!(!manifest.contains("package = \"kona-node-service-v2\""));
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
