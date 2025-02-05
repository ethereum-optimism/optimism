use crate::integration::{poll_logs, Arg, IntegrationError, Service, ServiceCommand};
use futures_util::Future;
use std::{
    net::SocketAddr,
    path::{Path, PathBuf},
    time::Duration,
};

#[derive(Default)]
pub struct RollupBoostConfig {
    jwt_path: Option<PathBuf>,
    l2_addr: Option<SocketAddr>,
    builder_addr: Option<SocketAddr>,
}

impl RollupBoostConfig {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn jwt_path<P: Into<PathBuf>>(mut self, path: P) -> Self {
        self.jwt_path = Some(path.into());
        self
    }

    pub fn l2_addr(mut self, addr: SocketAddr) -> Self {
        self.l2_addr = Some(addr);
        self
    }

    pub fn builder_addr(mut self, addr: SocketAddr) -> Self {
        self.builder_addr = Some(addr);
        self
    }
}

impl Service for RollupBoostConfig {
    fn command(&self) -> ServiceCommand {
        let mut bin_path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        bin_path.push("./target/debug/rollup-boost");

        let jwt_path = self.jwt_path.as_ref().expect("jwt_path not set");

        let l2_addr = self.l2_addr.expect("l2 addr not set");
        let l2_ip = l2_addr.ip();
        let l2_port = l2_addr.port();

        let builder_addr = self.builder_addr.expect("builder addr not set");
        let builder_ip = builder_addr.ip();
        let builder_port = builder_addr.port();

        let cmd = ServiceCommand::new(bin_path.to_str().unwrap())
            // jwt auth rpc secrets
            .arg("--l2.auth.jwtsecret.path")
            .arg(jwt_path.clone())
            .arg("--builder.auth.jwtsecret.path")
            .arg(jwt_path.clone())
            // builder and l2 address
            .arg("--l2.auth.addr")
            .arg(l2_ip.to_string())
            .arg("--l2.auth.port")
            .arg(l2_ip.to_string())
            .arg("--builder-url")
            .arg(self.builder_url.as_ref().expect("builder_url not set"))
            .arg("--rpc-port")
            .arg(Arg::Port {
                name: "rpc".into(),
                preferred: 8112,
            })
            .arg("--boost-sync");

        cmd
    }

    #[allow(clippy::manual_async_fn)]
    fn ready(&self, log_path: &Path) -> impl Future<Output = Result<(), IntegrationError>> + Send {
        async move {
            poll_logs(
                log_path,
                "Starting server on",
                Duration::from_millis(100),
                Duration::from_secs(60),
            )
            .await
        }
    }
}
