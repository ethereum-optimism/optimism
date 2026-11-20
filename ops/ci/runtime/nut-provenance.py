#!/usr/bin/env python3
"""Verify the provenance of every locked NUT bundle.

CircleCI verifies only the forks whose lock entry changed. This script verifies
every fork in fork_lock.toml. For each fork it runs the verifier that CircleCI
runs, with the Forge and Just versions that mise.toml pins at the commit the
lock records for that fork. The verifier owns every check of the lock entry.

RWX turns mise auto-install off. `--install-tools` installs those versions in
an earlier task, so the verdict task installs nothing.
"""
import argparse
from pathlib import Path
import subprocess
import sys
import tomllib

ROOT = Path(__file__).resolve().parents[3]
LOCK = 'op-core/nuts/fork_lock.toml'
TOOLS = 'ops/ci/runtime/nut-tools.sh'
VERIFIER = ['go', 'run', './ops/scripts/nut-provenance-verify']


def generator_pins(root=ROOT):
    """Return the Forge and Just versions of each locked fork.

    The versions come from mise.toml at the commit that the lock records for
    the fork, not from the mise.toml of the checkout.
    """
    pins = {}
    for fork, entry in sorted(tomllib.loads((root / LOCK).read_text()).items()):
        recorded = subprocess.check_output(['git', 'show', entry['commit'] + ':mise.toml'], cwd=root, text=True)
        tools = tomllib.loads(recorded)['tools']
        pins[fork] = {'forge': tools['forge'], 'just': tools['just']}
    return pins


def install_tools(root=ROOT):
    """Install the extra Solidity compiler and each distinct Forge and Just pair."""
    subprocess.run(['bash', TOOLS, 'solc'], cwd=root, check=True)
    for forge, just in sorted({(pin['forge'], pin['just']) for pin in generator_pins(root).values()}):
        subprocess.run(['bash', TOOLS, 'generators', forge, just], cwd=root, check=True)


def verify(root=ROOT):
    """Run the verifier for every locked fork.

    Return 0 if every verifier passes. Otherwise stop at the first verifier
    that fails and return its exit status.
    """
    for fork, pin in generator_pins(root).items():
        argv = ['mise', 'exec', 'forge@' + pin['forge'], 'just@' + pin['just'], '--', *VERIFIER, fork]
        status = subprocess.call(argv, cwd=root, stdin=subprocess.DEVNULL)
        if status:
            return status
    return 0


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--install-tools', action='store_true')
    if parser.parse_args().install_tools:
        install_tools()
    else:
        sys.exit(verify())
