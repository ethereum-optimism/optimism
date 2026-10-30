"""Offline report mechanics; runtime verification never repairs inputs."""
import importlib.util
import json
from pathlib import Path
import re


def helper(name):
    owner = Path(__file__).parent if name.startswith('compare-') or name in ('report-evidence', 'circle-alignment', 'flaky-report', 'selector-upload', 'selector-registry', 'l2-chains-sync-check') else Path(__file__).parents[1] / 'runtime'
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), owner / (name + '.py'))
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module)
    return module


REPORT = helper('ci-report')
normalize = REPORT.normalize


def verify_files(directory, hashes, *, required=(), missing_empty=None, recoverable=None, label='report'):
    """Recover only explicitly authorized manifest-declared empty originals."""
    directory = Path(directory)
    if not isinstance(hashes, dict) or not set(required) <= hashes.keys():
        raise ValueError('Incomplete original file manifest: ' + (label or 'report'))
    for name, sha in hashes.items():
        if not isinstance(name, str) or not isinstance(sha, str) or Path(name).is_absolute() or '..' in Path(name).parts or not re.fullmatch('[0-9a-f]{64}', sha):
            raise ValueError('Invalid original report path or hash')
        target = directory / name
        if not target.exists() and sha == REPORT.EMPTY_SHA256 and missing_empty is not None and (recoverable is None or name in recoverable):
            # An existing symlink must never authorize an out-of-tree write.
            if target.is_symlink() or any(p.is_symlink() for p in target.parents if p.is_relative_to(directory)):
                raise ValueError('Linked original report recovery path')
            target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(b'')
            missing_empty.append({'path': name, 'sha256': sha} | ({'report': label} if label else {}))
    return REPORT.verify_files(directory, hashes, required=required, label=label or 'report')


def originals(directory, required, empty, label):
    final = REPORT.read(directory / 'final.json')
    if final['exit_code'] != 0 or final['report_errors']:
        raise ValueError('Failed or incomplete original contract report: ' + label)
    return verify_files(directory, final['original_sha256'], required=required,
                        missing_empty=empty if label.startswith('circle/') else None, label=label)
