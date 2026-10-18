use kona_engine::EngineQueries;

/// Read-only request, retained until the RPC wrapper is removed.
#[derive(Debug)]
pub struct EngineRpcRequest(pub Box<EngineQueries>);
