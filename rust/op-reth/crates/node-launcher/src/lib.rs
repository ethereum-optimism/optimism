//! Non-generic launch of op-reth's own node.
//!
//! Launching a reth node compiles reth's launch stack (engine tree, pipeline, RPC registry,
//! network, payload service, ExEx manager) for the concrete node type, in the crate that names
//! that type. Every launch of [`OpRethNode`] goes through this crate's non-generic functions, so
//! that stack is compiled once, here: not in `reth-optimism-node`, and not again in the `op-reth`
//! binary or the tests that launch op-reth's node. The futures are boxed, so callers do not
//! compile their `poll` code either.
//!
//! A binary that launches a [`CustomOpNode`] with a replaced pool builder or payload service
//! compiles that launch itself, through [`LaunchOpNode`].
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use futures_util::future::BoxFuture;
use reth_db::DatabaseEnv;
use reth_node_builder::{
    Node, NodeAdapter, NodeBuilder, NodeBuilderWithComponents, NodeHandle, RethFullAdapter,
    WithLaunchContext,
};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::{
    CustomOpNode, LaunchOpNode, OpLaunchHooks, OpNode, OpNodeComponentBuilder,
    OpPayloadServiceBuilder, OpPoolBuilder,
};
use std::future::IntoFuture;

/// The node op-reth launches: an [`OpNode`] with the pool builder and payload service it installs
/// itself.
pub type OpRethNode = CustomOpNode<OpPoolBuilder, OpPayloadServiceBuilder>;

/// The node types, database and provider of a launched [`OpRethNode`].
type OpRethFullAdapter = RethFullAdapter<DatabaseEnv, OpRethNode>;

/// The components of a launched [`OpRethNode`].
pub type OpRethAdapter = NodeAdapter<OpRethFullAdapter>;

/// The add-ons of a launched [`OpRethNode`].
pub type OpRethAddOns = <OpRethNode as Node<OpRethFullAdapter>>::AddOns;

/// A node builder with the components and add-ons of an [`OpRethNode`], ready to [`launch`].
pub type OpRethLaunchBuilder = WithLaunchContext<
    NodeBuilderWithComponents<
        OpRethFullAdapter,
        OpNodeComponentBuilder<OpRethFullAdapter>,
        OpRethAddOns,
    >,
>;

/// Handle to a launched [`OpRethNode`].
pub type OpRethNodeHandle = NodeHandle<OpRethAdapter, OpRethAddOns>;

/// Launches the node with reth's debug capabilities, which are inert unless configured.
///
/// This is reth's launch without op-reth's additions: unlike [`launch_node`], it does not reject
/// state masking, install proof history, warn about Storage V1 or run [`OpLaunchHooks`], and it
/// returns once the node runs. It shares [`launch_node`]'s compiled code, so tests can launch the
/// node the binary launches without compiling reth's launch stack themselves.
pub fn launch(builder: OpRethLaunchBuilder) -> BoxFuture<'static, eyre::Result<OpRethNodeHandle>> {
    builder.launch_with_debug_capabilities().into_future()
}

/// Launches `node` the way op-reth does, then waits for it to exit.
///
/// This optionally installs proof history and fails if the engine config enables state masking;
/// see [`LaunchOpNode`].
pub fn launch_node(
    builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
    node: OpNode,
    hooks: OpLaunchHooks,
) -> BoxFuture<'static, eyre::Result<()>> {
    Box::pin(OpRethNode::from(node).launch(builder, hooks))
}
