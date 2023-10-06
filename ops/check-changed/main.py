import logging.config
import os
import re
import subprocess
import sys
import json

from github import Github

REBUILD_ALL_PATTERNS = [
    r'^\.circleci/\.*',
    r'^\.github/\.*',
    r'^package\.json',
    r'ops/check-changed/.*',
    r'^go\.mod',
    r'^go\.sum',
    r'ops/check-changed/.*'
]
with open("../../nx.json") as file:
    nx_json_data = json.load(file)
REBUILD_ALL_PATTERNS += nx_json_data["implicitDependencies"].keys()

WHITELISTED_BRANCHES = {
    'master',
    'develop'
}

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


def main():
    patterns = sys.argv[1].split(',')
    patterns = patterns + REBUILD_ALL_PATTERNS

    fp = os.path.realpath(__file__)
    monorepo_path = os.path.realpath(os.path.join(fp, '..', '..'))

    log.info('Discovered monorepo path: %s', monorepo_path)
    current_branch = git_cmd('rev-parse --abbrev-ref HEAD', monorepo_path)
    log.info('Current branch: %s', current_branch)

    if current_branch in WHITELISTED_BRANCHES:
        log.info('Current branch %s is whitelisted, triggering build', current_branch)
        exit_build()

    pr_urls = os.getenv('CIRCLE_PULL_REQUESTS', None)
    pr_urls = pr_urls.split(',') if pr_urls else []
    if len(pr_urls) == 0:
        log.info('Not a PR build, triggering build')
        exit_build()
    if len(pr_urls) > 1:
        log.warning('Multiple PR URLs found, choosing the first one. PRs found:')
        for url in pr_urls:
            log.warning(url)

    gh_token = os.getenv('GITHUB_ACCESS_TOKEN')
    if gh_token is None:
        log.info('No GitHub access token found - likely a fork. Triggering build')
        exit_build()

    g = Github(gh_token)
    try:
        g.get_user()
        repo = g.get_repo(os.getenv('CIRCLE_PROJECT_USERNAME') + '/' + os.getenv('CIRCLE_PROJECT_REPONAME'))
    except Exception:
        log.exception('Failed to get repo from GitHub')
        exit_build()

    pr = repo.get_pull(int(pr_urls[0].split('/')[-1]))
    log.info('Found PR: %s', pr.url)

    base_sha = pr.base.sha
    head_sha = pr.head.sha

    diffs = git_cmd('diff --name-only {}...{}'.format(base_sha, head_sha), monorepo_path).split('\n')
    log.info('Found diff. Checking for matches...')
    for diff in diffs:
        if match_path(diff, patterns):
            log.info('Match found, triggering build')
            exit_build()
        else:
            log.info('❌ no match found on %s', diff)

    log.info('No matches found, skipping build')
    exit_nobuild()


def git_cmd(cmd, cwd):
    return subprocess.check_output(['git'] + cmd.split(' '), cwd=cwd).decode('utf-8').strip()


def match_path(path, patterns):
    for pattern in patterns:
        if re.search(pattern, path):
            log.info('✅ match found on %s: %s', path, pattern)
            return True
    return False


def exit_build():
    sys.exit(0)


def exit_nobuild():
    subprocess.check_call(['circleci', 'step', 'halt'])
    sys.exit(0)


if __name__ == '__main__':
    try:
        main()
    except Exception:
        log.exception('Unhandled exception, triggering build')
        exit_build()
