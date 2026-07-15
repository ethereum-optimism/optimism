#!/usr/bin/env bash

set -euo pipefail

REGISTRY_URL="https://raw.githubusercontent.com/ethereum-optimism/superchain-registry/main"
PROXY_ARTIFACT="src/universal/Proxy.sol:Proxy"
# Every predeploy proxy is constructed with the L2ProxyAdmin predeploy.
PROXY_ARGS="0x0000000000000000000000004200000000000000000000000000000000000018"
ZERO_ADDRESS="0x0000000000000000000000000000000000000000"
MAX_PARALLEL_CHAINS=6

UPGRADE=""
NETWORK="mainnet"
DRY_RUN=false
WATCH=true
CHAINS=()
FAILURES=0
LAST_STATUS=""
LAST_OUTPUT=""
CHAIN_FAILURES=0
WORKER_PIDS=()

die() {
  echo "error: $*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage:
  just post-fork-ops --upgrade u19 --chains op,ink

Options:
  --upgrade <id>             Upgrade identifier. Currently supported: u19.
  --chains <slug[,slug...]>  Superchain registry chain slugs.
  --network <name>           Registry network. Default: mainnet.
  --dry-run                  Print forge commands without submitting them.
  --no-watch                 Submit without waiting for explorer results.
  -h, --help                 Show this help.
EOF
}

parse_args() {
  local chains=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --upgrade) [[ $# -ge 2 ]] || die "--upgrade requires a value"; UPGRADE="$2"; shift 2 ;;
      --upgrade=*) UPGRADE="${1#*=}"; shift ;;
      --chains) [[ $# -ge 2 ]] || die "--chains requires a value"; chains="$2"; shift 2 ;;
      --chains=*) chains="${1#*=}"; shift ;;
      --network) [[ $# -ge 2 ]] || die "--network requires a value"; NETWORK="$2"; shift 2 ;;
      --network=*) NETWORK="${1#*=}"; shift ;;
      --dry-run) DRY_RUN=true; shift ;;
      --no-watch) WATCH=false; shift ;;
      -h|--help) usage; exit 0 ;;
      *) die "unknown argument: $1" ;;
    esac
  done

  [[ -n "$UPGRADE" ]] || die "missing --upgrade <id>"
  [[ -n "$chains" ]] || die "missing --chains <slugs>"
  IFS=',' read -r -a CHAINS <<< "$chains"
  for i in "${!CHAINS[@]}"; do
    CHAINS[i]="${CHAINS[i]//[[:space:]]/}"
    [[ -n "${CHAINS[$i]}" ]] || die "empty chain slug"
  done
}

resolve_upgrade() {
  case "$(printf '%s' "$UPGRADE" | tr '[:upper:]' '[:lower:]')" in
    u19)
      RELEASE="op-contracts/v7.0.0-rc.3"
      # Karst's NUT bundle was generated here, before an inlined DevFeatures change.
      SOURCE_REF="79a83f3ce70b0f3c54cb058a1140fa50e57268bc"
      ;;
    *) die "unknown upgrade '$UPGRADE' (known: u19)" ;;
  esac
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

fetch_chain() {
  local slug="$1"
  local url="$REGISTRY_URL/superchain/configs/$NETWORK/$slug.toml"
  local body values

  # Chain RPC and explorer metadata belong to the registry, not this script.
  body="$(curl -fsSL "$url")" || die "fetching $url"
  values="$(printf '%s\n' "$body" | yq -p toml -r '[.name, .chain_id, .public_rpc, .explorer] | @tsv')"
  IFS=$'\t' read -r CHAIN_NAME CHAIN_ID CHAIN_RPC CHAIN_EXPLORER <<< "$values"
  CHAIN_SLUG="$slug"
  [[ -n "$CHAIN_NAME" && -n "$CHAIN_ID" && -n "$CHAIN_RPC" && -n "$CHAIN_EXPLORER" ]] \
    || die "$slug.toml is missing required chain metadata"
}

