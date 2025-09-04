use crate::FlashBlock;
use futures_util::{
    stream::{SplitSink, SplitStream},
    FutureExt, Sink, Stream, StreamExt,
};
use std::{
    fmt::{Debug, Formatter},
    future::Future,
    pin::Pin,
    task::{ready, Context, Poll},
};
use tokio::net::TcpStream;
use tokio_tungstenite::{
    connect_async,
    tungstenite::{Bytes, Error, Message},
    MaybeTlsStream, WebSocketStream,
};
use tracing::debug;
use url::Url;

/// An asynchronous stream of [`FlashBlock`] from a websocket connection.
///
/// The stream attempts to connect to a websocket URL and then decode each received item.
///
/// If the connection fails, the error is returned and connection retried. The number of retries is
/// unbounded.
pub struct WsFlashBlockStream<Stream, Sink, Connector> {
    ws_url: Url,
    state: State,
    connector: Connector,
    connect: ConnectFuture<Sink, Stream>,
    stream: Option<Stream>,
    sink: Option<Sink>,
}

impl WsFlashBlockStream<WsStream, WsSink, WsConnector> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn new(ws_url: Url) -> Self {
        Self {
            ws_url,
            state: State::default(),
            connector: WsConnector,
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
            sink: None,
        }
    }
}

impl<Stream, S, C> WsFlashBlockStream<Stream, S, C> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn with_connector(ws_url: Url, connector: C) -> Self {
        Self {
            ws_url,
            state: State::default(),
            connector,
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
            sink: None,
        }
    }
}

impl<Str, S, C> Stream for WsFlashBlockStream<Str, S, C>
where
    Str: Stream<Item = Result<Message, Error>> + Unpin,
    S: Sink<Message> + Send + Sync + Unpin,
    C: WsConnect<Stream = Str, Sink = S> + Clone + Send + Sync + 'static + Unpin,
{
    type Item = eyre::Result<FlashBlock>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        loop {
            if this.state == State::Initial {
                this.connect();
            }

            if this.state == State::Connect {
                match ready!(this.connect.poll_unpin(cx)) {
                    Ok((sink, stream)) => this.stream(sink, stream),
                    Err(err) => {
                        this.state = State::Initial;

                        return Poll::Ready(Some(Err(err)));
                    }
                }
            }

            while let State::Stream(pong) = &mut this.state {
                if pong.is_some() {
                    let mut sink = Pin::new(this.sink.as_mut().unwrap());
                    let _ = ready!(sink.as_mut().poll_ready(cx));
                    if let Some(pong) = pong.take() {
                        let _ = sink.as_mut().start_send(pong);
                    }
                    let _ = ready!(sink.as_mut().poll_flush(cx));
                }

                let Some(msg) = ready!(this
                    .stream
                    .as_mut()
                    .expect("Stream state should be unreachable without stream")
                    .poll_next_unpin(cx))
                else {
                    return Poll::Ready(None);
                };

                match msg {
                    Ok(Message::Binary(bytes)) => {
                        return Poll::Ready(Some(FlashBlock::decode(bytes)))
                    }
                    Ok(Message::Text(bytes)) => {
                        return Poll::Ready(Some(FlashBlock::decode(bytes.into())))
                    }
                    Ok(Message::Ping(bytes)) => this.ping(bytes),
                    Ok(msg) => debug!("Received unexpected message: {:?}", msg),
                    Err(err) => return Poll::Ready(Some(Err(err.into()))),
                }
            }
        }
    }
}

impl<Stream, S, C> WsFlashBlockStream<Stream, S, C>
where
    C: WsConnect<Stream = Stream, Sink = S> + Clone + Send + Sync + 'static,
{
    fn connect(&mut self) {
        let ws_url = self.ws_url.clone();
        let mut connector = self.connector.clone();

        Pin::new(&mut self.connect).set(Box::pin(async move { connector.connect(ws_url).await }));

        self.state = State::Connect;
    }

    fn stream(&mut self, sink: S, stream: Stream) {
        self.sink.replace(sink);
        self.stream.replace(stream);

        self.state = State::Stream(None);
    }

    fn ping(&mut self, pong: Bytes) {
        if let State::Stream(current) = &mut self.state {
            current.replace(Message::Pong(pong));
        }
    }
}

impl<Stream: Debug, S: Debug, C: Debug> Debug for WsFlashBlockStream<Stream, S, C> {
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
    Stream(Option<Message>),
}

type Ws = WebSocketStream<MaybeTlsStream<TcpStream>>;
type WsStream = SplitStream<Ws>;
type WsSink = SplitSink<Ws, Message>;
type ConnectFuture<Sink, Stream> =
    Pin<Box<dyn Future<Output = eyre::Result<(Sink, Stream)>> + Send + Sync + 'static>>;

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

    /// An associated `Sink` of [`Message`]s that this connection sends.
    type Sink;

    /// Asynchronously connects to a websocket hosted on `ws_url`.
    ///
    /// See the [`WsConnect`] documentation for details.
    fn connect(
        &mut self,
        ws_url: Url,
    ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send + Sync;
}

/// Establishes a secure websocket subscription.
///
/// See the [`WsConnect`] documentation for details.
#[derive(Debug, Clone)]
pub struct WsConnector;

impl WsConnect for WsConnector {
    type Stream = WsStream;
    type Sink = WsSink;

