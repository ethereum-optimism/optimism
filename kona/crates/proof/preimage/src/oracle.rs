use crate::{
    PreimageKey, PreimageOracleClient, PreimageOracleServer,
    errors::{PreimageOracleError, PreimageOracleResult},
    traits::{Channel, PreimageFetcher},
};
use alloc::{boxed::Box, vec::Vec};

/// An [OracleReader] is a high-level interface to the preimage oracle channel.
#[derive(Debug, Clone, Copy)]
pub struct OracleReader<C> {
    channel: C,
}

impl<C> OracleReader<C>
where
    C: Channel,
{
    /// Create a new [OracleReader] from a [Channel].
    pub const fn new(channel: C) -> Self {
        Self { channel }
    }

    /// Set the preimage key for the global oracle reader. This will overwrite any existing key, and
    /// block until the host has prepared the preimage and responded with the length of the
    /// preimage.
    async fn write_key(&self, key: PreimageKey) -> PreimageOracleResult<usize> {
        // Write the key to the host so that it can prepare the preimage.
        let key_bytes: [u8; 32] = key.into();
        self.channel.write(&key_bytes).await?;

        // Read the length prefix and reset the cursor.
        let mut length_buffer = [0u8; 8];
        self.channel.read_exact(&mut length_buffer).await?;
        Ok(u64::from_be_bytes(length_buffer) as usize)
    }
}

#[async_trait::async_trait]
impl<C> PreimageOracleClient for OracleReader<C>
where
    C: Channel + Send + Sync,
{
    /// Get the data corresponding to the currently set key from the host. Return the data in a new
    /// heap allocated `Vec<u8>`
    async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        trace!(target: "oracle_client", "Requesting data from preimage oracle. Key {key}");

        let length = self.write_key(key).await?;

        if length == 0 {
            return Ok(Default::default());
        }

        let mut data_buffer = alloc::vec![0; length];

        trace!(target: "oracle_client", "Reading data from preimage oracle. Key {key}");

        // Grab a read lock on the preimage channel to read the data.
        self.channel.read_exact(&mut data_buffer).await?;

        trace!(target: "oracle_client", "Successfully read data from preimage oracle. Key: {key}");

        Ok(data_buffer)
    }

    /// Get the data corresponding to the currently set key from the host. Write the data into the
    /// provided buffer
    async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
        trace!(target: "oracle_client", "Requesting data from preimage oracle. Key {key}");

        // Write the key to the host and read the length of the preimage.
        let length = self.write_key(key).await?;

        trace!(target: "oracle_client", "Reading data from preimage oracle. Key {key}");

        // Ensure the buffer is the correct size.
        if buf.len() != length {
            return Err(PreimageOracleError::BufferLengthMismatch(length, buf.len()));
        }

        if length == 0 {
            return Ok(());
        }

        self.channel.read_exact(buf).await?;

        trace!(target: "oracle_client", "Successfully read data from preimage oracle. Key: {key}");

        Ok(())
    }
}

/// An [OracleServer] is a router for the host to serve data back to the client [OracleReader].
#[derive(Debug, Clone, Copy)]
pub struct OracleServer<C> {
    channel: C,
}

impl<C> OracleServer<C>
where
    C: Channel,
{
    /// Create a new [OracleServer] from a [Channel].
    pub const fn new(channel: C) -> Self {
        Self { channel }
    }
}