predeploys() {
  # Supported releases use either the legacy list or structured records as their source of truth.
  perl -0ne '
    my %address;
    while (/address\s+internal\s+constant\s+(\w+)\s*=\s*(0x[0-9a-fA-F]{40});/g) {
      $address{$1} = lc $2;
    }

    if (my ($body) = /function getUpgradeablePredeploys\(\).*?\{(.*?)\n    \}/s) {
      my %name;
      while (/if \(_addr == (\w+)\) return "([^"]+)";/g) {
        $name{$1} = $2;
      }
      while ($body =~ /predeploys_\[\d+\]\s*=\s*Predeploys\.(\w+);/g) {
        my $constant = $1;
        die "missing address or name for $constant\n" if !$address{$constant} || !$name{$constant};
        print "$address{$constant}\t$name{$constant}\n";
      }
      next;
    }

    my $count = 0;
    while (/records_\[\d+\]\s*=\s*PredeployRecord\(\{(.*?)\n\s*\}\);/sg) {
      my $record = $1;
      next if $record !~ /isProxied:\s*true/ || $record !~ /isDeprecated:\s*false/;
      my ($constant) = $record =~ /proxy:\s*(\w+)/;
      my ($name) = $record =~ /variants:\s*_variants\(\s*"([^"]+)"/s;
      die "missing address or name for record\n" if !$constant || !$address{$constant} || !$name;
      print "$address{$constant}\t$name\n";
      $count++;
    }
    die "unsupported Predeploys.sol layout\n" if !$count;
  ' "$PREDEPLOYS_SOL"
}

retry() {
  # Retry transport failures only. Source and bytecode mismatches should fail immediately.
  local output status attempt
  for attempt in 1 2 3; do
    set +e
    output="$("$@" 2>&1)"
    status=$?
    set -e
    if [[ $status -eq 0 ]] || ! grep -Eiq \
      'connection reset|error sending request|deserialize response|internal server error|early EOF|invalid index-pack|unexpected disconnect' \
      <<< "$output"; then
      printf '%s\n' "$output"
      return "$status"
    fi
    if [[ $attempt -lt 3 ]]; then
      echo "transient failure, retrying ($attempt/3)" >&2
      sleep "$attempt"
    fi
  done
  printf '%s\n' "$output"
  return "$status"
}

artifact_for() {
  local name="$1" source=""
  # Supported releases have unique source basenames; find returns the first match if that changes.
  while IFS= read -r source; do break; done < <(find "$CONTRACTS_DIR/src" -name "$name.sol" -print)
  [[ -n "$source" ]] || return 1
  printf '%s:%s\n' "${source#"$CONTRACTS_DIR"/}" "$name"
}

compiler_version() {
  # Explorers require the full compiler build ID, which is more specific than the pragma.
  local artifact="$1" source pragma solc version
  source="$CONTRACTS_DIR/${artifact%%:*}"
  pragma="$(sed -n 's/^pragma solidity[[:space:]]*\([^;]*\);/\1/p' "$source" | head -n 1)"
  [[ "$pragma" == 0.8.* ]] || return 1
  solc="$HOME/.svm/$pragma/solc-$pragma"
  if [[ -x "$solc" ]]; then
    version="$("$solc" --version | sed -n 's/^Version: \([0-9][^+]*+commit\.[0-9a-fA-F]*\).*/v\1/p' | head -n 1)"
  fi
  printf '%s\n' "${version:-$pragma}"
}

api_url() {
  local explorer="${1%/}"
  [[ "$explorer" == */api ]] && printf '%s\n' "$explorer" || printf '%s/api\n' "$explorer"
}