    async fn connect(&mut self, ws_url: Url) -> eyre::Result<(WsSink, WsStream)> {
        let (stream, _response) = connect_async(ws_url.as_str()).await?;

        Ok(stream.split())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ExecutionPayloadBaseV1;
    use alloy_primitives::bytes::Bytes;
    use brotli::enc::BrotliEncoderParams;
    use std::{future, iter};
    use tokio_tungstenite::tungstenite::{protocol::frame::Frame, Error};

    /// A `FakeConnector` creates [`FakeStream`].
    ///
    /// It simulates the websocket stream instead of connecting to a real websocket.
    #[derive(Clone)]
    struct FakeConnector(FakeStream);

    /// A `FakeConnectorWithSink` creates [`FakeStream`] and [`FakeSink`].
    ///
    /// It simulates the websocket stream instead of connecting to a real websocket. It also accepts
    /// messages into an in-memory buffer.
    #[derive(Clone)]
    struct FakeConnectorWithSink(FakeStream);

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
                        Err(err) => Err(match err {
                            Error::AttackAttempt => Error::AttackAttempt,
                            err => unimplemented!("Cannot clone this error: {err}"),
                        }),
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

    #[derive(Clone)]
    struct NoopSink;

    impl<T> Sink<T> for NoopSink {
        type Error = ();

        fn poll_ready(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<Result<(), Self::Error>> {
            unimplemented!()
        }

        fn start_send(self: Pin<&mut Self>, _item: T) -> Result<(), Self::Error> {
            unimplemented!()
        }

        fn poll_flush(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<Result<(), Self::Error>> {
            unimplemented!()
        }

        fn poll_close(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<Result<(), Self::Error>> {
            unimplemented!()
        }
    }

    /// Receives [`Message`]s and stores them. A call to `start_send` first buffers the message
    /// to simulate flushing behavior.
    #[derive(Clone, Default)]
    struct FakeSink(Option<Message>, Vec<Message>);

    impl Sink<Message> for FakeSink {
        type Error = ();

        fn poll_ready(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
            self.poll_flush(cx)
        }

        fn start_send(self: Pin<&mut Self>, item: Message) -> Result<(), Self::Error> {
            self.get_mut().0.replace(item);
            Ok(())
        }

        fn poll_flush(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<Result<(), Self::Error>> {
            let this = self.get_mut();
            if let Some(item) = this.0.take() {
                this.1.push(item);
            }
            Poll::Ready(Ok(()))
        }

        fn poll_close(
            self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<Result<(), Self::Error>> {
            Poll::Ready(Ok(()))
        }
    }

    impl WsConnect for FakeConnector {
        type Stream = FakeStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send + Sync {
            future::ready(Ok((NoopSink, self.0.clone())))
        }
    }

    impl<T: IntoIterator<Item = Result<Message, Error>>> From<T> for FakeConnector {
        fn from(value: T) -> Self {
            Self(FakeStream(value.into_iter().collect()))
        }
    }

    impl WsConnect for FakeConnectorWithSink {
        type Stream = FakeStream;
        type Sink = FakeSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send + Sync {
            future::ready(Ok((FakeSink::default(), self.0.clone())))
        }
    }

    impl<T: IntoIterator<Item = Result<Message, Error>>> From<T> for FakeConnectorWithSink {
        fn from(value: T) -> Self {
            Self(FakeStream(value.into_iter().collect()))
        }
    }

    /// Repeatedly fails to connect with the given error message.
    #[derive(Clone)]
    struct FailingConnector(String);

    impl WsConnect for FailingConnector {
        type Stream = FakeStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send + Sync {
            future::ready(Err(eyre::eyre!("{}", &self.0)))
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

    #[test_case::test_case(Message::Pong(Bytes::from(b"test".as_slice())); "pong")]
    #[test_case::test_case(Message::Frame(Frame::pong(b"test".as_slice())); "frame")]
    #[tokio::test]
    async fn test_stream_ignores_unexpected_message(message: Message) {
        let messages = FakeConnector::from([Ok(message)]);
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, messages);
        assert!(stream.next().await.is_none());
    }

    #[tokio::test]
    async fn test_stream_passes_errors_through() {
        let messages = FakeConnector::from([Err(Error::AttackAttempt)]);
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, messages);

        let actual_messages: Vec<_> =
            stream.map(Result::unwrap_err).map(|e| format!("{e}")).collect().await;
        let expected_messages = vec!["Attack attempt detected".to_owned()];

        assert_eq!(actual_messages, expected_messages);
    }

    #[tokio::test]
    async fn test_connect_error_causes_retries() {
        let tries = 3;
        let error_msg = "test".to_owned();
        let messages = FailingConnector(error_msg.clone());
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, messages);

        let actual_errors: Vec<_> =
            stream.take(tries).map(Result::unwrap_err).map(|e| format!("{e}")).collect().await;
        let expected_errors: Vec<_> = iter::repeat_n(error_msg, tries).collect();

        assert_eq!(actual_errors, expected_errors);
    }

    #[tokio::test]
    async fn test_stream_pongs_ping() {
        const ECHO: [u8; 3] = [1u8, 2, 3];

        let messages = [Ok(Message::Ping(Bytes::from_static(&ECHO)))];
        let connector = FakeConnectorWithSink::from(messages);
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let _ = stream.next().await;

        let FakeSink(actual_buffered_messages, actual_sent_messages) = stream.sink.unwrap();

        assert!(
            actual_buffered_messages.is_none(),
            "buffer not flushed: {actual_buffered_messages:#?}"
        );

        let expected_sent_messages = vec![Message::Pong(Bytes::from_static(&ECHO))];

        assert_eq!(actual_sent_messages, expected_sent_messages);
    }
}
