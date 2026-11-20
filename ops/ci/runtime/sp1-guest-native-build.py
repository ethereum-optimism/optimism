#!/usr/bin/env python3
"""Build the SP1 guest ELFs in a clone at the paths that CircleCI uses.

The clone and its Cargo home have CircleCI's paths, so that ELF bytes that
embed those paths match CircleCI's build of the same commit.
"""
import os
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[3]
WORKSPACE = Path('/home/circleci/project')
CARGO = Path('/data/mise-data/.cargo')
TARGET = WORKSPACE/'rust/kona/sp1/programs/target'


def execute(features=""):
    """Run the build phase of sp1-guest.py in the clone and copy its hand-off directory here.

    Preconditions: WORKSPACE and CARGO do not exist, and this checkout has no
    uncommitted change to a tracked file. The clone contains only the commit,
    so the build would not contain such a change.

    Return the exit status of the build phase, or 1 if a precondition fails.
    The tool cache keeps Cargo home and the target directory inside this
    checkout. This function copies them to CARGO and TARGET before the build
    and, if the build passes, back.
    """
    cache=ROOT/'.ci/sp1-cache'
    output=ROOT/'.ci/sp1-guest/dependency';shutil.rmtree(output,ignore_errors=True)
    if (WORKSPACE.exists() or WORKSPACE.is_symlink() or CARGO.exists() or CARGO.is_symlink()
            or WORKSPACE.resolve()!=WORKSPACE or CARGO.resolve()!=CARGO):
        print('Native SP1 canonical directories are already occupied',file=sys.stderr);return 1
    if subprocess.call(['git','diff','--quiet','HEAD'],cwd=ROOT):
        print('The clone would not contain the uncommitted changes of this checkout',file=sys.stderr);return 1
    sha=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
    for argv in (['sudo','mkdir','-p',str(WORKSPACE),str(CARGO)],
                 ['sudo','chown',str(os.getuid())+':'+str(os.getgid()),str(WORKSPACE),str(CARGO)],
                 ['git','clone','--shared','--no-checkout',str(ROOT),str(WORKSPACE)],
                 ['git','-C',str(WORKSPACE),'checkout','--detach',sha],
                 ['mise','trust','--yes',str(WORKSPACE/'mise.toml')]):
        subprocess.run(argv,cwd=ROOT,check=True,stdin=subprocess.DEVNULL)
    for source,target in [(cache/'cargo',CARGO),(cache/'elf-target',TARGET)]:
        if source.exists():shutil.copytree(source,target,symlinks=True,dirs_exist_ok=True)
        else:target.mkdir(parents=True,exist_ok=True)
    status=subprocess.call([sys.executable,'ops/ci/runtime/sp1-guest.py','--phase','build','--features',features],
                           cwd=WORKSPACE,env={**os.environ,'SP1_GUEST_NATIVE_CACHE_ROOT':str(cache)})
    built=WORKSPACE/'.ci/sp1-guest/dependency'
    if built.is_dir():shutil.copytree(built,output)
    else:output.mkdir(parents=True)
    if not status:
        for source,target in [(CARGO,cache/'cargo'),(TARGET,cache/'elf-target')]:
            shutil.rmtree(target,ignore_errors=True);shutil.copytree(source,target,symlinks=True)
    return status
