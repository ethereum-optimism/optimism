#!/usr/bin/env python3
"""Compare real reporter fixtures and decode synthetic Go and JUnit edge cases.

The fixtures hold selected real records, not complete runs. The edge-case tests
are the only direct tests of ops/ci/runtime/ci-test-results.py.
"""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location('compare_ci', Path(__file__).resolve().parents[1] / 'compare-ci.py')
CI = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CI)
T = CI.T
FIXTURES = Path(__file__).parent / 'fixtures/ci-comparison'
GO_API = FIXTURES / 'circleci-go-cases.json'
FOUNDRY_API = FIXTURES / 'circleci-foundry-skips.json'
FOUNDRY_JUNIT = FIXTURES / 'rwx-foundry-skips.xml'
PACKAGE = 'github.com/ethereum-optimism/optimism/op-node/rollup'
RUN = {'Action': 'run', 'Package': PACKAGE, 'Test': 'TestSynthetic'}
PASS = {'Action': 'pass', 'Package': PACKAGE, 'Test': 'TestSynthetic'}
PACKAGE_PASS = {'Action': 'pass', 'Package': PACKAGE}
PACKAGE_FAIL = {'Action': 'fail', 'Package': PACKAGE}


class ReportTest(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)

    def write(self, name, text):
        path = self.root / name
        path.write_text(text)
        return path

    def api(self, name, items, **fields):
        return self.write(name, json.dumps({'items': items, **fields}))

    def events(self, name, events):
        return self.write(name, ''.join(json.dumps(event) + '\n' for event in events))


