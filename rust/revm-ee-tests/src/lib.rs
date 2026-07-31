//! Common test utilities for REVM crates.
//!
//! This crate provides shared test utilities that are used across different REVM crates.

use std::path::PathBuf;

use serde_json::Value;

/// Configuration for the test data comparison utility.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TestdataConfig {
    /// The directory where test data files are stored.
    pub testdata_dir: PathBuf,
}

impl Default for TestdataConfig {
    fn default() -> Self {
        Self { testdata_dir: PathBuf::from("tests/testdata") }
    }
}

/// Compares or saves the execution output to a testdata file.
///
/// This utility helps maintain consistent test behavior by comparing
/// execution results against known-good outputs stored in JSON files.
///
/// # Arguments
///
/// * `filename` - The name of the testdata file, relative to tests/testdata/
/// * `output` - The execution output to compare or save
///
/// # Panics
///
/// This function panics if:
/// - The output doesn't match the expected testdata (when testdata file exists)
/// - There's an error reading/writing files
/// - JSON serialization/deserialization fails
///
/// # Note
///
/// Tests using this function require the `serde` feature to be enabled:
/// ```bash
/// cargo test --features serde
/// ```
pub fn compare_or_save_testdata<T>(filename: &str, output: &T)
where
    T: serde::Serialize + for<'a> serde::Deserialize<'a> + PartialEq + std::fmt::Debug,
{
    compare_or_save_testdata_with_config(filename, output, TestdataConfig::default());
}

/// Compares or saves the execution output to a testdata file with custom configuration.
///
/// This is a more flexible version of [`compare_or_save_testdata`] that allows
/// specifying a custom testdata directory.
///
/// # Arguments
///
/// * `filename` - The name of the testdata file, relative to the testdata directory
/// * `output` - The execution output to compare or save
/// * `config` - Configuration for the test data comparison
pub fn compare_or_save_testdata_with_config<T>(filename: &str, output: &T, config: TestdataConfig)
where
    T: serde::Serialize + for<'a> serde::Deserialize<'a> + PartialEq + std::fmt::Debug,
{
    use std::fs;

    let testdata_file = config.testdata_dir.join(filename);

    // Create directory if it doesn't exist
    if !config.testdata_dir.exists() {
        fs::create_dir_all(&config.testdata_dir).unwrap();
    }

    // Serialize the output to serde Value.
    let output_json = serde_json::to_string(&output).unwrap();

    // convert to Value and sort all objects.
    let mut temp: Value = serde_json::from_str(&output_json).unwrap();
    temp.sort_all_objects();

    // serialize to pretty string
    let output_json = serde_json::to_string_pretty(&temp).unwrap();

    // If the testdata file doesn't exist, save the output
    if !testdata_file.exists() {
        fs::write(&testdata_file, &output_json).unwrap();
        println!("Saved testdata to {}", testdata_file.display());
        return;
    }

    // Read the expected output from the testdata file
    let expected_json = fs::read_to_string(&testdata_file).unwrap();

    // Deserialize to actual object for proper comparison
    let expected: T = serde_json::from_str(&expected_json).unwrap();

    // Compare the output objects directly
    assert!(
        *output == expected,
        "Value does not match testdata.\nExpected:\n{expected_json}\n\nActual:\n{output_json}"
    );
}

#[cfg(test)]
mod tests {
    use super::*;

    #[derive(Debug, PartialEq, serde::Serialize, serde::Deserialize)]
    struct TestData {
        value: u32,
        message: String,
    }

    #[test]
    fn test_compare_or_save_testdata() {
        let test_data = TestData { value: 42, message: "test message".to_string() };

        // This will save the test data on first run, then compare on subsequent runs
        compare_or_save_testdata("test_data.json", &test_data);
    }
}

#[cfg(test)]
mod op_revm_tests;
