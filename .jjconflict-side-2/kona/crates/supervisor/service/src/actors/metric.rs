use async_trait::async_trait;
use kona_supervisor_metrics::MetricsReporter;
use std::{io, sync::Arc, time::Duration};
use tokio::time::sleep;
use tokio_util::sync::CancellationToken;
use tracing::info;

use crate::SupervisorActor;

#[derive(derive_more::Constructor)]
pub struct MetricWorker<R> {
    interval: Duration,
    // list of reporters
    reporters: Vec<Arc<R>>,
    cancel_token: CancellationToken,
}

#[async_trait]
impl<R> SupervisorActor for MetricWorker<R>
where
    R: MetricsReporter + Send + Sync + 'static,
{
    type InboundEvent = ();
    type Error = io::Error;

    async fn start(mut self) -> Result<(), Self::Error> {
        info!(
            target: "supervisor::metric_worker",
            "Starting MetricWorker with interval: {:?}",
            self.interval
        );

        let reporters = self.reporters;
        let interval = self.interval;

        loop {
            if self.cancel_token.is_cancelled() {
                info!("MetricReporter actor is stopping due to cancellation.");
                break;
            }

            for reporter in &reporters {
                reporter.report_metrics();
            }
            sleep(interval).await;
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use mockall::{mock, predicate::*};
    use std::{sync::Arc, time::Duration};
    use tokio_util::sync::CancellationToken;

    mock! (
        #[derive(Debug)]
        pub Reporter {}

        impl MetricsReporter for Reporter {
            fn report_metrics(&self);
        }
    );

    #[tokio::test]
    async fn test_metric_worker_reports_metrics_and_stops_on_cancel() {
        let mut mock_reporter = MockReporter::new();
        mock_reporter.expect_report_metrics().return_const(());

        let reporter = Arc::new(mock_reporter);
        let cancel_token = CancellationToken::new();

        let worker = MetricWorker::new(
            Duration::from_millis(50),
            vec![reporter.clone()],
            cancel_token.clone(),
        );

        let handle = tokio::spawn(worker.start());

        tokio::time::sleep(Duration::from_millis(120)).await;
        cancel_token.cancel();

        let _ = handle.await;
    }

    #[tokio::test]
    async fn test_metric_worker_stops_immediately_on_cancel() {
        let mut mock_reporter = MockReporter::new();
        mock_reporter.expect_report_metrics().times(0);

        let reporter = Arc::new(mock_reporter);
        let cancel_token = CancellationToken::new();

        let worker = MetricWorker::new(
            Duration::from_millis(100),
            vec![reporter.clone()],
            cancel_token.clone(),
        );

        cancel_token.cancel();

        let _ = worker.start().await;
    }
}
