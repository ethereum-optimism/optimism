use crate::{FlashBlock, ws::FlashBlockDecoder};
use futures_util::{
    FutureExt, Sink, Stream, StreamExt,
    stream::{SplitSink, SplitStream},
};
use std::{
    fmt::{Debug, Formatter},
    future::Future,
    pin::Pin,
    task::{Context, Poll, ready},
    time::Duration,
};
use tokio::{
    net::TcpStream,
    time::{Instant, Sleep, sleep},
};
use tokio_tungstenite::{
    MaybeTlsStream, WebSocketStream, connect_async,
    tungstenite::{Bytes, Error, Message, protocol::CloseFrame},
};
use tracing::debug;
use url::Url;

/// Upstream silence after which the connection is considered dead.
///
/// Subblocks arrive every few hundred milliseconds, so this long without a single frame - not even
/// a websocket ping - means the connection has stopped delivering.
const IDLE_TIMEOUT: Duration = Duration::from_secs(5);

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
    decoder: Box<dyn FlashBlockDecoder>,
    connect: ConnectFuture<Sink, Stream>,
    stream: Option<Stream>,
    sink: Option<Sink>,
    /// Fires once [`IDLE_TIMEOUT`] has elapsed without a frame from the upstream.
    idle_deadline: Option<Pin<Box<Sleep>>>,
}

impl WsFlashBlockStream<WsStream, WsSink, WsConnector> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn new(ws_url: Url) -> Self {
        Self {
            ws_url,
            state: State::default(),
            connector: WsConnector,
            decoder: Box::new(()),
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
            sink: None,
            idle_deadline: None,
        }
    }

    /// Sets the [`FlashBlock`] decoder for the websocket stream.
    pub fn with_decoder(self, decoder: Box<dyn FlashBlockDecoder>) -> Self {
        Self { decoder, ..self }
    }
}

impl<Stream, S, C> WsFlashBlockStream<Stream, S, C> {
    /// Creates a new websocket stream over `ws_url`.
    pub fn with_connector(ws_url: Url, connector: C) -> Self {
        Self {
            ws_url,
            state: State::default(),
            decoder: Box::new(()),
            connector,
            connect: Box::pin(async move { Err(Error::ConnectionClosed)? }),
            stream: None,
            sink: None,
            idle_deadline: None,
        }
    }

    /// Restarts the idle deadline from now.
    fn reset_idle(&mut self) {
        match self.idle_deadline.as_mut() {
            Some(deadline) => deadline.as_mut().reset(Instant::now() + IDLE_TIMEOUT),
            None => self.idle_deadline = Some(Box::pin(sleep(IDLE_TIMEOUT))),
        }
    }

    /// Called where the connection would otherwise park the task: registers the idle deadline's
    /// wake-up and, once the deadline has passed, drops the connection and yields `error`.
    ///
    /// A websocket read that never yields returns `Poll::Pending` without scheduling a wake-up of
    /// its own, so this timer is the only thing that gets a silent connection polled again.
    fn park(
        &mut self,
        cx: &mut Context<'_>,
        error: &str,
    ) -> Poll<Option<eyre::Result<FlashBlock>>> {
        let deadline = self
            .idle_deadline
            .as_mut()
            .expect("idle deadline is armed while connecting or streaming");
        ready!(deadline.as_mut().poll(cx));
        self.disconnect();

        Poll::Ready(Some(Err(eyre::eyre!("{error} for {IDLE_TIMEOUT:?}"))))
    }

    /// Drops the connection, closing the socket, so the next poll reconnects.
    fn disconnect(&mut self) {
        // A handshake still in flight owns its socket too.
        self.connect = Box::pin(async move { Err(Error::ConnectionClosed)? });
        self.stream = None;
        self.sink = None;
        self.idle_deadline = None;
        self.state = State::Initial;
    }
}

