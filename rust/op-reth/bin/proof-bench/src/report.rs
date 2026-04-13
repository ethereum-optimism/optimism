use crate::rpc::Sample;
use hdrhistogram::Histogram;
use std::io::{self, Write};

// --- Per-Block Metrics ---

pub struct BenchMetrics {
    pub block: u64,
    pub p95_ms: f64,
    pub min_ms: f64,
    pub max_ms: f64,
    pub errors: usize,
    pub throughput: f64,
}

impl BenchMetrics {
    pub fn new(block: u64, samples: &[Sample], duration_secs: f64) -> Self {
        if samples.is_empty() {
            return Self::empty(block);
        }

        // 1. Prepare data
        let mut latencies: Vec<f64> = samples.iter().map(|s| s.latency_ms).collect();
        // Sorting is efficient enough for small N (batch size) and gives exact precision
        latencies.sort_by(|a, b| a.partial_cmp(b).unwrap());

        let errors = samples.iter().filter(|s| !s.success).count();

        // 2. Calculate Stats
        let min_ms = *latencies.first().unwrap_or(&0.0);
        let max_ms = *latencies.last().unwrap_or(&0.0);
        let p95_ms = calculate_percentile(&latencies, 0.95);

        let throughput =
            if duration_secs > 0.0 { samples.len() as f64 / duration_secs } else { 0.0 };

        Self { block, p95_ms, min_ms, max_ms, errors, throughput }
    }

    fn empty(block: u64) -> Self {
        Self { block, p95_ms: 0.0, min_ms: 0.0, max_ms: 0.0, errors: 0, throughput: 0.0 }
    }
}

// --- Global Accumulator ---

pub struct BenchSummary {
    pub hist: Histogram<u64>,
    pub total_errors: usize,
    pub total_requests: usize,
    pub min_ms: f64,
    pub max_ms: f64,
}

impl BenchSummary {
    pub fn new() -> Self {
        Self {
            hist: Histogram::<u64>::new_with_bounds(1, 3_600_000, 3).unwrap(),
            total_errors: 0,
            total_requests: 0,
            min_ms: f64::MAX,
            max_ms: 0.0,
        }
    }

    pub fn add(&mut self, sample: &Sample) {
        self.total_requests += 1;

        if !sample.success {
            self.total_errors += 1;
        }

        let lat = sample.latency_ms;

        if lat < self.min_ms {
            self.min_ms = lat;
        }
        if lat > self.max_ms {
            self.max_ms = lat;
        }

        // Update Histogram (saturating cast to avoid crashes on bad data)
        let val = (lat as u64).max(1);
        self.hist.record(val).ok();
    }
}

// --- Output Handling ---

pub struct Reporter;

impl Reporter {
    const SEP: &'static str =
        "---------------------------------------------------------------------------";

    pub fn print_header() {
        let header = format!(
            "{:<10} | {:<10} | {:<10} | {:<10} | {:<10} | {:<10}",
            "Block", "Req/s", "Min(ms)", "P95(ms)", "Max(ms)", "Errors"
        );

        let stdout = io::stdout();
        let mut handle = stdout.lock();
        writeln!(handle, "{header}").unwrap();
        writeln!(handle, "{}", Self::SEP).unwrap();
    }

    pub fn print_metrics(metrics: &BenchMetrics) {
        let line = format!(
            "{:<10} | {:<10.2} | {:<10.2} | {:<10.2} | {:<10.2} | {:<10}",
            metrics.block,
            metrics.throughput,
            metrics.min_ms,
            metrics.p95_ms,
            metrics.max_ms,
            metrics.errors
        );

        let stdout = io::stdout();
        let mut handle = stdout.lock();
        writeln!(handle, "{line}").unwrap();
    }

    pub fn print_summary(summary: &BenchSummary, total_duration: f64) {
        if summary.total_requests == 0 {
            println!("\nNo requests processed.");
            return;
        }

        let throughput = summary.total_requests as f64 / total_duration;

        // Histogram percentiles
        let p50 = summary.hist.value_at_quantile(0.50);
        let p95 = summary.hist.value_at_quantile(0.95);
        let p99 = summary.hist.value_at_quantile(0.99);

        // Sanity check min in case it stayed at MAX
        let min_print = if summary.min_ms == f64::MAX { 0.0 } else { summary.min_ms };

        println!("\n{:-<75}", "");
        println!("Summary:");
        println!("{:<20} {}", "Total Requests:", summary.total_requests);
        println!("{:<20} {:.2}s", "Total Time:", total_duration);
        println!("{:<20} {:.2}", "Throughput (Req/s):", throughput);
        println!("{:<20} {}", "Total Errors:", summary.total_errors);
        println!("{:-<35}", "");
        println!("{:<20} {:.2} ms", "Min Latency:", min_print);
        println!("{:<20} {} ms", "Median Latency:", p50);
        println!("{:<20} {} ms", "P95 Latency:", p95);
        println!("{:<20} {} ms", "P99 Latency:", p99);
        println!("{:<20} {:.2} ms", "Max Latency:", summary.max_ms);
        println!("{:-<75}", "");
    }
}

// --- Helpers ---

// Helper to extract clean math logic from struct initialization
fn calculate_percentile(sorted_data: &[f64], percentile: f64) -> f64 {
    if sorted_data.is_empty() {
        return 0.0;
    }
    let idx = ((sorted_data.len() as f64 * percentile).ceil() as usize).saturating_sub(1);
    sorted_data.get(idx).copied().unwrap_or(0.0)
}
