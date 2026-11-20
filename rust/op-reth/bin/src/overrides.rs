//! Startup overrides for a binary that embeds op-reth.

use futures_util::{FutureExt, future::LocalBoxFuture};
use reth_cli_commands::node::NoArgs;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{node_config::NodeConfig, version::RethCliVersionConsts};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_node::{LaunchOpNode, OpLaunchHooks, OpNode, args::RollupArgs};
use std::fmt;

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

/// Changes the node before launch. See [`OpRethOverrides::customize`].
type Customize<Ext, Node> = Box<
    dyn for<'a> FnOnce(
        &'a mut NodeConfig<OpChainSpec>,
        Ext,
        OpNode,
    ) -> LocalBoxFuture<'a, eyre::Result<(Node, OpLaunchHooks)>>,
>;

/// Launches the node `customize` returned, then waits for it to exit.
type Launch<Node> = fn(
    WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
    Node,
    OpLaunchHooks,
) -> LocalBoxFuture<'static, eyre::Result<()>>;

/// Startup overrides that [`main_with`](crate::main_with) applies.
///
/// The default overrides run op-reth itself. `Ext` holds the extra `node` arguments, and `Node` is
/// the node that launches: [`OpNode`], or a
/// [`CustomOpNode`](reth_optimism_node::CustomOpNode) with a replaced pool builder or payload
/// service.
pub struct OpRethOverrides<Ext = NoArgs, Node = OpNode> {
    /// The build metadata to report instead of op-reth's.
    pub(crate) version: Option<RethCliVersionConsts>,
    pub(crate) customize: Customize<Ext, Node>,
    pub(crate) launch: Launch<Node>,
}

impl Default for OpRethOverrides {
    fn default() -> Self {
        Self::new()
    }
}

impl OpRethOverrides {
    /// Creates overrides that run op-reth itself.
    pub fn new() -> Self {
        Self {
            version: None,
            customize: Box::new(|_, _, node| {
                async move { Ok((node, OpLaunchHooks::default())) }.boxed_local()
            }),
            // `launch_node` is not generic, so op-reth's own launch is compiled in
            // `reth-optimism-node-launcher` rather than here.
            launch: |builder, node, hooks| {
                reth_optimism_node_launcher::launch_node(builder, node, hooks)
            },
        }
    }
}

impl<Ext, Node> OpRethOverrides<Ext, Node> {
    /// Reports `version` wherever the client identity is observable: `--version`, the
    /// `--identity` default, `web3_clientVersion`, the devp2p identity and metrics.
    pub fn with_version(mut self, version: RethCliVersionConsts) -> Self {
        self.version = Some(version);
        self
    }

    /// Adds the arguments `NewExt` to the `node` command and runs `customize` before the node
    /// launches, replacing any earlier `customize`.
    ///
    /// `customize` gets the node's config, the parsed `NewExt` and the [`OpNode`] op-reth built
    /// from its own arguments. It returns the node to launch, which [`OpNode::with_pool`] and
    /// [`OpNode::with_payload`] can derive, and the [`OpLaunchHooks`] to launch it with.
    ///
    /// The node launches through [`LaunchOpNode`], so the launch is compiled in the crate that
    /// calls this.
    pub fn customize<NewExt, NewNode, F>(self, customize: F) -> OpRethOverrides<NewExt, NewNode>
    where
        F: for<'a> FnOnce(
                &'a mut NodeConfig<OpChainSpec>,
                NewExt,
                OpNode,
            )
                -> LocalBoxFuture<'a, eyre::Result<(NewNode, OpLaunchHooks)>>
            + 'static,
        NewNode: LaunchOpNode + 'static,
    {
        OpRethOverrides {
            version: self.version,
            customize: Box::new(customize),
            launch: |builder, node, hooks| node.launch(builder, hooks).boxed_local(),
        }
    }
}

impl<Ext, Node> fmt::Debug for OpRethOverrides<Ext, Node> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("OpRethOverrides").field("version", &self.version).finish_non_exhaustive()
    }
}