impl<Str, S, C> Stream for WsFlashBlockStream<Str, S, C>
where
    Str: Stream<Item = Result<Message, Error>> + Unpin,
    S: Sink<Message> + Send + Unpin,
    C: WsConnect<Stream = Str, Sink = S> + Clone + Send + 'static + Unpin,
{
    type Item = eyre::Result<FlashBlock>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        let this = self.get_mut();

        loop {
            if this.state == State::Initial {
                this.connect();
            }

            if this.state == State::Connect {
                match this.connect.poll_unpin(cx) {
                    Poll::Ready(Ok((sink, stream))) => this.stream(sink, stream),
                    Poll::Ready(Err(err)) => {
                        this.disconnect();

                        return Poll::Ready(Some(Err(err)));
                    }
                    // A peer can stall the TLS or HTTP upgrade handshake indefinitely.
                    Poll::Pending => return this.park(cx, "Connection attempt made no progress"),
                }
            }

            while matches!(this.state, State::Stream(_)) {
                if matches!(this.state, State::Stream(Some(_))) {
                    let mut sink = Pin::new(
                        this.sink.as_mut().expect("Stream state is unreachable without sink"),
                    );
                    if sink.as_mut().poll_ready(cx).is_pending() {
                        return this.park(cx, "Connection idle");
                    }
                    if let State::Stream(queued) = &mut this.state &&
                        let Some(pong) = queued.take()
                    {
                        let _ = sink.as_mut().start_send(pong);
                    }
                    if sink.as_mut().poll_flush(cx).is_pending() {
                        return this.park(cx, "Connection idle");
                    }
                }

                let msg = match this
                    .stream
                    .as_mut()
                    .expect("Stream state should be unreachable without stream")
                    .poll_next_unpin(cx)
                {
                    Poll::Ready(Some(msg)) => msg,
                    // Reported as an error so the service backs off: a peer that closes right
                    // after every handshake would otherwise be reconnected in a tight loop.
                    Poll::Ready(None) => {
                        this.disconnect();

                        return Poll::Ready(Some(Err(eyre::eyre!(
                            "Connection closed by upstream"
                        ))));
                    }
                    Poll::Pending => return this.park(cx, "Connection idle"),
                };

                // Any item proves the connection is alive.
                this.reset_idle();

                match msg {
                    Ok(Message::Binary(bytes)) => {
                        return Poll::Ready(Some(this.decoder.decode(bytes)));
                    }
                    Ok(Message::Text(bytes)) => {
                        return Poll::Ready(Some(this.decoder.decode(bytes.into())));
                    }
                    Ok(Message::Ping(bytes)) => this.ping(bytes),
                    Ok(Message::Close(frame)) => this.close(frame),
                    Ok(msg) => {
                        debug!(target: "flashblocks", "Received unexpected message: {:?}", msg)
                    }
                    Err(err) => return Poll::Ready(Some(Err(err.into()))),
                }
            }
        }
    }
}

