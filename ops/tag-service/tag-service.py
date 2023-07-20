#!/usr/bin/env python3
import logging.config
import os
import re
import subprocess
import sys

import click
import semver

# Minimum version numbers for packages migrating from legacy versioning.
MIN_VERSIONS = {
    'op-node': '0.10.14',
    'op-batcher': '0.10.14',
    'op-proposer': '0.10.14',
    'proxyd': '3.16.0',
    'indexer': '0.5.0',
    'fault-detector': '0.6.3',
    'ci-builder': '0.6.0'
}

VALID_BUMPS = ('major', 'minor', 'patch', 'prerelease', 'finalize-prerelease')

MESSAGE_TEMPLATE = '[tag-service-release] Tag {service} at {version}'

LOGGING_CONFIG = {
    'version': 1,
    'disable_existing_loggers': True,
    'formatters': {
        'standard': {
            'format': '%(asctime)s [%(levelname)s]: %(message)s'
        },
    },
    'handlers': {
        'default': {
            'level': 'INFO',
            'formatter': 'standard',
            'class': 'logging.StreamHandler',
            'stream': 'ext://sys.stderr'
        },
    },
    'loggers': {
        '': {
            'handlers': ['default'],
            'level': 'INFO',
            'propagate': False
        },
    }
}

logging.config.dictConfig(LOGGING_CONFIG)
log = logging.getLogger(__name__)


@click.command()
@click.option('--bump', required=True, type=click.Choice(VALID_BUMPS))
@click.option('--service', required=True, type=click.Choice(list(MIN_VERSIONS.keys())))
@click.option('--pre-release/--no-pre-release', default=False)
def tag_version(bump, service, pre_release):
    tags = subprocess.run(['git', 'tag', '--list'], capture_output=True, check=True) \
        .stdout.decode('utf-8').splitlines()

    # Filter out tags that don't match the service name, and tags
    # for prerelease versions.
    version_pattern = f'^{service}/v\\d+\\.\\d+\\.\\d+(-rc\\.\\d+)?$'
    svc_versions = [t.replace(f'{service}/v', '') for t in tags if re.match(version_pattern, t)]
    svc_versions = sorted(svc_versions, key=lambda v: semver.Version.parse(v), reverse=True)

    if pre_release and bump == 'prerelease':
        raise Exception('Cannot use --bump=prerelease with --pre-release')

    if pre_release and bump == 'finalize-prerelease':
        raise Exception('Cannot use --bump=finalize-prerelease with --pre-release')

    if len(svc_versions) == 0:
        latest_version = MIN_VERSIONS[service]
    else:
        latest_version = svc_versions[0]

    latest_version = semver.Version.parse(latest_version)

    log.info(f'Latest version: v{latest_version}')

    if bump == 'major':
        bumped = latest_version.bump_major()
    elif bump == 'minor':
        bumped = latest_version.bump_minor()
    elif bump == 'patch':
        bumped = latest_version.bump_patch()
    elif bump == 'prerelease':
        bumped = latest_version.bump_prerelease()
    elif bump == 'finalize-prerelease':
        bumped = latest_version.finalize_version()
    else:
        raise Exception('Invalid bump type: {}'.format(bump))

    if pre_release:
        bumped = bumped.bump_prerelease()

    new_version = 'v' + str(bumped)
    new_tag = f'{service}/{new_version}'

    log.info(f'Bumped version: {new_version}')

    log.info('Configuring git')
    # The below env vars are set by GHA.
    gh_actor = os.environ['GITHUB_ACTOR']
    gh_token = os.environ['INPUT_GITHUB_TOKEN']
    gh_repo = os.environ['GITHUB_REPOSITORY']
    origin_url = f'https://{gh_actor}:${gh_token}@github.com/{gh_repo}.git'
    subprocess.run(['git', 'config', 'user.name', gh_actor], check=True)
    subprocess.run(['git', 'config', 'user.email', f'{gh_actor}@users.noreply.github.com'], check=True)
    subprocess.run(['git', 'remote', 'set-url', 'origin', origin_url], check=True)

    log.info(f'Creating tag: {new_tag}')
    subprocess.run([
        'git',
        'tag',
        '-a',
        new_tag,
        '-m',
        MESSAGE_TEMPLATE.format(service=service, version=new_version)
    ], check=True)

    log.info('Pushing tag to origin')
    subprocess.run(['git', 'push', 'origin', new_tag], check=True)


if __name__ == '__main__':
    tag_version()
