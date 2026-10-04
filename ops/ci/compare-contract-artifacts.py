#!/usr/bin/env python3
"""Compare every bound contract input while retaining compiler metadata differences."""
from collections import Counter
from contextlib import contextmanager
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import tarfile
import tempfile

AST_IDS = {'id', 'scope', 'referencedDeclaration', 'functionReturnParameters', 'declaration', 'sourceUnit',
           'linearizedBaseContracts', 'usedErrors', 'usedEvents', 'contractDependencies', 'baseFunctions',
           'assignments', 'overloadedDeclarations'}
SOURCE_LOCATIONS = {'src', 'nativeSrc', 'nameLocation', 'nameLocations', 'memberLocation'}


def digest(data): return hashlib.sha256(data).hexdigest()


def normalize(value, root):
    if isinstance(value, str): return value.replace(root, '<repo>')
    if isinstance(value, dict): return {k: normalize(v, root) for k, v in value.items()}
    if isinstance(value, list): return [normalize(v, root) for v in value]
    return value


def ast_differences(a, b, counts, key='', exported=False):
    """Permit only compiler identifiers, retaining every structural/value check."""
    if a == b: return
    if type(a) is not type(b): raise ValueError('Contract AST value type differs')
    if isinstance(a, dict):
        if a.keys() != b.keys(): raise ValueError('Contract AST fields differ')
        for name in a: ast_differences(a[name], b[name], counts, name, exported or name == 'exportedSymbols')
    elif isinstance(a, list):
        if len(a) != len(b): raise ValueError('Contract AST structure differs')
        for x, y in zip(a, b): ast_differences(x, y, counts, key, exported)
    elif type(a) is int and (key in AST_IDS or exported) and min(a, b) >= 0:
        counts['node_identifiers'] += 1
    elif isinstance(a, str) and key in SOURCE_LOCATIONS:
        x, y = (re.fullmatch(r'(-?\d+):(\d+):(-?\d+)', v) for v in (a, b))
        if x is None or y is None or x.groups()[:2] != y.groups()[:2]:
            raise ValueError('Contract AST source span differs')
        counts['source_identifiers'] += 1
    elif isinstance(a, str) and key == 'typeIdentifier' and re.sub(r'_\$\d+', '_$<id>', a) == re.sub(r'_\$\d+', '_$<id>', b):
        counts['type_identifiers'] += 1
    else: raise ValueError('Unresolved contract AST difference at ' + key)


def contract(a, b, roots, counts):
    if a.keys() != b.keys(): raise ValueError('Contract artifact fields differ')
    for key in a:
        if a[key] == b[key]: continue
        if key in ('metadata', 'rawMetadata'):
            x, y = (json.loads(v) if key == 'rawMetadata' else v for v in (a[key], b[key]))
            if normalize(x, roots[0]) != normalize(y, roots[1]): raise ValueError('Contract compiler input metadata differs')
            counts['workspace_path_metadata'] += 1
        elif key == 'ast': ast_differences(a[key], b[key], counts)
        elif key == 'id' and type(a[key]) is int and type(b[key]) is int and min(a[key], b[key]) >= 0:
            counts['artifact_source_identifiers'] += 1
        else: raise ValueError('Contract runtime artifact differs at ' + key)


def archive(path, metadata):
    if digest(path.read_bytes()) != metadata['archive_sha256']: raise ValueError('Contract dependency archive changed')
    files = {}
    with tarfile.open(path, mode='r|*') as stream:
        for member in stream:
            name = member.name; relative = Path(name)
            if relative.is_absolute() or '..' in relative.parts: raise ValueError('Unsafe contract archive member')
            if member.isdir(): continue
            if not member.isfile() or name in files: raise ValueError('Invalid or duplicate contract archive member')
            data = stream.extractfile(member).read()
            if metadata['files'].get(name) != digest(data): raise ValueError('Contract input hash differs from original manifest')
            files[name] = data
    if files.keys() != metadata['files'].keys(): raise ValueError('Incomplete contract dependency archive')
    return files


def cache_inputs(value):
    """Validate content/configuration while preserving provider timestamps/build IDs."""
    if isinstance(value, dict):
        return {k: cache_inputs(v) for k, v in value.items() if k not in ('lastModificationDate', 'build_id', 'builds')}
    if isinstance(value, list): return [cache_inputs(v) for v in value]
    return value


def build_info(files, prefix):
    result = []
    for name, data in files.items():
        if name.startswith(prefix):
            row = json.loads(data)
            if set(row) != {'id', 'source_id_to_path', 'language'}: raise ValueError('Unexpected Foundry build-info schema')
            result.append({'language': row['language'], 'source_files': sorted(row['source_id_to_path'].values())})
    return sorted(result, key=lambda r: json.dumps(r, sort_keys=True))


@contextmanager
def embedded_stream(data):
    # Python 3.12 has no native zstd tar reader. Stream through the same zstd
    # tool used to create the deployer archive, without repeatedly seeking it.
    if data.startswith(b'\x28\xb5\x2f\xfd'):
        with tempfile.NamedTemporaryFile(suffix='.tzst') as source:
            source.write(data); source.flush()
            with subprocess.Popen(['zstd', '--decompress', '--stdout', source.name],
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE) as child:
                try:
                    with tarfile.open(fileobj=child.stdout, mode='r|') as stream: yield stream
                    child.stdout.read(); child.stderr.read()
                    if child.wait() != 0: raise ValueError('Corrupt embedded zstd archive')
                finally:
                    if child.poll() is None: child.kill()
    else:
        with tarfile.open(fileobj=io.BytesIO(data), mode='r|*') as stream: yield stream


