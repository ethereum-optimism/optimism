"""Shared helpers for the ops/ci unit tests."""
from contextlib import contextmanager
import os
from pathlib import Path


@contextmanager
def preserve_working_directory():
    """Restore the test process after a CLI runner changes its working directory."""
    previous = Path.cwd()
    try:
        yield
    finally:
        os.chdir(previous)
