"""Exercise fork and tool selection for NUT provenance against a real Git repository."""
import importlib.util
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location('nut_provenance', Path(__file__).resolve().parents[1] / 'runtime' / 'nut-provenance.py')
N = importlib.util.module_from_spec(SPEC); SPEC.loader.exec_module(N)


class ProvenanceTests(unittest.TestCase):

    def git(self, *argv):
        identity = ['-c', 'user.name=CI fixture', '-c', 'user.email=ci-fixture@example.invalid']
        return subprocess.check_output(['git', *identity, *argv], cwd=self.root, text=True).strip()

    def commit(self, files, message):
        for name, text in files.items():
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(text)
        self.git('add', '.')
        self.git('commit', '-qm', message)
        return self.git('rev-parse', 'HEAD')

    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.git('init', '-q')
        # Each fork records a commit with its own pins. The checkout pins other versions.
        recorded = {
            'karst': self.commit({'mise.toml': '[tools]\nforge = "1.2.3"\njust = "1.46.0"\n'}, 'karst generator'),
            'lagoon': self.commit({'mise.toml': '[tools]\nforge = "1.8.3"\njust = "1.46.0"\n'}, 'lagoon generator'),
        }
        recorded['mesa'] = recorded['karst']
        lock = ''.join('[' + fork + ']\n  commit = "' + sha + '"\n' for fork, sha in recorded.items())
        self.commit({'mise.toml': '[tools]\nforge = "9.9.9"\njust = "9.9.9"\n', N.LOCK: lock}, 'current checkout')

    def test_pins_come_from_the_commit_that_each_fork_records(self):
        old = {'forge': '1.2.3', 'just': '1.46.0'}
        self.assertEqual(N.generator_pins(self.root),
                         {'karst': old, 'lagoon': {'forge': '1.8.3', 'just': '1.46.0'}, 'mesa': old})

    def test_verify_runs_the_verifier_for_every_locked_fork_with_its_recorded_tools(self):
        with patch.object(N.subprocess, 'call', return_value=0) as call:
            self.assertEqual(N.verify(self.root), 0)
        self.assertEqual([c.args[0] for c in call.call_args_list], [
            ['mise', 'exec', 'forge@1.2.3', 'just@1.46.0', '--', *N.VERIFIER, 'karst'],
            ['mise', 'exec', 'forge@1.8.3', 'just@1.46.0', '--', *N.VERIFIER, 'lagoon'],
            ['mise', 'exec', 'forge@1.2.3', 'just@1.46.0', '--', *N.VERIFIER, 'mesa']])
        self.assertTrue(all(c.kwargs['cwd'] == self.root for c in call.call_args_list))

    def test_verify_stops_at_the_first_failing_verifier_and_returns_its_status(self):
        with patch.object(N.subprocess, 'call', side_effect=[0, 17, 0]) as call:
            self.assertEqual(N.verify(self.root), 17)
        self.assertEqual([c.args[0][-1] for c in call.call_args_list], ['karst', 'lagoon'])

    def test_install_tools_installs_each_distinct_forge_and_just_pair_once(self):
        installs, run = [], subprocess.run

        def record(argv, **kwargs):
            # generator_pins reads Git through subprocess.run; only the installer is replaced.
            if argv[0] != 'bash':
                return run(argv, **kwargs)
            installs.append(argv[1:])
            return subprocess.CompletedProcess(argv, 0)
        with patch.object(N.subprocess, 'run', side_effect=record):
            N.install_tools(self.root)
        self.assertEqual(installs, [[N.TOOLS, 'solc'], [N.TOOLS, 'generators', '1.2.3', '1.46.0'],
                                    [N.TOOLS, 'generators', '1.8.3', '1.46.0']])

    def test_installer_rejects_a_pin_that_is_not_an_exact_version_before_it_installs(self):
        result = subprocess.run(['bash', str(N.ROOT / N.TOOLS), 'generators', 'latest', '1.46.0'],
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertIn("'latest' '1.46.0'", result.stderr)


if __name__ == '__main__':
    unittest.main()
