#!/usr/bin/env bash
# Workflow routing decision tree for the CircleCI setup pipeline.
#
# Reads the change-detection / dispatch params already written to
# /tmp/pipeline-parameters.json by earlier steps and enables the matching
# c-run_* workflow flags based on the trigger source, branch, tag, and schedule.
#
# Inputs (set by the config.yml step environment):
#   BRANCH, TRIGGER_SOURCE, TAG, SCHEDULE_NAME
#
# Helpers (workflow-helpers.sh):
#   run foo     → sets c-run_foo: true in JSON (maps 1:1 to
#                 "when: << pipeline.parameters.c-run_foo >>" in a continuation config)
#   is_true x   → checks if c-x is true in JSON (set by earlier steps)
#   param x     → reads raw value of c-x from JSON
#   finalize    → strips intermediate params, keeps only the listed whitelist + all c-run_* flags
#
# How to add a new workflow:
#   1. Add "c-run_your_workflow: {type: boolean, default: false}" in the relevant continuation config
#   2. Add "run your_workflow" in the appropriate branch below
set -euo pipefail

# shellcheck disable=SC1091  # sourced helper resolved at runtime, not by shellcheck
source "$(dirname "${BASH_SOURCE[0]}")/workflow-helpers.sh"
init_json

case "${TRIGGER_SOURCE}" in

  # Scheduled pipelines: map schedule name → workflows
  scheduled_pipeline)
    case "${SCHEDULE_NAME}" in
      build_four_hours) run scheduled_todo_issues scheduled_cannon_full_tests ;;
      build_daily)      run scheduled_preimage_reproducibility scheduled_stale_check scheduled_heavy_fuzz_tests scheduled_daily_tests circleci_schedule_trigger_check ;;
    esac
    ;;

  # Webhook (push events)
  webhook)
    # --- Tag push ---
    if [[ -n "${TAG}" ]]; then
      run release

    # =========================================================
    #  Three mutually exclusive lifecycle stages:
    # =========================================================

    # 1. PR (feature branch push)
    #    Runs on every push to a feature branch.
    #    Path-based gating: only changed areas are tested.
    #    Docs-only changes skip main/release entirely.
    # ---------------------------------------------------------
    elif [[ "${BRANCH}" != "develop" && ! "${BRANCH}" =~ ^gh-readonly-queue/ ]]; then
      if is_true only_docs_changes; then
        run ci_gate_skip
        run contracts_feature_tests_short
        run rust_ci_gate_short
        run rust_e2e_gate_skip
      else
        run main
        run release
        if is_true contracts_changed; then
          run contracts_feature_tests
        else
          run contracts_feature_tests_short
        fi
        if is_true rust_changes_detected; then
          run rust_ci
          run rust_e2e_ci
        else
          run rust_ci_gate_short
          run rust_e2e_gate_skip
        fi
        if is_true circleci_changed; then
          run circleci_schedule_trigger_check
        fi
      fi

    # 2. Merge queue (pre-merge validation)
    #    Runs when GitHub merge queue picks up the PR.
    #    Full contract tests, path-gated rust/docs.
    # ---------------------------------------------------------
    elif [[ "${BRANCH}" =~ ^gh-readonly-queue/ ]]; then
      run main
      run release
      run contracts_feature_tests
      if is_true rust_changes_detected; then
        run rust_ci
        run rust_e2e_ci
      else
        run rust_ci_gate_short
        run rust_e2e_gate_skip
      fi
      if is_true circleci_changed; then
        run circleci_schedule_trigger_check
      fi

    # 3. After merge (develop push)
    #    Runs after the merge queue completes and pushes to develop.
    #    Adds expensive post-merge jobs: fault proofs, kontrol, prestate publishing.
    # ---------------------------------------------------------
    elif [[ "${BRANCH}" == "develop" ]]; then
      run main
      run release
      run publish_contract_artifacts
      run develop_fault_proofs
      run develop_kontrol_tests
      run contracts_feature_tests
      if is_true rust_changes_detected; then
        run rust_ci
        run rust_e2e_ci
        run kona_publish_prestates
      else
        run rust_ci_gate_short
        run rust_e2e_gate_skip
      fi
    fi
    ;;

  # API triggers: dispatch flags select workflows
  api)
    run release
    if is_true main_dispatch && [[ "$(param github-event-type)" == "__not_set__" ]]; then
      run main contracts_feature_tests
    fi
    if is_true fault_proofs_dispatch;     then run develop_fault_proofs; fi
    if is_true kontrol_dispatch;          then run develop_kontrol_tests; fi
    if is_true cannon_full_test_dispatch; then run scheduled_cannon_full_tests; fi
    if is_true reproducibility_dispatch;  then run scheduled_preimage_reproducibility; fi
    if is_true stale_check_dispatch;      then run scheduled_stale_check; fi
    if is_true heavy_fuzz_dispatch;       then run scheduled_heavy_fuzz_tests; fi
    if is_true publish_contract_artifacts_dispatch; then run publish_contract_artifacts; fi
    if is_true l2_fork_test_dispatch;     then run l2_fork_test; fi
    if is_true rust_ci_dispatch;          then run rust_ci; fi
    if is_true rust_e2e_dispatch;         then run rust_e2e_ci; fi
    if [[ "$(param github-event-type)" == "pull_request" && "$(param github-event-action)" == "labeled" ]]; then
      run close_issue
    fi
    ;;
esac

# Params to forward to continuation configs (everything else is stripped)
finalize "c-default_docker_image,c-rust_base_image,c-base_image,c-github-event-base64,c-go-cache-version,c-publish_contract_artifacts_ref"
