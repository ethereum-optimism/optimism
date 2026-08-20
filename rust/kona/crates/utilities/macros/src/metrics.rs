//! Macros for recording metrics.

/// Sets a metric value, optionally with a specified label.
#[macro_export]
macro_rules! set {
    (counter, $metric:path, $key:expr, $value:expr, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::counter!($metric, $key => $value).absolute($amount);
    };
    ($instrument:ident, $metric:path, $key:expr, $value:expr, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, $key => $value).set($amount);
    };
    (counter, $metric:path, $value:expr, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::counter!($metric, "type" => $value).absolute($amount);
    };
    ($instrument:ident, $metric:path, $value:expr, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, "type" => $value).set($amount);
    };
    (counter, $metric:path, $value:expr) => {
        #[cfg(feature = "metrics")]
        metrics::counter!($metric).absolute($value);
    };
    ($instrument:ident, $metric:path, $value:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric).set($value);
    };
    // Multi-label arms take the value first: a trailing `$value` after a repetition would be
    // ambiguous to the macro matcher.
    (counter, $metric:path, $value:expr, $($label_key:expr => $label_value:expr),+ $(,)?) => {
        #[cfg(feature = "metrics")]
        metrics::counter!($metric, $($label_key => $label_value),+).absolute($value);
    };
    ($instrument:ident, $metric:path, $value:expr, $($label_key:expr => $label_value:expr),+ $(,)?) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, $($label_key => $label_value),+).set($value);
    };
}

/// Increments a metric value, optionally with a specified label.
#[macro_export]
macro_rules! inc {
    ($instrument:ident, $metric:path, $value:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, "type" => $value).increment(1);
    };
    ($instrument:ident, $metric:path $(, $label_key:expr $(=> $label_value:expr)?)*$(,)?) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric $(, $label_key $(=> $label_value)?)*).increment(1);
    };
    ($instrument:ident, $metric:path, $value:expr $(, $label_key:expr $(=> $label_value:expr)?)*$(,)?) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric $(, $label_key $(=> $label_value)?)*).increment($value);
    };
}

/// Records a value, optionally with a specified label.
#[macro_export]
macro_rules! record {
    ($instrument:ident, $metric:path, $key:expr, $value:expr, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, $key => $value).record($amount);
    };
    ($instrument:ident, $metric:path, $amount:expr) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric).record($amount);
    };
    // Multi-label arm takes the amount first: a trailing `$amount` after a repetition would be
    // ambiguous to the macro matcher.
    ($instrument:ident, $metric:path, $amount:expr, $($label_key:expr => $label_value:expr),+ $(,)?) => {
        #[cfg(feature = "metrics")]
        metrics::$instrument!($metric, $($label_key => $label_value),+).record($amount);
    };
}