class ComparisonTest(ReportTest):
    def run_cli(self, baseline, candidate):
        output = io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(output):
            status = CI.main(['--baseline', *map(str, baseline), '--candidate', *map(str, candidate)])
        return status, output.getvalue()

    def test_real_circleci_go_records_keep_subtest_identity_and_skip_reason(self):
        cases = CI.load([GO_API])
        self.assertEqual(len(cases), 3)
        skipped = {key: reason for key, (outcome, reason) in cases.items() if outcome == 'skip'}
        self.assertEqual(len(skipped), 2)
        self.assertEqual(set(skipped.values()), {'only applicable to span batches'})
        self.assertTrue(all('/' in name for _, name in skipped))

    def test_real_foundry_api_and_junit_records_are_equivalent(self):
        # The two fixtures come from different commits. The test compares the
        # reporter conversion only: suite identity and skip reasons.
        api, junit = CI.load([FOUNDRY_API]), CI.load([FOUNDRY_JUNIT])
        self.assertEqual(api, junit)
        self.assertEqual(len(junit), 4)
        reasons = [reason for _, reason in junit.values()]
        self.assertEqual(reasons.count(None), 2)
        self.assertIn('Skipping: standard configs incompatible with SUPER_ROOT_GAMES_MIGRATION', reasons)
        self.assertTrue(all(suite.startswith('test/') and ':' in suite for suite, _ in junit))
        status, output = self.run_cli([FOUNDRY_API], [FOUNDRY_JUNIT])
        self.assertEqual(status, CI.EQUIVALENT, output)
        self.assertIn('equivalent (4 baseline cases, 4 candidate cases)', output)

    def test_equal_counts_do_not_hide_missing_extra_and_different_cases(self):
        first, second, third = T.read_json(GO_API)['items']
        candidate = self.api('candidate.json', [{**first, 'name': 'TestSubstitute'}, {**second, 'result': 'failure'}, third])
        self.assertEqual(CI.compare(CI.load([GO_API]), CI.load([candidate])),
                         {'missing': [(first['classname'], first['name'])],
                          'extra': [(first['classname'], 'TestSubstitute')],
                          'different': [(second['classname'], second['name'])]})
        status, output = self.run_cli([GO_API], [candidate])
        self.assertEqual(status, CI.DIFFERENT, output)
        self.assertIn(f"missing from candidate: {first['classname']} {first['name']}: pass", output)
        self.assertIn(f"extra in candidate: {first['classname']} TestSubstitute: pass", output)
        self.assertIn('baseline skip (only applicable to span batches), candidate fail', output)

    def test_cases_and_packages_that_fail_on_both_sides_are_equivalent(self):
        first = T.read_json(GO_API)['items'][0]
        failing = self.api('failing.json', [{**first, 'result': 'failure'}])
        self.assertEqual(self.run_cli([failing], [failing])[0], CI.EQUIVALENT)
        package_fails = self.events('fails.json', [RUN, PASS, PACKAGE_FAIL])
        package_passes = self.events('passes.json', [RUN, PASS, PACKAGE_PASS])
        self.assertEqual(self.run_cli([package_fails], [package_fails])[0], CI.EQUIVALENT)
        status, output = self.run_cli([package_fails], [package_passes])
        self.assertEqual(status, CI.DIFFERENT, output)
        self.assertIn(f'missing from candidate: {PACKAGE} (package): fail', output)

    def test_shards_are_joined_and_a_case_in_two_reports_is_rejected(self):
        items = T.read_json(GO_API)['items']
        shards = [self.api('shard-0.json', items[:1]), self.write('shard-1.json', json.dumps(items[1:]))]
        self.assertEqual(CI.load(shards), CI.load([GO_API]))
        self.assertEqual(self.run_cli(shards, [GO_API])[0], CI.EQUIVALENT)
        with self.assertRaisesRegex(ValueError, 'reports a case twice'):
            CI.load([GO_API, shards[0]])
        self.assertEqual(self.run_cli([GO_API, shards[0]], [GO_API])[0], CI.REJECTED)
        # Test-level shards run the same package. Its failure in each shard is one failure.
        failed = [self.events(f'go-{index}.json', [{**RUN, 'Test': name}, {**PASS, 'Test': name}, PACKAGE_FAIL])
                  for index, name in enumerate(('TestOne', 'TestTwo'))]
        self.assertEqual(CI.load(failed), {(PACKAGE, 'TestOne'): ('pass', None), (PACKAGE, 'TestTwo'): ('pass', None),
                                           (PACKAGE, CI.PACKAGE): ('fail', None)})

    def test_incomplete_empty_or_malformed_responses_are_rejected(self):
        row = T.read_json(GO_API)['items'][0]
        for message, path in (('one page of several', self.api('page.json', [row], next_page_token='still-more')),
                              ('no test case', self.api('empty.json', [])),
                              ('no `items` list', self.write('error.json', '{"message": "Not Found"}')),
                              ('unknown CircleCI test result', self.api('result.json', [{**row, 'result': 'mystery'}])),
                              ('finite nonnegative', self.api('time.json', [{**row, 'run_time': -1}])),
                              ('duplicate JSON key', self.write('keys.json', '{"items": [], "items": []}'))):
            with self.subTest(path=path.name):
                with self.assertRaisesRegex(ValueError, message):
                    CI.load([path])
                self.assertEqual(self.run_cli([path], [GO_API])[0], CI.REJECTED)
        self.assertEqual(self.run_cli([self.root / 'absent.json'], [GO_API])[0], CI.REJECTED)

    def test_sides_without_a_common_case_are_incomparable(self):
        status, output = self.run_cli([GO_API], [FOUNDRY_JUNIT])
        self.assertEqual(status, CI.INCOMPARABLE, output)
        self.assertIn('incomparable (3 baseline cases, 4 candidate cases, none in common)', output)

    def test_format_follows_the_xml_extension_or_the_content(self):
        self.assertEqual(CI.report_format(FOUNDRY_JUNIT), 'junit')
        self.assertEqual(CI.report_format(self.write('junit.txt', '\n<testsuite/>')), 'junit')
        self.assertEqual(CI.report_format(GO_API), 'circleci-tests')
        self.assertEqual(CI.report_format(self.api('compact.json', [])), 'circleci-tests')
        events = self.events('events.json', [RUN, PASS, PACKAGE_PASS])
        self.assertEqual(CI.report_format(events), 'go-json')
        junit = self.write('junit.xml', f'<testsuite><testcase classname="{PACKAGE}" name="TestSynthetic"/></testsuite>')
        self.assertEqual(self.run_cli([events], [junit])[0], CI.EQUIVALENT)


