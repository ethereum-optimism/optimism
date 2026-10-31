#!/usr/bin/env python3
"""Check the current Circle gate prerequisites against permanent native policy."""
import argparse
import importlib.util
from pathlib import Path

_SPEC = importlib.util.spec_from_file_location('report_evidence', Path(__file__).with_name('report-evidence.py'))
E=importlib.util.module_from_spec(_SPEC);_SPEC.loader.exec_module(E)
G=E.helper('pr-gate')
MAPPING=Path(__file__).with_name('circle-gates.json')


def configuration(gate=None):
    manifest,selections=G.configuration(gate)
    mapping=G.read(MAPPING)
    if mapping['version']!=1 or mapping['repository']!=manifest['repository'] or set(mapping['gates'])!=set(manifest['gates']):
        raise ValueError('Wrong Circle gate mapping')
    for name,selected in selections.items():
        row=mapping['gates'][name]
        jobs=G.yaml(row['circle_config'])['workflows'][row['workflow']]['jobs']
        matches=[job[name] for job in jobs if isinstance(job,dict) and name in job]
        if len(matches)!=1 or matches[0].get('always-succeed',False) or matches[0]['requires']!=[{n:'terminal'} for n in selected['requires']]:
            raise ValueError('Missing, duplicate, renamed or changed Circle gate dependency')
    return manifest,selections


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--gate');args=parser.parse_args()
    configuration(args.gate)
