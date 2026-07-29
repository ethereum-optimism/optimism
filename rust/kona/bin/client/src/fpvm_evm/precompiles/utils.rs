//! Utility functions for precompiles

/// A macro that generates an async block that sends a hint to the host, constructs a key hash
/// from the hint data, fetches the result of the precompile run from the host, and returns the
/// result data.
///
/// The macro takes the following arguments:
/// - `hint_data`: The hint data to send to the host.
#[macro_export]
macro_rules! precompile_run {
    ($hint_writer:expr, $oracle_reader:expr, $hint_data:expr) => {
        async move {
            use alloc::{string::ToString, vec::Vec};
            use kona_preimage::{PreimageKey, PreimageKeyType, errors::PreimageOracleError};
            use kona_proof::{HintType, errors::OracleProviderError};

            // Write the hint for the precompile run.
            let hint_data = $hint_data;
            HintType::L1Precompile.with_data(hint_data).send($hint_writer).await?;

            // Construct the key hash for the precompile run.
            let raw_key_data = hint_data.iter().copied().flatten().copied().collect::<Vec<u8>>();
            let key_hash = alloy_primitives::keccak256(&raw_key_data);

            // Fetch the result of the precompile run from the host.
            let result_data = $oracle_reader
                .get(PreimageKey::new(*key_hash, PreimageKeyType::Precompile))
                .await
                .map_err(OracleProviderError::Preimage)?;

            // Ensure we've received valid result data.
            if result_data.is_empty() {
                return Err(OracleProviderError::Preimage(PreimageOracleError::Other(
                    "Invalid result data".to_string(),
                )));
            }

            // Ensure we've not received an error from the host.
            if result_data[0] == 0 {
                return Err(OracleProviderError::Preimage(PreimageOracleError::Other(
                    "Error executing precompile in host".to_string(),
                )));
            }

            // Return the result data.
            Ok(result_data[1..].to_vec())
        }
    };
}

pub(crate) use precompile_run;
