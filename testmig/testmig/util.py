import os
import socket
import subprocess
import time

import requests
from tqdm import tqdm


def download_with_progress(url, f):
    res = requests.get(url, stream=True, allow_redirects=True)
    total_len = res.headers.get('content-length')
    if total_len is None:
        raise Exception('no total length, bailing out')

    total_len = int(total_len)

    with tqdm(total=total_len, unit='b', unit_scale=True) as pbar:
        for data in res.iter_content(chunk_size=4096):
            f.write(data)
            pbar.update(len(data))


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


def wait_up(port, retries=10, wait_secs=1):
    for i in range(0, retries):
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        try:
            s.connect(('127.0.0.1', int(port)))
            s.shutdown(2)
            return True
        except Exception:
            time.sleep(wait_secs)

    raise Exception(f'Timed out waiting for port {port}.')
