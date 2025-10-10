#!/usr/bin/env bash
set -euo pipefail

# ci_flake_shake_prepare_slack.sh
#
# Purpose:
#   Used in CI by the flake-shake promote job to parse the promoter output
#   (`promotion-ready.json`) and prepare environment variables consumed by the
#   Slack orb step.
#
# Inputs (positional):
#   $1 PROMO_JSON – path to the promoter output `promotion-ready.json`.
#       Default: ./final-promotion/promotion-ready.json
#
# Outputs (env):
#   Exports into $BASH_ENV (for subsequent steps):
#     - SLACK_BLOCKS_PAYLOAD: compact JSON array of Slack Block Kit blocks for the message body
#
# Requirements:
#   - jq must be available in PATH
#
# Block count constraints:
#   Slack allows max 50 blocks per message. Our layout uses:
#     - 3 header/link/divider blocks
#     - 4 blocks per candidate (2 sections + 1 context + 1 divider)
#     - +1 optional overflow notice block
#   This caps candidates at 11; if more, we add a final notice linking to the job.

PROMO_JSON=${1:-./final-promotion/promotion-ready.json}

SLACK_BLOCKS="[]"
if [ -f "$PROMO_JSON" ]; then
  # Determine URL to the flake-shake report job (artifacts live there),
  # falling back to the current job URL if not resolvable.
  REPORT_JOB_URL="${CIRCLE_BUILD_URL:-}"
  if [ -n "${CIRCLE_WORKFLOW_ID:-}" ] && [ -n "${CIRCLE_API_TOKEN:-}" ]; then
    JOBS_JSON=$(curl -sfL -H "Circle-Token: ${CIRCLE_API_TOKEN}" "https://circleci.com/api/v2/workflow/${CIRCLE_WORKFLOW_ID}/jobs?limit=100" || true)
    if [ -n "${JOBS_JSON:-}" ]; then
      # Prefer web_url if available; otherwise construct URL from job_number
      REPORT_WEB_URL=$(printf '%s' "$JOBS_JSON" | jq -r '.items[] | select(.name=="op-acceptance-tests-flake-shake-report") | .web_url // empty' | head -n1)
      if [ -n "$REPORT_WEB_URL" ] && [ "$REPORT_WEB_URL" != "null" ]; then
        REPORT_JOB_URL="$REPORT_WEB_URL"
      else
        REPORT_JOB_NUM=$(printf '%s' "$JOBS_JSON" | jq -r '.items[] | select(.name=="op-acceptance-tests-flake-shake-report") | .job_number // empty' | head -n1)
        if [ -n "$REPORT_JOB_NUM" ] && [ "$REPORT_JOB_NUM" != "null" ] && [ -n "${CIRCLE_PROJECT_USERNAME:-}" ] && [ -n "${CIRCLE_PROJECT_REPONAME:-}" ]; then
          REPORT_JOB_URL="https://circleci.com/gh/${CIRCLE_PROJECT_USERNAME}/${CIRCLE_PROJECT_REPONAME}/${REPORT_JOB_NUM}"
        fi
      fi
    fi
  fi

  # Ensure URL points to the artifacts page of the report job
  REPORT_ARTIFACTS_URL="$REPORT_JOB_URL"
  if [ -n "$REPORT_ARTIFACTS_URL" ]; then
    if ! printf '%s' "$REPORT_ARTIFACTS_URL" | grep -q '/artifacts\($\|[?#]\)'; then
      REPORT_ARTIFACTS_URL="${REPORT_ARTIFACTS_URL%/}/artifacts"
    fi
  fi

  # Build Block Kit blocks (header + link + divider + per-candidate sections)
  # See: https://docs.slack.dev/block-kit
  SLACK_BLOCKS=$(jq -c \
    --arg url "${REPORT_ARTIFACTS_URL}" \
    --slurpfile meta "${PROMO_JSON%/*}/metadata.json" '
    def name_or_pkg(t): (if ((t.test_name|tostring)|length) == 0 then "(package)" else t.test_name end);
    def owner_or_unknown(t): (if ((t.owner|tostring)|length) == 0 then "unknown" else t.owner end);
    def pkg_link(t): (
      (t.package|tostring) as $p |
      (if ($p|test("^github\\.com/ethereum-optimism/optimism/")) then
         ("https://github.com/ethereum-optimism/optimism/tree/develop/" + ($p | sub("^github\\.com/ethereum-optimism/optimism/"; "")))
       else "" end) as $u |
      (if $u != "" then ("<" + $u + "|" + $p + ">") else $p end)
    );
    def testblocks(t): [
      {"type":"section","fields":[
        {"type":"mrkdwn","text":"*Test:*\n\(name_or_pkg(t))"},
        {"type":"mrkdwn","text":"*Owner:*\n\(owner_or_unknown(t))"}
      ]},
      {"type":"section","fields":[
        {"type":"mrkdwn","text":"*Runs:*\n\(t.total_runs)"},
        {"type":"mrkdwn","text":"*Pass Rate:*\n\((t.pass_rate|tostring))%"}
      ]},
      {"type":"context","elements":[{"type":"mrkdwn","text": pkg_link(t) }]},
      {"type":"divider"}
    ];
    . as $root |
    ($meta | if length>0 then .[0] else {} end) as $meta |
    ($meta.date // "") as $date |
    ($meta.gate // "flake-shake") as $gate |
    ($meta.pr_url // "") as $pr_url |
    ( if (($meta.flake_gate_tests // 0) == 0) then
        [
          {"type":"header","text":{"type":"plain_text","text":":partywizard: Acceptance Tests: Flake-Shake — Gate Empty"}},
          {"type":"section","text":{"type":"mrkdwn","text":"No tests in flake-shake gate; nothing to promote. Artifacts: <\($url)|CircleCI Job>"}}
        ]
      elif ($root.candidates|length) == 0 then
        [
          {"type":"header","text":{"type":"plain_text","text":":partywizard: Acceptance Tests: No Flake-Shake Promotion Candidates — \(if $date != "" then $date else (now|strftime("%Y-%m-%d")) end)"}},
          {"type":"section","text":{"type":"mrkdwn","text":"No promotions today. Artifacts: <\($url)|CircleCI Job>"}}
        ]
      else
        (
          [
            {"type":"header","text":{"type":"plain_text","text":":partywizard: Acceptance Tests: Flake-Shake Promotion Candidates (\($root.candidates|length)) — \(if $date != "" then $date else (now|strftime("%Y-%m-%d")) end)"}},
            {"type":"section","text":{"type":"mrkdwn","text": (if $pr_url != "" then "PR: <\($pr_url)|Open PR>  •  Artifacts: <\($url)|CircleCI Job>" else "Artifacts: <\($url)|CircleCI Job>" end) }},
            {"type":"divider"}
          ]
        )
        + ( ($root.candidates[:11] | map(testblocks(.)) | add) )
        + ( if ($root.candidates|length) > 11 then
              [ {"type":"section","text":{"type":"mrkdwn","text":"Too many tests; see the report: <\($url)|CircleCI Job>"}} ]
            else [] end )
      end )
  ' "$PROMO_JSON")
fi

echo "export SLACK_BLOCKS_PAYLOAD='$SLACK_BLOCKS'" >> "$BASH_ENV"

echo "Prepared Slack env: blocks generated"

echo "[debug] SLACK_BLOCKS: $SLACK_BLOCKS"