#[async_trait::async_trait]
impl<C> PreimageOracleServer for OracleServer<C>
where
    C: Channel + Send + Sync,
{
    async fn next_preimage_request<F>(&self, fetcher: &F) -> Result<(), PreimageOracleError>
    where
        F: PreimageFetcher + Send + Sync,
    {
        // Read the preimage request from the client, and throw early if there isn't any.
        let mut buf = [0u8; 32];
        self.channel.read_exact(&mut buf).await?;
        let preimage_key = PreimageKey::try_from(buf)?;

        trace!(target: "oracle_server", "Fetching preimage for key {preimage_key}");

        // Fetch the preimage value from the preimage getter.
        let value = fetcher.get_preimage(preimage_key).await?;

        // Write the length as a big-endian u64 followed by the data.
        self.channel.write(value.len().to_be_bytes().as_ref()).await?;
        self.channel.write(value.as_ref()).await?;

        trace!(target: "oracle_server", "Successfully wrote preimage data for key {preimage_key}");

        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{PreimageKeyType, native_channel::BidirectionalChannel};
    use alloc::sync::Arc;
    use alloy_primitives::keccak256;
    use std::collections::HashMap;
    use tokio::sync::Mutex;

    struct TestFetcher {
        preimages: Arc<Mutex<HashMap<PreimageKey, Vec<u8>>>>,
    }

    #[async_trait::async_trait]
    impl PreimageFetcher for TestFetcher {
        async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            let read_lock = self.preimages.lock().await;
            read_lock.get(&key).cloned().ok_or(PreimageOracleError::KeyNotFound)
        }
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn test_oracle_reader_get_exact() {
        const MOCK_DATA_A: &[u8] = b"1234567890";
        const MOCK_DATA_B: &[u8] = b"FACADE";
        let key_a: PreimageKey =
            PreimageKey::new(*keccak256(MOCK_DATA_A), PreimageKeyType::Keccak256);
        let key_b: PreimageKey =
            PreimageKey::new(*keccak256(MOCK_DATA_B), PreimageKeyType::Keccak256);

        let preimages = {
            let mut preimages = HashMap::default();
            preimages.insert(key_a, MOCK_DATA_A.to_vec());
            preimages.insert(key_b, MOCK_DATA_B.to_vec());
            Arc::new(Mutex::new(preimages))
        };

        let preimage_channel = BidirectionalChannel::new().unwrap();

        let client = tokio::task::spawn(async move {
            let oracle_reader = OracleReader::new(preimage_channel.client);
            let mut contents_a = [0u8; 10];
            let mut contents_b = [0u8; 6];
            oracle_reader.get_exact(key_a, &mut contents_a).await.unwrap();
            oracle_reader.get_exact(key_b, &mut contents_b).await.unwrap();

            (contents_a, contents_b)
        });
        tokio::task::spawn(async move {
            let oracle_server = OracleServer::new(preimage_channel.host);
            let test_fetcher = TestFetcher { preimages: Arc::clone(&preimages) };

            loop {
                match oracle_server.next_preimage_request(&test_fetcher).await {
                    Err(PreimageOracleError::IOError(_)) => break,
                    Err(e) => panic!("Unexpected error: {e:?}"),
                    Ok(_) => {}
                }
            }
        });

        let (c,) = tokio::join!(client);
        let (contents_a, contents_b) = c.unwrap();
        assert_eq!(contents_a, MOCK_DATA_A);
        assert_eq!(contents_b, MOCK_DATA_B);
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn test_oracle_client_and_host() {
        const MOCK_DATA_A: &[u8] = b"1234567890";
        const MOCK_DATA_B: &[u8] = b"FACADE";
        let key_a: PreimageKey =
            PreimageKey::new(*keccak256(MOCK_DATA_A), PreimageKeyType::Keccak256);
        let key_b: PreimageKey =
            PreimageKey::new(*keccak256(MOCK_DATA_B), PreimageKeyType::Keccak256);

        let preimages = {
            let mut preimages = HashMap::default();
            preimages.insert(key_a, MOCK_DATA_A.to_vec());
            preimages.insert(key_b, MOCK_DATA_B.to_vec());
            Arc::new(Mutex::new(preimages))
        };

        let preimage_channel = BidirectionalChannel::new().unwrap();

        let client = tokio::task::spawn(async move {
            let oracle_reader = OracleReader::new(preimage_channel.client);
            let contents_a = oracle_reader.get(key_a).await.unwrap();
            let contents_b = oracle_reader.get(key_b).await.unwrap();

            (contents_a, contents_b)
        });
        tokio::task::spawn(async move {
            let oracle_server = OracleServer::new(preimage_channel.host);
            let test_fetcher = TestFetcher { preimages: Arc::clone(&preimages) };

            loop {
                match oracle_server.next_preimage_request(&test_fetcher).await {
                    Err(PreimageOracleError::IOError(_)) => break,
                    Err(e) => panic!("Unexpected error: {e:?}"),
                    Ok(_) => {}
                }
            }
        });

        let (c,) = tokio::join!(client);
        let (contents_a, contents_b) = c.unwrap();
        assert_eq!(contents_a, MOCK_DATA_A);
        assert_eq!(contents_b, MOCK_DATA_B);
    }
}
