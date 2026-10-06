"""Suite-independent report bytes, artifact checks and subprocess evidence.

Callers choose required files, report layout, working directory and redaction.
Callers own selection, source/settings binding, retries and verdict rules.
"""
from concurrent.futures import ThreadPoolExecutor
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import sys
import time

EMPTY_SHA256 = hashlib.sha256(b'').hexdigest()


def read(path):
    return json.loads(Path(path).read_text())


def write(path, value):
    Path(path).write_text(json.dumps(value, indent=2, sort_keys=True) + '\n')


def digest(path):
    with Path(path).open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def command(argv, *, cwd=None):
    return subprocess.check_output(argv, cwd=cwd, text=True).strip()


def normalize(value, root):
    """Replace only the caller's workspace path in nested report values."""
    if isinstance(value, str):
        return value.replace(root, '<repo>')
    if isinstance(value, dict):
        return {key: normalize(item, root) for key, item in value.items()}
    if isinstance(value, list):
        return [normalize(item, root) for item in value]
    return value


def file_hashes(directory, *, exclude=()):
    directory = Path(directory)
    return {str(path.relative_to(directory)): digest(path)
            for path in sorted(directory.rglob('*'))
            if path.is_file() and path.name not in exclude}


def verify_files(directory, hashes, *, required=(), missing_empty=None, label='report'):
    """Verify listed bytes. Only explicit offline callers may recover empty files.

    A missing_empty list authorizes recreation of manifest-declared empty files
    and records each recovery. It never authorizes a missing nonempty file.
    This function does not interpret report status or revision/settings fields.
    """
    if not isinstance(hashes, dict) or not set(required) <= hashes.keys():
        raise ValueError('Incomplete original file manifest: ' + label)
    directory = Path(directory)
    for name, sha in hashes.items():
        if not isinstance(name, str) or not isinstance(sha, str):
            raise ValueError('Invalid original report path or hash')
        path = Path(name)
        if path.is_absolute() or '..' in path.parts or not re.fullmatch('[0-9a-f]{64}', sha):
            raise ValueError('Invalid original report path or hash')
        target = directory / path
        if not target.exists() and sha == EMPTY_SHA256 and missing_empty is not None:
            missing_empty.append({'report': label, 'path': name, 'sha256': sha})
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(b'')
        if digest(target) != sha:
            raise ValueError('Missing or corrupt original report: ' + label + '/' + name)
    return hashes


@contextmanager
def forwarding_signals(child):
    """Forward cancellation to the child's process group; restore parent handlers."""
    previous = {}
    def cancel(signum, _frame):
        try:
            os.killpg(child.pid, signum)
        except ProcessLookupError:
            pass
    try:
        for signum in (signal.SIGINT, signal.SIGTERM):
            previous[signum] = signal.signal(signum, cancel)
        yield
    finally:
        for signum, handler in previous.items():
            signal.signal(signum, handler)


def stage(directory, name, argv, *, cwd, layout='combined', stdout_json=False,
          stdout_file=None, stdin=None, redactor=lambda data: data):
    """Run once, retain original streams and signal, and return a shell exit code.

    combined preserves Rust stage logs and optional separate stdout. split
    preserves contract stdout/stderr files and redacts before saving or echoing.
    Suite adapters choose the layout; this helper never retries the command.
    """
    if layout not in ('combined', 'split'):
        raise ValueError('Unknown stage report layout')
    if stdout_file is not None and (stdout_json or Path(stdout_file).name != stdout_file or layout != 'combined'):
        raise ValueError('Invalid separated original stdout destination')
    directory = Path(directory)
    started = time.time()
    row = {'argv': [redactor(arg.encode()).decode() for arg in argv], 'cwd': str(cwd),
           'started_at': started, 'exit_code': None}
    if stdout_file is not None:
        row['stdout_file'] = stdout_file
    if stdin == subprocess.DEVNULL:
        row['stdin'] = 'devnull'
    record = directory / (name + '.stage.json')
    write(record, row)

    def copy(stream, path, echo):
        with path.open('wb') as output:
            for line in iter(stream.readline, b''):
                line = redactor(line)
                output.write(line); output.flush()
                if echo:
                    sys.stdout.buffer.write(line); sys.stdout.buffer.flush()

    if layout == 'split':
        stdout = directory / (name + ('.json' if stdout_json else '.log'))
        stderr = directory / (name + '.stderr.log')
        child = subprocess.Popen(argv, cwd=cwd, stdin=stdin, stdout=subprocess.PIPE,
                                 stderr=subprocess.PIPE, start_new_session=True)
        try:
            with forwarding_signals(child), ThreadPoolExecutor(max_workers=2) as pool:
                readers = [pool.submit(copy, child.stdout, stdout, not stdout_json),
                           pool.submit(copy, child.stderr, stderr, True)]
                status = child.wait()
                for reader in readers:
                    reader.result()
        finally:
            child.stdout.close(); child.stderr.close()
        row.update(stdout_sha256=digest(stdout), stderr_sha256=digest(stderr))
    else:
        separate = stdout_json or stdout_file is not None
        log = directory / (name + '.log')
        stdout = directory / (stdout_file or name + '.json')
        with log.open('wb') as output, (stdout.open('wb') if separate else open(os.devnull, 'wb')) as out:
            child = subprocess.Popen(argv, cwd=cwd, stdin=stdin, start_new_session=True,
                                     stdout=out if separate else subprocess.PIPE,
                                     stderr=output if separate else subprocess.STDOUT)
            try:
                with forwarding_signals(child):
                    if not separate:
                        for line in iter(child.stdout.readline, b''):
                            output.write(line); output.flush()
                            sys.stdout.buffer.write(line); sys.stdout.buffer.flush()
                    status = child.wait()
            finally:
                if child.stdout is not None:
                    child.stdout.close()
        row['log_sha256'] = digest(log)
        if stdout_file is not None:
            row['stdout_sha256'] = digest(stdout)
    row.update(exit_code=status, elapsed_seconds=time.time() - started)
    write(record, row)
    return status if status >= 0 else 128 - status
