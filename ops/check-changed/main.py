import http.client
import logging.config
import json
import os
import re
import subprocess
import sys

REBUILD_ALL_PATTERNS = [
    r'^\.circleci/\.*',
    r'^\.github/\.*',
    r'^package\.json',
    r'ops/check-changed/.*',
    r'^mise.toml',
]

GO_PATTERNS = [
    r'^go\.mod',
    r'^go\.sum',
]

WHITELISTED_BRANCHES = {
    'master',
    'develop'
}

GITHUB_REPO_PART_PATTERN = re.compile(r'^[A-Za-z0-9_.-]+$')

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
    patterns = sys.argv[1].split(',') + REBUILD_ALL_PATTERNS
    no_go_deps = os.getenv('CHECK_CHANGED_NO_GO_DEPS')
    if no_go_deps is None:
        patterns = patterns + GO_PATTERNS

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

    # If we successfully extracted a PR number and did not find PRs from CIRCLE_PULL_REQUESTS,
    # we are on a merge queue branch and can reconstruct the original PR URL from the PR number.
    pr_number = extract_pr_number(current_branch)
    if not pr_urls and pr_number is not None:
        log.info('No PR URLs found but extracted branch number, constructing PR URL')
        base_url = "https://github.com/ethereum-optimism/optimism/pull/"
        pr_urls = [base_url + pr_number]

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

    try:
        pr_number = int(pr_urls[0].split('/')[-1])
        pr = fetch_pull_request(pr_number, gh_token)
        pr_url = pr['url']
        base_sha = pr['base']['sha']
        head_sha = pr['head']['sha']
    except Exception:
        log.exception('Failed to get PR metadata from GitHub')
        exit_build()

    log.info('Found PR: %s', pr_url)

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


def fetch_pull_request(pr_number, token):
    owner = os.getenv('CIRCLE_PROJECT_USERNAME')
    repo = os.getenv('CIRCLE_PROJECT_REPONAME')
    if not owner or not repo:
        raise RuntimeError('missing CircleCI project environment')
    if not GITHUB_REPO_PART_PATTERN.fullmatch(owner) or not GITHUB_REPO_PART_PATTERN.fullmatch(repo):
        raise RuntimeError('invalid CircleCI project environment')

    conn = http.client.HTTPSConnection('api.github.com', timeout=30)
    try:
        conn.request(
            'GET',
            '/repos/{}/{}/pulls/{}'.format(owner, repo, pr_number),
            headers={
                'Accept': 'application/vnd.github+json',
                'Authorization': 'Bearer {}'.format(token),
                'X-GitHub-Api-Version': '2022-11-28',
                'User-Agent': 'optimism-check-changed',
            },
        )
        response = conn.getresponse()
        body = response.read()
    finally:
        conn.close()

    if response.status >= 400:
        raise RuntimeError('GitHub API returned status {}'.format(response.status))
    return json.loads(body.decode('utf-8'))


def match_path(path, patterns):
    for pattern in patterns:
        if re.search(pattern, path):
            log.info('✅ match found on %s: %s', path, pattern)
            return True
    return False

def extract_pr_number(branch_name):
    # Merge queue branches are named: gh-readonly-queue/{base_branch}/pr-{number}-{sha}
    match = re.search(r'/pr-(\d+)-', branch_name)
    if match:
        pr_number = match.group(1)
        log.info('Extracted PR number: %s', pr_number)
        return pr_number
    else:
        return None

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
