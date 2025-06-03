use std::{fs::File, time::Duration};

use crate::Args;
use clap::Parser;
use tokio::task::JoinHandle;
use tracing::subscriber::DefaultGuard;
use tracing_subscriber::fmt;

use crate::tests::common::{TEST_DATA, get_available_port};

#[derive(Debug)]
pub struct RollupBoost {
    args: Args,
    pub _handle: JoinHandle<eyre::Result<()>>,
    pub _tracing_guard: DefaultGuard,
}

impl RollupBoost {
    pub fn args(&self) -> &Args {
        &self.args
    }

    pub fn rpc_endpoint(&self) -> String {
        format!("http://localhost:{}", self.args.rpc_port)
    }

    pub fn metrics_endpoint(&self) -> String {
        format!("http://localhost:{}", self.args.metrics_port)
    }

    pub fn debug_endpoint(&self) -> String {
        format!("http://localhost:{}", self.args.debug_server_port)
    }

    pub async fn get_metrics(&self) -> eyre::Result<String> {
        let response = reqwest::get(self.metrics_endpoint() + "/metrics").await?;
        let body = response.text().await?;
        Ok(body)
    }
}

#[derive(Clone, Debug)]
pub struct RollupBoostConfig {
    pub args: Args,
}

impl Default for RollupBoostConfig {
    fn default() -> Self {
        let mut args = Args::parse_from([
            "rollup-boost",
            &format!("--l2-jwt-path={}/jwt_secret.hex", *TEST_DATA),
            &format!("--builder-jwt-path={}/jwt_secret.hex", *TEST_DATA),
            "--log-level=trace",
        ]);

        args.rpc_port = get_available_port();
        args.metrics_port = get_available_port();
        args.debug_server_port = get_available_port();

        Self { args }
    }
}

impl RollupBoostConfig {
    pub async fn start(self) -> RollupBoost {
        let args = self.args.clone();

        // Create a custom log subscriber only for this task
        let log_file = args.log_file.as_ref().unwrap();
        let file = File::create(log_file).unwrap();

        let subscriber = fmt::Subscriber::builder()
            .with_writer(file)
            .with_max_level(tracing::Level::DEBUG)
            .with_ansi(false)
            .finish();

        let guard = tracing::subscriber::set_default(subscriber);

        let _handle = tokio::spawn(async move {
            let res = args.clone().run().await;
            if let Err(e) = &res {
                eprintln!("Error: {:?}", e);
            }
            res
        });

        // Allow some time for the app to startup
        tokio::time::sleep(Duration::from_secs(4)).await;

        RollupBoost {
            args: self.args,
            _handle,
            _tracing_guard: guard,
        }
    }
}
