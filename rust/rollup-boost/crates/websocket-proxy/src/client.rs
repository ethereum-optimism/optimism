use crate::rate_limit::Ticket;
use axum::extract::ws::WebSocket;
use std::net::IpAddr;

pub struct ClientConnection {
    client_addr: IpAddr,
    _ticket: Ticket,
    pub(crate) websocket: WebSocket,
}

impl ClientConnection {
    pub fn new(client_addr: IpAddr, ticket: Ticket, websocket: WebSocket) -> Self {
        Self {
            client_addr,
            _ticket: ticket,
            websocket,
        }
    }

    pub fn id(&self) -> String {
        self.client_addr.to_string()
    }
}
