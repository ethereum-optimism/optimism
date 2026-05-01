//! [`DatabaseMetrics`](reth_db::database_metrics::DatabaseMetrics) implementation for
//! [`MdbxProofsStorageV2`].
//!
//! Reports per-table size, page counts, and entry counts.

use super::{MdbxProofsStorageV2, Tables};
use eyre::WrapErr;
use metrics::{Label, gauge};
use reth_db::Database;
use tracing::error;

impl reth_db::database_metrics::DatabaseMetrics for MdbxProofsStorageV2 {
    fn report_metrics(&self) {
        for (name, value, labels) in self.gauge_metrics() {
            gauge!(name, labels).set(value);
        }
    }

    fn gauge_metrics(&self) -> Vec<(&'static str, f64, Vec<Label>)> {
        let mut metrics = Vec::new();

        let _ = self
            .env
            .view(|tx| {
                for table in Tables::ALL.iter().map(Tables::name) {
                    let table_db =
                        tx.inner().open_db(Some(table)).wrap_err("Could not open db.")?;

                    let stats = tx
                        .inner()
                        .db_stat(table_db.dbi())
                        .wrap_err(format!("Could not find table: {table}"))?;

                    let page_size = stats.page_size() as usize;
                    let leaf_pages = stats.leaf_pages();
                    let branch_pages = stats.branch_pages();
                    let overflow_pages = stats.overflow_pages();
                    let num_pages = leaf_pages + branch_pages + overflow_pages;
                    let table_size = page_size * num_pages;
                    let entries = stats.entries();

                    metrics.push((
                        "optimism_proof_storage.table_size",
                        table_size as f64,
                        vec![Label::new("table", table)],
                    ));
                    metrics.push((
                        "optimism_proof_storage.table_pages",
                        leaf_pages as f64,
                        vec![Label::new("table", table), Label::new("type", "leaf")],
                    ));
                    metrics.push((
                        "optimism_proof_storage.table_pages",
                        branch_pages as f64,
                        vec![Label::new("table", table), Label::new("type", "branch")],
                    ));
                    metrics.push((
                        "optimism_proof_storage.table_pages",
                        overflow_pages as f64,
                        vec![Label::new("table", table), Label::new("type", "overflow")],
                    ));
                    metrics.push((
                        "optimism_proof_storage.table_entries",
                        entries as f64,
                        vec![Label::new("table", table)],
                    ));
                }

                Ok::<(), eyre::Report>(())
            })
            .map_err(|error| error!(%error, "Failed to read db table stats"));

        if let Ok(freelist) =
            self.env.freelist().map_err(|error| error!(%error, "Failed to read db.freelist"))
        {
            metrics.push(("optimism_proof_storage.freelist", freelist as f64, vec![]));
        }

        if let Ok(stat) = self.env.stat().map_err(|error| error!(%error, "Failed to read db.stat"))
        {
            metrics.push(("optimism_proof_storage.page_size", stat.page_size() as f64, vec![]));
        }

        metrics.push((
            "optimism_proof_storage.timed_out_not_aborted_transactions",
            self.env.timed_out_not_aborted_transactions() as f64,
            vec![],
        ));

        metrics
    }
}
