#!/usr/bin/env bash
#
# kona-sp1 range/aggregation proof benchmark suite.
#
# Three host roles, run as separate modes against a shared data dir:
#
#   plan [--check]       Print the planned ranges (no RPC fetch, no proving).
#                        --check: read-only report of the anchor state and which
#                        stdin/proof files already exist (writes nothing).
#   fetch_range [-j N]   Fetch + save the SP1 stdin for every planned range.
#                        Runs on a host with RPC access; does NOT prove and (via
#                        range-bench --no-execute) does NOT execute. -j sets how
#                        many fetches run in parallel (default 1).
#   fetch_agg P...       Fetch + save the RPC-derived aggregation inputs (L1
#                        checkpoint + header preimages) for an explicit, contiguous
#                        list of range proofs, via agg-bench --save-agg-inputs.
#                        Runs on a host with RPC access; does NOT prove. (agg-bench
#                        still sets up keys and runs one execute pass, so the ELFs
#                        and the range proofs must be present here.)
#   prove                Generate a compressed range proof for every saved stdin.
#                        Runs on the proving host; needs no RPC.
#   agg [--inputs F] [--plonk] P...
#                        Aggregate an explicit, contiguous list of range proofs.
#                        Compressed by default; --plonk for the final PLONK proof.
#                        Uses saved agg inputs (--inputs F, else the conventional
#                        agg_inputs_<s>_<e>.cbor) to run without RPC; otherwise
#                        fetches the L1 checkpoint + header preimages from RPC.
#
# Ranges are identified by their L2 block start/end pair and stored as:
#   <DATA_DIR>/stdin_<start>_<end>.bin       (fetch_range output)
#   <DATA_DIR>/agg_inputs_<start>_<end>.cbor (fetch_agg output)
#   <DATA_DIR>/proof_<start>_<end>.bin       (prove output, compressed)
#   <DATA_DIR>/agg_<start>_<end>.bin         (compressed aggregation proof)
#   <DATA_DIR>/agg_<start>_<end>.plonk.bin   (PLONK aggregation proof)
#
# Timing data is appended to <DATA_DIR>/timings.tsv; full per-job output goes to
# <DATA_DIR>/logs/. Both are meant to be analyzed later.
#
set -euo pipefail

# ============================================================================
# Configuration -- fill these in (or export them before running).
# ============================================================================

# RPC endpoints (same vars the bench binaries read). Needed for `plan`, `fetch`,
# and `agg`. Not needed for `prove`.
export L1_RPC="${L1_RPC:-}"
export L2_RPC="${L2_RPC:-}"
export L2_NODE_RPC="${L2_NODE_RPC:-}"
# Required for post-Ecotone / blob-backed ranges:
export L1_BEACON_RPC="${L1_BEACON_RPC:-}"

# Proving backend, passed straight through to SP1's ProverClient::from_env().
# Unset/empty = local CPU; "cuda" = local GPU; "network"/"hosted" = SP1 network
# (also set NETWORK_PRIVATE_KEY for those).
export SP1_PROVER="${SP1_PROVER:-}"
export NETWORK_PRIVATE_KEY="${NETWORK_PRIVATE_KEY:-}"

# Extra cargo features for the bench binaries (e.g. "gpu" for CUDA proving).
FEATURES="${FEATURES:-}"

# Where stdin, proofs, logs and timings live. Safe to share across hosts.
DATA_DIR="${DATA_DIR:-$HOME/kona-sp1-bench-data}"

# End of the variable-size contiguous range set, walking backwards. "auto" uses
# the chain's latest finalized block, then PERSISTS it to <DATA_DIR>/anchor.txt
# and reuses it on later runs so block numbers (and filenames) stay stable and
# already-fetched ranges are skipped instead of re-fetched. Delete anchor.txt (or
# set a fixed number here) to re-anchor. IMPORTANT: to reuse ranges fetched
# before this anchor was persisted, set this to the block number you used then.
ANCHOR_BLOCK="${ANCHOR_BLOCK:-auto}"