class ParserTest(ReportTest):
    def go_cases(self, events):
        source = {'feature': None, 'id': 'events'}
        cases, failures = T.go_cases(self.events('events.json', events), source, {})
        return cases, failures, source

    def test_go_retry_keeps_every_attempt_and_the_final_verdict(self):
        # Synthetic events in the order that gotestsum writes them when it reruns a failed test.
        events = [{'Action': action, 'Package': PACKAGE, 'Test': 'TestSynthetic', 'Elapsed': 1}
                  for action in ('run', 'fail', 'run', 'pass')]
        events.insert(2, PACKAGE_FAIL)
        events.append(PACKAGE_PASS)
        cases, failures, source = self.go_cases(events)
        self.assertEqual([case['outcome'] for case in cases], ['pass'])
        self.assertEqual([attempt['outcome'] for attempt in cases[0]['attempts']], ['fail', 'pass'])
        self.assertEqual(failures, [])
        self.assertEqual(source['package_attempts'], [{'suite': PACKAGE, 'outcomes': ['fail', 'pass']}])

    def test_go_package_failure_outside_passing_cases_is_kept(self):
        cases, failures, _ = self.go_cases([RUN, PASS, PACKAGE_FAIL])
        self.assertEqual([case['outcome'] for case in cases], ['pass'])
        self.assertEqual(failures, [PACKAGE])

    def test_go_build_events_identified_by_import_path_keep_the_package_verdict(self):
        # Go 1.24+ emits build events with ImportPath and no Package field.
        passing = [{'ImportPath': PACKAGE + ' [' + PACKAGE + '.test]', 'Action': 'build-output', 'Output': '# compiling\n'},
                   RUN, PASS, PACKAGE_PASS]
        cases, failures, _ = self.go_cases(passing)
        self.assertEqual([case['outcome'] for case in cases], ['pass'])
        self.assertEqual(failures, [])
        broken = PACKAGE + '/broken'
        target = broken + ' [' + broken + '.test]'
        failed = passing + [{'ImportPath': target, 'Action': 'build-output', 'Output': 'x.go:1: undefined: y\n'},
                            {'ImportPath': target, 'Action': 'build-fail'},
                            {'Action': 'start', 'Package': broken},
                            {'Action': 'fail', 'Package': broken, 'FailedBuild': target}]
        _, failures, source = self.go_cases(failed)
        self.assertEqual(failures, [broken])
        self.assertEqual(source['observed_packages'], [PACKAGE, broken])
        with self.assertRaisesRegex(ValueError, 'ImportPath'):
            self.go_cases([{'Action': 'build-fail'}] + passing)

    def test_go_package_without_tests_is_observed(self):
        _, failures, source = self.go_cases([RUN, PASS, PACKAGE_PASS, {'Action': 'skip', 'Package': PACKAGE + '/no-tests'}])
        self.assertEqual(source['observed_packages'], [PACKAGE, PACKAGE + '/no-tests'])
        self.assertEqual(failures, [])

    def test_truncated_or_ambiguous_go_events_are_rejected(self):
        for message, events in (('tests without terminal verdicts', [RUN]), ('without matching run', [PASS]),
                                ('started twice', [RUN, RUN]), ('packages without terminal verdicts', [RUN, PASS])):
            with self.subTest(events=events), self.assertRaisesRegex(ValueError, message):
                self.go_cases(events)

    def test_malformed_junit_is_rejected(self):
        for message, xml in (('root must be', '<other/>'),
                             ('classname or parent suite name', '<testsuite><testcase name="MissingClass"/></testsuite>'),
                             ('DTDs/entities', '<!DOCTYPE testsuite><testsuite/>'),
                             ('ambiguous', f'<testsuite><testcase classname="{PACKAGE}" name="test"><failure/><skipped/></testcase></testsuite>')):
            with self.subTest(xml=xml), self.assertRaisesRegex(ValueError, message):
                T.junit_cases(self.write('results.xml', xml), {'feature': None, 'id': 'junit'}, {})


if __name__ == '__main__':
    unittest.main()
