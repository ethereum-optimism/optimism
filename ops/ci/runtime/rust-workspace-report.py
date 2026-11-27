#!/usr/bin/env python3
"""Record the stages, tool versions and exit status of one Rust workspace job."""
import importlib.util
import json
import os
from pathlib import Path
import sys
import time
import xml.etree.ElementTree as ET


SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).with_name('ci-report.py'))
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)

digest = REPORT.digest
write = REPORT.write

# Jobs whose command writes no test report. RWX shows one case for each of them.
CHECK_JOBS = ('wasm-unknown', 'wasm-wasi', 'zepter', 'typos', 'deny', 'registry', 'interop')
# A tool that only one job uses, and the command that prints its version.
JOB_TOOLS = {'deny': ('cargo', 'deny', '--version'), 'zepter': ('zepter', '--version'), 'typos': ('typos', '--version'),
             'wasm-unknown': ('cargo', 'hack', '--version'),
             'wasm-wasi': ('cargo', 'hack', '--version'), 'interop': ('go', 'version')}


def command(*args):
    return REPORT.command(args)


def begin(directory, job):
    settings = {'suite': 'rust-workspace', 'job': job, 'source_sha': command('git', 'rev-parse', 'HEAD'),
                'rwx_run_id': os.environ.get('RWX_RUN_ID'),
                'rwx_task_attempt': os.environ.get('RWX_TASK_ATTEMPT_NUMBER'),
                'rustc': command('rustc', '--version'), 'cargo': command('cargo', '--version'),
                'nextest': command('cargo', 'nextest', '--version'),
                'started_at': time.time(), 'cpus': os.cpu_count()}
    if job in JOB_TOOLS:
        settings['extra_tool'] = command(*JOB_TOOLS[job])
    write(directory / 'settings.json', settings)


def stage(directory, name, args, stdout_json=False, cwd='rust', stdin=None, stdout_file=None):
    return REPORT.stage(directory, name, args, cwd=cwd, stdout_json=stdout_json,
                        stdin=stdin, stdout_file=stdout_file)


def check_junit(directory, job, passed):
    suite = ET.Element('testsuite', name=job, tests='1')
    case = ET.SubElement(suite, 'testcase', classname=job, name=job)
    if not passed:
        ET.SubElement(case, 'failure', message='See original stage logs and final.json')
    ET.ElementTree(suite).write(directory / 'checks.junit.xml', encoding='utf-8', xml_declaration=True)


def finish(directory, status):
    """Write the job's exit status and stage records. The caller's commands own the verdict."""
    job = json.loads((directory / 'settings.json').read_text())['job']
    stages = {p.stem.removesuffix('.stage'): json.loads(p.read_text()) for p in directory.glob('*.stage.json')}
    if job in CHECK_JOBS:
        check_junit(directory, job, status == 0)
    write(directory / 'final.json', {'exit_code': status, 'stages': stages})
    return status


if __name__ == '__main__':
    mode, name = sys.argv[1:3]
    directory = Path(name).resolve()
    if mode == 'begin':
        begin(directory, sys.argv[3])
    elif mode in ('stage', 'json-stage'):
        sys.exit(stage(directory, sys.argv[3], sys.argv[4:], mode == 'json-stage'))
    elif mode == 'stage-at':
        sys.exit(stage(directory, sys.argv[4], sys.argv[5:], cwd=sys.argv[3]))
    elif mode == 'finish':
        sys.exit(finish(directory, int(sys.argv[3])))
    else:
        raise ValueError('Unknown Rust report mode')