# Variable-size set: a descending sweep from 3600 down to 100, laid out
# contiguously (ending at the anchor) so their proofs can be aggregated. Editing
# this list changes the contiguous layout and therefore every range's block
# numbers -- keep it stable once you have fetched data. Short proving-time data
# points live in EXTRA_SIZES below instead, so they don't perturb this set.
SIZES=(3600 1800 1350 900 450 225 100)

# Extra standalone ranges for short proving-time data points (~10-20h on CPU at
# the observed ~2h/1B cycles). Placed contiguously just below the variable-size
# set, but kept out of SIZES so adding/removing them never shifts the larger
# ranges' block numbers (and thus never orphans their already-fetched files).
EXTRA_SIZES=(40 20)

# Freshness set: same-size ranges sampled at different archival depths, to
# measure how fetch cost grows with age. Not contiguous (independent points in
# time); not meant to be aggregated.
FRESHNESS_SIZE="${FRESHNESS_SIZE:-20}"
FRESHNESS_LABELS=(1h 1d 1w 2w 3w)
FRESHNESS_SECONDS=(3600 86400 604800 1209600 1814400)

# ============================================================================
# Internals
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TIMINGS="$DATA_DIR/timings.tsv"
LOG_DIR="$DATA_DIR/logs"
ANCHOR_FILE="$DATA_DIR/anchor.txt"

# Activate mise so the pinned `just` / toolchain are on PATH, if available.
if command -v mise >/dev/null 2>&1; then
  eval "$(mise activate bash)"
fi

die() { echo "error: $*" >&2; exit 1; }

# Sub-second wall clock. `date +%N` is GNU-only (BSD/macOS date lacks it), so use
# python3 (already required by the range planner) for portability.
now_s() { python3 -c 'import time; print(time.time())'; }
elapsed() { awk "BEGIN{printf \"%.3f\", $2 - $1}"; }

# Append one row to the timings file. Parallel fetch jobs append concurrently;
# a single short line written with >> is atomic on a local filesystem (well under
# PIPE_BUF), so no flock is needed -- which also keeps this portable to macOS.
log_timing() {
  # columns: iso_time  phase  kind  label  start  end  size  wall_s  extra  log
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$@" >> "$TIMINGS"
}

iso_now() { date -u +%Y-%m-%dT%H:%M:%SZ; }

ensure_data_dir() {
  mkdir -p "$DATA_DIR" "$LOG_DIR"
  if [ ! -f "$TIMINGS" ]; then
    printf 'iso_time\tphase\tkind\tlabel\tstart\tend\tsize\twall_s\textra\tlog\n' > "$TIMINGS"
  fi
}

require_rpc() {
  [ -n "$L2_RPC" ] || die "L2_RPC is not set (edit the config block at the top of this script)"
  [ -n "$L1_RPC" ] || die "L1_RPC is not set"
  [ -n "$L2_NODE_RPC" ] || die "L2_NODE_RPC is not set"
}

require_elfs() {
  [ -s "$SCRIPT_DIR/elf/range-elf" ] || die "elf/range-elf missing or empty -- build it first with: just build-elfs"
  [ -s "$SCRIPT_DIR/elf/aggregation-elf" ] || die "elf/aggregation-elf missing or empty -- build it first with: just build-elfs"
}

# SP1 cycle-tracker noise. The kona guest prints `cycle-tracker-report-start/end:`
# markers per section and per precompile call; with SP1's `profiling` feature off
# (our default) the executor echoes each as `stdout: <line>` to stderr, carrying
# no cycle counts -- pure noise that dominates large-range logs. The `[┌└]╴` glyph
# branch additionally strips the tree output SP1 emits when profiling IS enabled.
CYCLE_TRACKER_RE='cycle-tracker-(report-)?(start|end):|[┌└]╴'

# Run a bench `just` target, filtering cycle-tracker noise out of the captured
# log. Piping through the filter would lose the bench's exit code, so it is
# written to a temp file inside the pipeline and read back afterwards (bash waits
# for the whole pipeline, so the file is complete by then). No `sed -i`, which is
# not portable between GNU and BSD/macOS.
run_bench() {
  local logfile="$1"; shift
  local rcfile; rcfile="$(mktemp "${TMPDIR:-/tmp}/kona-bench-rc.XXXXXX")"
  ( cd "$SCRIPT_DIR" && just features="$FEATURES" "$@" 2>&1; echo "$?" > "$rcfile" ) \
    | grep -avE "$CYCLE_TRACKER_RE" > "$logfile"
  local rc; rc="$(cat "$rcfile")"; rm -f "$rcfile"
  return "${rc:-1}"
}

