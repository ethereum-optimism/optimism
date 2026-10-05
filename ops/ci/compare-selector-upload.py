#!/usr/bin/env python3
"""Compare complete original private selector publisher executions on one SHA."""
import argparse
from collections import Counter
import datetime
import hashlib
import importlib.util
import json
from pathlib import Path
import re
import subprocess


def helper(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


U = helper('selector-upload')
G = helper('compare-pr-gates')
R = U.R
read, check = U.read, R.check


def normalized(value, workspace):
    """Only an actual workspace path prefix is provider-specific."""
    if isinstance(value, dict):
        return {key: normalized(item, workspace) for key, item in value.items()}
    if isinstance(value, list):
        return [normalized(item, workspace) for item in value]
    if isinstance(value, str) and (value == workspace or value.startswith(workspace + '/')):
        return '$WORKSPACE' + value[len(workspace):]
    return value


def fingerprint(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(',', ':')).encode()).hexdigest()


def authority(settings):
    sha = settings['source_sha']
    check(re.fullmatch('[0-9a-f]{40}', sha), 'Invalid original source revision')
    expected = G.source_inputs(sha)
    inputs = settings['inputs']
    # The runtime reader checked every recursive submodule blob against its Git
    # object. Independently bind the complete parent tree here, including modes.
    tree = subprocess.check_output(['git', 'ls-tree', '-rz', sha], cwd=G.ROOT)
    for entry in tree.split(b'\0'):
        if not entry:
            continue
        header, name = entry.split(b'\t', 1)
        mode, kind, oid = header.decode().split()
        name = name.decode()
        row = inputs[name]
        check(row['mode'] == mode, 'Changed original source mode: ' + name)
        if kind == 'commit':
            check(row == {'mode': mode, 'gitlink': oid, 'checkout': 'initialized'}
                  and any(key.startswith(name + '/') for key in inputs), 'Missing initialized source submodule')
        else:
            wanted = (hashlib.sha256(expected[name]['symlink'].encode()).hexdigest()
                      if mode == '120000' else expected[name])
            check(row == {'mode': mode, 'sha256': wanted}, 'Changed committed selector source: ' + name)
    modules = [name for name, row in inputs.items() if row.get('mode') == '160000']
    check(set(inputs) - set(expected) == {name for name in inputs if any(name.startswith(m + '/') for m in modules)},
          'Foreign source inputs outside initialized submodules')
    check(settings['implementation'] == {name: hashlib.sha256(G.source('ops/ci/' + name, sha)).hexdigest()
          for name in U.IMPLEMENTATION}, 'Uncommitted selector implementation')
    check(settings['version'] == 1 and settings['profile'] == 'default'
          and settings['command'] == U.COMMAND and settings['branch'] == 'codex/rwx-ci-pilot'
          and set(settings['compilers']) == set(U.COMPILERS),
          'Wrong original selector selection or profile')


def compiler(directory, workspace):
    cache = read(directory / 'compiler-cache.json')
    units = {p.stem: read(p) for p in sorted((directory / 'compiler-units').glob('*.json'))}
    artifacts = {str(p.relative_to(directory / 'compiler-artifacts')): read(p)
                 for p in sorted((directory / 'compiler-artifacts').rglob('*.json'))}
    # artifacts/build-info is outside forge-artifacts in this repository.
    selection = U.catalog(cache, units, artifacts)
    check(selection == read(directory / 'selection.json'), 'Changed original complete ABI selection')
    fingerprints = {}
    for key, unit in units.items():
        value = normalized({name: data for name, data in unit.items() if name != 'id'}, workspace)
        fingerprints[key] = fingerprint(value)
    check(len(set(fingerprints.values())) == len(fingerprints), 'Duplicate original compiler unit')
    logical = normalized(selection, workspace)
    for row in logical['declarations'].values():
        row['build_id'] = fingerprints[row['build_id']]
        # Foundry picks an ID from the first completed profile; U.catalog binds
        # it to the same source/version and validates the exact unit separately.
        row.pop('artifact_source_id')
    logical_cache = normalized(cache, workspace)
    logical_cache['builds'] = sorted(fingerprints.values())
    logical_cache['mocks'] = sorted(logical_cache['mocks'])
    for row in logical_cache['files'].values():
        check(type(row['lastModificationDate']) is int and row['lastModificationDate'] >= 0,
              'Invalid original compiler source mtime')
        row.pop('lastModificationDate')
        for versions in row['artifacts'].values():
            for profiles in versions.values():
                for artifact in profiles.values():
                    artifact['build_id'] = fingerprints[artifact['build_id']]
    return {'selection': logical, 'cache': logical_cache, 'units': sorted(fingerprints.values())}


