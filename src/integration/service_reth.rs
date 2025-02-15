use crate::integration::{Arg, ReadyParams, Service, ServiceCommand};
use std::{path::PathBuf, time::Duration};

#[derive(Default)]
pub struct RethConfig {
    jwt_secret_path: Option<PathBuf>,
    chain_config_path: Option<PathBuf>,
    p2p_secret_key: Option<String>,
    trusted_peer: Option<String>,
}

impl RethConfig {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn jwt_secret_path<P: Into<PathBuf>>(mut self, path: P) -> Self {
        self.jwt_secret_path = Some(path.into());
        self
    }

    pub fn chain_config_path<P: Into<PathBuf>>(mut self, path: P) -> Self {
        self.chain_config_path = Some(path.into());
        self
    }

    pub fn p2p_secret_key(mut self, key: String) -> Self {
        self.p2p_secret_key = Some(key);
        self
    }

    pub fn trusted_peer(mut self, trusted_peer: String) -> Self {
        self.trusted_peer = Some(trusted_peer);
        self
    }
}

impl Service for RethConfig {
    fn command(&self) -> ServiceCommand {
        let mut cmd = ServiceCommand::new("op-reth")
            .arg("node")
            .arg("--authrpc.port")
            .arg(Arg::Port {
                name: "authrpc".into(),
                preferred: 8551,
            })
            .arg("--authrpc.jwtsecret")
            .arg(
                self.jwt_secret_path
                    .as_ref()
                    .expect("jwt_secret_path not set"),
            )
            .arg("--chain")
            .arg(
                self.chain_config_path
                    .as_ref()
                    .expect("chain_config_path not set"),
            )
            .arg("--datadir")
            .arg(Arg::Dir {
                name: "data".into(),
            })
            .arg("--disable-discovery")
            .arg("--port")
            .arg(Arg::Port {
                name: "p2p".into(),
                preferred: 30303, // We do not use this port but it cannot be disabled
            })
            .arg("--color")
            .arg("never")
            .arg("--ipcdisable");

        if let Some(p2p_secret_key) = &self.p2p_secret_key {
            cmd = cmd.arg("--p2p-secret-key").arg(Arg::FilePath {
                name: "p2p_secret_key".into(),
                content: p2p_secret_key.clone(),
            });
        }

        if let Some(trusted_peer) = &self.trusted_peer {
            cmd = cmd.arg("--trusted-peers").arg(trusted_peer);
        }

        cmd
    }

    fn ready(&self) -> ReadyParams {
        ReadyParams {
            log_pattern: "Starting consensus".to_string(),
            duration: Duration::from_secs(5),
        }
    }
}