# ----------------------------------------------------------------------------
# Range planning (emits TSV rows: kind <tab> label <tab> start <tab> end <tab> size)
# ----------------------------------------------------------------------------
plan_ranges() {
  [ -n "$L2_RPC" ] || die "L2_RPC is not set (edit the config block at the top of this script)"
  # In auto mode, reuse a previously persisted anchor so the layout (and hence
  # every filename) stays stable across runs and already-fetched ranges are
  # skipped rather than re-fetched.
  local pinned=""
  if [ "$ANCHOR_BLOCK" = "auto" ] && [ -f "$ANCHOR_FILE" ]; then
    pinned="$(cat "$ANCHOR_FILE")"
  fi
  local out
  out="$(
  SIZES_STR="${SIZES[*]}" \
  EXTRA_SIZES_STR="${EXTRA_SIZES[*]}" \
  FRESHNESS_LABELS_STR="${FRESHNESS_LABELS[*]}" \
  FRESHNESS_SECONDS_STR="${FRESHNESS_SECONDS[*]}" \
  FRESHNESS_SIZE="$FRESHNESS_SIZE" \
  ANCHOR_BLOCK="$ANCHOR_BLOCK" \
  PINNED_ANCHOR="$pinned" \
  L2_RPC="$L2_RPC" \
  python3 - <<'PY'
import os, json, ssl, urllib.request

L2 = os.environ["L2_RPC"]

# Do not verify the RPC's TLS certificate (e.g. self-signed devnet endpoints).
SSL_CTX = ssl._create_unverified_context()

def rpc(method, params):
    req = urllib.request.Request(
        L2,
        data=json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}).encode(),
        headers={"content-type": "application/json"},
    )
    with urllib.request.urlopen(req, timeout=30, context=SSL_CTX) as r:
        d = json.load(r)
    if d.get("error"):
        raise SystemExit(f"RPC error {method}: {d['error']}")
    return d["result"]

def block(tag):
    b = rpc("eth_getBlockByNumber", [tag, False])
    if b is None:
        raise SystemExit(f"block {tag} not found")
    return b

sizes = [int(x) for x in os.environ["SIZES_STR"].split()]
extra_sizes = [int(x) for x in os.environ.get("EXTRA_SIZES_STR", "").split()]
labels = os.environ["FRESHNESS_LABELS_STR"].split()
secs = [int(x) for x in os.environ["FRESHNESS_SECONDS_STR"].split()]
fsize = int(os.environ["FRESHNESS_SIZE"])
anchor_cfg = os.environ.get("ANCHOR_BLOCK", "auto")
pinned = os.environ.get("PINNED_ANCHOR", "").strip()

latest = block("latest")
latest_num = int(latest["number"], 16)
latest_ts = int(latest["timestamp"], 16)

# Resolve the anchor: explicit config > persisted (pinned) > live finalized.
if anchor_cfg != "auto":
    anchor = int(anchor_cfg, 0)
elif pinned:
    anchor = int(pinned, 0)
else:
    anchor = int(block("finalized")["number"], 16)

total = sum(sizes)
start_set = anchor - total
if start_set < 1:
    raise SystemExit(f"anchor {anchor} too small for total span {total}")

rows = []
cur = start_set
for sz in sizes:
    rows.append(("var", "-", cur, cur + sz, sz))
    cur += sz  # contiguous: each range's end is the next range's start

# Extra standalone ranges: laid out contiguously just below the variable-size
# set (ending at start_set). They are independent of `sizes`, so adding or
# removing them never moves the variable-size ranges' block numbers.
extra_cur = start_set - sum(extra_sizes)
if extra_sizes and extra_cur < 1:
    raise SystemExit(f"anchor {anchor} too small for extra span {sum(extra_sizes)}")
for sz in extra_sizes:
    rows.append(("extra", "-", extra_cur, extra_cur + sz, sz))
    extra_cur += sz

