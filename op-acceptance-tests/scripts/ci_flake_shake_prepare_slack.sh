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
  # Build Block Kit blocks (header + link + divider + per-candidate sections)
  SLACK_BLOCKS=$(jq -c \
    --arg url "${CIRCLE_BUILD_URL:-}" \
    --slurpfile meta "${PROMO_JSON%/*}/metadata.json" '
    def name_or_pkg(t): (if ((t.test_name|tostring)|length) == 0 then "(package)" else t.test_name end);
    def testblocks(t): [
      {"type":"section","fields":[
        {"type":"mrkdwn","text":"*Test:*\n\(name_or_pkg(t))"},
        {"type":"mrkdwn","text":"*Package:*\n\(t.package)"}
      ]},
      {"type":"section","fields":[
        {"type":"mrkdwn","text":"*Runs:*\n\(t.total_runs)"},
        {"type":"mrkdwn","text":"*Pass Rate:*\n\((t.pass_rate|tostring))%"}
      ]},
      {"type":"context","elements":[{"type":"mrkdwn","text": t.package }]},
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
