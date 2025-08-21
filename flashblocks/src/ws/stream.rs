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
    tungstenite::{handshake::client::Response, Error, Message},
    MaybeTlsStream, WebSocketStream,
};
use url::Url;

/// An asynchronous stream of [`FlashBlock`] from a websocket connection.
///
/// The stream attempts to connect to a websocket URL and then decode each received item.
///
/// If the connection fails, the error is returned and connection retried. The number of retries is
/// unbounded.
pub struct FlashBlockWsStream {
    ws_url: Url,
    state: State,
    connect: ConnectFuture,
    stream: Option<SplitStream<WebSocketStream<MaybeTlsStream<TcpStream>>>>,
}

impl Stream for FlashBlockWsStream {
    type Item = eyre::Result<FlashBlock>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        if self.state == State::Initial {
            self.connect();
        }

        if self.state == State::Connect {
            match ready!(self.connect.poll_unpin(cx)) {
                Ok((stream, _)) => self.stream(stream),
                Err(err) => {
                    self.state = State::Initial;

                    return Poll::Ready(Some(Err(err.into())))
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

impl FlashBlockWsStream {
    fn connect(&mut self) {
        let ws_url = self.ws_url.clone();

        Pin::new(&mut self.connect)
            .set(Box::pin(async move { connect_async(ws_url.as_str()).await }));

        self.state = State::Connect;
    }

    fn stream(&mut self, stream: WebSocketStream<MaybeTlsStream<TcpStream>>) {
        self.stream.replace(stream.split().1);

        self.state = State::Stream;
    }
}

impl Debug for FlashBlockWsStream {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FlashBlockStream")
            .field("ws_url", &self.ws_url)
            .field("state", &self.state)
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

type ConnectFuture = Pin<
    Box<
        dyn Future<Output = Result<(WebSocketStream<MaybeTlsStream<TcpStream>>, Response), Error>>
            + Send
            + Sync
            + 'static,
    >,
>;
