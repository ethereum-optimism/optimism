#!/usr/bin/env python3
"""Generates vectors.json: edge cases for the interop message expiry rule, shared by the Go and Rust tests.

Spec (protocol, https://specs.optimism.io/interop/messaging.html#invalid-messages), on u64 block timestamps:
    valid  iff  init <= exec  and  exec - init <= W
    W = override if override > 0 else 604800 (7 days)
The expected values here are computed with Python integers (no overflow), independently of both implementations.
"reason" is the FIRST failing rule: "future" (init > exec), "expired" (exec - init > W), or "ok".

Every vector has init >= 1 and exec >= 1 so that op-supernode's activation rule (activation 0, block time 1) never
fires first; activation is orthogonal to the window rule and covered by the existing tests.

Overrides above the 7-day cap (604800) must be REJECTED when the dependency set is built or parsed (op-core: the
constructor, JSON and TOML; kona: serde parsing), never clamped. Each such override gets one vector with
"configRejected": true and no timestamp expectation. Every override at or below the cap must be accepted and used
verbatim (0 = unset = 604800).
"""
import json
import os

U64 = 2**64 - 1
DEFAULT = 604800
CAP = 604800  # protocol cap on overrides (7 days)

OVERRIDES = [0, 1, 2, 3600, DEFAULT - 1, DEFAULT, DEFAULT + 1, 8 * 86400, U64 - 1, U64]


def window(override):
    return override if override > 0 else DEFAULT


def expect(init, exec_, w):
    if init > exec_:
        return False, "future"
    if exec_ - init > w:
        return False, "expired"
    return True, "ok"


vectors = []
seen = set()


def add(name, init, exec_, override):
    if not (1 <= init <= U64 and 1 <= exec_ <= U64):
        return
    if override > CAP:
        init, exec_, name = 1, 1, f"override={override} (above the cap: config rejected)"
    key = (init, exec_, override)
    if key in seen:
        return
    seen.add(key)
    w = window(override)
    valid, reason = expect(init, exec_, w)
    vectors.append(
        {
            "name": name,
            "init": init,
            "exec": exec_,
            "override": override,
            "window": w,
            "configRejected": override > CAP,
            "valid": valid and override <= CAP,
            "reason": reason if override <= CAP else "configRejected",
        }
    )


for ov in OVERRIDES:
    w = window(ov)
    tag = f"override={ov}"
    for base_name, base in [("init=1", 1), ("init=1e9", 10**9), ("init=2^63", 2**63), ("init=max-W-1", U64 - w - 1)]:
        for d_name, d in [("W-1", w - 1), ("W", w), ("W+1", w + 1), ("0", 0), ("1", 1)]:
            add(f"{tag} {base_name} exec-init={d_name}", base, base + d, ov)
        for d_name, d in [("1", 1), ("W", w), ("W+1", w + 1)]:
            add(f"{tag} {base_name} init-exec={d_name} (init in future)", base + d, base, ov)
    # Overflow-ish extremes.
    add(f"{tag} init=1 exec=max", 1, U64, ov)
    add(f"{tag} init=max exec=max", U64, U64, ov)
    add(f"{tag} init=max-W exec=max", U64 - w, U64, ov)
    add(f"{tag} init=max-W-1 exec=max", U64 - w - 1, U64, ov)
    # init far in the future: a lone `exec - init` (u64 wrapping) would come out tiny and pass the window.
    add(f"{tag} init=max exec=1 (wraps to 2)", U64, 1, ov)
    add(f"{tag} init=max exec=W (wraps to W+1)", U64, w, ov)
    add(f"{tag} init=max exec=W-1 (wraps to W)", U64, w - 1, ov)

out = {
    "spec": "valid iff init <= exec && exec - init <= W; W = override > 0 ? override : 604800",
    "defaultWindow": DEFAULT,
    "cap": CAP,
    "vectors": vectors,
}
path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "vectors.json")
with open(path, "w") as f:
    json.dump(out, f, indent=1)
    f.write("\n")
print(f"wrote {len(vectors)} vectors to {path}")
