#!/usr/bin/env python3
"""One-time encrypted export of the existing CI-only archive RPC inputs."""

import os
from pathlib import Path
import subprocess


root = Path(__file__).resolve().parents[2]
output = root / "tmp/rwx-rpc-encrypted"
output.mkdir(parents=True, exist_ok=True)
for name in ("OP_CI_MAINNET_L1_ARCHIVE_RPC_URL", "OP_CI_SEPOLIA_L1_ARCHIVE_RPC_URL"):
    value = os.environ.get(name)
    if not value:
        raise SystemExit(f"Missing required CI input: {name}")
    # Values only enter OpenSSL through stdin. The retained artifact is RSA
    # OAEP ciphertext; the corresponding private key is never committed.
    subprocess.run(["openssl", "pkeyutl", "-encrypt", "-pubin", "-inkey",
                    str(root / "ops/ci/rwx-rpc-recipient.pem"),
                    "-pkeyopt", "rsa_padding_mode:oaep", "-pkeyopt", "rsa_oaep_md:sha256",
                    "-pkeyopt", "rsa_mgf1_md:sha256", "-out", str(output / (name + ".enc"))],
                   input=value.encode(), check=True, stdout=subprocess.DEVNULL)
print("Encrypted both required archive RPC inputs.")