def protocol(directory, wanted):
    actual = U.imports(directory, wanted)
    frames = []
    for i in range(actual['requests']):
        request = read(directory / ('http-' + str(i) + '-request.json'))
        response = read(directory / ('http-' + str(i) + '-response.json'))
        check(sum(len(values) for values in request.values()) > 0, 'Empty publisher POST')
        frames.append({'request': {kind: sorted(values) for kind, values in request.items()}, 'response': response})
    return actual, frames


def process(directory, argv, cwd):
    row = read(directory / 'execution.json')
    check(row['argv'] == argv and row['cwd'] == cwd and type(row['exit_code']) is int and row['exit_code'] == 0
          and row['timed_out'] is False and row['signals'] == []
          and isinstance(row['elapsed_seconds'], (int, float)) and row['elapsed_seconds'] > 0
          and isinstance(row['started_at'], (int, float)) and row['started_at'] > 0
          and row['stdout_sha256'] == R.digest(directory / 'stdout.log')
          and row['stderr_sha256'] == R.digest(directory / 'stderr.log'), 'Missing, failed or corrupt real execution')
    isolation = read(directory / 'isolation.json')
    check(type(isolation['uid']) is int and isolation['uid'] > 0
          and type(isolation['gid']) is int and isolation['gid'] > 0 and isolation['pid'] == 1
          and isolation['interfaces'] == [[1, 'lo']] and isolation['no_new_privileges'] is True
          and isolation['destination'] == 'https://' + R.HOST + R.IMPORT
          and {v['address'] for v in isolation['external_blocked']} == {'1.1.1.1', '2606:4700:4700::1111'}
          and all(type(v['errno']) is int and v['errno'] == 101 for v in isolation['external_blocked']),
          'Publisher destination isolation changed')
    return row


def readback(directory, wanted):
    hashes = wanted['hash32']
    expected = {'function': {}, 'event': {}}
    for signature, digest in hashes.items():
        check(re.fullmatch('0x[0-9a-f]{64}', digest), 'Invalid full selector digest')
        expected['function'].setdefault(digest[:10], []).append(signature)
        expected['event'].setdefault(digest, []).append(signature)
    observed = read(directory / 'lookup.json')
    check(set(observed) == set(expected), 'Incomplete original readback kinds')
    for kind, values in expected.items():
        check(set(observed[kind]) == set(values), 'Missing or extra readback digest')
        for digest, names in values.items():
            rows = observed[kind][digest]
            check(all(set(row) == {'name', 'filtered', 'hasVerifiedContract'} and type(row['filtered']) is bool
                      and row['hasVerifiedContract'] is False for row in rows)
                  and sorted(row['name'] for row in rows) == sorted(names), 'Missing, foreign or corrupt readback signature')
    # Re-derive the aggregate from every original captured GET and response.
    import urllib.parse
    combined = {'function': {}, 'event': {}}
    metadata = [p for p in (directory / 'registry/lookup').glob('http-*.json') if re.fullmatch(r'http-\d+', p.stem)]
    metadata.sort(key=lambda p: int(p.stem.split('-')[1]))
    for i, path in enumerate(metadata):
        row = read(path)
        check(path.stem == 'http-' + str(i) and row['method'] == 'GET' and row['host'] == R.HOST
              and row['status'] == 200, 'Missing or failed original lookup request')
        request, response = path.with_name(path.stem + '-request.json'), path.with_name(path.stem + '-response.json')
        check(request.read_bytes() == b'' and R.digest(request) == row['request_sha256']
              and R.digest(response) == row['response_sha256'], 'Corrupt original lookup payload')
        url = urllib.parse.urlsplit(row['path'])
        query = urllib.parse.parse_qs(url.query)
        kinds = set(query) - {'filter'}
        check(url.path == R.LOOKUP and query.get('filter') == ['false'] and len(kinds) == 1
              and kinds <= set(combined), 'Changed real lookup query')
        kind = kinds.pop()
        check(len(query[kind]) == 1, 'Repeated lookup query')
        keys = query[kind][0].split(',')
        value = read(response)
        other = 'function' if kind == 'event' else 'event'
        check(0 < len(keys) <= 50 and len(set(keys)) == len(keys) and not (set(keys) & set(combined[kind]))
              and value['ok'] is True and set(value['result']) == set(combined)
              and set(value['result'][kind]) == set(keys) and value['result'][other] == {},
              'Incomplete or repeated original lookup response')
        combined[kind].update(value['result'][kind])
    check(combined == observed, 'Derived readback differs from complete original HTTP responses')
    return {kind: {key: sorted(value, key=lambda row: row['name']) for key, value in rows.items()}
            for kind, rows in observed.items()}


