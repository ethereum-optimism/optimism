#!/usr/bin/env python3
"""Produce a bounded native Go report while retaining the original JSON unchanged."""
import argparse
from collections import deque
import hashlib
import json
from pathlib import Path

LIMIT = 64 * 1024


def events(path):
    with path.open() as stream:
        for line in stream:
            event = json.loads(line)
            if not isinstance(event, dict) or not isinstance(event.get('Action'), str):
                raise ValueError('Invalid original Go event')
            yield event


def key(event):
    return event.get('Package'), event.get('Test')


def project(source, destination):
    # Retain failure and skip messages from every attempt, including recovered failures.
    interesting = {key(e) for e in events(source) if e['Action'] in ('fail', 'skip')}
    buffered, sizes, truncated = {}, {}, set()
    original_count = reported_count = 0
    destination.parent.mkdir(parents=True, exist_ok=True)
    with destination.open('w') as output:
        def emit(event):
            nonlocal reported_count
            output.write(json.dumps(event, separators=(',', ':')) + '\n')
            reported_count += 1

        def flush(identity, terminal):
            if identity in truncated:
                emit({**terminal, 'Action': 'output', 'Output': '[Native report output shortened; complete original events are in the go-json artifact.]\n'})
            for event in buffered.pop(identity, []):
                emit(event)
            sizes.pop(identity, None)
            truncated.discard(identity)

        for event in events(source):
            original_count += 1
            identity = key(event)
            if event['Action'] == 'output':
                if identity not in interesting:
                    continue
                queue = buffered.setdefault(identity, deque())
                size = len(event.get('Output', '').encode())
                if size > LIMIT:
                    event = {**event, 'Output': event['Output'].encode()[-LIMIT:].decode(errors='replace')}
                    size = len(event['Output'].encode())
                    truncated.add(identity)
                queue.append(event)
                sizes[identity] = sizes.get(identity, 0) + size
                while sizes[identity] > LIMIT and queue:
                    sizes[identity] -= len(queue.popleft().get('Output', '').encode())
                    truncated.add(identity)
                continue
            if event['Action'] in ('pass', 'fail', 'skip'):
                flush(identity, event)
            emit(event)
        # Partial/canceled reports stay partial: never synthesize terminal verdicts.
        for identity, queue in list(buffered.items()):
            if queue:
                flush(identity, queue[-1])
    with source.open('rb') as stream:
        digest = hashlib.file_digest(stream, 'sha256').hexdigest()
    destination.with_suffix('.metadata.json').write_text(json.dumps({
        'original_sha256': digest, 'original_events': original_count,
        'native_events': reported_count, 'output_limit_bytes_per_attempt': LIMIT,
        'rule': 'all non-output events; bounded output for every failed or skipped case/package'
    }, indent=2) + '\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('destination', type=Path)
    args = parser.parse_args()
    project(args.source, args.destination)