blockscout_verified() {
  # Blockscout can report failure before eth-bytecode-db finishes importing the source.
  local address="$1" response attempt
  [[ "$CHAIN_SLUG" != "unichain" ]] || return 1
  for attempt in 1 2 3; do
    response="$(curl -fsSL "${CHAIN_EXPLORER%/}/api/v2/smart-contracts/$address" 2>/dev/null || true)"
    jq -e '.is_verified == true' <<< "$response" >/dev/null 2>&1 && return 0
    [[ $attempt -eq 3 ]] || sleep 2
  done
  return 1
}

last_line() {
  sed '/^[[:space:]]*$/d' <<< "$1" | tail -n 1
}

single_line() {
  printf '%s' "$1" | tr '\t\n' '  '
}

record() {
  # One normalized row per target drives both the summary and detailed report.
  printf '%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$1" "$CHAIN_SLUG" "$(single_line "$2")" "$3" "$4" "$(single_line "$5")" >> "$REPORT_FILE"
}

run_verify() {
  local label="$1" address="$2" artifact="$3" constructor_args="${4:-}"
  local version output status result
  local args=(verify-contract "$address" "$artifact" --chain "$CHAIN_ID" --rpc-url "$CHAIN_RPC")

  version="$(compiler_version "$artifact")" || {
    LAST_STATUS="failed"; LAST_OUTPUT="cannot resolve compiler version for $artifact"; return 1;
  }
  args+=(--compiler-version "$version" --compilation-profile default --retries 10 --delay 10)
  # Unichain uses Etherscan; the other supported explorers use Blockscout.
  if [[ "$CHAIN_SLUG" == "unichain" ]]; then
    args+=(--verifier etherscan)
  else
    args+=(--verifier blockscout --verifier-url "$(api_url "$CHAIN_EXPLORER")")
  fi
  [[ -z "$constructor_args" ]] || args+=(--constructor-args "$constructor_args")
  [[ "$WATCH" == false ]] || args+=(--watch)

  if [[ "$DRY_RUN" == true ]]; then
    printf -v LAST_OUTPUT 'cd %q && forge' "$CONTRACTS_DIR"
    printf -v output ' %q' "${args[@]}"
    LAST_OUTPUT+="$output"
    LAST_STATUS="dry-run"
    echo "[$CHAIN_SLUG] dry-run $label $address"
    return 0
  fi

  echo "[$CHAIN_SLUG] verifying $label $address"
  set +e
  output="$(cd "$CONTRACTS_DIR" && retry forge "${args[@]}")"
  status=$?
  set -e
  LAST_OUTPUT="$output"

  if grep -qi 'already verified' <<< "$output"; then
    LAST_STATUS="already-verified"; result=0
  elif grep -qi 'Contract successfully verified' <<< "$output"; then
    LAST_STATUS="verified"; result=0
  elif grep -Eqi 'Details: .?Fail|Fail -|Unable to verify|Response: .?NOTOK|Failed verify submission|Error: Failed to verify' <<< "$output"; then
    if blockscout_verified "$address"; then
      LAST_STATUS="already-verified"; result=0
    else
      LAST_STATUS="failed"; result=1
    fi
  elif grep -qi 'Submitted contract for verification' <<< "$output"; then
    LAST_STATUS="submitted"; result=0
  elif [[ $status -eq 0 ]]; then
    LAST_STATUS="verified"; result=0
  else
    LAST_STATUS="failed"; result=$status
  fi
  echo "[$CHAIN_SLUG] $LAST_STATUS $label $address"
  return "$result"
}

verify_target() {
  local label="$1" address="$2" artifact="$3" constructor_args="${4:-}"
  if run_verify "$label" "$address" "$artifact" "$constructor_args"; then
    record "$LAST_STATUS" "$label" "$address" "$artifact" "$(last_line "$LAST_OUTPUT")"
  else
    record failed "$label" "$address" "$artifact" "$(last_line "$LAST_OUTPUT")"
    FAILURES=$((FAILURES + 1))
  fi
}