# Derive L2 block time from a wide sample to place the freshness ranges.
ref_num = max(1, latest_num - 5000)
ref_ts = int(block(hex(ref_num))["timestamp"], 16)
span = latest_num - ref_num
block_time = (latest_ts - ref_ts) / span if span > 0 else 2.0
if block_time <= 0:
    block_time = 2.0

for label, sec in zip(labels, secs):
    s = latest_num - int(round(sec / block_time))
    if s < 1:
        continue
    rows.append(("fresh", label, s, s + fsize, fsize))

# First line carries the resolved anchor so the caller can persist it.
print(f"#anchor\t{anchor}")
for r in rows:
    print("\t".join(str(x) for x in r))
PY
  )" || die "range planning failed (see error above)"

  # Persist the resolved anchor (first run, auto mode) so later runs are stable.
  local resolved
  resolved="$(printf '%s\n' "$out" | sed -n 's/^#anchor[[:space:]]*//p')"
  if [ "${PLAN_READONLY:-0}" != 1 ] && [ "$ANCHOR_BLOCK" = "auto" ] && [ -n "$resolved" ] &&
    [ ! -f "$ANCHOR_FILE" ]; then
    mkdir -p "$DATA_DIR"
    printf '%s\n' "$resolved" > "$ANCHOR_FILE"
    echo "persisted anchor $resolved -> $ANCHOR_FILE (delete to re-anchor)" >&2
  fi

  # Emit the plan rows, dropping the #anchor header line.
  printf '%s\n' "$out" | grep -v '^#anchor'
}

# ----------------------------------------------------------------------------
# Modes
# ----------------------------------------------------------------------------
mode_plan() {
  local check=0
  while [ $# -gt 0 ]; do
    case "$1" in
      --check) check=1; shift ;;
      *) die "unknown plan arg: $1" ;;
    esac
  done
  [ "$check" -eq 1 ] && { mode_plan_check; return; }

  echo "Data dir : $DATA_DIR"
  echo "Anchor   : $ANCHOR_BLOCK"
  echo "Backend  : ${SP1_PROVER:-cpu}${FEATURES:+ (features: $FEATURES)}"
  echo
  printf '%-6s %-6s %-12s %-12s %-7s\n' kind label start end size
  printf '%-6s %-6s %-12s %-12s %-7s\n' ---- ----- ----- --- ----
  plan_ranges | while IFS=$'\t' read -r kind label start end size; do
    printf '%-6s %-6s %-12s %-12s %-7s\n' "$kind" "$label" "$start" "$end" "$size"
  done
}

# Read-only status report: anchor state + which stdin/proof files exist for the
# planned ranges. Touches nothing on disk (no anchor persist, no dirs created).
mode_plan_check() {
  PLAN_READONLY=1
  echo "Data dir : $DATA_DIR $([ -d "$DATA_DIR" ] && echo '(exists)' || echo '(does not exist yet)')"
  echo "Anchor cfg : $ANCHOR_BLOCK"
  if [ -f "$ANCHOR_FILE" ]; then
    echo "Anchor pin : $(cat "$ANCHOR_FILE") (from $ANCHOR_FILE)"
  elif [ "$ANCHOR_BLOCK" != "auto" ]; then
    echo "Anchor pin : $ANCHOR_BLOCK (from config; not yet persisted)"
  else
    echo "Anchor pin : (none -- auto will compute from finalized and persist on first fetch)"
  fi
  echo

  local plan; plan="$(plan_ranges)"
  printf '%-6s %-5s %-12s %-12s %-6s %-8s %-8s\n' kind label start end size stdin proof
  printf '%-6s %-5s %-12s %-12s %-6s %-8s %-8s\n' ---- ----- ----- --- ---- ----- -----
  local n=0 ns=0 np=0
  while IFS=$'\t' read -r kind label start end size; do
    [ -n "$kind" ] || continue
    local sflag="MISSING" pflag="--"
    if [ -s "$DATA_DIR/stdin_${start}_${end}.bin" ]; then sflag="present"; ns=$((ns + 1)); fi
    if [ -s "$DATA_DIR/proof_${start}_${end}.bin" ]; then pflag="present"; np=$((np + 1)); fi
    n=$((n + 1))
    printf '%-6s %-5s %-12s %-12s %-6s %-8s %-8s\n' "$kind" "$label" "$start" "$end" "$size" "$sflag" "$pflag"
  done <<< "$plan"
  echo
  echo "summary: stdin ${ns}/${n} present | proof ${np}/${n} present"

  # List any aggregation artifacts already on disk (keyed by proof-set range, so
  # not derivable from the plan).
  shopt -s nullglob
  local agg=("$DATA_DIR"/agg_inputs_*.cbor "$DATA_DIR"/agg_*.bin "$DATA_DIR"/agg_*.plonk.bin)
  shopt -u nullglob
  if [ "${#agg[@]}" -gt 0 ]; then
    echo
    echo "aggregation artifacts present:"
    local f
    for f in "${agg[@]}"; do echo "  $(basename "$f")"; done
  fi
}

