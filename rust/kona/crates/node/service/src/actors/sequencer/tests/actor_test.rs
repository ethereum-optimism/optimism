#[cfg(test)]
use crate::{
    EngineClientError, SequencerActorError,
    actors::{
        MockOriginSelector, MockSequencerEngineClient, sequencer::tests::test_util::test_actor,
    },
};
use alloy_rpc_types_engine::PayloadId;
use kona_derive::{BuilderError, PipelineErrorKind, test_utils::TestAttributesBuilder};
use kona_engine::BuildTaskError;
use kona_protocol::{BlockInfo, L2BlockInfo};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use rstest::rstest;

#[rstest]
#[case::temp(PipelineErrorKind::Temporary(BuilderError::Custom(String::new()).into()), false)]
#[case::reset(PipelineErrorKind::Reset(BuilderError::Custom(String::new()).into()), false)]
#[case::critical(PipelineErrorKind::Critical(BuilderError::Custom(String::new()).into()), true)]
#[tokio::test]
async fn test_build_unsealed_payload_prepare_payload_attributes_error(
    #[case] forced_error: PipelineErrorKind,
    #[case] expect_err: bool,
) {
    let mut client = MockSequencerEngineClient::new();

    let unsafe_head = L2BlockInfo::default();
    client.expect_get_unsafe_head().times(1).return_once(move || Ok(unsafe_head));
    // Must not be called on critical error
    client.expect_start_build_block().times(0);
    if let PipelineErrorKind::Reset(_) = &forced_error {
        client.expect_reset_engine_forkchoice().times(1).return_once(move || Ok(()));
    }

    let l1_origin = BlockInfo::default();
    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(1).return_once(move |_, _| Ok(l1_origin));

    let attributes_builder = TestAttributesBuilder { attributes: vec![Err(forced_error)] };

    let mut actor = test_actor();
    actor.origin_selector = origin_selector;
    actor.engine_client = client;
    actor.attributes_builder = attributes_builder;

    let result = actor.build_unsealed_payload().await;
    if expect_err {
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            SequencerActorError::AttributesBuilder(PipelineErrorKind::Critical(_))
        ));
    } else {
        assert!(result.is_ok());
    }
}

#[rstest]
#[case::stale_parent(EngineClientError::StartBuildError(
    BuildTaskError::UnsafeHeadChangedSinceBuild,
))]
#[case::request_dropped(EngineClientError::ResponseError("response channel closed.".to_string()))]
#[tokio::test]
async fn test_build_unsealed_payload_retries_when_the_engine_rejects_the_build(
    #[case] forced_error: EngineClientError,
) {
    // The engine reorged onto a derived chain (or reset) after the unsafe head was read, so these
    // attributes no longer extend it. The sequencer must re-build on the new head next tick rather
    // than treat this as fatal.
    let payload_id = PayloadId::new([7u8; 8]);
    let reorged_head = L2BlockInfo {
        block_info: BlockInfo { number: 42, ..Default::default() },
        ..Default::default()
    };

    let mut client = MockSequencerEngineClient::new();
    let mut heads = vec![reorged_head, L2BlockInfo::default()];
    client.expect_get_unsafe_head().times(2).returning(move || Ok(heads.pop().unwrap()));
    let mut results = vec![Ok(payload_id), Err(forced_error)];
    client.expect_start_build_block().times(2).returning(move |_| results.pop().unwrap());

    let mut origin_selector = MockOriginSelector::new();
    origin_selector.expect_next_l1_origin().times(2).returning(|_, _| Ok(BlockInfo::default()));

    let mut actor = test_actor();
    actor.origin_selector = origin_selector;
    actor.engine_client = client;
    actor.attributes_builder = TestAttributesBuilder {
        attributes: vec![Ok(OpPayloadAttributes::default()), Ok(OpPayloadAttributes::default())],
    };

    assert!(actor.build_unsealed_payload().await.expect("a rejected build is not fatal").is_none());

    // The engine has since published the head it reorged to, so the next attempt succeeds there.
    let handle = actor
        .build_unsealed_payload()
        .await
        .expect("the retry builds")
        .expect("a payload was started");
    assert_eq!(handle.payload_id, payload_id);
    assert_eq!(handle.attributes_with_parent.parent(), &reorged_head);
}
