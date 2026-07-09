#!/usr/bin/env bash
# Workflow routing policy for the CircleCI setup pipeline.
#
# Decides WHICH continuation workflows run, based on the trigger source, branch,
# tag, schedule name, and the change-detection / dispatch params already written
# to /tmp/pipeline-parameters.json by earlier steps. The workflow lists
# themselves live in routing.yml — this script only holds the conditions.
#
# Each enabled workflow sets c-run_<name>: true in the JSON, which maps 1:1 to
# "when: << pipeline.parameters.c-run_<name> >>" in a continuation config.
#
# Inputs (set by the config.yml step environment):
#   BRANCH, TRIGGER_SOURCE, TAG, SCHEDULE_NAME
#
# Helpers (workflow-helpers.sh):
#   run <wf...>            enable workflows by literal name
#   run_group <sec> <key>  enable the workflows listed under routing.yml sec.key
#   is_true <x>            true if c-x is true in the JSON
#   param <x>              raw value of c-x from the JSON
#   finalize               strip intermediate params, keep c-run_* + passthrough
#
# How to add a new workflow:
#   1. Declare "c-run_<name>: {type: boolean, default: false}" in a continuation
#      config under .circleci/continue/.
#   2. Wire it in here (literal "run <name>" for a one-off, or add it to the
#      relevant routing.yml list and use run_group).
set -euo pipefail

# shellcheck disable=SC1091  # sourced helper resolved at runtime, not by shellcheck
source "$(dirname "${BASH_SOURCE[0]}")/workflow-helpers.sh"
init_json

case "${TRIGGER_SOURCE}" in

  # Scheduled pipelines: map schedule name -> workflows (routing.yml schedules).
  scheduled_pipeline)
    run_group schedules "${SCHEDULE_NAME}"
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

  # API triggers: dispatch flags select workflows (routing.yml api_dispatch).
  api)
    run release
    # main_dispatch only fires for genuine API dispatches, not github-event triggers.
    if is_true main_dispatch && [[ "$(param github-event-type)" == "__not_set__" ]]; then
      run_group api_dispatch main_dispatch
    fi
    # Simple dispatch flags: each enables its workflows when the flag is set.
    # main_dispatch and labeled_pr have bespoke conditions, handled separately.
    for flag in $(yq -r '.api_dispatch | keys | .[]' "${ROUTING}"); do
      # Keep this skip-list in sync with bespoke api_dispatch conditions.
      case "${flag}" in
        main_dispatch | labeled_pr) continue ;;
      esac
      if is_true "${flag}"; then
        run_group api_dispatch "${flag}"
      fi
    done
    # GitHub "pull_request labeled" event triggers issue-close automation.
    if [[ "$(param github-event-type)" == "pull_request" && "$(param github-event-action)" == "labeled" ]]; then
      run_group api_dispatch labeled_pr
    fi
    ;;
esac

finalize
