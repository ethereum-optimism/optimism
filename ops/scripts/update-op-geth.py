#!/usr/bin/env python3


import urllib3
import json
import subprocess
import os


GETH_VERSION='v1.10.26'


def main():
	for project in ('op-service', 'op-node', 'op-proposer', 'op-batcher', 'op-bindings', 'op-chain-ops', 'op-e2e'):
		print(f'Updating {project}...')
		update_mod(project)

	# pull the replacer from one of the modules above, since
	# go work edit -replace does not resolve the branch name
	# into a pseudo-version.
	pv = None
	with open('./op-service/go.mod') as f:
		for line in f:
			if line.startswith(f'replace github.com/ethereum/go-ethereum {GETH_VERSION}'):
				splits = line.split(' => ')
				pv = splits[1].strip()
				pv = pv.split(' ')[1]
				break

	if pv is None:
		raise Exception('Pseudo version not found.')

	print('Updating go.work...')
	subprocess.run([
		'go',
		'work',
		'edit',
		'-replace',
		f'github.com/ethereum/go-ethereum@{GETH_VERSION}=github.com/ethereum-optimism/op-geth@{pv}'
	], cwd=os.path.join(project), check=True)


def update_mod(project):
	print('Replacing...')
	subprocess.run([
		'go',
		'mod',
		'edit',
		'-replace',
		f'github.com/ethereum/go-ethereum@{GETH_VERSION}=github.com/ethereum-optimism/op-geth@optimism-history'
	], cwd=os.path.join(project), check=True)
	print('Tidying...')
	subprocess.run([
		'go',
		'mod',
		'tidy'
	], cwd=os.path.join(project), check=True)


if __name__ == '__main__':
	main()
