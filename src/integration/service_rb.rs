use crate::integration::{Arg, ReadyParams, Service, ServiceCommand};
use std::{path::PathBuf, time::Duration};

#[derive(Default)]
pub struct RollupBoostConfig {
    jwt_path: Option<PathBuf>,
    l2_url: Option<String>,
    builder_url: Option<String>,
}

impl RollupBoostConfig {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn jwt_path<P: Into<PathBuf>>(mut self, path: P) -> Self {
        self.jwt_path = Some(path.into());
        self
    }

    pub fn l2_url(mut self, url: String) -> Self {
        self.l2_url = Some(url);
        self
    }

    pub fn builder_url(mut self, url: String) -> Self {
        self.builder_url = Some(url);
        self
    }
}

impl Service for RollupBoostConfig {
    fn command(&self) -> ServiceCommand {
        let mut bin_path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        bin_path.push("./target/debug/rollup-boost");

        let jwt_path = self.jwt_path.as_ref().expect("jwt_path not set");

        let cmd = ServiceCommand::new(bin_path.to_str().unwrap())
            .arg("--l2-jwt-path")
            .arg(jwt_path.clone())
            .arg("--builder-jwt-path")
            .arg(jwt_path.clone())
            .arg("--l2-url")
            .arg(self.l2_url.as_ref().expect("l2_url not set"))
            .arg("--builder-url")
            .arg(self.builder_url.as_ref().expect("builder_url not set"))
            .arg("--rpc-port")
            .arg(Arg::Port {
                name: "rpc".into(),
                preferred: 8112,
            })
            .arg("--debug-server-port")
            .arg(Arg::Port {
                name: "debug".into(),
                preferred: 5555,
            });

        cmd
    }

    fn ready(&self) -> ReadyParams {
        ReadyParams {
            log_pattern: "Starting server on".to_string(),
            duration: Duration::from_secs(5),
        }
    }
}
