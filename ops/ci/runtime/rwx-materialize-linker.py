#!/usr/bin/env python3
"""Detach native linker inputs from restored RWX layers before WASM builds."""
import errno
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import time


# WASM build scripts still link native executables. Ask the image's default C
# driver for its inputs rather than pinning a GCC version or following every
# file in /usr/lib. Include the targets of glibc/GCC linker scripts as well.
INPUTS = (
    'crt1.o', 'Scrt1.o', 'crti.o', 'crtn.o', 'crtbegin.o', 'crtbeginS.o',
    'crtbeginT.o', 'crtend.o', 'crtendS.o', 'libgcc.a', 'libgcc_eh.a',
    'libgcc_s.so', 'libgcc_s.so.1', 'libc.so', 'libc.so.6', 'libc_nonshared.a',
    'libm.so', 'libm.so.6', 'libmvec.so.1', 'libpthread.a', 'libdl.a',
    'librt.a', 'libutil.a', 'ld-linux-x86-64.so.2',
)


def linker_inputs():
    paths = set()
    for name in INPUTS:
        value = subprocess.check_output(
            ['/usr/bin/cc', '-print-file-name=' + name], text=True, timeout=30).strip()
        path = Path(value)
        if not path.is_absolute():
            raise ValueError('C driver could not locate linker input: ' + name)
        path = path.resolve(strict=True)
        if not path.is_relative_to('/usr/lib'):
            raise ValueError('Linker input outside /usr/lib: ' + str(path))
        paths.add(path)
    return sorted(paths)


def materialize(path, record):
    started = time.monotonic()
    record.update(path=str(path), read_attempts=0, read_timeouts=0)
    try:
        for attempt in range(3):
            record['read_attempts'] += 1
            try:
                before = path.stat()
                if not stat.S_ISREG(before.st_mode) or before.st_size > 32 * 1024 * 1024:
                    raise ValueError('Unexpected linker input type or size: ' + str(path))
                data = path.read_bytes()
                after = path.stat()
                if (before.st_ino, before.st_size, before.st_mtime_ns) != (
                        after.st_ino, after.st_size, after.st_mtime_ns) or len(data) != before.st_size:
                    raise ValueError('Linker input changed while reading: ' + str(path))
                break
            except OSError as error:
                # Retry only the filesystem failure observed in hosted builds.
                # Never retry Cargo, the linker, or a compilation/test verdict.
                if error.errno != errno.ETIMEDOUT:
                    raise
                record['read_timeouts'] += 1
                if attempt == 2:
                    raise
                time.sleep(attempt + 1)
        expected = hashlib.sha256(data).hexdigest()
        fd, temporary = tempfile.mkstemp(prefix='.rwx-linker-', dir=path.parent)
        try:
            with os.fdopen(fd, 'wb') as target:
                target.write(data)
                os.fchown(target.fileno(), before.st_uid, before.st_gid)
                os.fchmod(target.fileno(), stat.S_IMODE(before.st_mode))
            # A real write to a fresh inode is intentional: reading the old file
            # alone only warms a page cache that the kernel can evict again.
            if hashlib.sha256(Path(temporary).read_bytes()).hexdigest() != expected:
                raise ValueError('Local linker input digest mismatch: ' + str(path))
            os.utime(temporary, ns=(before.st_atime_ns, before.st_mtime_ns))
            os.replace(temporary, path)
        finally:
            Path(temporary).unlink(missing_ok=True)
        current = path.stat()
        if current.st_ino == before.st_ino or current.st_nlink != 1:
            raise ValueError('Linker input was not detached: ' + str(path))
        record.update(bytes=len(data), sha256=expected, original_inode=before.st_ino,
                      local_inode=current.st_ino, detached=True)
    finally:
        record['seconds'] = time.monotonic() - started


def main():
    report = {'schema': 1, 'files': [], 'success': False}
    try:
        for path in linker_inputs():
            record = {}
            report['files'].append(record)
            materialize(path, record)
        report['success'] = True
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        report['error'] = str(error)
    print(json.dumps(report, sort_keys=True), flush=True)
    return 0 if report['success'] else 1


if __name__ == '__main__':
    sys.exit(main())
