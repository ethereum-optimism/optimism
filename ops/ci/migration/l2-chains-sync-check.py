#!/usr/bin/env python3
"""Transitional verification of Circle schedules against its archive RPC inventory."""
import importlib.util
import json
from pathlib import Path

_SPEC=importlib.util.spec_from_file_location('report_evidence',Path(__file__).with_name('report-evidence.py'))
E=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(E)
M=E.helper('main-checks')
ROOT=M.ROOT
COMMAND=['bash','.circleci/scripts/check-l2-chains-sync.sh']
stage, retain_sources = M.stage, M.retain_sources


def l2_selection(configuration, rpcs):
    matrices = [body['contracts-bedrock-tests-l2-fork']['matrix']['parameters']['fork_op_chain']
                for body in configuration['workflows']['scheduled-daily-tests']['jobs']
                if isinstance(body, dict) and 'contracts-bedrock-tests-l2-fork' in body]
    if len(matrices) != 1 or not matrices[0] or len(set(matrices[0])) != len(matrices[0]):
        raise ValueError('Missing or duplicate complete scheduled L2 fork matrix')
    if not isinstance(rpcs, dict) or not rpcs: raise ValueError('Missing complete L2 RPC chain selection')
    return {'rpc_chains': sorted(rpcs), 'matrix_chains': sorted(matrices[0])}

def discovery(directory, job):
    stage(directory, 'configuration', ['yq', '-o=json', '.', '.circleci/continue/main.yml'], json_output=True)
    configuration = json.loads((directory / 'configuration.json').read_text())
    retain_sources(directory, ['.circleci/continue/main.yml', '.circleci/l2-rpcs.json'])
    return l2_selection(configuration, json.loads((ROOT / '.circleci/l2-rpcs.json').read_text()))


if __name__=='__main__':
    raise SystemExit(M.execute('l2-chains-sync-check', argv=COMMAND, discover=discovery))