fetch_one() {
  local kind="$1" label="$2" start="$3" end="$4" size="$5"
  local out="$DATA_DIR/stdin_${start}_${end}.bin"
  local log="$LOG_DIR/fetch_${start}_${end}.log"
  if [ -s "$out" ]; then
    echo "skip fetch ${start}-${end} (stdin exists)"
    return 0
  fi
  echo "fetch ${kind} ${label} ${start}-${end} (size ${size})"
  local t0 t1 wall
  t0="$(now_s)"
  if run_bench "$log" range-bench --start "$start" --end "$end" --save-stdin "$out" --no-execute; then
    t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
    log_timing "$(iso_now)" fetch "$kind" "$label" "$start" "$end" "$size" "$wall" - "$log"
  else
    t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
    log_timing "$(iso_now)" fetch-FAILED "$kind" "$label" "$start" "$end" "$size" "$wall" - "$log"
    echo "  FAILED -- see $log" >&2
  fi
}

mode_fetch_range() {
  local jobs=1
  while [ $# -gt 0 ]; do
    case "$1" in
      -j) jobs="$2"; shift 2 ;;
      -j*) jobs="${1#-j}"; shift ;;
      *) die "unknown fetch arg: $1" ;;
    esac
  done
  if ! [[ "$jobs" =~ ^[0-9]+$ ]] || [ "$jobs" -lt 1 ]; then
    die "-j must be a positive integer"
  fi
  ensure_data_dir
  require_rpc

  # Resolve the plan once so parallel re-runs share identical block numbers.
  local plan; plan="$(plan_ranges)"
  [ -n "$plan" ] || die "empty plan"

  # Warm the build serially so parallel jobs don't race the cargo compile.
  echo "warming build (cargo compile of range-bench)..."
  ( cd "$SCRIPT_DIR" && just features="$FEATURES" range-bench --help ) >/dev/null 2>&1 || true

  echo "fetching with -j ${jobs}"
  while IFS=$'\t' read -r kind label start end size; do
    [ -n "$kind" ] || continue
    # Poll instead of `wait -n` (the latter needs bash 4.3+; macOS ships 3.2).
    while [ "$(jobs -rp | wc -l)" -ge "$jobs" ]; do sleep 0.2; done
    fetch_one "$kind" "$label" "$start" "$end" "$size" &
  done <<< "$plan"
  wait
  echo "fetch complete. timings -> $TIMINGS"
}

mode_prove() {
  ensure_data_dir
  require_elfs
  shopt -s nullglob
  local stdins=("$DATA_DIR"/stdin_*.bin)
  shopt -u nullglob
  [ "${#stdins[@]}" -gt 0 ] || die "no stdin_*.bin files in $DATA_DIR -- run fetch_range first"

  for stdin in "${stdins[@]}"; do
    local base se start end
    base="$(basename "$stdin" .bin)"; se="${base#stdin_}"
    start="${se%_*}"; end="${se#*_}"
    local proof="$DATA_DIR/proof_${start}_${end}.bin"
    local log="$LOG_DIR/prove_${start}_${end}.log"
    if [ -s "$proof" ]; then
      echo "skip prove ${start}-${end} (proof exists)"
      continue
    fi
    echo "prove ${start}-${end}"
    local t0 t1 wall extra
    t0="$(now_s)"
    if run_bench "$log" range-bench --load-stdin "$stdin" --prove --save-proof "$proof"; then
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
      extra="$(extract_metrics "$log")"
      log_timing "$(iso_now)" prove range - "$start" "$end" "$((end - start))" "$wall" "$extra" "$log"
    else
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
      log_timing "$(iso_now)" prove-FAILED range - "$start" "$end" "$((end - start))" "$wall" - "$log"
      echo "  FAILED -- see $log" >&2
    fi
  done
  echo "prove complete. timings -> $TIMINGS"
}

