#!/usr/bin/env python3
"""Export optimized private/profile and projection bytecode after `mise x -- just build-source`."""

import json
from pathlib import Path

root = Path(__file__).resolve().parents[2]
for name in (
    "L1Block", "L2ToL1MessagePasser", "L2ToL2CrossDomainMessenger", "SuperchainETHBridge",
    "L2ToL2CrossDomainMessengerReplay", "ClaimRegistry", "EventReplayer", "CrossL2Inbox",
):
    artifact_path = root / "packages/contracts-bedrock/forge-artifacts" / f"{name}.sol" / f"{name}.json"
    artifact = json.loads(artifact_path.read_text())
    metadata = artifact["metadata"]
    if isinstance(metadata, str):
        metadata = json.loads(metadata)
    settings = metadata["settings"]
    if settings["optimizer"] != {"enabled": True, "runs": 999999}:
        raise SystemExit(f"{name}: rebuild with the default optimized profile")
    code = artifact["deployedBytecode"]
    if code.get("immutableReferences") or code.get("linkReferences"):
        raise SystemExit(f"{name}: unresolved deployment references")
    raw = code["object"].removeprefix("0x")
    bytes.fromhex(raw)  # Reject unresolved library placeholders.
    if not raw:
        raise SystemExit(f"{name}: empty runtime bytecode")
    (Path(__file__).parent / "bytecode" / f"{name}.hex").write_text(raw + "\n")
