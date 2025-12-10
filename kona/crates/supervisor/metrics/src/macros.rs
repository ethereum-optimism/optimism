/// Macro to observe a call to a storage method and record metrics.
#[macro_export]
macro_rules! observe_metrics_for_result {
    (
        $success_metric:expr,
        $error_metric:expr,
        $duration_metric:expr,
        $method_name:expr,
        $block:expr $(, $tag_key:expr => $tag_val:expr )*
    ) => {{
        let start_time = std::time::Instant::now();
        let result = $block;
        let duration = start_time.elapsed().as_secs_f64();

        if result.is_ok() {
            metrics::counter!(
                $success_metric,
                "method" => $method_name
                $(, $tag_key => $tag_val )*
            ).increment(1);
        } else {
            metrics::counter!(
                $error_metric,
                "method" => $method_name
                $(, $tag_key => $tag_val )*
            ).increment(1);
        }

        metrics::histogram!(
            $duration_metric,
            "method" => $method_name
            $(, $tag_key => $tag_val )*
        ).record(duration);

        result
    }};
}

/// Macro to observe a call to an async function and record metrics.
#[macro_export]
macro_rules! observe_metrics_for_result_async {
    (
        $success_metric:expr,
        $error_metric:expr,
        $duration_metric:expr,
        $method_name:expr,
        $block:expr $(, $tag_key:expr => $tag_val:expr )*
    ) => {{
        let start_time = std::time::Instant::now();
        let result = $block.await;
        let duration = start_time.elapsed().as_secs_f64();

        if result.is_ok() {
            metrics::counter!(
                $success_metric,
                "method" => $method_name
                $(, $tag_key => $tag_val )*
            ).increment(1);
        } else {
            metrics::counter!(
                $error_metric,
                "method" => $method_name
                $(, $tag_key => $tag_val )*
            ).increment(1);
        }

        metrics::histogram!(
            $duration_metric,
            "method" => $method_name
            $(, $tag_key => $tag_val )*
        ).record(duration);

        result
    }};
}