# Validate an explicit, contiguous, ascending list of compressed range proofs.
# Sets REPLY_FIRST, REPLY_LAST, REPLY_JOINED on success.
verify_proofs() {
  [ "$#" -ge 1 ] || die "need an explicit list of compressed range proofs (in chain order)"
  local prev_end="" first_start="" last_end=""
  for p in "$@"; do
    [ -s "$p" ] || die "proof not found or empty: $p"
    local base se s e
    base="$(basename "$p" .bin)"
    [[ "$base" == proof_* ]] || die "proof file must be named proof_<start>_<end>.bin: $p"
    se="${base#proof_}"; s="${se%_*}"; e="${se#*_}"
    [ -z "$first_start" ] && first_start="$s"
    if [ -n "$prev_end" ] && [ "$s" != "$prev_end" ]; then
      die "non-contiguous proofs: $prev_end != $s (proofs must tile a contiguous range in order)"
    fi
    prev_end="$e"; last_end="$e"
  done
  REPLY_FIRST="$first_start"
  REPLY_LAST="$last_end"
  REPLY_JOINED="$(IFS=,; echo "$*")"
}

mode_fetch_agg() {
  verify_proofs "$@"
  ensure_data_dir
  require_rpc
  require_elfs
  local first="$REPLY_FIRST" last="$REPLY_LAST" joined="$REPLY_JOINED"
  local out="$DATA_DIR/agg_inputs_${first}_${last}.cbor"
  local log="$LOG_DIR/fetch_agg_${first}_${last}.log"
  if [ -s "$out" ]; then
    echo "skip fetch_agg ${first}-${last} (agg inputs exist)"
    return 0
  fi
  echo "fetch_agg ${first}-${last} from $# proofs"
  local t0 t1 wall
  t0="$(now_s)"
  if run_bench "$log" agg-bench --proofs "$joined" --save-agg-inputs "$out"; then
    t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
    log_timing "$(iso_now)" fetch-agg agg - "$first" "$last" "$((last - first))" "$wall" - "$log"
    echo "saved agg inputs -> $out"
  else
    t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
    log_timing "$(iso_now)" fetch-agg-FAILED agg - "$first" "$last" "$((last - first))" "$wall" - "$log"
    die "fetch_agg failed -- see $log"
  fi
}