def report(directory):
    U.original(directory, 'passed')
    prepared = directory / 'prepared'
    U.original(prepared, 'compiled')
    settings = read(directory / 'settings.json')
    authority(settings)
    check(all(read(prepared / 'settings.json')[key] == settings[key] for key in
              ('version', 'source_sha', 'branch', 'profile', 'workspace_root', 'tools', 'compilers',
               'command', 'configuration', 'inputs', 'implementation')), 'Stale preparation identity')
    workspace = settings['workspace_root']
    check(isinstance(workspace, str) and Path(workspace).is_absolute(), 'Missing original workspace')
    stable, initial = compiler(prepared, workspace), compiler(prepared / 'initial-discovery', workspace)
    check(read(prepared / 'foundry-config.json') == settings['configuration'], 'Changed original Foundry settings')
    wanted = read(prepared / 'expected.json')
    selected = stable['selection']
    groups = {'function': set(), 'event': set()}
    for key in selected['selected']:
        for row in selected['declarations'][key]['signatures']:
            groups['event' if row['kind'] == 'event' else 'function'].add(row['signature'])
    check(set(wanted) == {'signatures', 'hash32'} and set(wanted['signatures']) == set(groups)
          and set(wanted['hash32']) == set.union(*groups.values()), 'Changed complete selector expectation')
    for kind, names in groups.items():
        check(wanted['signatures'][kind] == {name: wanted['hash32'][name] if kind == 'event'
              else wanted['hash32'][name][:10] for name in names}, 'Corrupt selected digest binding')
    compilers = read(prepared / 'selected-compilers.json')
    initial_versions = {read(p)['solcVersion'] for p in (prepared / 'initial-discovery/compiler-units').glob('*.json')}
    stable_versions = {read(p)['solcVersion'] for p in (prepared / 'compiler-units').glob('*.json')}
    check(compilers == read(prepared / 'initial-discovery/selected-compilers.json')
          and set(compilers) == initial_versions and stable_versions <= initial_versions
          and all(compilers[key] == settings['compilers'][key] for key in set(compilers) & set(settings['compilers'])),
          'Unstable original compiler resolution')
    recipe = read(directory / 'recipe.stage.json')
    check(recipe['argv'] == ['just', '--dry-run', 'update-selectors'] and recipe['exit_code'] == 0
          and (directory / 'recipe.log').read_bytes() == b''
          and (directory / 'recipe.stderr.log').read_text().strip() == 'forge selectors up --all', 'Changed original publisher recipe')
    upload = directory / 'registry/upload'
    execution = process(upload, U.COMMAND, workspace + '/packages/contracts-bedrock')
    names = Counter(selected['declarations'][key]['contract'] for key in selected['selected'])
    actual_names = Counter(re.findall(r'^Uploading selectors for ([A-Za-z_$][A-Za-z0-9_$]*)\.\.\.$',
                          (upload / 'stderr.log').read_text(), re.MULTILINE))
    check(names == actual_names and 'Selectors successfully uploaded to OpenChain' in (upload / 'stdout.log').read_text(),
          'Actual publisher omitted declarations or success output')
    imports, frames = protocol(upload, wanted)
    check(imports == read(directory / 'imports.json') and all(not row['duplicated'] for row in imports['outcomes'].values()),
          'Changed original upload summary or reused destination')
    check(read(directory / 'registry/initial-rows.json') == [], 'Destination contains restored results')
    rows = read(upload / 'database-rows.json')
    check(len(rows) == len(wanted['hash32']) and {row['signature']: row['hash32'] for row in rows} == wanted['hash32'],
          'Missing, duplicate or foreign database signatures')
    for row in rows:
        check(set(row) == {'signature', 'hash4', 'hash32', 'created_at'} and row['hash4'] == row['hash32'][:10],
              'Corrupt database signature')
        when = datetime.datetime.fromisoformat(row['created_at'].replace('Z', '+00:00')).timestamp()
        check(execution['started_at'] - 1 <= when <= execution['started_at'] + execution['elapsed_seconds'] + 1,
              'Database row was not created by this fresh upload')
    lookup_execution = read(directory / 'registry/lookup/execution.json')
    argv = lookup_execution['argv']
    check(len(argv) == 5 and argv[2] == 'lookup-client'
          and argv[1] == workspace + '/ops/ci/selector-upload.py'
          and argv[4] == workspace + '/.ci/selector-upload/run/lookup.json', 'Changed real readback command')
    process(directory / 'registry/lookup', argv, workspace + '/packages/contracts-bedrock')
    source = read(directory / 'registry/registry-source.json')
    check(source['repository'] == 'https://github.com/argotorg/sourcify' and source['source_sha'] == R.SOURCIFY_SHA
          and source['helper_sha256'] == settings['implementation']['selector-registry.py'], 'Wrong actual registry source')
    for key, path in [('schema_sha256', R.SCHEMA), ('dockerfile_sha256', 'services/4byte/Dockerfile'),
                      ('lockfile_sha256', 'package-lock.json')]:
        check(source[key] == source['inputs'][path]['sha256'], 'Changed registry build input')
    service = read(directory / 'registry/service-content.json')
    for name, digest in service.items():
        U.safe_relative(name)
        check(re.fullmatch('[0-9a-f]{64}', digest), 'Corrupt running registry byte digest')
        committed = source['inputs'].get('services/4byte/' + name)
        if committed:
            check(committed['sha256'] == digest, 'Running service differs from committed source')
    check('dist/cli.js' in service and 'src/cli.ts' in service, 'Missing running service byte inventory')
    images = read(directory / 'registry/image-ids.json')
    check(set(images) == {R.REGISTRY_IMAGE, R.POSTGRES_IMAGE}
          and all(re.fullmatch('sha256:[0-9a-f]{64}', value) for value in images.values()), 'Wrong actual image identities')
    containers = read(directory / 'registry/containers.json')
    check(len(containers) == 2 and containers[0]['HostConfig']['NetworkMode'] == 'none'
          and containers[1]['HostConfig']['NetworkMode'] == 'container:' + containers[0]['Id']
          and all(not row['HostConfig']['PortBindings'] and row['State']['Running'] for row in containers)
          and containers[0]['Image'] == images[R.POSTGRES_IMAGE] and containers[1]['Image'] == images[R.REGISTRY_IMAGE]
          and containers[1]['Config']['User'] == '1000:1000' and containers[1]['HostConfig']['ReadonlyRootfs'] is True,
          'Actual registry containers differ from isolated producers')
    return {'settings': settings, 'initial': initial, 'stable': stable, 'expected': wanted, 'compilers': compilers,
            'imports': imports, 'frames': frames, 'readback': readback(directory, wanted),
            'database': [{key: value for key, value in row.items() if key != 'created_at'} for row in rows],
            'registry_source': source, 'service': service}


