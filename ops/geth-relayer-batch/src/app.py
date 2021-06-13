import os
import logging
import traceback
import time
import json
import requests
from flask import Flask
from flask import request

from datetime import datetime
import subprocess
from env import MyEnv


app = Flask(__name__)

SEARCH_DOMAIN = os.environ.get('SEARCH_DOMAIN')

DIR = '/l2-node/l2'

print(os.environ)


@app.route('/')
def health():
    return 'Ok'
    
@app.route('/v1/chain/update',methods=['POST'])
def update_chain():
    mount_path = request.args.get("mount_path")
    efs_id = request.args.get("efs_id")
    access_point_id = request.args.get("access_point_id")
    kill_pids()
    the_path = mount_path or '/metis'
    logging.warn(f'update_chain mount_path:{mount_path},efs_id:{efs_id},access_point_id:{access_point_id}')
    _try_cmd(['umount', the_path])
    _try_cmd(['umount', '/root/.ethereum'])
    output = None
    if efs_id is not None and access_point_id is not None:
        output=_try_cmd(['mount', '-t', 'efs', '-o', f'tls,accesspoint={access_point_id}',f'{efs_id}:', the_path])
        logging.warn(f'mount to ap: {efs_id},{access_point_id},{output}')
        _try_cmd(['mkdir', '-p', f'{the_path}/.ethereum'])
        _try_cmd(['mount','--bind',f'{the_path}/.ethereum','/root/.ethereum'])
        logging.warn(f"mount to root ethereum: mount --bind {the_path}/.ethereum /root/.ethereum")
    if output is None or len(output)<=0:
        body=request.get_data(as_text=True)
        output=_update_chain(json.loads(body))
    response = {
        'data': output.decode('utf-8')
    }
    return response

def kill_pids():
    output = _try_cmd_string("/app/process_kill.sh")
    logging.warn(output)

def _try_cmd(cmds):
    try:
        return subprocess.check_output(cmds)
    except Exception as e:
        logging.warn(f'exce cmd in update chain error {cmds}')
        return ""
        
def _try_cmd_string(cmd):
    try:
        return subprocess.check_output(cmd, shell=True)
    except Exception as e:
        logging.warn(f'exce cmd string in update chain error {cmd}')
        return "" 
        
def _update_chain(body):
    logging.warn(f'update_chain to file:{body}')
    if body is not None:
        myEnv = MyEnv('')
        myEnv.SetEnvFile("/app/env.sh")
        myEnv.envs=body
        myEnv.Save()
    logging.warn(f'update_chain to file2:{myEnv.envs}')

    output = _try_cmd(['cat','/app/env.sh'])
    logging.warn(output)
    
    output = _try_cmd([f'/app/restart.sh','/app/env.sh'])
    logging.warn(output)
    return output

@app.route('/v1/chain/stop',methods=['POST'])
def stop_chain():
    logging.warn('stop_chain...')
    kill_pids()
    response = {
        'data': "success"
    }
    logging.warn(response)
    return response 
    
@app.route('/v1/shell/exec',methods=['POST'])
def exec_shell():
    logging.warn('appending to file')
    bodys=request.get_data()
    logging.warn(bodys)
    body = json.loads(bodys)
    cmd = body['cmd']
    tokens=cmd.split(' ')
    logging.warn(tokens)
    ls_output = subprocess.check_output(tokens)
    logging.warn(ls_output)
    response = {
        'data': ls_output.decode('utf-8')
    }
    logging.warn(response)
    return response

@app.route('/v1/metis/l2/geth')
def metis_l2_geth():
    logging.warn('mount to ap')
    efs_id = request.args.get("efs_id")
    access_point_id = request.args.get("access_point_id")
    mount_path = request.args.get("mount_path")
    the_path = mount_path or '/metis'
    ls_output=subprocess.check_output(['mount', '-t', 'efs', '-o', f'tls,accesspoint={access_point_id}',f'{efs_id}:', the_path])
    logging.warn(ls_output)
    response = {
        'data': ls_output.decode('utf-8')
    }
    logging.warn(response)
    return response


@app.route('/v1/batch/submitter')
def talk_to_batch_submitter():
    host = f'http://batch_submitter_{SEARCH_DOMAIN}:4567' if SEARCH_DOMAIN else 'http://batch_submitter:4567'
    print(f'Calling batch submitter host: {host}')
    response = requests.get(f'{host}/v1/batch_submitter')
    print(response.content)

    return {
        'data': response.json()
    }, response.status_code


@app.route('/v1/dtl')
def talk_to_dtl():
    host = f'http://data_transport_layer_{SEARCH_DOMAIN}:7878' if SEARCH_DOMAIN else 'http://data_transport_layer:7878'
    print(f'Calling batch submitter host: {host}')
    response = requests.get(f'{host}/v1/dtl')
    print(response.content)

    return {
        'data': response.json()
    }, response.status_code


if __name__ == "__main__" :
    _update_chain({'a':'1','b':'2', 'l2v':'10', 'l2r':'http://ip-172-31-12-82.us-east-2.compute.internal:8089/state-dump.latest.json'})