use core::{
    fmt::{Debug, Formatter},
    net::SocketAddr,
    pin::Pin,
    sync::atomic::{AtomicUsize, Ordering},
    task::{Context, Poll},
};
use futures::{Sink, SinkExt, StreamExt};
use op_alloy_rpc_types_engine::OpFlashblockPayload;
use std::{io, net::TcpListener, sync::Arc};
use tokio::{
    net::TcpStream,
    sync::{
        broadcast::{self, Receiver, error::RecvError},
        watch,
    },
};
use tokio_tungstenite::WebSocketStream;
use tokio_tungstenite::tungstenite::Utf8Bytes;
use tokio_tungstenite::{accept_async, tungstenite::Message};

/// A WebSockets publisher that accepts connections from client websockets and broadcasts to them
/// updates about new flashblocks. It maintains a count of sent messages and active subscriptions.
///
/// This is modelled as a `futures::Sink` that can be used to send `OpFlashblockPayload` messages.
pub struct WebSocketPublisher {
    sent: Arc<AtomicUsize>,
    subs: Arc<AtomicUsize>,
    term: watch::Sender<bool>,
    pipe: broadcast::Sender<Utf8Bytes>,
}

impl WebSocketPublisher {
    pub fn new(addr: SocketAddr) -> io::Result<Self> {
        let (pipe, _) = broadcast::channel(100);
        let (term, _) = watch::channel(false);

        let sent = Arc::new(AtomicUsize::new(0));
        let subs = Arc::new(AtomicUsize::new(0));
        let listener = TcpListener::bind(addr)?;

        tokio::spawn(listener_loop(
            listener,
            pipe.subscribe(),
            term.subscribe(),
            Arc::clone(&sent),
            Arc::clone(&subs),
        ));

        Ok(Self {
            sent,
            subs,
            term,
            pipe,
        })
    }

    pub fn publish(&self, payload: &OpFlashblockPayload) -> io::Result<()> {
        // Serialize the payload to a UTF-8 string
        // serialize only once, then just copy around only a pointer
        // to the serialized data for each subscription.
        let serialized = serde_json::to_string(payload)?;
        let utf8_bytes = Utf8Bytes::from(serialized);

        // Send the serialized payload to all subscribers
        self.pipe
            .send(utf8_bytes)
            .map_err(|e| io::Error::new(io::ErrorKind::ConnectionAborted, e))?;
        Ok(())
    }
}

impl Drop for WebSocketPublisher {
    fn drop(&mut self) {
        // Notify the listener loop to terminate
        let _ = self.term.send(true);
        tracing::info!("WebSocketPublisher dropped, terminating listener loop");
    }
}

async fn listener_loop(
    listener: TcpListener,
    receiver: Receiver<Utf8Bytes>,
    term: watch::Receiver<bool>,
    sent: Arc<AtomicUsize>,
    subs: Arc<AtomicUsize>,
) {
    listener
        .set_nonblocking(true)
        .expect("Failed to set TcpListener socket to non-blocking");

    let listener = tokio::net::TcpListener::from_std(listener)
        .expect("Failed to convert TcpListener to tokio TcpListener");

    let listen_addr = listener
        .local_addr()
        .expect("Failed to get local address of listener");
    tracing::info!("Flashblocks WebSocketPublisher listening on {listen_addr}");

    let mut term = term;

    loop {
        let subs = Arc::clone(&subs);

        tokio::select! {
            // drop this connection if the `WebSocketPublisher` is dropped
            _ = term.changed() => {
                if *term.borrow() {
                    return;
                }
            }

            // Accept new connections on the websocket listener
            // when a new connection is established, spawn a dedicated task to handle
            // the connection and broadcast with that connection.
            Ok((connection, peer_addr)) = listener.accept() => {
                let sent = Arc::clone(&sent);
                let term = term.clone();
                let receiver_clone = receiver.resubscribe();

                match accept_async(connection).await {
                    Ok(stream) => {
                        tokio::spawn(async move {
                            subs.fetch_add(1, Ordering::Relaxed);
                            tracing::debug!("WebSocket connection established with {}", peer_addr);

                            // Handle the WebSocket connection in a dedicated task
                            broadcast_loop(stream, term, receiver_clone, sent).await;

                            subs.fetch_sub(1, Ordering::Relaxed);
                            tracing::debug!("WebSocket connection closed for {}", peer_addr);
                        });
                    }
                    Err(e) => {
                        tracing::warn!("Failed to accept WebSocket connection from {peer_addr}: {e}");
                    }
                }
            }
        }
    }
}