verify_implementation() {
  local name="$1" address="$2" variant artifact
  local errors=()
  # On CGT chains, Forge may exhaust retries on the default artifact before trying the CGT variant.
  for variant in "$name" "${name}CGT"; do
    artifact="$(artifact_for "$variant")" || continue
    if run_verify "$variant Implementation" "$address" "$artifact"; then
      record "$LAST_STATUS" "$variant Implementation" "$address" "$artifact" "$(last_line "$LAST_OUTPUT")"
      return
    fi
    errors+=("$artifact: $(last_line "$LAST_OUTPUT")")
  done
  record failed "$name Implementation" "$address" "$name" "${errors[*]}"
  FAILURES=$((FAILURES + 1))
}

rpc_code() {
  retry cast code "$1" --rpc-url "$CHAIN_RPC"
}

implementation() {
  # Proxy allows address(0) for eth_call introspection; other callers may hit fallback.
  retry cast call --from "$ZERO_ADDRESS" "$1" 'implementation()(address)' --rpc-url "$CHAIN_RPC"
}

verify_chain() {
  local slug="$1" proxy name code impl
  fetch_chain "$slug"
  echo "Verifying U19 predeploys on $CHAIN_NAME ($CHAIN_ID)"

  # Verify every deployed proxy, then verify its current implementation when enabled.
  while IFS=$'\t' read -r proxy name; do
    if ! code="$(rpc_code "$proxy")"; then
      record failed "$name Proxy" "$proxy" "$PROXY_ARTIFACT" "cast code failed"
      FAILURES=$((FAILURES + 1)); continue
    fi
    if [[ "$code" == "0x" ]]; then
      record skipped "$name" "$proxy" "$PROXY_ARTIFACT" "no proxy code"; continue
    fi
    verify_target "$name Proxy" "$proxy" "$PROXY_ARTIFACT" "$PROXY_ARGS"

    if ! impl="$(implementation "$proxy")"; then
      record failed "$name Implementation" "$proxy" "$name" "implementation() failed"
      FAILURES=$((FAILURES + 1)); continue
    fi
    if [[ "$(tr '[:upper:]' '[:lower:]' <<< "$impl")" == "$ZERO_ADDRESS" ]]; then
      record skipped "$name Implementation" "$ZERO_ADDRESS" "$name" "disabled (zero implementation)"; continue
    fi
    if ! code="$(rpc_code "$impl")" || [[ "$code" == "0x" ]]; then
      record failed "$name Implementation" "$impl" "$name" "implementation has no code"
      FAILURES=$((FAILURES + 1)); continue
    fi
    verify_implementation "$name" "$impl"
  done < "$PREDEPLOY_FILE"
}

run_chain_batch() {
  local start="$1" end="$2" i pid log report
  local logs=() reports=()
  WORKER_PIDS=()

  # Each worker gets isolated output so concurrent chains never interleave logs or report rows.
  for ((i = start; i < end; i++)); do
    log="$WORKTREE/chain.$i.log"
    report="$WORKTREE/report.$i.tsv"
    : > "$report"
    echo "[${CHAINS[$i]}] started"
    REPORT_FILE="$report" verify_chain "${CHAINS[$i]}" > "$log" 2>&1 &
    pid=$!
    WORKER_PIDS+=("$pid")
    logs+=("$log")
    reports+=("$report")
  done

  # Replay complete chain blocks in input order after the batch has run concurrently.
  for i in "${!WORKER_PIDS[@]}"; do
    if ! wait "${WORKER_PIDS[$i]}"; then
      CHAIN_FAILURES=$((CHAIN_FAILURES + 1))
    fi
    WORKER_PIDS[i]=""
    cat "${logs[$i]}"
    cat "${reports[$i]}" >> "$REPORT_FILE"
  done
}

