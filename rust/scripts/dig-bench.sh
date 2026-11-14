#!/usr/bin/env bash
# Bench-only helper (mds/bench-dig-rust-2). Run from rust/.
set -uo pipefail
ms() { date +%s%3N; }
mkdir -p /tmp/dig-art
echo "RUSTUP_AUTO_INSTALL=${RUSTUP_AUTO_INSTALL:-unset}"
rustup show active-toolchain
ls -1 "$RUSTUP_HOME/toolchains"

s=$(ms)
cargo test --no-run --workspace --all-features --locked --timings > /tmp/build.log 2>&1 || { tail -40 /tmp/build.log; exit 1; }
echo "BUILD test --no-run: $(( ($(ms) - s) / 1000 )) s"
cp target/cargo-timings/cargo-timing.html /tmp/dig-art/test-build-timing.html
sccache --show-stats | grep -E '^(Compile requests|Cache hits|Cache misses|Non-cacheable calls) '
echo "librocksdb archives (dev, line-tables):"; find target -name 'librocksdb.a' -exec du -h {} \;
du -sh target

touch op-reth/crates/node/src/lib.rs
cargo test --no-run --workspace --all-features --locked -v > /tmp/v.log 2>&1 || { tail -40 /tmp/v.log; exit 1; }
grep -F 'Running `' /tmp/v.log | grep -F -- '--test' | grep -F 'op-reth/crates/node/src/lib.rs' | head -1 \
  | sed -e 's/^ *Running `//' -e 's/`$//' -e 's# [^ ]*sccache # #' > /tmp/cmd-lib.txt
[ -s /tmp/cmd-lib.txt ] || { echo "no lib test command found"; exit 0; }
ef=$(grep -o 'extra-filename=[^ ]*' /tmp/cmd-lib.txt | head -1 | cut -d= -f2)
bin="target/debug/deps/reth_optimism_node${ef}"
for v in base opt0; do
  extra=""; [ "$v" = opt0 ] && extra="-C opt-level=0"
  s=$(ms); bash -c "$(cat /tmp/cmd-lib.txt) $extra" > "/tmp/dig-art/c-$v.log" 2>&1; rc=$?
  echo "COMPILE lib-$v rc=$rc wall_ms=$(( $(ms) - s )) size=$(du -h "$bin" 2>/dev/null | cut -f1)"
  [ $rc -eq 0 ] || { tail -20 "/tmp/dig-art/c-$v.log"; continue; }
  s=$(ms); timeout 900 "$bin" -q > "/tmp/dig-art/r-$v.log" 2>&1; rc=$?
  echo "RUN lib-$v rc=$rc wall_ms=$(( $(ms) - s )) $(grep -a 'test result' /tmp/dig-art/r-$v.log | tail -1)"
done

for v in dbg nodbg; do
  cfg=""; [ "$v" = nodbg ] && cfg="--config profile.dev.package.librocksdb-sys.debug=false"
  s=$(ms)
  RUSTC_WRAPPER= CARGO_TARGET_DIR=/tmp/rdb-$v cargo build -p librocksdb-sys --locked $cfg --timings > /tmp/dig-art/rdb-$v.log 2>&1; rc=$?
  echo "ROCKSDB $v rc=$rc wall_ms=$(( $(ms) - s ))"
  grep -E 'Finished|error' /tmp/dig-art/rdb-$v.log | tail -2
  find /tmp/rdb-$v -name 'librocksdb.a' -exec du -h {} \;
  du -sh /tmp/rdb-$v
  cp /tmp/rdb-$v/cargo-timings/cargo-timing.html /tmp/dig-art/rdb-$v.html 2>/dev/null
  rm -rf /tmp/rdb-$v
done
s=$(ms)
RUSTC_WRAPPER= CARGO_TARGET_DIR=/tmp/jem cargo build -p tikv-jemalloc-sys --locked > /tmp/dig-art/jem.log 2>&1; echo "JEMALLOC rc=$? wall_ms=$(( $(ms) - s ))"
