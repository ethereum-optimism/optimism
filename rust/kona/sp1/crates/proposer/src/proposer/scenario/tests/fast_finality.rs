use super::*;

fn config(limit: u64) -> ProposerConfig {
    let mut config = scenario_config();
    config.fast_finality_mode = true;
    config.fast_finality_proving_limit = NonZeroU64::new(limit).unwrap();
    config.proposal_interval_seconds = 100;
    config
}

#[tokio::test]
async fn created_games_are_accelerated_only_when_fast_finality_is_enabled() {
    for enabled in [false, true] {
        let world = ScenarioWorld::new();
        world.set_horizons(1, 1);
        let mut config = config(1);
        config.proposal_interval_seconds = 1;
        config.fast_finality_mode = enabled;
        let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

        let creation = scenario.tick().await.unwrap();
        creation.task_id_for(|operation| {
            matches!(operation, OperationSummary::ProposeGame { sequence_number: 1, .. })
        });
        scenario.settle_scheduled(&creation).await.unwrap();
        let created = world.observation().games[0].clone();
        let target = created.target();
        assert_eq!(created.creator, ScenarioWorld::proposer_address());
        assert_eq!(created.proposal_status, ProposalStatus::Unchallenged);

        let proving = scenario.tick().await.unwrap();
        if !enabled {
            assert!(
                !proving
                    .scheduled
                    .iter()
                    .any(|task| matches!(task.operation, OperationSummary::ProveGame { .. }))
            );
            scenario.settle_scheduled(&proving).await.unwrap();
            assert!(world.proof_record(&target, 1).is_none());
            assert_eq!(world.observation().games[0].proposal_status, ProposalStatus::Unchallenged);
            continue;
        }
        proving.task_id_for(|operation| {
            matches!(operation, OperationSummary::ProveGame {
                address, purpose: ProvingPurpose::FastFinality, ..
            } if *address == target.address)
        });
        scenario.settle_scheduled(&proving).await.unwrap();
        assert_eq!(
            world.action_record(&ActionTarget::Prove(target.clone()), 1).unwrap().effect,
            CommittedEffect::Proven { game: target.address }
        );
        assert_eq!(
            world.observation().games[0].proposal_status,
            ProposalStatus::UnchallengedAndValidProofProvided
        );

        let resolution = scenario.tick().await.unwrap();
        scenario.settle_scheduled(&resolution).await.unwrap();
        assert_eq!(
            world.action_record(&ActionTarget::Resolve(target.clone()), 1).unwrap().effect,
            CommittedEffect::Resolved { game: target.address }
        );
        let resolved = world.observation();
        assert_eq!(resolved.games[0].status, GameStatus::DefenderWins);
        assert!(resolved.latest_l1.timestamp < created.deadline);
    }
}

#[tokio::test]
async fn foreign_games_are_defended_but_only_owned_unchallenged_games_are_accelerated() {
    let world = ScenarioWorld::new();
    let foreign = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    let defended =
        ScenarioGame::new(1, u32::MAX, 2, ScenarioWorld::default_prestate()).challenged();
    let mut owned = ScenarioGame::new(2, u32::MAX, 3, ScenarioWorld::default_prestate());
    owned.creator = ScenarioWorld::proposer_address();
    let foreign_target = foreign.target();
    let defended_target = defended.target();
    let owned_target = owned.target();
    for game in [foreign, defended, owned] {
        world.add_game(game);
    }
    world.set_horizons(3, 3);
    let mut scenario = ScenarioHarness::new(world.clone(), config(2)).await.unwrap();

    let tick = scenario.tick().await.unwrap();
    let mut proofs = tick
        .scheduled
        .iter()
        .filter_map(|scheduled| match scheduled.operation {
            OperationSummary::ProveGame { address, purpose, .. } => Some((address, purpose)),
            _ => None,
        })
        .collect::<Vec<_>>();
    proofs.sort_unstable();
    assert_eq!(
        proofs,
        vec![
            (defended_target.address, ProvingPurpose::Defense),
            (owned_target.address, ProvingPurpose::FastFinality),
        ]
    );
    scenario.settle_scheduled(&tick).await.unwrap();
    assert!(world.proof_record(&foreign_target, 1).is_none());
    for target in [defended_target, owned_target] {
        assert_eq!(
            world.action_record(&ActionTarget::Prove(target.clone()), 1).unwrap().effect,
            CommittedEffect::Proven { game: target.address }
        );
    }
}

