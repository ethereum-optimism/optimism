//! Tests for starting a driver: where it listens, what it refuses to speak, and which
//! discovered peers it would dial.

use super::*;
use crate::{GossipTransport, StartError, builder::tests::memory_transport};
use futures::future;
use libp2p::{
    StreamProtocol, Transport, TransportError,
    core::{
        muxing::StreamMuxerBox,
        transport::{self, TransportEvent},
    },
};
use std::{
    pin::Pin,
    task::{Context, Poll},
};

/// A started driver reports the address it listens on, where another driver can reach it.
#[tokio::test]
async fn test_peer_addr_is_where_the_driver_listens() {
    let mut driver_1 = memory_driver().start().await.expect("listen");
    let addr_1 = driver_1.peer_addr();
    assert_eq!(driver_1.peer_addr(), addr_1);

    let mut driver_2 = memory_driver().start().await.expect("listen");
    driver_2.dial_multiaddr(addr_1).expect("dial");
    poll_until_connected(&mut driver_1, &mut driver_2).await;
}

/// A transport whose listeners close with an error before they report an address.
///
/// The memory transport always reports the address first, so it can't test this.
#[derive(Default)]
struct ClosingTransport {
    /// The listener to close on the next poll.
    listener_id: Option<ListenerId>,
}

impl Transport for ClosingTransport {
    type Output = (PeerId, StreamMuxerBox);
    type Error = std::io::Error;
    type ListenerUpgrade = future::Pending<Result<Self::Output, Self::Error>>;
    type Dial = future::Pending<Result<Self::Output, Self::Error>>;

    fn listen_on(
        &mut self,
        id: ListenerId,
        _addr: Multiaddr,
    ) -> Result<(), TransportError<Self::Error>> {
        self.listener_id = Some(id);
        Ok(())
    }

    fn remove_listener(&mut self, _id: ListenerId) -> bool {
        false
    }

    fn dial(
        &mut self,
        addr: Multiaddr,
        _opts: transport::DialOpts,
    ) -> Result<Self::Dial, TransportError<Self::Error>> {
        Err(TransportError::MultiaddrNotSupported(addr))
    }

    fn poll(
        mut self: Pin<&mut Self>,
        _cx: &mut Context<'_>,
    ) -> Poll<TransportEvent<Self::ListenerUpgrade, Self::Error>> {
        self.listener_id.take().map_or(Poll::Pending, |listener_id| {
            Poll::Ready(TransportEvent::ListenerClosed {
                listener_id,
                reason: Err(std::io::Error::other("listener failed")),
            })
        })
    }
}

/// A listener that closes before it reports an address fails the start, instead of leaving
/// it waiting for an address that never comes.
#[tokio::test]
async fn test_start_fails_when_listener_closes() {
    let starter = memory_driver_builder()
        .with_transport(GossipTransport::Other(Box::new(|_| {
            Ok(ClosingTransport::default().boxed())
        })))
        .build()
        .expect("build");

    let start = tokio::time::timeout(Duration::from_secs(10), starter.start()).await;
    let Ok(Err(StartError::ListenerClosed(Some(reason)))) = start else {
        panic!("expected the listener to close, got {start:?}");
    };
    assert_eq!(reason.to_string(), "listener failed");
}

/// A driver whose listener has already closed is dropped like any other: removing the
/// listener again does nothing.
#[tokio::test]
async fn test_driver_with_a_closed_listener_drops() {
    let mut driver = memory_driver().start().await.expect("listen");
    assert!(driver.swarm.remove_listener(driver.listener_id));
    poll_one_until(&mut driver, |driver| driver.swarm.listeners().next().is_none()).await;
    drop(driver);
}

/// kona doesn't serve op-node's retired sync request/response protocol, `payload_by_number`.
/// A peer that still asks for it is told it isn't supported, so it can't send the node a
/// single byte on it, let alone the 128 MiB it once could.
#[tokio::test]
async fn test_sync_request_protocol_is_unsupported() {
    let mut driver = memory_driver().start().await.expect("listen");
    let peer_id = *driver.local_peer_id();

    let GossipTransport::Other(transport) = memory_transport() else {
        unreachable!("the memory transport is a custom one");
    };
    let mut peer = libp2p::SwarmBuilder::with_new_identity()
        .with_tokio()
        .with_other_transport(transport)
        .expect("transport")
        .with_behaviour(|_| libp2p_stream::Behaviour::new())
        .expect("behaviour")
        .build();
    peer.dial(Multiaddr::from(driver.peer_addr())).expect("dial");

    let chain_id = driver.handler().rollup_config.l2_chain_id;
    let protocol =
        StreamProtocol::try_from_owned(format!("/opstack/req/payload_by_number/{chain_id}/0/"))
            .expect("protocol");
    let mut control = peer.behaviour().new_control();
    let mut open = std::pin::pin!(control.open_stream(peer_id, protocol));
    let opened = tokio::time::timeout(Duration::from_secs(10), async {
        loop {
            tokio::select! {
                opened = &mut open => break opened,
                Some(event) = driver.next() => { driver.handle_event(event); }
                _ = peer.select_next_some() => {}
            }
        }
    })
    .await
    .expect("the peer gets an answer in time");

    assert!(
        matches!(opened, Err(libp2p_stream::OpenStreamError::UnsupportedProtocol(_))),
        "{opened:?}"
    );
}