def circle_replay_job(source, compiled, job_name):
    """Bind Circle's generated job alias to the original isolated publisher."""
    original = source['workflows']['selector-upload-replay']['jobs']
    check(len(original) == 1 and set(original[0]) == {'contracts-bedrock-upload'}
          and original[0]['contracts-bedrock-upload'].get('selector_shadow') is True,
          'Wrong original Circle replay selection')
    selected = compiled['workflows']['selector-upload-replay']['jobs']
    check(re.fullmatch(r'contracts-bedrock-upload(?:-[1-9][0-9]*)?', job_name)
          and len(selected) == 1 and set(selected[0]) == {job_name}
          and job_name in compiled['jobs'], 'Wrong actual Circle replay selection')
    return compiled['jobs'][job_name]


def hosted(circle, native, run_path, github_path):
    s = read(native / 'settings.json')
    run = read(run_path)
    check(run['ID'] == run['RunID'] == s['native_run_id'] and run['CommitSha'] == s['source_sha']
          and run['Branch'] == s['branch'] and run['Trigger'] == 'github.push'
          and run['DefinitionPath'] == '.rwx/selector-upload.yml' and run['TargetedTaskKeys'] is None,
          'Foreign, prototype or manually targeted native execution')
    tasks = G.named(run['Tasks'], 'Key')
    G.fresh(tasks['upload']); G.fresh(tasks['helper-tests'])
    check(s['native_task_attempt'] == '1', 'Uninvestigated native upload retry')
    sha = s['source_sha']
    c = read(circle / 'settings.json')
    check(c == {'provider': 'circleci', 'source_sha': sha, 'branch': s['branch'],
                'pipeline_id': c['pipeline_id'], 'workflow_id': c['workflow_id'], 'job_number': c['job_number']},
          'Changed Circle evidence identity')
    envelope = read(circle / 'final.json')
    check(envelope['sha256'] == {str(p.relative_to(circle)): R.digest(p) for p in sorted(circle.rglob('*'))
          if p.is_file() and p != circle / 'final.json'}, 'Missing or corrupt Circle collection')
    pipeline, workflow, job = (read(circle / name) for name in ('pipeline.json', 'workflow.json', 'job.json'))
    check(pipeline['id'] == c['pipeline_id'] and pipeline['vcs']['revision'] == sha
          and pipeline['vcs']['branch'] == s['branch'] and not pipeline['errors']
          and workflow['id'] == c['workflow_id'] and workflow['pipeline_id'] == c['pipeline_id']
          and workflow['name'] == 'selector-upload-replay' and workflow['status'] == 'success', 'Foreign Circle replay')
    pages = sorted((circle / 'api-pages').glob('jobs-*.json'), key=lambda p: int(p.stem.split('-')[1]))
    check([p.name for p in pages] == ['jobs-' + str(i) + '.json' for i in range(len(pages))], 'Missing Circle job page')
    jobs = []
    for i, p in enumerate(pages):
        page = read(p); jobs.extend(page['items'])
        check(bool(page.get('next_page_token')) == (i < len(pages) - 1), 'Incomplete Circle pagination')
    check(len(jobs) == 1 and jobs[0]['status'] == 'success'
          and jobs[0]['job_number'] == c['job_number'] and job['workflows']['job_id'] == jobs[0]['id']
          and job['build_num'] == c['job_number'] and job['vcs_revision'] == sha and job['branch'] == s['branch']
          and job['workflows']['workflow_id'] == c['workflow_id'] and job['status'] == job['outcome'] == 'success'
          and not job['retry_of'] and all(job[key] is False for key in ('failed', 'timedout', 'canceled', 'infrastructure_fail')),
          'Failed, skipped or retried original Circle publisher')
    config = read(circle / 'pipeline-config.json')
    check(config['compiled'] == (circle / 'compiled.yml').read_text()
          and config['source'] == (circle / 'source.yml').read_text(), 'Changed original compiled Circle configuration')
    compiled = G.yaml(circle / 'compiled.yml')
    publisher = circle_replay_job(G.yaml(circle / 'source.yml'), compiled, jobs[0]['name'])
    commands = [step['run']['command'] for step in publisher['steps']
                if isinstance(step, dict) and isinstance(step.get('run'), dict)]
    check(any('python3 ops/ci/selector-upload.py run .ci/selector-upload/run' in command for command in commands)
          and 'just update-selectors' not in commands, 'Actual Circle publisher was not isolated')
    preparation = circle / 'archive/prepare.console.log'
    check(preparation.is_file() and preparation.stat().st_size > 0
          and any('> .ci/selector-upload/prepare.console.log 2>&1' in command for command in commands),
          'Missing complete original Circle preparation console')
    # The preparation artifact and its full stream are retained separately
    # from the publisher report; both remain bound by the collection seal.
    check((circle / 'selector-registry-originals.tar.gz').is_file(), 'Missing original Circle registry preparation archive')
    index = read(circle / 'step-index.json')
    actions = [(i, j) for i, step in enumerate(job['steps']) for j, action in enumerate(step['actions']) if action.get('output_url')]
    check([(row['step'], row['action']) for row in index] == actions, 'Missing complete original Circle step logs')
    for row in index:
        events = read(circle / row['path'])
        check(all(event.get('truncated') is False and isinstance(event['message'], str) for event in events), 'Truncated Circle originals')
    original_steps = [row for row in index if row['name'] == 'Run original selector publisher against private official registry']
    check(len(original_steps) == 1 and original_steps[0]['status'] == 'success', 'Missing executed original Circle publisher')
    github = read(github_path)
    checks = G.named(github['checks'], 'name')
    native_check = checks['RWX: optimism-selector-upload-shadow']
    check(github['sha'] == sha and native_check['state'] == 'success'
          and '/runs/' + run['ID'] in native_check['link'], 'Missing successful same-SHA optional native check')
    return {'circle_pipeline': pipeline['number'], 'circle_job': job['build_num'], 'native_run': run['ID']}


