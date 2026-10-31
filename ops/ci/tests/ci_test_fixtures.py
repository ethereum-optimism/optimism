"""Report construction mechanics for suite-specific comparison fixtures."""
import importlib.util
from contextlib import contextmanager
import json
import os
from pathlib import Path

SPEC = importlib.util.spec_from_file_location('ci_report', Path(__file__).resolve().parents[1] / 'runtime' / 'ci-report.py')
REPORT = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(REPORT)


@contextmanager
def preserve_working_directory():
    """Restore the test process after a CLI runner changes its working directory."""
    previous = Path.cwd()
    try:
        yield
    finally:
        os.chdir(previous)


class ReportFixtures:
    # Suite tests supply the original selection, settings, commands and cases.
    write = staticmethod(REPORT.write)

    def seal(self, directory):
        self.write(directory / 'final.json', {
            'exit_code': 0, 'report_errors': [],
            'original_sha256': REPORT.file_hashes(directory, exclude=('final.json',)),
        })

    def mutate(self, directory, filename, change):
        path = directory / filename
        value = json.loads(path.read_text())
        change(value)
        self.write(path, value)
        self.seal(directory)
