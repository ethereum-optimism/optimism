#!/usr/bin/env python3
"""Compare complete original historical NUT regeneration evidence."""
import argparse
import importlib.util
from pathlib import Path

_SPEC = importlib.util.spec_from_file_location('report_evidence', Path(__file__).with_name('report-evidence.py'))
E = importlib.util.module_from_spec(_SPEC); _SPEC.loader.exec_module(E)
N = E.helper('nut-provenance')
G, S = N.G, N.S
originals, full_selection, validate_fork = N.originals, N.full_selection, N.validate_fork


def compare(circle,native,output):
    result={'passed':False,'errors':[],'full_original_comparison':True,'coverage':{}}
    try:
        E.verify_files(circle, G.read(circle/'final.json')['original_sha256'], missing_empty=[], label='circle/nut-provenance'); originals(circle);originals(native)
        a,b=G.read(circle/'settings.json'),G.read(native/'settings.json')
        for key in ('source_sha','branch','input_sha256','environment','go_version','prepare_only'):
            if a[key]!=b[key]:raise ValueError('NUT provider settings differ: '+key)
        if a['provider']!='circleci' or b['provider']!='rwx' or a['prepare_only']:raise ValueError('Wrong original NUT provider or preparation-only evidence')
        selection,base_a=full_selection(circle);other,base_b=full_selection(native)
        result['base_discovery']={'circleci':base_a,'rwx':base_b}
        if selection!=other:
            raise ValueError('Missing or different full NUT workload selection')
        if (circle/'fork_lock.toml').read_bytes()!=(native/'fork_lock.toml').read_bytes():
            raise ValueError('Original complete NUT lock inputs differ')
        for fork,entry in selection['entries'].items():
            left,right=circle/'forks'/fork,native/'forks'/fork
            ca,cb=validate_fork(left,entry,fork),validate_fork(right,entry,fork)
            if ca!=cb:raise ValueError('Complete original NUT build selection differs: '+fork)
            for name in ('mise.toml','contracts.justfile','foundry.toml','locked-bundle.json','regenerated-bundle.json','tracked-stage.bin','worktree-status.txt','submodule-inventories.json'):
                if (left/'originals'/name).read_bytes()!=(right/'originals'/name).read_bytes():raise ValueError('Original NUT input or artifact differs: '+fork+'/'+name)
            tools_a,tools_b=G.read(left/'tools.json'),G.read(right/'tools.json')
            if tools_a['go']['version']!=a['go_version'] or tools_b['go']['version']!=b['go_version']:
                raise ValueError('Original NUT verifier Go differs from settings')
            if {k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in tools_a.items()}!={k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in tools_b.items()}:
                raise ValueError('Original NUT generator binary or toolchain differs')
            compilers_a,compilers_b=G.read(left/'compilers.json'),G.read(right/'compilers.json')
            if {k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in compilers_a.items()}!={k:{n:v for n,v in row.items() if n!='binary_path'} for k,row in compilers_b.items()}:
                raise ValueError('Original NUT Solidity compiler binaries differ')
            result['coverage'][fork]={'recorded_commit':entry['commit'],'bundle_sha256':entry['hash'],'historical_tools':entry['historical_tools'],
                    'historical_source_files':len(ca['git_inventory']),'submodules':len(ca['submodules']),'compiler_partitions':ca['compiler_partitions'],'outcome':'passed','tests':0,'retries':0}
        result['passed']=True
    except Exception as error:result['errors'].append(str(error))
    S.write(output,result);return 0 if result['passed'] else 1

if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--compare', nargs=2, type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args=parser.parse_args()
    raise SystemExit(compare(*args.compare, args.output))