def compare(circle, native, run_path, github_path):
    ids = hosted(circle, native, run_path, github_path)
    values = {'circleci': report(circle / 'report'), 'rwx': report(native)}
    c, n = values['circleci'], values['rwx']
    check(c['settings']['provider'] == 'circleci' and n['settings']['provider'] == 'rwx', 'Wrong original providers')
    for key in ('version', 'source_sha', 'branch', 'profile', 'tools', 'compilers', 'command', 'inputs', 'implementation'):
        check(c['settings'][key] == n['settings'][key], 'Original publisher identity differs: ' + key)
    check(normalized(c['settings']['configuration'], c['settings']['workspace_root'])
          == normalized(n['settings']['configuration'], n['settings']['workspace_root']), 'Original Foundry settings differ')
    for key in ('initial', 'stable', 'expected', 'compilers', 'imports', 'frames', 'readback', 'database', 'registry_source', 'service'):
        check(c[key] == n[key], 'Complete original selector parity differs: ' + key)
    return {'verified_parity': True, 'source_sha': c['settings']['source_sha'], **ids,
            'selected_declarations': len(c['stable']['selection']['selected']),
            'compiled_sources': len(c['stable']['selection']['compiled_sources']),
            'signatures': {kind: len(rows) for kind, rows in c['expected']['signatures'].items()},
            'database_signatures': len(c['database']), 'requests': c['imports']['requests'],
            'original_manifest_sha256': {'circleci': R.digest(circle / 'report/final.json'), 'rwx': R.digest(native / 'final.json')},
            'complete_original_files': {'circleci': len(U.files(circle / 'report')) + 1, 'rwx': len(U.files(native)) + 1},
            'normalizations': ['workspace path prefixes', 'compiler source mtimes',
                               'profile completion dependent artifact IDs after exact unit validation',
                               'POST signature array ordering', 'database insertion timestamps', 'API result array ordering']}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ('circle', 'native', 'run', 'github', 'output'):
        parser.add_argument('--' + key, type=Path, required=True)
    args = parser.parse_args()
    result = compare(args.circle, args.native, args.run, args.github)
    R.write(args.output, result)
    print(json.dumps(result))
