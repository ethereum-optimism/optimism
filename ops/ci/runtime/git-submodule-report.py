#!/usr/bin/env python3
"""Validate original recursive Git submodule status by path and commit identity."""
import re


def revisions(text):
    rows = []
    for index, line in enumerate(text.splitlines()):
        match = re.fullmatch(r'([ +\-U]?)([0-9a-f]{40}) ([^\s]+)(?: \(([^\n()]*)\))?', line)
        if not match or match[1] not in ('', ' ') or (not match[1] and index != 0):
            raise ValueError('Invalid, uninitialized or mismatched submodule status')
        prefix, sha, path, description = match.groups()
        if path.startswith('/') or '..' in path.split('/'):
            raise ValueError('Invalid original submodule path')
        rows.append({'path': path, 'sha': sha})
    if not rows or len({r['path'] for r in rows}) != len(rows):
        raise ValueError('Empty or duplicate original submodule selection')
    return sorted(rows, key=lambda r: r['path'])
