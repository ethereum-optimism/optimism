#!/usr/bin/env python3
"""Compare the test cases that two CI providers report.

Each side is one or more report files, one per shard. A report is JUnit XML,
`go test -json` events, or one complete response of CircleCI's job
test-metadata API. The tool takes the format from the `.xml` extension or from
the content. It joins the reports of a side into
{(suite, name): (outcome, skip reason)}. It prints the cases that are missing
from the candidate, extra in the candidate, or different. A Go package that
fails outside its test cases is the case `(package)` of that package.

The caller selects reports of the same commit and workload. The tool does not
check the commit, and it does not check that a provider found every test.

Exit status: 0 equivalent, 1 different, 2 rejected input, 3 incomparable (the
sides have no case in common).
"""
import argparse
import importlib.util
import json
from pathlib import Path
import sys
import xml.etree.ElementTree as ET

EQUIVALENT, DIFFERENT, REJECTED, INCOMPARABLE = 0, 1, 2, 3
PACKAGE = '(package)'


def helper(name):
    spec = importlib.util.spec_from_file_location(name.replace('-', '_'), Path(__file__).resolve().parents[1] / 'runtime' / (name + '.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


T = helper('ci-test-results')


def report_format(path):
    """Return `junit`, `go-json` or `circleci-tests` for one report file."""
    text = path.read_text(encoding='utf-8').lstrip()
    if path.suffix == '.xml' or text.startswith('<'):
        return 'junit'
    try:
        first = json.loads(text.split('\n', 1)[0])
    except ValueError:
        return 'circleci-tests'  # One JSON document that spans several lines.
    return 'go-json' if isinstance(first, dict) and 'Action' in first else 'circleci-tests'


def circleci_cases(path, source):
    """Decode one complete response of CircleCI's job test-metadata API.

    The API returns a large job in pages. Before the call, the caller joins the
    `items` of every page into one response, or into one bare list.
    """
    data = T.read_json(path)
    if isinstance(data, dict):
        if data.get('next_page_token'):
            raise ValueError(f'CircleCI tests response is one page of several; join every page first: {path}')
        data = data.get('items')
    if not isinstance(data, list):
        raise ValueError(f'CircleCI tests response has no `items` list: {path}')
    cases = []
    for row in data:
        T.json_object(row, 'CircleCI test')
        outcome = T.RESULTS.get(row.get('result'))
        if outcome is None:
            raise ValueError(f'unknown CircleCI test result {row.get("result")!r}: {path}')
        cases.append(T.make_case(row.get('classname'), row.get('name'), outcome, row.get('run_time'),
                                 T.skip_reason(row.get('message')) if outcome == 'skip' else None, source))
    return cases


def load(paths):
    """Return {(suite, name): (outcome, skip reason)} for the reports of one side.

    Raises ValueError if a report has no case, or if two reports of the side
    contain the same case.
    """
    result = {}
    for path in paths:
        source, failed = {'feature': None, 'id': str(path)}, []
        kind = report_format(path)
        if kind == 'junit':
            cases = T.junit_cases(path, source, {})
        elif kind == 'go-json':
            cases, failed = T.go_cases(path, source, {})
        else:
            cases = circleci_cases(path, source)
        if not cases and not failed:
            raise ValueError(f'report contains no test case: {path}')
        # Test-level shards run the same package, so a package can fail in each.
        result.update(((suite, PACKAGE), ('fail', None)) for suite in failed)
        for case in cases:
            key = case['suite'], case['name']
            if key in result:
                raise ValueError(f'one side reports a case twice: {key[0]} {key[1]} ({path})')
            result[key] = case['outcome'], case['skip_reason']
    return result


def compare(baseline, candidate):
    """Return the sorted case keys that are missing, extra and different in the candidate."""
    shared = baseline.keys() & candidate.keys()
    return {'missing': sorted(baseline.keys() - shared), 'extra': sorted(candidate.keys() - shared),
            'different': sorted(key for key in shared if baseline[key] != candidate[key])}


def verdict(value):
    outcome, reason = value
    return outcome if reason is None else f'{outcome} ({reason})'


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    for side in ('baseline', 'candidate'):
        parser.add_argument('--' + side, type=Path, nargs='+', required=True, metavar='REPORT',
                            help=f'report files of the {side} provider, one per shard')
    args = parser.parse_args(argv)
    try:
        baseline, candidate = load(args.baseline), load(args.candidate)
    except (OSError, ValueError, ET.ParseError) as error:
        print(f'CI comparison rejected input: {error}', file=sys.stderr)
        return REJECTED
    sizes = f'{len(baseline)} baseline cases, {len(candidate)} candidate cases'
    if not baseline.keys() & candidate.keys():
        print(f'CI comparison: incomparable ({sizes}, none in common)')
        return INCOMPARABLE
    report = compare(baseline, candidate)
    print(f'CI comparison: {"different" if any(report.values()) else "equivalent"} ({sizes})')
    for suite, name in report['missing']:
        print(f'missing from candidate: {suite} {name}: {verdict(baseline[suite, name])}')
    for suite, name in report['extra']:
        print(f'extra in candidate: {suite} {name}: {verdict(candidate[suite, name])}')
    for suite, name in report['different']:
        print(f'different: {suite} {name}: baseline {verdict(baseline[suite, name])}, candidate {verdict(candidate[suite, name])}')
    return DIFFERENT if any(report.values()) else EQUIVALENT


if __name__ == '__main__':
    sys.exit(main())
