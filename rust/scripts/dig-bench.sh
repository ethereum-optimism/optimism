#!/usr/bin/env bash
# Bench-only helper (mds/bench-dig-rust-*). Run from rust/.
set -uo pipefail
mode=${1:-probe}

probe() {
  echo "== toolchain probe"
  echo "RUSTUP_HOME=${RUSTUP_HOME:-unset} CARGO_HOME=${CARGO_HOME:-unset} RUSTUP_TOOLCHAIN=${RUSTUP_TOOLCHAIN:-unset} RUSTUP_AUTO_INSTALL=${RUSTUP_AUTO_INSTALL:-unset}"
  type -a cargo rustc rustup mold 2>&1 | head -12
  ls -1 "$RUSTUP_HOME/toolchains"
  du -sh "$RUSTUP_HOME"/toolchains/* 2>/dev/null
  echo "-- active toolchain with the new pin:"
  rustup show active-toolchain || echo "PROBE: new pin not installed"
  for t in $(ls "$RUSTUP_HOME/toolchains"); do
    echo "-- $t"
    rustup component list --installed --toolchain "$t" | tr '\n' ' '; echo
    "$RUSTUP_HOME/toolchains/$t/bin/rustc" -vV | grep -E 'release|commit-hash'
  done
  mkdir -p /tmp/oldpin
  printf '[toolchain]\nchannel = "1.99"\n' > /tmp/oldpin/rust-toolchain.toml
  echo "-- active toolchain with the old pin (1.99), no auto-install:"
  (cd /tmp/oldpin && RUSTUP_AUTO_INSTALL=0 rustup show active-toolchain) || echo "PROBE: old pin 1.99 not installed without the toolchain cache"
}

ms() { date +%s%3N; }

variant() {
  label=$1; cmdfile=$2; wrap=$3; shift 3
  printf '%s\n' "RUSTC_BOOTSTRAP=1 $(cat "$cmdfile") -Ztime-passes $*" > "/tmp/v-$label.sh"
  s=$(ms)
  if [ "$wrap" = mold ]; then
    mold -run bash "/tmp/v-$label.sh" > "/tmp/tp-$label.log" 2>&1
  else
    bash "/tmp/v-$label.sh" > "/tmp/tp-$label.log" 2>&1
  fi
  rc=$?
  e=$(ms)
  echo "VARIANT $label rc=$rc wall_ms=$((e - s))"
  if [ $rc -ne 0 ]; then tail -25 "/tmp/tp-$label.log"; return 0; fi
  grep -E 'time:' "/tmp/tp-$label.log" \
    | grep -E '[[:space:]](total|run_linker|link|link_binary|link_crate|LLVM_passes|codegen_crate|finish_ongoing_codegen|type_check_crate|MIR_borrow_checking|monomorphization_collector|analysis|expand_crate|macro_expand_crate|generate_crate_metadata|MIR_effect_checking|codegen_to_LLVM_IR)$' \
    | sed "s/^/  [$label] /"
  ef=$(grep -o 'extra-filename=[^ ]*' "$cmdfile" | head -1 | cut -d= -f2)
  cn=$(grep -o -- '--crate-name [^ ]*' "$cmdfile" | head -1 | awk '{print $2}')
  bin="target/debug/deps/${cn}${ef}"
  if [ -f "$bin" ]; then
    echo "  [$label] binary $(du -h "$bin" | cut -f1) linker: $(readelf -p .comment "$bin" 2>/dev/null | grep -o -i -E 'mold [0-9.]+|LLD [0-9.]+' | sort -u | tr '\n' ' ')"
  fi
}

tests() {
  s=$(ms)
  cargo test --no-run --workspace --all-features --locked --timings > /tmp/build.log 2>&1 || { tail -40 /tmp/build.log; exit 1; }
  echo "BUILD test --no-run: $(( ($(ms) - s) / 1000 )) s"
  sccache --show-stats | grep -E '^(Compile requests|Cache hits|Cache misses|Non-cacheable calls) '
  mkdir -p /tmp/dig-art
  cp target/cargo-timings/cargo-timing.html /tmp/dig-art/test-build-timing.html
  du -sh target
  touch op-reth/crates/node/src/lib.rs
  s=$(ms)
  cargo test --no-run --workspace --all-features --locked -v > /tmp/v.log 2>&1 || { tail -40 /tmp/v.log; exit 1; }
  echo "REBUILD after touching reth-optimism-node lib.rs: $(( ($(ms) - s) / 1000 )) s"
  cp /tmp/v.log /tmp/dig-art/rebuild-v.log
  grep -F 'Running `' /tmp/v.log | grep -F -- '--test' | grep -F 'op-reth/crates/node/src/lib.rs' | head -1 \
    | sed -e 's/^ *Running `//' -e 's/`$//' -e 's# [^ ]*sccache # #' > /tmp/cmd-lib.txt
  grep -F 'Running `' /tmp/v.log | grep -F -- '--test' | grep -F 'op-reth/crates/node/tests/it/main.rs' | head -1 \
    | sed -e 's/^ *Running `//' -e 's/`$//' -e 's# [^ ]*sccache # #' > /tmp/cmd-it.txt
  wc -c /tmp/cmd-lib.txt /tmp/cmd-it.txt
  head -c 600 /tmp/cmd-lib.txt; echo
  [ -s /tmp/cmd-lib.txt ] || { echo "no lib test command found"; grep -F 'Running `' /tmp/v.log | grep -F 'op-reth/crates/node' | head -5 | cut -c1-400; exit 0; }
  variant lib-base /tmp/cmd-lib.txt none
  variant lib-threads8 /tmp/cmd-lib.txt none -Zthreads=8
  variant lib-mold /tmp/cmd-lib.txt mold
  variant lib-dbg0 /tmp/cmd-lib.txt none -Cdebuginfo=0
  variant lib-dbg0-mold /tmp/cmd-lib.txt mold -Cdebuginfo=0
  if [ -s /tmp/cmd-it.txt ]; then
    variant it-base /tmp/cmd-it.txt none
    variant it-threads8 /tmp/cmd-it.txt none -Zthreads=8
    variant it-mold /tmp/cmd-it.txt mold
  fi
  cp /tmp/tp-*.log /tmp/dig-art/ 2>/dev/null || true
}

case "$mode" in
  probe) probe ;;
  tests) probe; tests ;;
esac
