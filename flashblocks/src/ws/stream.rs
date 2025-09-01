use crate::FlashBlock;
use eyre::eyre;
use futures_util::{stream::SplitStream, FutureExt, Stream, StreamExt};
use std::{
    fmt::{Debug, Formatter},
    future::Future,
    pin::Pin,
    task::{ready, Context, Poll},
};
use tokio::net::TcpStream;
use tokio_tungstenite::{
    connect_async,
    tungstenite::{Error, Message},
    MaybeTlsStream, WebSocketStream,
};
use url::Url;

/// An asynchronous stream of [`FlashBlock`] from a websocket connection.
///
/// The stream attempts to connect to a websocket URL and then decode each received item.
///
/// If the connection fails, the error is returned and connection retried. The number of retries is
/// unbounded.
pub struct WsFlashBlockStream<Stream, Connector> {
    ws_url: Url,
    state: State,
    connector: Connector,
    connect: ConnectFuture<Stream>,
    stream: Option<Stream>,
}

impl WsFlashBlockStream<WssStream, WsConnector> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn new(ws_url: Url) -> Self {
        Self {
            ws_url,
            state: State::default(),
            connector: WsConnector,
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
        }
    }
}

impl<S, C> WsFlashBlockStream<S, C> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn with_connector(ws_url: Url, connector: C) -> Self {
        Self {
            ws_url,
            state: State::default(),
            connector,
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
        }
    }
}

impl<S, C> Stream for WsFlashBlockStream<S, C>
where
    S: Stream<Item = Result<Message, Error>> + Unpin,
    C: WsConnect<Stream = S> + Clone + Send + Sync + 'static + Unpin,
{
    type Item = eyre::Result<FlashBlock>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        if self.state == State::Initial {
            self.connect();
        }

        if self.state == State::Connect {
            match ready!(self.connect.poll_unpin(cx)) {
                Ok(stream) => self.stream(stream),
                Err(err) => {
                    self.state = State::Initial;

                    return Poll::Ready(Some(Err(err)));
                }
            }
        }

        let msg = ready!(self
            .stream
            .as_mut()
            .expect("Stream state should be unreachable without stream")
            .poll_next_unpin(cx));

        Poll::Ready(msg.map(|msg| match msg {
            Ok(Message::Binary(bytes)) => FlashBlock::decode(bytes),
            Ok(msg) => Err(eyre!("Unexpected websocket message: {msg:?}")),
            Err(err) => Err(err.into()),
        }))
    }
}

impl<S, C> WsFlashBlockStream<S, C>
where
    C: WsConnect<Stream = S> + Clone + Send + Sync + 'static,
{
    fn connect(&mut self) {
        let ws_url = self.ws_url.clone();
        let mut connector = self.connector.clone();

        Pin::new(&mut self.connect).set(Box::pin(async move { connector.connect(ws_url).await }));

        self.state = State::Connect;
    }

    fn stream(&mut self, stream: S) {
        self.stream.replace(stream);

        self.state = State::Stream;
    }
}

impl<S: Debug, C: Debug> Debug for WsFlashBlockStream<S, C> {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FlashBlockStream")
            .field("ws_url", &self.ws_url)
            .field("state", &self.state)
            .field("connector", &self.connector)
            .field("connect", &"Pin<Box<dyn Future<..>>>")
            .field("stream", &self.stream)
            .finish()
    }
}

#[derive(Default, Debug, Eq, PartialEq)]
enum State {
    #[default]
    Initial,
    Connect,
    Stream,
}

type WsStream = WebSocketStream<MaybeTlsStream<TcpStream>>;
type WssStream = SplitStream<WsStream>;
type ConnectFuture<Stream> =
    Pin<Box<dyn Future<Output = eyre::Result<Stream>> + Send + Sync + 'static>>;

/// The `WsConnect` trait allows for connecting to a websocket.
///
/// Implementors of the `WsConnect` trait are called 'connectors'.
///
/// Connectors are defined by one method, [`connect()`]. A call to [`connect()`] attempts to
/// establish a secure websocket connection and return an asynchronous stream of [`Message`]s
/// wrapped in a [`Result`].
///
/// [`connect()`]: Self::connect
pub trait WsConnect {
    /// An associated `Stream` of [`Message`]s wrapped in a [`Result`] that this connection returns.
    type Stream;

