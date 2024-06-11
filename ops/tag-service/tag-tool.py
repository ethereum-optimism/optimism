import argparse
import subprocess
import re
import semver

SERVICES  = [
    'ci-builder',
    'ci-builder-rust',
    'chain-mon',
    'op-node',
    'op-batcher',
    'op-challenger',
    'op-dispute-mon',
    'op-proposer',
    'da-server',
    'proxyd',
    'op-heartbeat',
    'op-contracts',
    'test',
    'op-stack', # special case for tagging op-node, op-batcher, and op-proposer together
    'op-conductor',
]
VERSION_PATTERN = '^{service}/v\\d+\\.\\d+\\.\\d+(-rc\\.\\d+)?$'
GIT_TAG_COMMAND = 'git tag -a {tag} -m "{message}"'
GIT_PUSH_COMMAND = 'git push origin {tag}'

def new_tag(service, version, bump):
    if bump == 'major':
        bumped = version.bump_major()
    elif bump == 'minor':
        bumped = version.bump_minor()
    elif bump == 'patch':
        bumped = version.bump_patch()
    elif bump == 'prerelease':
        bumped = version.bump_prerelease()
    elif bump == 'finalize-prerelease':
        bumped = version.finalize_version()
    else:
        raise Exception('Invalid bump type: {}'.format(bump))
    return f'{service}/v{bumped}'

def latest_version(service):
    # Get the list of tags from the git repository.
    tags = subprocess.run(['git', 'tag', '--list', f'{service}/v*'], capture_output=True, check=True) \
        .stdout.decode('utf-8').splitlines()
    # Filter out tags that don't match the service name, and tags for prerelease versions.
    svc_versions = sorted([t.replace(f'{service}/v', '') for t in tags])
    if len(svc_versions) == 0:
        raise Exception(f'No tags found for service: {service}')
    return svc_versions[-1]

def latest_among_services(services):
    latest = '0.0.0'
    for service in services:
        candidate = latest_version(service)
        if semver.compare(candidate, latest) > 0:
            latest = candidate
    return latest

def main():
    parser = argparse.ArgumentParser(description='Create a new git tag for a service')
    parser.add_argument('--service', type=str, help='The name of the Service')
    parser.add_argument('--bump', type=str, help='The type of bump to apply to the version number')
    parser.add_argument('--message', type=str, help='Message to include in git tag', default='[tag-tool-release]')
    args = parser.parse_args()

    service = args.service

    if service == 'op-stack':
      latest = latest_among_services(['op-node', 'op-batcher', 'op-proposer'])
    else:
      latest = latest_version(service)

    bumped = new_tag(service, semver.VersionInfo.parse(latest), args.bump)

    print(f'latest tag: {latest}')
    print(f'new tag: {bumped}')
    print('run the following commands to create the new tag:\n')
    # special case for tagging op-node, op-batcher, and op-proposer together. All three would share the same semver
    if args.service == 'op-stack':
        print(GIT_TAG_COMMAND.format(tag=bumped.replace('op-stack', 'op-node'), message=args.message))
        print(GIT_PUSH_COMMAND.format(tag=bumped.replace('op-stack', 'op-node')))
        print(GIT_TAG_COMMAND.format(tag=bumped.replace('op-stack', 'op-batcher'), message=args.message))
        print(GIT_PUSH_COMMAND.format(tag=bumped.replace('op-stack', 'op-batcher')))
        print(GIT_TAG_COMMAND.format(tag=bumped.replace('op-stack', 'op-proposer'), message=args.message))
        print(GIT_PUSH_COMMAND.format(tag=bumped.replace('op-stack', 'op-proposer')))
    else:
        print(GIT_TAG_COMMAND.format(tag=bumped, message=args.message))
        print(GIT_PUSH_COMMAND.format(tag=bumped))

if __name__ == "__main__":
    main()

