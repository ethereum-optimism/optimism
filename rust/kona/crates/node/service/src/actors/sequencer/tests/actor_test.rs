#[cfg(test)]
use crate::{
    SequencerActor, SequencerActorError,
    actors::{
        MockConductor, MockOriginSelector, MockSequencerEngineClient, MockUnsafePayloadGossipClient,
    },
};
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo};
use rstest::rstest;
use std::sync::Arc;
use tokio::sync::mpsc;

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    client.expect_start_build_block().times(0);
    if let PipelineErrorKind::Reset(_) = &forced_error {
        client.expect_reset_engine_forkchoice().times(1).return_once(move || Ok(()));
    }

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };
    let (_admin_api_tx, admin_api_rx) = mpsc::channel(20);
    let mut actor = SequencerActor::<_, MockConductor, _, _, _>::new(
        admin_api_rx,
        attributes_builder,
        None,
        client,
        true,
        false,
        origin_selector,
        Arc::new(RollupConfig { block_time: 2, ..Default::default() }),
        MockUnsafePayloadGossipClient::new(),
    );

    let result = actor.worker.as_mut().unwrap().build_payload().await;
    if expect_err {
        assert!(matches!(
            result.unwrap_err(),
            SequencerActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.is_ok());
    }
}