    /// Asynchronously connects to a websocket hosted on `ws_url`.
    ///
    /// See the [`WsConnect`] documentation for details.
    fn connect(
        &mut self,
        ws_url: Url,
    ) -> impl Future<Output = eyre::Result<Self::Stream>> + Send + Sync;
}

/// Establishes a secure websocket subscription.
///
/// See the [`WsConnect`] documentation for details.
#[derive(Debug, Clone)]
pub struct WsConnector;

impl WsConnect for WsConnector {
    type Stream = WssStream;

    async fn connect(&mut self, ws_url: Url) -> eyre::Result<WssStream> {
        let (stream, _response) = connect_async(ws_url.as_str()).await?;

        Ok(stream.split().1)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ExecutionPayloadBaseV1;
    use alloy_primitives::bytes::Bytes;
    use brotli::enc::BrotliEncoderParams;
    use std::future;
    use tokio_tungstenite::tungstenite::Error;

    /// A `FakeConnector` creates [`FakeStream`].
    ///
    /// It simulates the websocket stream instead of connecting to a real websocket.
    #[derive(Clone)]
    struct FakeConnector(FakeStream);

    /// Simulates a websocket stream while using a preprogrammed set of messages instead.
    #[derive(Default)]
    struct FakeStream(Vec<Result<Message, Error>>);

    impl Clone for FakeStream {
        fn clone(&self) -> Self {
            Self(
                self.0
                    .iter()
                    .map(|v| match v {
                        Ok(msg) => Ok(msg.clone()),
                        Err(err) => unimplemented!("Cannot clone this error: {err}"),
                    })
                    .collect(),
            )
        }
    }

    impl Stream for FakeStream {
        type Item = Result<Message, Error>;

        fn poll_next(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
            let this = self.get_mut();

            Poll::Ready(this.0.pop())
        }
    }

    impl WsConnect for FakeConnector {
        type Stream = FakeStream;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<Self::Stream>> + Send + Sync {
            future::ready(Ok(self.0.clone()))
        }
    }

    impl<T: IntoIterator<Item = Result<Message, Error>>> From<T> for FakeConnector {
        fn from(value: T) -> Self {
            Self(FakeStream(value.into_iter().collect()))
        }
    }

    fn to_json_message(block: &FlashBlock) -> Result<Message, Error> {
        Ok(Message::Binary(Bytes::from(serde_json::to_vec(block).unwrap())))
    }

    fn to_brotli_message(block: &FlashBlock) -> Result<Message, Error> {
        let json = serde_json::to_vec(block).unwrap();
        let mut compressed = Vec::new();
        brotli::BrotliCompress(
            &mut json.as_slice(),
            &mut compressed,
            &BrotliEncoderParams::default(),
        )?;

        Ok(Message::Binary(Bytes::from(compressed)))
    }

    #[test_case::test_case(to_json_message; "json")]
    #[test_case::test_case(to_brotli_message; "brotli")]
    #[tokio::test]
    async fn test_stream_decodes_messages_successfully(
        to_message: impl Fn(&FlashBlock) -> Result<Message, Error>,
    ) {
        let flashblocks = [FlashBlock {
            payload_id: Default::default(),
            index: 0,
            base: Some(ExecutionPayloadBaseV1 {
                parent_beacon_block_root: Default::default(),
                parent_hash: Default::default(),
                fee_recipient: Default::default(),
                prev_randao: Default::default(),
                block_number: 0,
                gas_limit: 0,
                timestamp: 0,
                extra_data: Default::default(),
                base_fee_per_gas: Default::default(),
            }),
            diff: Default::default(),
            metadata: Default::default(),
        }];

        let messages = FakeConnector::from(flashblocks.iter().map(to_message));
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, messages);

        let actual_messages: Vec<_> = stream.map(Result::unwrap).collect().await;
        let expected_messages = flashblocks.to_vec();

        assert_eq!(actual_messages, expected_messages);
    }
}
