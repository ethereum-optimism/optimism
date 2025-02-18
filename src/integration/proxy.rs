use bytes::Bytes;
use http::header;
use http_body_util::{combinators::BoxBody, BodyExt, Full};
use hyper::client::conn::http1::Builder;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response};
use hyper_util::rt::TokioIo;
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::net::{TcpListener, TcpStream};

#[derive(Debug, Serialize, Deserialize, Clone)]
struct JsonRpcRequest {
    jsonrpc: String,
    method: String,
    params: Value,
    id: Value,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct JsonRpcResponse {
    jsonrpc: String,
    #[serde(default)]
    result: Option<Value>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    error: Option<JsonRpcError>,
    id: Value,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct JsonRpcError {
    code: i32,
    message: String,
    data: Option<Value>,
}

pub type DynHandlerFn = Box<dyn Fn(&str, Value, Value) -> Option<Value> + Send + Sync>;

// Structure to hold the target address that we'll pass to the proxy function
#[derive(Clone)]
struct ProxyConfig {
    target_addr: SocketAddr,
    handler: Arc<DynHandlerFn>,
}

async fn proxy(
    config: ProxyConfig,
    req: Request<hyper::body::Incoming>,
) -> Result<Response<BoxBody<Bytes, hyper::Error>>, Box<dyn std::error::Error + Send + Sync>> {
    let (parts, body) = req.into_parts();
    let bytes = body.collect().await?.to_bytes();

    let json_rpc_request = serde_json::from_slice::<JsonRpcRequest>(&bytes).unwrap();
    let req = Request::from_parts(parts, Full::new(bytes));

    let stream = TcpStream::connect(config.target_addr).await?;
    let io = TokioIo::new(stream);

    let (mut sender, conn) = Builder::new()
        .preserve_header_case(true)
        .title_case_headers(true)
        .handshake(io)
        .await?;

    tokio::task::spawn(async move {
        if let Err(err) = conn.await {
            println!("Connection failed: {:?}", err);
        }
    });

    let resp = sender.send_request(req).await?;

    let (parts, body) = resp.into_parts();
    let bytes = body.collect().await?.to_bytes();

    let json_rpc_response = serde_json::from_slice::<JsonRpcResponse>(&bytes).unwrap();
    let bytes = if let Some(result) = json_rpc_response.clone().result {
        let value = (config.handler)(&json_rpc_request.method, json_rpc_request.params, result);
        if let Some(value) = value {
            // If the handler returns a value, we replace the result with the new value
            // The callback only returns the result of the jsonrpc request so we have to wrap it up
            // again in a JsonRpcResponse
            let mut new_json_rpc_resp = json_rpc_response;
            new_json_rpc_resp.result = Some(value);
            Bytes::from(serde_json::to_vec(&new_json_rpc_resp).unwrap())
        } else {
            // If the handler returns None, we return the original response
            bytes
        }
    } else {
        bytes
    };

    let bytes_len = bytes.len();
    let mut resp = Response::from_parts(parts, Full::new(bytes).map_err(|_| unreachable!()));

    // We have to update the content length to the new bytes length
    resp.headers_mut()
        .insert(header::CONTENT_LENGTH, bytes_len.into());

    Ok(resp.map(|b| b.boxed()))
}

pub async fn start_proxy_server(
    handler: DynHandlerFn,
    listen_port: u16,
    target_port: u16,
) -> Result<(), Box<dyn std::error::Error>> {
    let listen_addr = SocketAddr::from(([127, 0, 0, 1], listen_port));
    let target_addr = SocketAddr::from(([127, 0, 0, 1], target_port));

    let config = ProxyConfig {
        target_addr,
        handler: Arc::new(handler),
    };
    let listener = TcpListener::bind(listen_addr).await?;

    tokio::spawn(async move {
        loop {
            let (stream, _) = listener.accept().await.unwrap();
            let io = TokioIo::new(stream);
            let config = config.clone();

            tokio::task::spawn(async move {
                if let Err(err) = http1::Builder::new()
                    .preserve_header_case(true)
                    .title_case_headers(true)
                    .serve_connection(io, service_fn(move |req| proxy(config.clone(), req)))
                    .await
                {
                    println!("Failed to serve connection: {:?}", err);
                }
            });
        }
    });

    Ok(())
}