def embedded(data):
    files, links = {}, {}
    with embedded_stream(data) as stream:
        for member in stream:
            name = member.name; relative = Path(name)
            if relative.is_absolute() or '..' in relative.parts: raise ValueError('Unsafe embedded contract archive')
            if not member.isdir() and (name in files or name in links): raise ValueError('Duplicate embedded contract member')
            if member.isfile():
                files[name] = digest(stream.extractfile(member).read())
            elif member.issym() or member.islnk():
                links[name] = member.linkname
            elif not member.isdir(): raise ValueError('Unexpected embedded contract member')
    return files, links


def compare(directories, sha):
    metadata = {p: json.loads((d / 'metadata.json').read_text()) for p, d in directories.items()}
    for field in ('version', 'kind', 'commit_sha', 'settings', 'mise_sha256', 'tool_versions'):
        if metadata['circle'][field] != metadata['rwx'][field]: raise ValueError('Contract producer binding differs at ' + field)
    if metadata['circle']['commit_sha'] != sha or metadata['circle']['kind'] != 'contracts-e2e':
        raise ValueError('Contract producer source or suite differs')
    files = {p: archive(d / 'files.tar.gz', metadata[p]) for p, d in directories.items()}
    a, b = files['circle'], files['rwx']; counts = Counter()
    prefix = 'packages/contracts-bedrock/'; info_prefix = prefix + 'artifacts/build-info/'
    if {n for n in a if not n.startswith(info_prefix)} != {n for n in b if not n.startswith(info_prefix)}:
        raise ValueError('Contract input file inventories differ')
    if build_info(a, info_prefix) != build_info(b, info_prefix): raise ValueError('Complete compiler source graphs differ')
    roots = []
    for rows in (a, b):
        artifact = json.loads(rows[next(n for n in rows if n.startswith(prefix + 'forge-artifacts/') and n.endswith('.json') and 'metadata' in json.loads(rows[n]))])
        remapping = next(r for r in artifact['metadata']['settings']['remappings'] if '/packages/contracts-bedrock/lib/lib-keccak/' in r)
        roots.append(remapping.split('/packages/contracts-bedrock/', 1)[0])
    contract_files = 0
    embedded_name = 'op-deployer/pkg/deployer/artifacts/forge-artifacts/artifacts.tzst'
    for name in sorted(a.keys() & b.keys()):
        if name.startswith(info_prefix) or name == embedded_name: continue
        if name.startswith(prefix + 'forge-artifacts/') and name.endswith('.json'):
            contract(json.loads(a[name]), json.loads(b[name]), roots, counts); contract_files += 1
        elif name == prefix + 'cache/solidity-files-cache.json':
            if cache_inputs(json.loads(a[name])) != cache_inputs(json.loads(b[name])):
                raise ValueError('Foundry compiler content or configuration cache differs')
        elif a[name] != b[name]: raise ValueError('Unresolved contract dependency file differs: ' + name)
    ea, eb = (embedded(rows[embedded_name]) for rows in (a, b)); fa, la = ea; fb, lb = eb
    if la != lb: raise ValueError('Embedded source links differ')
    def source_inputs(rows): return {n: h for n, h in rows.items() if not n.startswith(('forge-artifacts/', 'artifacts/build-info/', 'cache/')) and n != '.gitcommit'}
    if source_inputs(fa) != source_inputs(fb): raise ValueError('Complete embedded source files differ')
    for rows in (fa, fb):
        if '.gitcommit' in rows and rows['.gitcommit'] != digest((sha + '\n').encode()): raise ValueError('Embedded revision metadata differs')
    # The native source package carries the exact .gitcommit fallback used when
    # deployed from a package without Git. All other source bytes must agree.
    for rows, outer in ((fa, a), (fb, b)):
        expected = {n.removeprefix(prefix): digest(data) for n, data in outer.items() if n.startswith(prefix)}
        if any(rows.get(n) != h for n, h in expected.items()): raise ValueError('Embedded compiler outputs differ from producer outputs')
        if {n for n in rows if n.startswith(('forge-artifacts/', 'artifacts/build-info/', 'cache/'))} != expected.keys():
            raise ValueError('Embedded compilation inventory differs from producer outputs')
    return {'source_sha': sha, 'verified_runtime_input_parity': True, 'contract_artifacts': contract_files,
            'compiler_source_graphs': len(build_info(a, info_prefix)), 'embedded_source_files': len(source_inputs(fa)),
            'resolved_metadata_differences': dict(counts), 'embedded_revision_fallback': {p: '.gitcommit' in rows for p, rows in [('circle', fa), ('rwx', fb)]},
            'original_metadata': metadata, 'embedded_original_sha256': {'circle': fa, 'rwx': fb},
            'archive_bytes_equal': metadata['circle']['archive_sha256'] == metadata['rwx']['archive_sha256']}
