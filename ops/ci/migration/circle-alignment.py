#!/usr/bin/env python3
"""Check the current Circle gate prerequisites against permanent native policy."""
import argparse
import importlib.util
from pathlib import Path


def helper(name):
    spec=importlib.util.spec_from_file_location(name.replace('-','_'),Path(__file__).resolve().parents[1]/'runtime'/(name+'.py'))
    module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module)
    return module


G=helper('pr-gate')
E2E=helper('rust-e2e')
MAPPING=Path(__file__).with_name('circle-gates.json')


def gate_requires(row,name):
    jobs=G.yaml(row['circle_config'])['workflows'][row['workflow']]['jobs']
    matches=[job[name] for job in jobs if isinstance(job,dict) and name in job]
    if len(matches)!=1 or matches[0].get('always-succeed',False):
        raise ValueError('Missing, duplicate, renamed or changed Circle gate dependency')
    return matches[0]['requires']


def configuration(gate=None):
    manifest,selections=G.configuration(gate)
    mapping=G.read(MAPPING)
    if mapping['version']!=1 or mapping['repository']!=manifest['repository'] or set(mapping['gates'])!=set(manifest['gates']):
        raise ValueError('Wrong Circle gate mapping')
    for name,selected in selections.items():
        if gate_requires(mapping['gates'][name],name)!=[{n:'terminal'} for n in selected['requires']]:
            raise ValueError('Missing, duplicate, renamed or changed Circle gate dependency')
    return manifest,selections


def e2e_configuration():
    """Compare the RWX Rust E2E jobs with Circle's required-rust-e2e prerequisites.

    The RWX E2E gate requires one full-workspace release build and one verdict
    per E2E.JOBS entry. Circle splits that build across `release_jobs`. Circle's
    gate must require exactly those jobs and the E2E jobs, each as terminal.
    """
    row=G.read(MAPPING)['e2e_gate']
    native=[*row['release_jobs'],*(job['circle_job'] for job in E2E.JOBS.values())]
    circle=gate_requires(row,row['gate'])
    if len(native)!=len(set(native)) or sorted(circle,key=str)!=sorted(({n:'terminal'} for n in native),key=str):
        raise ValueError('Missing, duplicate, renamed or changed Circle E2E gate dependency')
    return native


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--gate');args=parser.parse_args()
    if args.gate in (None,'required-rust-e2e'):e2e_configuration()
    if args.gate!='required-rust-e2e':configuration(args.gate)
