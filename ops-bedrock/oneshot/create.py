import argparse
import logging
import os
import subprocess
from logging.config import dictConfig

log_level = os.getenv('LOG_LEVEL')

log_config = {
    'version': 1,
    'loggers': {
        '': {
            'handlers': ['console'],
            'level': log_level if log_level is not None else 'INFO'
        },
    },
    'handlers': {
        'console': {
            'formatter': 'stderr',
            'class': 'logging.StreamHandler',
            'stream': 'ext://sys.stdout'
        }
    },
    'formatters': {
        'stderr': {
            'format': '[%(levelname)s|%(asctime)s] %(message)s',
            'datefmt': '%m-%d-%Y %I:%M:%S'
        }
    },
}

dictConfig(log_config)

lgr = logging.getLogger()

parser = argparse.ArgumentParser(description='Creates a Bedrock oneshot container.')
parser.add_argument('--op-node-image', help='op-node image to use inside the container.', required=True)
parser.add_argument('--op-geth-image', help='op-geth image to use inside the container.', required=True)
parser.add_argument('--network-name', help='Network name.', required=True)
parser.add_argument('--tag', help='Docker tag.', required=True)


def main():
    args = parser.parse_args()
    full_tag = f'us-central1-docker.pkg.dev/bedrock-goerli-development/images/bedrock-oneshot:{args.tag}'

    build_args = (
        ('op_node_image', args.op_node_image),
        ('op_geth_image', args.op_geth_image),
        ('network_name', args.network_name)
    )
    cmd_args = ['docker', 'build', '-f', 'Dockerfile.oneshot', '-t', full_tag]
    for arg in build_args:
        cmd_args.append('--build-arg')
        cmd_args.append(f'{arg[0]}={arg[1]}')
    cmd_args.append(os.getcwd())
    run_command(cmd_args)


def run_command(args, check=True, shell=False, cwd=None, env=None):
    env = env if env else {}
    return subprocess.run(
        args,
        check=check,
        shell=shell,
        env={
            **os.environ,
            **env
        },
        cwd=cwd
    )


if __name__ == '__main__':
    main()