#[tokio::test]
async fn nearest_deadline_proof_holds_capacity_through_failure_and_retry_then_creation_resumes() {
    let world = ScenarioWorld::new();
    let mut later = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    later.creator = ScenarioWorld::proposer_address();
    later.deadline = 6_000;
    let mut earlier = ScenarioGame::new(1, 0, 2, ScenarioWorld::default_prestate());
    earlier.creator = ScenarioWorld::proposer_address();
    earlier.deadline = 5_000;
    let later_target = later.target();
    let earlier_target = earlier.target();
    for game in [later, earlier] {
        world.add_game(game);
    }
    world.set_horizons(3, 3);
    world.block_proof(earlier_target.clone(), 1, ProofOutcome::Failure, "earlier proof");
    let mut config = config(1);
    config.proposal_interval_seconds = 1;
    let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

    let first = scenario.tick().await.unwrap();
    let first_id = first.task_id_for(|operation| {
        matches!(operation, OperationSummary::ProveGame {
            address, purpose: ProvingPurpose::FastFinality, ..
        } if *address == earlier_target.address)
    });
    scenario.wait_for_proof_barrier(first_id, &earlier_target, 1).await.unwrap();
    scenario.settle(&first.task_ids_except(first_id)).await.unwrap();
    assert!(
        !first
            .scheduled
            .iter()
            .any(|task| matches!(task.operation, OperationSummary::ProposeGame { .. }))
    );
    assert!(world.proof_record(&later_target, 1).is_none());

    let blocked = scenario.tick().await.unwrap();
    assert!(blocked.snapshot.active_tasks.iter().any(|task| task.task_id == first_id));
    assert!(!blocked.scheduled.iter().any(|task| matches!(
        task.operation,
        OperationSummary::ProveGame { .. } | OperationSummary::ProposeGame { .. }
    )));
    scenario.settle_scheduled(&blocked).await.unwrap();
    scenario.release_proof_barrier(&earlier_target, 1).unwrap();
    scenario.settle(&[first_id]).await.unwrap();
    assert_eq!(world.proof_record(&earlier_target, 1).unwrap().lifecycle, ProofLifecycle::Failed);
    assert!(world.action_record(&ActionTarget::Prove(earlier_target.clone()), 1).is_none());

    let retry = scenario.tick().await.unwrap();
    retry.task_id_for(|operation| {
        matches!(operation, OperationSummary::ProveGame {
            address, purpose: ProvingPurpose::FastFinality, ..
        } if *address == earlier_target.address)
    });
    assert!(
        !retry
            .scheduled
            .iter()
            .any(|task| matches!(task.operation, OperationSummary::ProposeGame { .. }))
    );
    assert!(world.proof_record(&later_target, 1).is_none());
    scenario.settle_scheduled(&retry).await.unwrap();
    assert_eq!(
        world.proof_record(&earlier_target, 2).unwrap().lifecycle,
        ProofLifecycle::Succeeded
    );
    assert_eq!(world.observation().games.len(), 2);

    let next = scenario.tick().await.unwrap();
    next.task_id_for(|operation| {
        matches!(operation, OperationSummary::ProveGame { address, .. } if *address == later_target.address)
    });
    assert!(
        !next
            .scheduled
            .iter()
            .any(|task| matches!(task.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario.settle_scheduled(&next).await.unwrap();

    let resumed = scenario.tick().await.unwrap();
    resumed.task_id_for(|operation| {
        matches!(
            operation,
            OperationSummary::ProposeGame { sequence_number: 3, parent_game_index: 1 }
        )
    });
    scenario.settle_scheduled(&resumed).await.unwrap();
    assert!(matches!(
        world
            .action_record(&ActionTarget::Create { sequence_number: 3, parent_game_index: 1 }, 1)
            .unwrap()
            .effect,
        CommittedEffect::Created { .. }
    ));
    assert!(world.proof_record(&earlier_target, 3).is_none());
    assert!(world.proof_record(&later_target, 2).is_none());
}

#[tokio::test]
async fn parked_fast_finality_does_not_consume_defense_slots_but_defense_counts_toward_creation_capacity()
 {
    let world = ScenarioWorld::new();
    let mut owned = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    owned.creator = ScenarioWorld::proposer_address();
    let owned_target = owned.target();
    world.add_game(owned);
    world.set_horizons(1, 1);
    world.block_proof(owned_target.clone(), 1, ProofOutcome::Success, "fast finality proof");
    let mut config = config(2);
    config.proposal_interval_seconds = 1;
    config.max_concurrent_defense_tasks = NonZeroU64::MIN;
    let mut scenario = ScenarioHarness::new(world.clone(), config).await.unwrap();

    let first = scenario.tick().await.unwrap();
    let fast_id = first.task_id_for(|operation| {
        matches!(operation,
            OperationSummary::ProveGame { address, purpose: ProvingPurpose::FastFinality, .. }
            if *address == owned_target.address
        )
    });
    scenario.wait_for_proof_barrier(fast_id, &owned_target, 1).await.unwrap();
    scenario.settle(&first.task_ids_except(fast_id)).await.unwrap();

    let mut defended = ScenarioGame::new(1, 0, 2, ScenarioWorld::default_prestate()).challenged();
    defended.deadline = 5_000;
    let defended_target = defended.target();
    let mut waiting = ScenarioGame::new(2, 1, 3, ScenarioWorld::default_prestate()).challenged();
    waiting.deadline = 6_000;
    let waiting_target = waiting.target();
    for game in [defended, waiting] {
        world.add_game(game);
    }
    world.set_horizons(3, 3);
    world.block_proof(defended_target.clone(), 1, ProofOutcome::Success, "defense proof");
    let second = scenario.tick().await.unwrap();
    let defense_id = second.task_id_for(|operation| {
        matches!(operation,
            OperationSummary::ProveGame { address, purpose: ProvingPurpose::Defense, .. }
            if *address == defended_target.address
        )
    });
    assert!(
        !second
            .scheduled
            .iter()
            .any(|task| matches!(task.operation, OperationSummary::ProposeGame { .. }))
    );
    scenario.wait_for_proof_barrier(defense_id, &defended_target, 1).await.unwrap();
    scenario.settle(&second.task_ids_except(defense_id)).await.unwrap();

    world.set_horizons(4, 4);
    let full = scenario.tick().await.unwrap();
    assert!(!full.scheduled.iter().any(|task| matches!(
        task.operation,
        OperationSummary::ProveGame { .. } | OperationSummary::ProposeGame { .. }
    )));
    scenario.settle_scheduled(&full).await.unwrap();
    assert!(world.proof_record(&waiting_target, 1).is_none());
    scenario.release_proof_barrier(&defended_target, 1).unwrap();
    scenario.settle(&[defense_id]).await.unwrap();

    let next = scenario.tick().await.unwrap();
    next.task_id_for(|operation| {
        matches!(operation,
            OperationSummary::ProveGame { address, purpose: ProvingPurpose::Defense, .. }
            if *address == waiting_target.address
        )
    });
    scenario.settle_scheduled(&next).await.unwrap();
    scenario.release_proof_barrier(&owned_target, 1).unwrap();
    scenario.settle(&[fast_id]).await.unwrap();
    for target in [owned_target, defended_target, waiting_target] {
        assert_eq!(world.proof_record(&target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
    }
}

#[tokio::test]
async fn challenge_during_fast_finality_reuses_the_inflight_proof() {
    let world = ScenarioWorld::new();
    let mut owned = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    owned.creator = ScenarioWorld::proposer_address();
    let target = owned.target();
    world.add_game(owned);
    world.set_horizons(1, 1);
    world.block_proof(target.clone(), 1, ProofOutcome::Success, "challenge during proof");
    let mut scenario = ScenarioHarness::new(world.clone(), config(1)).await.unwrap();

    let started = scenario.tick().await.unwrap();
    let proof_id = started.task_id_for(|operation| {
        matches!(operation,
            OperationSummary::ProveGame { address, purpose: ProvingPurpose::FastFinality, .. }
            if *address == target.address
        )
    });
    scenario.wait_for_proof_barrier(proof_id, &target, 1).await.unwrap();
    scenario.settle(&started.task_ids_except(proof_id)).await.unwrap();
    world.update_game(&target, |game| game.proposal_status = ProposalStatus::Challenged);

    let challenged = scenario.tick().await.unwrap();
    assert!(challenged.snapshot.active_tasks.iter().any(|task| task.task_id == proof_id));
    assert!(!challenged.scheduled.iter().any(|task| matches!(task.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario.settle_scheduled(&challenged).await.unwrap();
    scenario.release_proof_barrier(&target, 1).unwrap();
    scenario.settle(&[proof_id]).await.unwrap();
    assert_eq!(
        world.observation().games[0].proposal_status,
        ProposalStatus::ChallengedAndValidProofProvided
    );
    assert_eq!(
        world.action_record(&ActionTarget::Prove(target.clone()), 1).unwrap().effect,
        CommittedEffect::Proven { game: target.address }
    );
    let resolved = scenario.tick().await.unwrap();
    scenario.settle_scheduled(&resolved).await.unwrap();
    assert_eq!(world.observation().games[0].status, GameStatus::DefenderWins);
    assert!(world.proof_record(&target, 2).is_none());
}

#[tokio::test]
async fn expired_fast_finality_proof_is_not_submitted_and_the_game_resolves_normally() {
    let world = ScenarioWorld::new();
    let mut owned = ScenarioGame::new(0, u32::MAX, 1, ScenarioWorld::default_prestate());
    owned.creator = ScenarioWorld::proposer_address();
    owned.deadline = 1_500;
    let target = owned.target();
    world.add_game(owned);
    world.set_horizons(1, 1);
    world.set_host_time(10_000);
    world.block_proof(target.clone(), 1, ProofOutcome::Success, "expired acceleration");
    let mut scenario = ScenarioHarness::new(world.clone(), config(1)).await.unwrap();

    let started = scenario.tick().await.unwrap();
    let proof_id = started.task_id_for(|operation| {
        matches!(operation,
            OperationSummary::ProveGame { address, purpose: ProvingPurpose::FastFinality, .. }
            if *address == target.address
        )
    });
    scenario.wait_for_proof_barrier(proof_id, &target, 1).await.unwrap();
    scenario.settle(&started.task_ids_except(proof_id)).await.unwrap();
    world.set_latest_l1_time(1_501);
    scenario.release_proof_barrier(&target, 1).unwrap();
    scenario.settle(&[proof_id]).await.unwrap();
    assert_eq!(world.proof_record(&target, 1).unwrap().lifecycle, ProofLifecycle::Succeeded);
    assert!(world.action_record(&ActionTarget::Prove(target.clone()), 1).is_none());

    let expired = scenario.tick().await.unwrap();
    assert!(!expired.scheduled.iter().any(|task| matches!(task.operation,
        OperationSummary::ProveGame { address, .. } if address == target.address
    )));
    scenario.settle_scheduled(&expired).await.unwrap();
    assert_eq!(
        world.action_record(&ActionTarget::Resolve(target.clone()), 1).unwrap().effect,
        CommittedEffect::Resolved { game: target.address }
    );
    assert_eq!(world.observation().games[0].status, GameStatus::DefenderWins);
    assert!(world.proof_record(&target, 2).is_none());
}
