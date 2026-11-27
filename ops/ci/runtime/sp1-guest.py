#!/usr/bin/env python3
"""Run the steps of CircleCI's kona-build-sp1-elfs job in two RWX tasks.

`--phase build` checks the guest lock, builds the guest ELFs and applies
CircleCI's two ELF checks. `--phase checks` receives those ELFs and runs the
guest tests, the guest lints and the vkey crate checks. The exit status of
each recipe is the verdict.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tomllib

ROOT = Path(__file__).resolve().parents[3]
RUST = ROOT / 'rust'
SP1 = RUST / 'kona/sp1'
REPORTS = ROOT / '.ci/sp1-guest'
# The guest programs that build-elfs-native builds, as CircleCI's marker check lists them.
PROGRAMS = ('log-commitment', 'super-range', 'super-aggregation')
TEST_CONFIG_MARKER = b'KONA_SP1_UNSAFE_TEST_CONFIG_FALLBACK{fd6d88e711058eef5eff1512237c8ad3}'


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).with_name(name + '.py'))
    value = importlib.util.module_from_spec(spec); spec.loader.exec_module(value)
    return value


REPORT = helper('ci-report')


def command(*argv): return REPORT.command(argv, cwd=ROOT)


def run(*argv, cwd=RUST):
    """Run one command. Raise CalledProcessError if it fails."""
    subprocess.run(argv, cwd=cwd, check=True, stdin=subprocess.DEVNULL)


def tools():
    version = tomllib.loads((ROOT / 'mise.toml').read_text())['tools']['github:succinctlabs/sp1']['version']
    result = {name: command(*argv) for name, argv in {
        'rustc': ['rustc', '--version'], 'cargo': ['cargo', '--version'],
        'cargo-prove': ['cargo', 'prove', '--version'], 'succinct-rustc': ['rustc', '+succinct', '--version'],
        'nightly-rustfmt': ['cargo', '+' + tomllib.loads((ROOT / 'mise.toml').read_text())['tools']['rust'][1]['version'], 'fmt', '--version'],
        'just': ['just', '--version'], 'protoc': ['protoc', '--version']}.items()}
    directory=Path(command('mise','where','github:succinctlabs/sp1')).resolve()
    binary=Path(shutil.which('cargo-prove') or '/missing-cargo-prove').resolve()
    if directory.name!=version or not binary.is_file() or not binary.is_relative_to(directory):
        raise ValueError('Cargo-prove executable does not come from the pinned mise installation')
    # This release reports a Git descriptor ('sp1'), not the numeric SDK version.
    # Retain that original descriptor and verify the actual installed binary bytes.
    result['cargo-prove-sha256']=REPORT.digest(binary)
    rustc = Path(command('rustup', 'which', '--toolchain', 'succinct', 'rustc'))
    if not rustc.is_file(): raise ValueError('Missing actual Succinct compiler')
    result['succinct-rustc-sha256'] = REPORT.digest(rustc)
    library = rustc.parent.parent/'lib'
    result['succinct-library-sha256'] = {str(p.relative_to(library)):REPORT.digest(p) for p in sorted(library.rglob('*')) if p.is_file()}
    if not result['succinct-library-sha256']:raise ValueError('Missing actual Succinct linker and standard-library inputs')
    result['sp1-version'] = version
    return result


def check_elfs(directory, sha, features):
    """Apply CircleCI's two checks to the built ELFs. Raise ValueError if one fails.

    1. vkeys.toml has the names that its consumers read.
    2. Each ELF embeds exactly one build marker, and that marker names `sha`,
       with the suffix -test if the build selected features. The super-range
       ELF has the test-config-fallback marker if, and only if, the build
       selected features.
    """
    # CircleCI's "Check vkeys manifest names expected by consumers" step.
    if not {'super-aggregation', 'log-commitment'} <= set(tomllib.loads((directory / 'vkeys.toml').read_text())):
        raise ValueError('vkeys manifest lacks a name that its consumers read')
    expected = 'KONA_SP1_BUILD{git_sha=' + sha + ('-test' if features else '') + '}'
    for name in PROGRAMS:
        data = (directory / (name + '-elf')).read_bytes()
        markers = set(m.decode() for m in re.findall(rb'KONA_SP1_BUILD\{git_sha=[0-9A-Za-z._-]*\}', data))
        if markers != {expected}:
            raise ValueError('Guest ELF has missing, extra or stale source markers')
        if name == 'super-range' and (TEST_CONFIG_MARKER in data) != bool(features):
            raise ValueError('Guest ELF test-config-fallback marker differs from selected features')


def cache_start(phase, features):
    """Point Cargo and sccache at the tool-cache directories of the phase and start sccache.

    If the compiler inputs differ from the ones that built the target
    directory of the phase, delete that directory first.
    """
    cache = ROOT / '.ci/sp1-cache'
    target = cache / (phase + '-target')
    cargo_home = cache/'cargo'
    if phase=='elf':
        # The clone at CircleCI's paths keeps sccache data in the checkout that started it.
        native=helper('sp1-guest-native-build')
        cache=Path(os.environ['SP1_GUEST_NATIVE_CACHE_ROOT'])
        target=native.TARGET;cargo_home=native.CARGO
    mode = os.environ.get('SP1_GUEST_TARGET_MODE', 'keep')
    if mode not in ('keep', 'sccache-only'): raise ValueError('Invalid SP1 compiler-target probe mode')
    identity={'phase':phase,'features':features,'tools':tools(),'rustflags':'' if phase=='elf' else '-Dwarnings','incremental':'0'}
    identity_sha256=hashlib.sha256(json.dumps(identity,sort_keys=True).encode()).hexdigest()
    identity_path=target/'.sp1-compiler-inputs.json'
    if not identity_path.is_file() or REPORT.read(identity_path)!=identity:
        shutil.rmtree(target,ignore_errors=True)
    if mode == 'sccache-only': shutil.rmtree(target, ignore_errors=True)
    target.mkdir(parents=True,exist_ok=True);REPORT.write(identity_path,identity)
    os.environ.update(CARGO_HOME=str(cargo_home), CARGO_TARGET_DIR=str(target),
        RUSTC_WRAPPER='sccache', SCCACHE_DIR=str(cache/(phase+'-sccache')/identity_sha256),
        SCCACHE_BASEDIRS=str(ROOT), SCCACHE_CACHE_SIZE='10G', SCCACHE_IDLE_TIMEOUT='0')
    for name in ('CARGO_HOME', 'CARGO_TARGET_DIR', 'SCCACHE_DIR'):
        Path(os.environ[name]).mkdir(parents=True, exist_ok=True)
    run('sccache', '--start-server', cwd=ROOT)
    run('sccache', '--zero-stats', cwd=ROOT)
    run('python3', 'ops/ci/runtime/rust-target-cache.py', 'prepare', cwd=ROOT)


def cache_finish(status):
    """Print the sccache statistics and stop sccache. Commit the target cache if nothing failed.

    Return `status` if it is not 0. Otherwise return the exit status of the
    first cache command that fails, or 0.
    """
    for argv in (['sccache', '--show-stats'], ['sccache', '--stop-server']):
        code = subprocess.call(argv, cwd=ROOT)
        status = status or code
    if not status:
        status = subprocess.call(['python3', 'ops/ci/runtime/rust-target-cache.py', 'commit'], cwd=ROOT)
    return status


def cached(phase, features, steps):
    """Run `steps` between cache_start and cache_finish. Return the exit status of the phase."""
    cache_start(phase, features)
    try:
        steps()
        status = 0
    except subprocess.CalledProcessError as error:
        # A negative status is a signal number; report it as a shell does.
        status = error.returncode if error.returncode > 0 else 128 - error.returncode
    except (OSError, ValueError) as error:
        print(error, file=sys.stderr)
        status = 1
    return cache_finish(status)


def build(features):
    """Build the guest ELFs and write them to the hand-off directory. Return the exit status.

    sp1-guest-native-build.py owns the build location. If this checkout is not
    at that location, delegate to it; it runs this phase again inside its clone.
    """
    native = helper('sp1-guest-native-build')
    if ROOT != native.WORKSPACE:
        return native.execute(features)
    files = REPORTS / 'dependency/files'
    shutil.rmtree(files.parent, ignore_errors=True)
    files.mkdir(parents=True)

    def steps():
        # CircleCI checks the committed guest lock first: build-elfs-native regenerates it in place.
        run('just', 'check-sp1-guest-lock')
        run('just', 'build-elfs-native', features, cwd=SP1)
        check_elfs(SP1 / 'elf', command('git', 'rev-parse', 'HEAD'), features)
        for name in (*(program + '-elf' for program in PROGRAMS), 'vkeys.toml'):
            shutil.copyfile(SP1 / 'elf' / name, files / name)
    return cached('elf', features, steps)


def checks(artifact, features):
    """Run CircleCI's test, lint and vkey crate steps on the ELFs in `artifact`. Return the exit status."""
    def steps():
        # The guest lints and the vkey crates read the ELFs and vkeys.toml that the build task made.
        (SP1 / 'elf').mkdir(exist_ok=True)
        for path in (artifact / 'files').iterdir():
            shutil.copyfile(path, SP1 / 'elf' / path.name)
        os.environ['RUSTFLAGS'] = '-Dwarnings'
        for recipe in ('test-sp1-guest', 'lint-sp1-guest', 'check-vkey-crates'):
            run('just', recipe)
    return cached('checks', features, steps)


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--phase',choices=('build','checks'),required=True)
    parser.add_argument('--features',choices=('', 'test-config-fallback'),default='')
    parser.add_argument('--artifact',type=Path,default=REPORTS/'dependency')
    args=parser.parse_args()
    sys.exit(build(args.features) if args.phase=='build' else checks(args.artifact,args.features))
