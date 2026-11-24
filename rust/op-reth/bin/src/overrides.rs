//! Startup overrides for a binary that embeds op-reth.

use futures_util::{FutureExt, future::LocalBoxFuture};
use reth_cli_commands::node::NoArgs;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{node_config::NodeConfig, version::RethCliVersionConsts};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::{LaunchOpNode, OpLaunchHooks, OpNode, args::RollupArgs};
use std::{fmt, ops::AsyncFnOnce};

/// The arguments of the `node` command: op-reth's own, then those of the binary that embeds it.
#[derive(Debug, Clone, clap::Args)]
pub struct OpRethArgs<Ext: clap::Args = NoArgs> {
    /// op-reth's rollup arguments.
    #[command(flatten)]
    pub rollup: RollupArgs,
    /// The embedding binary's arguments.
    #[command(flatten)]
    pub ext: Ext,
}

/// Maps the configured node and returns it with its launch hooks.
type NodeMapper<Ext> = Box<
    dyn for<'a> FnOnce(
        &'a mut NodeConfig<OpChainSpec>,
        Ext,
        OpNode,
    )
        -> LocalBoxFuture<'a, eyre::Result<(Box<dyn MappedOpNode>, OpLaunchHooks)>>,
>;

/// Holds a mapped OP node without exposing its builder types on `OpRethOverrides`.
pub(crate) trait MappedOpNode {
    fn launch(
        self: Box<Self>,
        builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
        hooks: OpLaunchHooks,
    ) -> LocalBoxFuture<'static, eyre::Result<()>>;
}

impl<Pool, Payload> MappedOpNode for OpNode<Pool, Payload>
where
    Self: LaunchOpNode + 'static,
{
    fn launch(
        self: Box<Self>,
        builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
        hooks: OpLaunchHooks,
    ) -> LocalBoxFuture<'static, eyre::Result<()>> {
        LaunchOpNode::launch(*self, builder, hooks).boxed_local()
    }
}

/// Startup overrides that [`main_with`](crate::main_with) applies.
///
/// The default overrides run op-reth itself. `Ext` holds the extra `node` arguments.
pub struct OpRethOverrides<Ext = NoArgs> {
    /// The build metadata to report instead of op-reth's.
    pub(crate) version: Option<RethCliVersionConsts>,
    pub(crate) node_mapper: Option<NodeMapper<Ext>>,
}

impl Default for OpRethOverrides {
    fn default() -> Self {
        Self::new()
    }
}

impl OpRethOverrides {
    /// Creates overrides that run op-reth itself.
    pub fn new() -> Self {
        Self { version: None, node_mapper: None }
    }
}

impl<Ext> OpRethOverrides<Ext> {
    /// Reports `version` wherever the client identity is observable: `--version`, the
    /// `--identity` default, `web3_clientVersion`, the devp2p identity and metrics.
    pub fn with_version(mut self, version: RethCliVersionConsts) -> Self {
        self.version = Some(version);
        self
    }

    /// Adds `NewExt` to the `node` command and installs an async mapper that runs before launch.
    ///
    /// The mapper receives the mutable node config, the parsed extra arguments, and the [`OpNode`]
    /// built from op-reth's own arguments. It returns the configured [`OpNode`] and its
    /// [`OpLaunchHooks`] for op-reth to launch.
    /// Use [`OpNode::with_pool`] or [`OpNode::with_payload`] to change the node's builder types.
    /// Calling this again replaces the previous mapper.
    ///
    /// ```no_run
    /// use op_reth::{OpRethOverrides, main_with, node::OpLaunchHooks};
    /// use reth_cli_commands::node::NoArgs;
    /// use reth_node_builder::components::BasicPayloadServiceBuilder;
    ///
    /// let overrides: OpRethOverrides = OpRethOverrides::new().map_node(async |_, _: NoArgs, node| {
    ///     let payload = BasicPayloadServiceBuilder::new(node.payload_builder());
    ///     Ok((node.with_payload(payload), OpLaunchHooks::default()))
    /// });
    /// main_with(overrides)?;
    /// # Ok::<(), eyre::Report>(())
    /// ```
    pub fn map_node<NewExt, Pool, Payload, F>(self, mapper: F) -> OpRethOverrides<NewExt>
    where
        F: for<'a> AsyncFnOnce(
                &'a mut NodeConfig<OpChainSpec>,
                NewExt,
                OpNode,
            ) -> eyre::Result<(OpNode<Pool, Payload>, OpLaunchHooks)>
            + 'static,
        NewExt: 'static,
        OpNode<Pool, Payload>: LaunchOpNode + 'static,
    {
        OpRethOverrides {
            version: self.version,
            node_mapper: Some(Box::new(move |config, args, node| {
                async move {
                    let (node, hooks) = mapper(config, args, node).await?;
                    let node: Box<dyn MappedOpNode> = Box::new(node);
                    Ok((node, hooks))
                }
                .boxed_local()
            })),
        }
    }
}

impl<Ext> fmt::Debug for OpRethOverrides<Ext> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpRethOverrides").field("version", &self.version).finish_non_exhaustive()
    }
}
