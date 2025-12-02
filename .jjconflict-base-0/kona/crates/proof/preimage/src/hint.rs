use crate::{
    Channel, HintReaderServer,
    errors::{PreimageOracleError, PreimageOracleResult},
    traits::{HintRouter, HintWriterClient},
};
use alloc::{boxed::Box, format, string::String, vec};
use async_trait::async_trait;

/// A [HintWriter] is a high-level interface to the hint channel. It provides a way to write hints
/// to the host.
#[derive(Debug, Clone, Copy)]
pub struct HintWriter<C> {
    channel: C,
}

impl<C> HintWriter<C> {
    /// Create a new [HintWriter] from a [Channel].
    pub const fn new(channel: C) -> Self {
        Self { channel }
    }
}

#[async_trait]
impl<C> HintWriterClient for HintWriter<C>
where
    C: Channel + Send + Sync,
{
    /// Write a hint to the host. This will overwrite any existing hint in the channel, and block
    /// until all data has been written.
    async fn write(&self, hint: &str) -> PreimageOracleResult<()> {
        trace!(target: "hint_writer", "Writing hint \"{hint}\"");

        // Form the hint into a byte buffer. The format is a 4-byte big-endian length prefix
        // followed by the hint string.
        self.channel.write(u32::to_be_bytes(hint.len() as u32).as_ref()).await?;
        self.channel.write(hint.as_bytes()).await?;

        trace!(target: "hint_writer", "Successfully wrote hint");

        // Read the hint acknowledgement from the host.
        let mut hint_ack = [0u8; 1];
        self.channel.read_exact(&mut hint_ack).await?;

        trace!(target: "hint_writer", "Received hint acknowledgement");

        Ok(())
    }
}

/// A [HintReader] is a router for hints sent by the [HintWriter] from the client program. It
/// provides a way for the host to prepare preimages for reading.
#[derive(Debug, Clone, Copy)]
pub struct HintReader<C> {
    channel: C,
}

impl<C> HintReader<C>
where
    C: Channel,
{
    /// Create a new [HintReader] from a [Channel].
    pub const fn new(channel: C) -> Self {
        Self { channel }
    }
}

#[async_trait]
impl<C> HintReaderServer for HintReader<C>
where
    C: Channel + Send + Sync,
{
    async fn next_hint<R>(&self, hint_router: &R) -> PreimageOracleResult<()>
    where
        R: HintRouter + Send + Sync,
    {
        // Read the length of the raw hint payload.
        let mut len_buf = [0u8; 4];
        self.channel.read_exact(&mut len_buf).await?;
        let len = u32::from_be_bytes(len_buf);

        // Read the raw hint payload.
        let mut raw_payload = vec![0u8; len as usize];
        self.channel.read_exact(raw_payload.as_mut_slice()).await?;
        let payload = match String::from_utf8(raw_payload) {
            Ok(p) => p,
            Err(e) => {
                // Write back on error to prevent blocking the client.
                self.channel.write(&[0x00]).await?;

                return Err(PreimageOracleError::Other(format!(
                    "Failed to decode hint payload: {e}"
                )));
            }
        };

        trace!(target: "hint_reader", "Successfully read hint: \"{payload}\"");

        // Route the hint
        if let Err(e) = hint_router.route_hint(payload).await {
            // Write back on error to prevent blocking the client.
            self.channel.write(&[0x00]).await?;

            error!(target: "hint_reader", "Failed to route hint: {e}");
            return Err(e);
        }

        // Write back an acknowledgement to the client to unblock their process.
        self.channel.write(&[0x00]).await?;

        trace!(target: "hint_reader", "Successfully routed and acknowledged hint");

        Ok(())
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::native_channel::BidirectionalChannel;
    use alloc::{sync::Arc, vec::Vec};
    use tokio::sync::Mutex;

    struct TestRouter {
        incoming_hints: Arc<Mutex<Vec<String>>>,
    }

    #[async_trait]
    impl HintRouter for TestRouter {
        async fn route_hint(&self, hint: String) -> PreimageOracleResult<()> {
            self.incoming_hints.lock().await.push(hint);
            Ok(())
        }
    }

    struct TestFailRouter;

    #[async_trait]
    impl HintRouter for TestFailRouter {
        async fn route_hint(&self, _hint: String) -> PreimageOracleResult<()> {
            Err(PreimageOracleError::KeyNotFound)
        }
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn test_unblock_on_bad_utf8() {
        let mock_data = [0xf0, 0x90, 0x28, 0xbc];

        let hint_channel = BidirectionalChannel::new().unwrap();

        let client = tokio::task::spawn(async move {
            let hint_writer = HintWriter::new(hint_channel.client);

            #[allow(invalid_from_utf8_unchecked)]
            hint_writer.write(unsafe { alloc::str::from_utf8_unchecked(&mock_data) }).await
        });
        let host = tokio::task::spawn(async move {
            let router = TestRouter { incoming_hints: Default::default() };

            let hint_reader = HintReader::new(hint_channel.host);
            hint_reader.next_hint(&router).await
        });

        let (c, h) = tokio::join!(client, host);
        c.unwrap().unwrap();
        assert!(h.unwrap().is_err_and(|e| {
            let PreimageOracleError::Other(e) = e else {
                return false;
            };
            e.contains("Failed to decode hint payload")
        }));
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn test_unblock_on_fetch_failure() {
        const MOCK_DATA: &str = "test-hint 0xfacade";

        let hint_channel = BidirectionalChannel::new().unwrap();

        let client = tokio::task::spawn(async move {
            let hint_writer = HintWriter::new(hint_channel.client);

            hint_writer.write(MOCK_DATA).await
        });
        let host = tokio::task::spawn(async move {
            let hint_reader = HintReader::new(hint_channel.host);
            hint_reader.next_hint(&TestFailRouter).await
        });

        let (c, h) = tokio::join!(client, host);
        c.unwrap().unwrap();
        assert!(h.unwrap().is_err_and(|e| matches!(e, PreimageOracleError::KeyNotFound)));
    }

    #[tokio::test(flavor = "multi_thread", worker_threads = 2)]
    async fn test_hint_client_and_host() {
        const MOCK_DATA: &str = "test-hint 0xfacade";

        let incoming_hints = Arc::new(Mutex::new(Vec::new()));
        let hint_channel = BidirectionalChannel::new().unwrap();

        let client = tokio::task::spawn(async move {
            let hint_writer = HintWriter::new(hint_channel.client);

            hint_writer.write(MOCK_DATA).await
        });
        let host = tokio::task::spawn({
            let incoming_hints_ref = Arc::clone(&incoming_hints);
            async move {
                let router = TestRouter { incoming_hints: incoming_hints_ref };

                let hint_reader = HintReader::new(hint_channel.host);
                hint_reader.next_hint(&router).await.unwrap();
            }
        });

        let _ = tokio::join!(client, host);
        let mut hints = incoming_hints.lock().await;

        assert_eq!(hints.len(), 1);
        let h = hints.remove(0);
        assert_eq!(h, MOCK_DATA);
    }
}
