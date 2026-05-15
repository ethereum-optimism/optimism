use crate::Message;
use eyre::Context;
use futures::stream::FuturesUnordered;
use libp2p::{PeerId, StreamProtocol, swarm::Stream};
use std::collections::HashMap;
use tracing::{debug, warn};

pub(crate) struct StreamsHandler {
    peers_to_stream: HashMap<PeerId, HashMap<StreamProtocol, Stream>>,
}

impl StreamsHandler {
    pub(crate) fn new() -> Self {
        Self {
            peers_to_stream: HashMap::new(),
        }
    }

    pub(crate) fn has_peer(&self, peer: &PeerId) -> bool {
        self.peers_to_stream.contains_key(peer)
    }

    pub(crate) fn insert_peer_and_stream(
        &mut self,
        peer: PeerId,
        protocol: StreamProtocol,
        stream: Stream,
    ) {
        self.peers_to_stream
            .entry(peer)
            .or_default()
            .insert(protocol, stream);
    }

    pub(crate) fn remove_peer(&mut self, peer: &PeerId) {
        self.peers_to_stream.remove(peer);
    }

    pub(crate) async fn broadcast_message<M: Message>(&mut self, message: M) -> eyre::Result<()> {
        use futures::{SinkExt as _, StreamExt as _};
        use tokio_util::{
            codec::{FramedWrite, LinesCodec},
            compat::FuturesAsyncReadCompatExt as _,
        };

        let protocol = message.protocol();
        let payload = message
            .to_string()
            .wrap_err("failed to serialize payload")?;

        let peers = self.peers_to_stream.keys().cloned().collect::<Vec<_>>();
        let mut futures = FuturesUnordered::new();
        for peer in peers {
            let protocol_to_stream = self
                .peers_to_stream
                .get_mut(&peer)
                .expect("stream map must exist for peer");
            let Some(stream) = protocol_to_stream.remove(&protocol) else {
                warn!("no stream for protocol {protocol:?} to peer {peer}");
                continue;
            };
            let stream = stream.compat();
            let payload = payload.clone();
            let fut = async move {
                let mut writer = FramedWrite::new(stream, LinesCodec::new());
                writer
                    .send(payload)
                    .await
                    .wrap_err("failed to send message to peer")?;
                Ok::<(PeerId, libp2p::swarm::Stream), eyre::ErrReport>((
                    peer,
                    writer.into_inner().into_inner(),
                ))
            };
            futures.push(fut);
        }

        while let Some(result) = futures.next().await {
            match result {
                Ok((peer, stream)) => {
                    let protocol_to_stream = self
                        .peers_to_stream
                        .get_mut(&peer)
                        .expect("stream map must exist for peer");
                    protocol_to_stream.insert(protocol.clone(), stream);
                }
                Err(e) => {
                    warn!("failed to send payload to peer: {e:?}");
                }
            }
        }

        debug!(
            "broadcasted message to {} peers",
            self.peers_to_stream.len()
        );

        Ok(())
    }
}