mode_agg() {
  local plonk=0 inputs=""
  local -a proofs=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --plonk) plonk=1; shift ;;
      --inputs) inputs="$2"; shift 2 ;;
      *) proofs+=("$1"); shift ;;
    esac
  done
  # ${arr[@]+...} guards against "unbound variable" on an empty array under
  # set -u in bash 3.2 (macOS system bash).
  verify_proofs ${proofs[@]+"${proofs[@]}"}
  ensure_data_dir
  require_elfs
  local first="$REPLY_FIRST" last="$REPLY_LAST" joined="$REPLY_JOINED"

  # Skip if the aggregation proof already exists (don't redo / overwrite).
  local out
  if [ "$plonk" -eq 1 ]; then
    out="$DATA_DIR/agg_${first}_${last}.plonk.bin"
  else
    out="$DATA_DIR/agg_${first}_${last}.bin"
  fi
  if [ -s "$out" ]; then
    echo "skip agg ${first}-${last} ($(basename "$out") exists)"
    return 0
  fi

  # Use saved agg inputs (explicit --inputs, else the conventional path) to skip
  # RPC; otherwise the bench fetches the checkpoint + headers itself.
  [ -n "$inputs" ] || inputs="$DATA_DIR/agg_inputs_${first}_${last}.cbor"
  local -a input_args=()
  if [ -s "$inputs" ]; then
    input_args=(--load-agg-inputs "$inputs")
    echo "using saved agg inputs: $inputs (no RPC)"
  else
    require_rpc
    echo "no saved agg inputs at $inputs -- fetching from RPC"
  fi

  local t0 t1 wall extra log
  t0="$(now_s)"
  if [ "$plonk" -eq 1 ]; then
    log="$LOG_DIR/agg_plonk_${first}_${last}.log"
    echo "agg (plonk) ${first}-${last} from ${#proofs[@]} proofs"
    if run_bench "$log" plonk-prove-bench --proofs "$joined" ${input_args[@]+"${input_args[@]}"} --save-proof "$out"; then
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"; extra="$(extract_metrics "$log")"
      log_timing "$(iso_now)" agg-plonk agg - "$first" "$last" "$((last - first))" "$wall" "$extra" "$log"
      echo "saved PLONK proof -> $out"
    else
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
      log_timing "$(iso_now)" agg-plonk-FAILED agg - "$first" "$last" "$((last - first))" "$wall" - "$log"
      die "PLONK aggregation failed -- see $log"
    fi
  else
    log="$LOG_DIR/agg_compressed_${first}_${last}.log"
    echo "agg (compressed) ${first}-${last} from ${#proofs[@]} proofs"
    if run_bench "$log" agg-bench --proofs "$joined" ${input_args[@]+"${input_args[@]}"} --prove --save-proof "$out"; then
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"; extra="$(extract_metrics "$log")"
      log_timing "$(iso_now)" agg-compressed agg - "$first" "$last" "$((last - first))" "$wall" "$extra" "$log"
      echo "saved compressed agg proof -> $out"
    else
      t1="$(now_s)"; wall="$(elapsed "$t0" "$t1")"
      log_timing "$(iso_now)" agg-compressed-FAILED agg - "$first" "$last" "$((last - first))" "$wall" - "$log"
      die "compressed aggregation failed -- see $log"
    fi
  fi
  echo "agg complete. timings -> $TIMINGS"
}

# Pull a few headline metrics out of a bench log into a compact key=val string.
extract_metrics() {
  local log="$1"
  local cycles gas prove verify calldata
  cycles="$(grep -oE '[0-9]+ cycles' "$log" | grep -oE '[0-9]+' | head -1)"
  gas="$(grep -oE 'gas Some\([0-9]+\)' "$log" | grep -oE '[0-9]+' | head -1)"
  prove="$(grep -oE 'prove wall-clock: [^,;]+' "$log" | head -1 | sed 's/prove wall-clock: //')"
  verify="$(grep -oE 'local verify: [^,;]+' "$log" | head -1 | sed 's/local verify: //')"
  calldata="$(grep -oE 'on-chain calldata: [0-9]+ bytes' "$log" | grep -oE '[0-9]+' | head -1)"
  local out=""
  [ -n "$cycles" ] && out+="cycles=$cycles;"
  [ -n "$gas" ] && out+="gas=$gas;"
  [ -n "$prove" ] && out+="prove=$prove;"
  [ -n "$verify" ] && out+="verify=$verify;"
  [ -n "$calldata" ] && out+="calldata_bytes=$calldata;"
  [ -n "$out" ] && echo "${out%;}" || echo "-"
}

usage() {
  # Print the leading comment block (skip the shebang, stop at the first
  # non-comment line).
  awk 'NR==1{next} /^#/{sub(/^# ?/,""); print; next} {exit}' "${BASH_SOURCE[0]}"
  exit "${1:-0}"
}

main() {
  local cmd="${1:-}"; shift || true
  case "$cmd" in
    plan)  mode_plan "$@" ;;
    fetch_range) mode_fetch_range "$@" ;;
    fetch_agg)   mode_fetch_agg "$@" ;;
    prove) mode_prove "$@" ;;
    agg)   mode_agg "$@" ;;
    ""|-h|--help|help) usage 0 ;;
    *) echo "unknown mode: $cmd" >&2; usage 1 ;;
  esac
}

main "$@"