run_chains() {
  local start end total="${#CHAINS[@]}"
  echo "Verifying $total chain(s) with up to $MAX_PARALLEL_CHAINS workers"
  for ((start = 0; start < total; start += MAX_PARALLEL_CHAINS)); do
    end=$((start + MAX_PARALLEL_CHAINS))
    if ((end > total)); then end="$total"; fi
    run_chain_batch "$start" "$end"
  done
}

count() {
  awk -F '\t' -v status="$1" -v chain="${2:-}" \
    '$1 == status && (chain == "" || $2 == chain) { n++ } END { print n + 0 }' "$REPORT_FILE"
}

report_details() {
  local status="$1" title="$2"
  [[ "$(count "$status")" != 0 ]] || return 0
  echo
  echo "$title"
  awk -F '\t' -v status="$status" '$1 == status {
    printf "- [%s] %s %s (%s) %s\n", $2, $3, $5, $6, $4
  }' "$REPORT_FILE"
}

print_report() {
  local chain
  echo
  echo "Verification summary"
  printf '%-12s %8s %8s %9s %8s %8s %7s\n' chain existing verified submitted dry-run skipped failed
  for chain in "${CHAINS[@]}"; do
    printf '%-12s %8s %8s %9s %8s %8s %7s\n' "$chain" \
      "$(count already-verified "$chain")" "$(count verified "$chain")" \
      "$(count submitted "$chain")" "$(count dry-run "$chain")" \
      "$(count skipped "$chain")" "$(count failed "$chain")"
  done
  report_details verified "Verified this run"
  report_details submitted "Submitted"
  report_details dry-run "Dry run"
  report_details skipped "Skipped"
  report_details failed "Failed"
}

cleanup() {
  # Release builds and reports stay in a temporary worktree, leaving the caller's checkout alone.
  local pid
  for pid in "${WORKER_PIDS[@]}"; do
    [[ -z "$pid" ]] || kill "$pid" >/dev/null 2>&1 || true
  done
  for pid in "${WORKER_PIDS[@]}"; do
    [[ -z "$pid" ]] || wait "$pid" >/dev/null 2>&1 || true
  done
  [[ -z "${WORKTREE:-}" || ! -d "$WORKTREE" ]] || git -C "$REPO_ROOT" worktree remove --force "$WORKTREE" >/dev/null 2>&1
}

main() {
  parse_args "$@"
  resolve_upgrade
  for command in git curl perl yq jq cast forge; do need "$command"; done

  REPO_ROOT="$(git rev-parse --show-toplevel)"
  WORKTREE="$(mktemp -d "${TMPDIR:-/tmp}/post-fork-ops.XXXXXX")"
  trap cleanup EXIT
  git -C "$REPO_ROOT" worktree add --detach "$WORKTREE" "$SOURCE_REF"
  retry git -C "$WORKTREE" submodule update --init --recursive packages/contracts-bedrock/lib

  CONTRACTS_DIR="$WORKTREE/packages/contracts-bedrock"
  PREDEPLOYS_SOL="$CONTRACTS_DIR/src/libraries/Predeploys.sol"
  PREDEPLOY_FILE="$WORKTREE/predeploys.tsv"
  REPORT_FILE="$WORKTREE/report.tsv"
  : > "$REPORT_FILE"
  predeploys > "$PREDEPLOY_FILE"
  [[ -s "$PREDEPLOY_FILE" ]] || die "no upgradeable predeploys found"

  echo "Upgrade $UPGRADE uses release $RELEASE"
  echo "Verification source: $SOURCE_REF"
  echo "Found $(wc -l < "$PREDEPLOY_FILE" | tr -d ' ') upgradeable predeploys"
  run_chains
  FAILURES="$(count failed)"
  print_report
  [[ $CHAIN_FAILURES -eq 0 ]] || die "$CHAIN_FAILURES chain worker(s) failed"
  [[ $FAILURES -eq 0 ]] || die "$FAILURES verification target(s) failed"
}

main "$@"
