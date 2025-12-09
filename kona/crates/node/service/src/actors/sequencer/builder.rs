//! Builder for [`SequencerActor`].

use crate::{
    UnsafePayloadGossipClient,
    actors::{
        BlockBuildingClient,
        sequencer::{Conductor, OriginSelector, SequencerActor, SequencerAdminQuery},
    },
};
use kona_derive::AttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// Builder for constructing a [`SequencerActor`].
#[derive(Debug, Default)]
pub struct SequencerActorBuilder<
    AttributesBuilder_,
    BlockBuildingClient_,
    Conductor_,
    OriginSelector_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    BlockBuildingClient_: BlockBuildingClient,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Receiver for admin API requests.
    pub admin_api_rx: Option<mpsc::Receiver<SequencerAdminQuery>>,
    /// The attributes builder used for block building.
    pub attributes_builder: Option<AttributesBuilder_>,
    /// The struct used to build blocks.
    pub block_building_client: Option<BlockBuildingClient_>,
    /// The cancellation token, shared between all tasks.
    pub cancellation_token: Option<CancellationToken>,
    /// The optional conductor RPC client.
    pub conductor: Option<Conductor_>,
    /// Whether the sequencer is active.
    pub is_active: Option<bool>,
    /// Whether the sequencer is in recovery mode.
    pub in_recovery_mode: Option<bool>,
    /// The struct used to determine the next L1 origin.
    pub origin_selector: Option<OriginSelector_>,
    /// The rollup configuration.
    pub rollup_config: Option<Arc<RollupConfig>>,
    /// A client to asynchronously sign and gossip built payloads to the network actor.
    pub unsafe_payload_gossip_client: Option<UnsafePayloadGossipClient_>,
}

impl<
    AttributesBuilder_,
    BlockBuildingClient_,
    Conductor_,
    OriginSelector_,
    UnsafePayloadGossipClient_,
>
    SequencerActorBuilder<
        AttributesBuilder_,
        BlockBuildingClient_,
        Conductor_,
        OriginSelector_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    BlockBuildingClient_: BlockBuildingClient,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Creates a new empty [`SequencerActorBuilder`].
    pub const fn new() -> Self {
        Self {
            admin_api_rx: None,
            attributes_builder: None,
            block_building_client: None,
            cancellation_token: None,
            conductor: None,
            unsafe_payload_gossip_client: None,
            is_active: None,
            in_recovery_mode: None,
            origin_selector: None,
            rollup_config: None,
        }
    }

    /// Sets whether the sequencer is active.
    pub const fn with_active_status(mut self, is_active: bool) -> Self {
        self.is_active = Some(is_active);
        self
    }

    /// Sets whether the sequencer is in recovery mode.
    pub const fn with_recovery_mode_status(mut self, is_recovery_mode: bool) -> Self {
        self.in_recovery_mode = Some(is_recovery_mode);
        self
    }

    /// Sets the rollup configuration.
    pub fn with_rollup_config(mut self, rollup_config: Arc<RollupConfig>) -> Self {
        self.rollup_config = Some(rollup_config);
        self
    }

    /// Sets the admin API receiver.
    pub fn with_admin_api_receiver(
        mut self,
        admin_api_rx: mpsc::Receiver<SequencerAdminQuery>,
    ) -> Self {
        self.admin_api_rx = Some(admin_api_rx);
        self
    }

    /// Sets the attributes builder.
    pub fn with_attributes_builder(mut self, attributes_builder: AttributesBuilder_) -> Self {
        self.attributes_builder = Some(attributes_builder);
        self
    }

    /// Sets the conductor.
    pub fn with_conductor(mut self, conductor: Conductor_) -> Self {
        self.conductor = Some(conductor);
        self
    }

    /// Sets the origin selector.
    pub fn with_origin_selector(mut self, origin_selector: OriginSelector_) -> Self {
        self.origin_selector = Some(origin_selector);
        self
    }

    /// Sets the block engine.
    pub fn with_block_building_client(
        mut self,
        block_building_client: BlockBuildingClient_,
    ) -> Self {
        self.block_building_client = Some(block_building_client);
        self
    }

    /// Sets the cancellation token.
    pub fn with_cancellation_token(mut self, token: CancellationToken) -> Self {
        self.cancellation_token = Some(token);
        self
    }

    /// Sets the gossip payload sender.
    pub fn with_unsafe_payload_gossip_client(
        mut self,
        gossip_client: UnsafePayloadGossipClient_,
    ) -> Self {
        self.unsafe_payload_gossip_client = Some(gossip_client);
        self
    }

    /// Builds the [`SequencerActor`].
    ///
    /// # Panics
    ///
    /// Panics if any required field is not set.
    pub fn build(
        self,
    ) -> Result<
        SequencerActor<
            AttributesBuilder_,
            BlockBuildingClient_,
            Conductor_,
            OriginSelector_,
            UnsafePayloadGossipClient_,
        >,
        String,
    > {
        Ok(SequencerActor {
            admin_api_rx: self.admin_api_rx.expect("admin_api_rx is required"),
            attributes_builder: self.attributes_builder.expect("attributes_builder is required"),
            block_building_client: self
                .block_building_client
                .expect("block_building_client is required"),
            cancellation_token: self.cancellation_token.expect("cancellation is required"),
            conductor: self.conductor,
            is_active: self.is_active.expect("initial active status not set"),
            in_recovery_mode: self.in_recovery_mode.expect("initial recovery mode status not set"),
            origin_selector: self.origin_selector.expect("origin_selector is required"),
            rollup_config: self.rollup_config.expect("rollup_config is required"),
            unsafe_payload_gossip_client: self
                .unsafe_payload_gossip_client
                .expect("unsafe_payload_gossip_client is required"),
        })
    }
}
