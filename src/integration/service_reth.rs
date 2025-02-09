use crate::integration::{Arg, IntegrationError, Service, ServiceCommand, ServiceInstance};
use futures_util::Future;
use std::{path::PathBuf, time::Duration};

#[derive(Default)]
pub struct RethConfig {
    jwt_secret_path: Option<PathBuf>,
    chain_config_path: Option<PathBuf>,
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
}

impl Service for RethConfig {
    fn command(&self) -> ServiceCommand {
        ServiceCommand::new("op-reth")
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
            .arg("--ipcdisable")
    }

    fn ready(
        &self,
        service: &mut ServiceInstance,
    ) -> impl Future<Output = Result<(), IntegrationError>> + Send {
        async move { service.wait_for_log("Starting consensus", Duration::from_secs(5)) }
    }
}
