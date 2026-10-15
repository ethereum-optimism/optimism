"""Verify source bytes and submodule identities against a committed revision."""
import hashlib
import json
import os
from pathlib import Path
import subprocess

def check(value, message):
    if not value:
        raise ValueError(message)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def write(path, data):
    Path(path).parent.mkdir(parents=True, exist_ok=True)
    Path(path).write_text(json.dumps(data, indent=2, sort_keys=True) + '\n')


def command(argv, **kwargs):
    return subprocess.check_output(argv, **kwargs).decode().strip()


def source_inputs(source, revision, allow_gitlinks=False):
    """Hash every real input and verify it against the pinned committed tree."""
    result = {}
    for entry in subprocess.check_output(['git', '-C', str(source), 'ls-tree', '-rz', revision]).split(b'\0'):
        if not entry:
            continue
        header, filename = entry.split(b'\t', 1)
        mode, kind, oid = header.split()
        name = filename.decode()
        path = Path(source) / name
        check(name not in result and '..' not in Path(name).parts,
              'Unsupported or duplicate source input')
        if kind == b'commit':
            check(mode == b'160000' and path.is_dir(), 'Missing source gitlink: ' + name)
            if allow_gitlinks:
                check(command(['git', '-C', str(path), 'rev-parse', 'HEAD']) == oid.decode(),
                      'Mismatched source submodule: ' + name)
                result[name] = {'mode': mode.decode(), 'gitlink': oid.decode(), 'checkout': 'initialized'}
                result.update({name + '/' + key: value for key, value in
                               source_inputs(path, oid.decode(), True).items()})
            else:
                # The official service image is built without recursive
                # checkout. database-specs is an unused empty gitlink; the
                # schema loaded below is a separate committed SQL blob.
                check(not any(path.iterdir()), 'Unexpected content in unused source gitlink: ' + name)
                result[name] = {'mode': mode.decode(), 'gitlink': oid.decode(), 'checkout': 'uninitialized'}
            continue
        check(kind == b'blob', 'Unsupported source tree entry')
        if mode == b'120000':
            check(path.is_symlink(), 'Source link is missing')
            data = os.readlink(path).encode()
        else:
            check(mode in (b'100644', b'100755') and path.is_file() and not path.is_symlink(),
                  'Source file is missing')
            check(bool(path.stat().st_mode & 0o111) == (mode == b'100755'),
                  'Changed committed source executable mode: ' + name)
            data = path.read_bytes()
        check(hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest() == oid.decode(),
              'Changed committed source input: ' + name)
        result[name] = {'mode': mode.decode(), 'sha256': hashlib.sha256(data).hexdigest()}
    check(result, 'Empty source tree')
    return result