/// An instance of this loop is spawned for each connected WebSocket client.
/// It listens for broadcast updates about new flashblocks and sends them to the client.
/// It also handles termination signals to gracefully close the connection.
/// Any connectivity errors will terminate the loop, which will in turn
/// decrement the subscription count in the `WebSocketPublisher`.
async fn broadcast_loop(
    stream: WebSocketStream<TcpStream>,
    term: watch::Receiver<bool>,
    blocks: broadcast::Receiver<Utf8Bytes>,
    sent: Arc<AtomicUsize>,
) {
    let mut term = term;
    let mut blocks = blocks;
    let Ok(peer_addr) = stream.get_ref().peer_addr() else {
        return;
    };
    let (mut sink, mut stream_read) = stream.split();

    loop {
        tokio::select! {
            // Check if the publisher is terminated
            _ = term.changed() => {
                if *term.borrow() {
                    tracing::info!("WebSocketPublisher is terminating, closing broadcast loop");
                    return;
                }
            }

            // Handle incoming WebSocket messages (including pings)
            msg = stream_read.next() => {
                match msg {
                    Some(Ok(_)) => {
                        // Ignore all inbound frames.
                        // Tungstenite will auto-respond to Ping and handle Close internally.
                    }
                    Some(Err(e)) => {
                        tracing::debug!("WebSocket error from {peer_addr}: {e}");
                        break;
                    }
                    None => {
                        tracing::debug!("WebSocket stream ended for {peer_addr}");
                        break;
                    }
                }
            }

            // Receive payloads from the broadcast channel
            payload = blocks.recv() => match payload {
                Ok(payload) => {
                    // Here you would typically send the payload to the WebSocket clients.
                    // For this example, we just increment the sent counter.
                    sent.fetch_add(1, Ordering::Relaxed);

                    tracing::debug!("Broadcasted payload: {:?}", payload);
                    if let Err(e) = sink.send(Message::Text(payload)).await {
                        tracing::debug!("Closing flashblocks subscription for {peer_addr}: {e}");
                        break; // Exit the loop if sending fails
                    }
                }
                Err(RecvError::Closed) => {
                    tracing::debug!("Broadcast channel closed, exiting broadcast loop");
                    return;
                }
                Err(RecvError::Lagged(_)) => {
                    tracing::warn!("Broadcast channel lagged, some messages were dropped");
                }
            },
        }
    }
}

impl Debug for WebSocketPublisher {
    fn fmt(&self, f: &mut Formatter<'_>) -> core::fmt::Result {
        let subs = self.subs.load(Ordering::Relaxed);
        let sent = self.sent.load(Ordering::Relaxed);

        f.debug_struct("WebSocketPublisher")
            .field("subs", &subs)
            .field("payloads_sent", &sent)
            .finish()
    }
}

impl Sink<&OpFlashblockPayload> for WebSocketPublisher {
    type Error = eyre::Report;

    fn poll_ready(self: Pin<&mut Self>, _: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        Poll::Ready(Ok(()))
    }

    fn start_send(self: Pin<&mut Self>, item: &OpFlashblockPayload) -> Result<(), Self::Error> {
        self.publish(item)?;
        Ok(())
    }

    fn poll_flush(self: Pin<&mut Self>, _: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        Poll::Ready(Ok(()))
    }

    fn poll_close(self: Pin<&mut Self>, _: &mut Context<'_>) -> Poll<Result<(), Self::Error>> {
        Poll::Ready(Ok(()))
    }
}