impl<Stream, S, C> WsFlashBlockStream<Stream, S, C>
where
    C: WsConnect<Stream = Stream, Sink = S> + Clone + Send + 'static,
{
    fn connect(&mut self) {
        let ws_url = self.ws_url.clone();
        let mut connector = self.connector.clone();

        Pin::new(&mut self.connect).set(Box::pin(async move { connector.connect(ws_url).await }));
        self.reset_idle();

        self.state = State::Connect;
    }

    fn stream(&mut self, sink: S, stream: Stream) {
        self.sink.replace(sink);
        self.stream.replace(stream);
        self.reset_idle();

        self.state = State::Stream(None);
    }

    fn ping(&mut self, pong: Bytes) {
        if let State::Stream(current) = &mut self.state {
            current.replace(Message::Pong(pong));
        }
    }

    fn close(&mut self, frame: Option<CloseFrame>) {
        if let State::Stream(current) = &mut self.state {
            current.replace(Message::Close(frame));
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
    Pin<Box<dyn Future<Output = eyre::Result<(Sink, Stream)>> + Send + 'static>>;

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
    ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send;
}

/// Establishes a secure websocket subscription.
///
/// See the [`WsConnect`] documentation for details.
#[derive(Debug, Clone)]
#[allow(unnameable_types)]
pub struct WsConnector;

impl WsConnect for WsConnector {
    type Stream = WsStream;
    type Sink = WsSink;

    async fn connect(&mut self, ws_url: Url) -> eyre::Result<(WsSink, WsStream)> {
        ensure_crypto_provider();

        let (stream, _response) = connect_async(ws_url.as_str()).await?;

        Ok(stream.split())
    }
}

/// Installs a process-level rustls [`CryptoProvider`] for `wss://` (TLS) connections.
///
/// `rustls` cannot automatically select a provider when more than one is present in the
/// dependency graph (both `aws-lc-rs` and `ring` are), so it panics on the first TLS handshake
/// unless a default has been installed. We install the `aws-lc-rs` provider once, but only if no
/// other component already installed one, so this stays idempotent and non-clobbering.
///
/// `aws-lc-rs` is chosen over `ring` because it is the provider most likely to have already been
/// installed by an upstream dependency (`rustls-webpki` and others pull it in). Since
/// [`install_default`] is process-global and first-wins, matching `aws-lc-rs` minimizes the chance
/// this lazy install races with and disagrees with whatever a dependency installed first.
///
/// [`CryptoProvider`]: rustls::crypto::CryptoProvider
/// [`install_default`]: rustls::crypto::CryptoProvider::install_default
fn ensure_crypto_provider() {
    static INIT: std::sync::LazyLock<()> = std::sync::LazyLock::new(|| {
        if rustls::crypto::CryptoProvider::get_default().is_none() {
            let _ = rustls::crypto::aws_lc_rs::default_provider().install_default();
        }
    });
    std::sync::LazyLock::force(&INIT);
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::bytes::Bytes;
    use brotli::enc::BrotliEncoderParams;
    use std::{future, iter, sync::Arc};
    use tokio::time;
    use tokio_tungstenite::tungstenite::{
        Error,
        protocol::frame::{Frame, coding::CloseCode},
    };

    #[test]
    fn ensure_crypto_provider_installs_a_default() {
        ensure_crypto_provider();
        assert!(
            rustls::crypto::CryptoProvider::get_default().is_some(),
            "a process-level CryptoProvider should be installed after the call"
        );
    }

    #[test]
    fn ensure_crypto_provider_is_idempotent() {
        // Must not panic on repeated calls, and the installed provider must not change.
        ensure_crypto_provider();
        let first = rustls::crypto::CryptoProvider::get_default();
        for _ in 0..5 {
            ensure_crypto_provider();
        }
        let after = rustls::crypto::CryptoProvider::get_default();
        assert!(
            std::ptr::eq(first.unwrap(), after.unwrap()),
            "repeated calls must not replace the provider"
        );
    }

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

    impl FakeStream {
        fn new(mut messages: Vec<Result<Message, Error>>) -> Self {
            messages.reverse();

            Self(messages)
        }
    }

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

    /// Simulates an upstream that delivers `messages` and then goes silent *without* closing the
    /// connection: it yields no further item and never wakes the task again.
    ///
    /// This is the half-open socket from <https://github.com/ethereum-optimism/optimism/issues/22816>.
    #[derive(Clone)]
    struct SilentStream(Vec<Message>);

    impl SilentStream {
        fn new(mut messages: Vec<Message>) -> Self {
            messages.reverse();

            Self(messages)
        }
    }

    impl Stream for SilentStream {
        type Item = Result<Message, Error>;

        fn poll_next(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
            self.get_mut().0.pop().map_or(Poll::Pending, |msg| Poll::Ready(Some(Ok(msg))))
        }
    }

    /// Simulates an upstream that delivers one message every `interval` and then goes silent.
    struct HeartbeatStream {
        messages: Vec<Message>,
        interval: Duration,
        next: Pin<Box<Sleep>>,
    }

    impl HeartbeatStream {
        fn new(mut messages: Vec<Message>, interval: Duration) -> Self {
            messages.reverse();

            Self { messages, interval, next: Box::pin(sleep(interval)) }
        }
    }

    impl Stream for HeartbeatStream {
        type Item = Result<Message, Error>;

        fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
            let this = self.get_mut();

            ready!(this.next.as_mut().poll(cx));

            let Some(msg) = this.messages.pop() else { return Poll::Pending };
            this.next.as_mut().reset(Instant::now() + this.interval);

            Poll::Ready(Some(Ok(msg)))
        }
    }

    /// Creates [`HeartbeatStream`]s.
    #[derive(Clone)]
    struct HeartbeatConnector {
        messages: Vec<Message>,
        interval: Duration,
    }

    impl WsConnect for HeartbeatConnector {
        type Stream = HeartbeatStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            future::ready(Ok((
                NoopSink,
                HeartbeatStream::new(self.messages.clone(), self.interval),
            )))
        }
    }

    /// Simulates a peer that accepts the connection attempt and then never completes the
    /// handshake, leaving the connect future pending with nothing to wake it.
    ///
    /// Each connect future holds a clone of the `Arc`, standing in for the socket it owns.
    #[derive(Clone)]
    struct HangingConnector(Arc<()>);

    impl WsConnect for HangingConnector {
        type Stream = FakeStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            let socket = self.0.clone();

            async move {
                let _socket = socket;
                future::pending().await
            }
        }
    }

    /// Creates [`SilentStream`]s.
    #[derive(Clone)]
    struct SilentConnector(SilentStream);

    impl WsConnect for SilentConnector {
        type Stream = SilentStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            future::ready(Ok((NoopSink, self.0.clone())))
        }
    }

    impl WsConnect for FakeConnector {
        type Stream = FakeStream;
        type Sink = NoopSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            future::ready(Ok((NoopSink, self.0.clone())))
        }
    }

    impl<T: IntoIterator<Item = Result<Message, Error>>> From<T> for FakeConnector {
        fn from(value: T) -> Self {
            Self(FakeStream::new(value.into_iter().collect()))
        }
    }

    impl WsConnect for FakeConnectorWithSink {
        type Stream = FakeStream;
        type Sink = FakeSink;

        fn connect(
            &mut self,
            _ws_url: Url,
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            future::ready(Ok((FakeSink::default(), self.0.clone())))
        }
    }

    impl<T: IntoIterator<Item = Result<Message, Error>>> From<T> for FakeConnectorWithSink {
        fn from(value: T) -> Self {
            Self(FakeStream::new(value.into_iter().collect()))
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
        ) -> impl Future<Output = eyre::Result<(Self::Sink, Self::Stream)>> + Send {
            future::ready(Err(eyre::eyre!("{}", &self.0)))
        }
    }

    fn to_json_message<B: TryFrom<Bytes, Error: Debug>, F: Fn(B) -> Message>(
        wrapper_f: F,
    ) -> impl Fn(&FlashBlock) -> Result<Message, Error> + use<F, B> {
        move |block| to_json_message_using(block, &wrapper_f)
    }

    fn to_json_binary_message(block: &FlashBlock) -> Result<Message, Error> {
        to_json_message_using(block, Message::Binary)
    }

    fn to_json_message_using<B: TryFrom<Bytes, Error: Debug>, F: Fn(B) -> Message>(
        block: &FlashBlock,
        wrapper_f: F,
    ) -> Result<Message, Error> {
        Ok(wrapper_f(B::try_from(Bytes::from(serde_json::to_vec(block).unwrap())).unwrap()))
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

    fn flashblock() -> FlashBlock {
        Default::default()
    }

    #[test_case::test_case(to_json_message(Message::Binary); "json binary")]
    #[test_case::test_case(to_json_message(Message::Text); "json UTF-8")]
    #[test_case::test_case(to_brotli_message; "brotli")]
    #[tokio::test]
    async fn test_stream_decodes_messages_successfully(
        to_message: impl Fn(&FlashBlock) -> Result<Message, Error>,
    ) {
        let flashblocks = [flashblock()];
        let connector = FakeConnector::from(flashblocks.iter().map(to_message));
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let actual_messages: Vec<_> = stream.take(1).map(Result::unwrap).collect().await;
        let expected_messages = flashblocks.to_vec();

        assert_eq!(actual_messages, expected_messages);
    }

    #[test_case::test_case(Message::Pong(Bytes::from(b"test".as_slice())); "pong")]
    #[test_case::test_case(Message::Frame(Frame::pong(b"test".as_slice())); "frame")]
    #[tokio::test]
    async fn test_stream_ignores_unexpected_message(message: Message) {
        let flashblock = flashblock();
        let connector = FakeConnector::from([Ok(message), to_json_binary_message(&flashblock)]);
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let expected_message = flashblock;
        let actual_message =
            stream.next().await.expect("Binary message should not be ignored").unwrap();

        assert_eq!(actual_message, expected_message)
    }

    #[tokio::test]
    async fn test_stream_passes_errors_through() {
        let connector = FakeConnector::from([Err(Error::AttackAttempt)]);
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let actual_messages: Vec<_> =
            stream.take(1).map(Result::unwrap_err).map(|e| format!("{e}")).collect().await;
        let expected_messages = vec!["Attack attempt detected".to_owned()];

        assert_eq!(actual_messages, expected_messages);
    }

    #[tokio::test]
    async fn test_connect_error_causes_retries() {
        let tries = 3;
        let error_msg = "test".to_owned();
        let connector = FailingConnector(error_msg.clone());
        let ws_url = "http://localhost".parse().unwrap();
        let stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let actual_errors: Vec<_> =
            stream.take(tries).map(Result::unwrap_err).map(|e| format!("{e}")).collect().await;
        let expected_errors: Vec<_> = iter::repeat_n(error_msg, tries).collect();

        assert_eq!(actual_errors, expected_errors);
    }

    #[test_case::test_case(
        Message::Close(Some(CloseFrame { code: CloseCode::Normal, reason: "test".into() })),
        Message::Close(Some(CloseFrame { code: CloseCode::Normal, reason: "test".into() }));
        "close"
    )]
    #[test_case::test_case(
        Message::Ping(Bytes::from_static(&[1u8, 2, 3])),
        Message::Pong(Bytes::from_static(&[1u8, 2, 3]));
        "ping"
    )]
    #[tokio::test]
    async fn test_stream_responds_to_messages(msg: Message, expected_response: Message) {
        let flashblock = flashblock();
        let messages = [Ok(msg), to_json_binary_message(&flashblock)];
        let connector = FakeConnectorWithSink::from(messages);
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let _ = stream.next().await;

        let expected_response = vec![expected_response];
        let FakeSink(actual_buffer, actual_response) = stream.sink.unwrap();

        assert!(actual_buffer.is_none(), "buffer not flushed: {actual_buffer:#?}");
        assert_eq!(actual_response, expected_response);
    }

    #[tokio::test(start_paused = true)]
    async fn test_stream_reconnects_when_the_upstream_goes_silent() {
        let flashblock = flashblock();
        let connector =
            SilentConnector(SilentStream::new(vec![to_json_binary_message(&flashblock).unwrap()]));
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        assert_eq!(stream.next().await.expect("stream should not end").unwrap(), flashblock);

        // The upstream is now silent while the connection stays open, so the inner websocket read
        // parks without ever waking us again. Only the idle deadline can recover from this.
        let received = time::timeout(Duration::from_secs(60), stream.next())
            .await
            .expect("a silent upstream should not park the stream forever");
        let err = received.expect("stream should not end").unwrap_err();

        assert!(err.to_string().contains("idle"), "unexpected error: {err}");
        assert!(stream.stream.is_none(), "the dead connection should be dropped, not held open");
        assert_eq!(stream.state, State::Initial, "the next poll should reconnect");
    }

    #[tokio::test(start_paused = true)]
    async fn test_stream_gives_up_on_a_handshake_that_never_completes() {
        let socket = Arc::new(());
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream =
            WsFlashBlockStream::with_connector(ws_url, HangingConnector(socket.clone()));

        let received = time::timeout(Duration::from_secs(60), stream.next())
            .await
            .expect("a handshake that never completes should not park the stream forever");
        let err = received.expect("stream should not end").unwrap_err();

        assert!(err.to_string().contains("no progress"), "unexpected error: {err}");
        // Only the test's and the connector's handles remain.
        assert_eq!(Arc::strong_count(&socket), 2, "the stalled handshake should be dropped");
    }

    #[tokio::test]
    async fn test_stream_end_is_reported_as_an_error() {
        let flashblock = flashblock();
        let connector = FakeConnector::from([to_json_binary_message(&flashblock)]);
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        assert_eq!(stream.next().await.expect("stream should not end").unwrap(), flashblock);

        let err = stream.next().await.expect("stream should not end").unwrap_err();

        assert!(err.to_string().contains("closed by upstream"), "unexpected error: {err}");
        assert_eq!(stream.state, State::Initial, "the next poll should reconnect");
    }

    #[tokio::test(start_paused = true)]
    async fn test_slow_consumer_is_not_mistaken_for_a_silent_upstream() {
        let flashblock = flashblock();
        let connector = SilentConnector(SilentStream::new(vec![
            Message::Binary(Bytes::from_static(b"not a flashblock")),
            to_json_binary_message(&flashblock).unwrap(),
        ]));
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        assert!(stream.next().await.expect("stream should not end").is_err());

        // The service backs off after a stream error without polling the stream, while the next
        // frame waits in the socket.
        time::sleep(Duration::from_secs(10)).await;

        assert_eq!(stream.next().await.expect("stream should not end").unwrap(), flashblock);
    }

    #[tokio::test(start_paused = true)]
    async fn test_idle_deadline_restarts_on_every_frame() {
        let flashblock = flashblock();
        let connector = HeartbeatConnector {
            // The flashblock only arrives after 9s, well past the idle timeout, but each pong in
            // front of it restarts the deadline.
            messages: vec![
                Message::Pong(Bytes::from_static(b"1")),
                Message::Pong(Bytes::from_static(b"2")),
                to_json_binary_message(&flashblock).unwrap(),
            ],
            interval: Duration::from_secs(3),
        };
        let ws_url = "http://localhost".parse().unwrap();
        let mut stream = WsFlashBlockStream::with_connector(ws_url, connector);

        let received = stream.next().await.expect("stream should not end").unwrap();

        assert_eq!(
            received, flashblock,
            "traffic within the idle timeout should keep the connection alive"
        );
    }
}
