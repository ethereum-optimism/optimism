import subprocess
import shlex
import json
import os, sys
import hashlib
import base64
import requests
from pprint import pprint
from urllib.parse import quote
import logging
import re


logging.basicConfig(format='%(asctime)s - %(levelname)s - %(message)s',stream=sys.stdout,level=logging.INFO)

def send_request(url, method="GET", headers=None, data=None, access_token=None):

    """
    Send an HTTP request and return the response.

    :param url: URL to send the request to
    :param method: HTTP method (GET or POST)
    :param headers: Dictionary of headers to send with the request
    :param data: Data to send with the request. For GET requests, these will be converted to URL parameters; for POST requests, this will be the request body.
    :return: A dictionary with the status code, response data, and any error message.
    """
    try:
        if not url.startswith("https://"):
            raise ValueError("URL must start with https:// for security reasons")
        if not access_token:
            try:
                access_token=print_access_token()
            except Exception as e:
                return {
                    "status_code": None,
                    "data": None,
                    "error": str(e)
                }
        # Ensure headers and data are not None
        if headers is None:
            headers = {}
        if data is None:
            data = {}
        headers['Content-Type']="application/json"
        headers['Authorization']=f"Bearer {access_token}"
        # Choose the request method
        if method.upper() == "GET":
            response = requests.get(url, headers=headers)
        elif method.upper() == "POST":
            response = requests.post(url, headers=headers, data=data)
        else:
            return {"error": "Unsupported method specified"}

        # Check if the response was successful
        response.raise_for_status()

        # Return the response status code and content
        try:
            response_data = response.json()
        except ValueError:
            response_data = response.text
        return {
            "status_code": response.status_code,
            "data": response_data,  # or response.text if expecting text
            "error": None
        }
    except requests.RequestException as e:
        # Handle any errors that occur during the request
        return {
            "status_code": None,
            "data": None,
            "error": str(e)
        }


def execute_shell_command(cmd,timeout=5):
    """
    Executes a shell command and returns the output.

    Parameters:
    - cmd (str): The command to execute.

    Returns:
    - A tuple containing the command's standard output and standard error.
    """
    if not isinstance(cmd, str) or ';' in cmd or '&&' in cmd or '||' in cmd:
        raise ValueError("Invalid command. Command must be a safe string.")
    try:
        # Use shlex.split to handle command parsing.
        process = subprocess.run(shlex.split(cmd), check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True,timeout=timeout)
        stdout = process.stdout
        return stdout, None  # Return stdout and None for stderr in case of success.
    except subprocess.CalledProcessError as e:
        return e.stdout, e.stderr  # Return both stdout and stderr in case of error.
    except subprocess.TimeoutExpired as e:
        return e.stdout, e.stderr

def print_access_token():
    cmd = "gcloud auth print-access-token"
    stdout, stderr = execute_shell_command(cmd,timeout=2)

    if stderr is None:
        return stdout.strip()
    else:
        raise Exception("Failed to print access token. Please ensure you are properly authenticated and try again.")

def get_current_user_email():
    url = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"
    response = send_request(url=url, method="GET", headers=None, data=None)
    if response['data'] and  "email" in response['data']:
        return response['data']['email']
    return None

def generate_image_description_payload(image_info):
    payload = {
        "critical": {
            "identity": {
                "docker-reference": image_info['path']
            },
            "image": {
                "docker-manifest-digest": image_info['digest']
            },
            "type": "Google cloud binauthz container signature"
        }
    }
    str_payload=json.dumps(payload, indent=0)
    return str_payload.encode('utf-8')

def generate_attestation_payload(image_info,attestor_info,serialized_payload,payload_signature):
    payload = {
    "resourceUri": image_info['fully_qualified_digest'],
    "note_name": attestor_info['note_reference'],
    "attestation": {
        "serialized_payload": serialized_payload,
        "signatures": [
            {
                "public_key_id": attestor_info['key_id'],
                "signature": payload_signature
            }]
        }
    }
    str_payload=json.dumps(payload,indent=None)
    return str_payload.encode('utf-8')

def get_base64_encoded_hash(payload):
    sha512_hash = hashlib.sha512(payload).digest()
    base64_encoded_hash = base64.b64encode(sha512_hash).decode('utf-8')
    return base64_encoded_hash

def generate_image_payload_signature(base64_encoded_hash,key_info,attestor_info):

    # Validate or sanitize key_info values before using them in URL construction
    for key, value in key_info.items():
        if not re.match(r'^[\w-]+$', value):
            raise ValueError(f"Invalid value for {key}: {value}")
    url =f"https://cloudkms.googleapis.com/v1/projects/{key_info['project_id']}/locations/{key_info['location']}/keyRings/{key_info['keyring']}/cryptoKeys/{key_info['key']}/cryptoKeyVersions/{key_info['version']}:asymmetricSign?alt=json"
    headers = {
        "x-goog-user-project": f"{attestor_info['project_id']}"
    }
    data=json.dumps({"digest":{"sha512":base64_encoded_hash}})
    response = send_request(url=url, method="POST", headers=headers, data=data)
    if response['error'] or not response['data'] or  "signature" not in response['data'] :
        return None
    return response['data']['signature']

def retrieve_attestor_info(attestor_info,current_user_email):
    if not attestor_info['project_id']:
        #assuming user email project as attestor project id
        project_id=current_user_email.split("@")[1].split(".")[0]
        attestor_info['project_id']=project_id

    url =f"https://binaryauthorization.googleapis.com/v1/projects/{attestor_info['project_id']}/attestors/{attestor_info['attestor']}"
    print(url)
    headers = {
        "x-goog-user-project": f"{attestor_info['project_id']}"
    }
    response = send_request(url=url, method="GET", headers=headers, data=None)
    if response['data']:
        return response['data']
    return None



def retrieve_image_info(image_input):
    image_info={
        "path":None,
        "digest":None,
        "tag":None,
        "fully_qualified_digest":None
    }

    pattern = r'^(?P<path>[\w\.\-\/]+)(?::(?P<tag>[\w\.\-]+))?(?:@sha256:(?P<digest>[a-fA-F0-9]{64}))?$'
    match = re.match(pattern, image_input)
    if match:
        image_info.update(match.groupdict())

    if not image_info['digest']:
        cmd=f"gcloud container images describe {image_info['path']}:{image_info['tag']} --format=json"
        stdout, stderr = execute_shell_command(cmd=cmd,timeout=2)
        try:
            json_obj=json.loads(stdout.strip())['image_summary']
            image_info['fully_qualified_digest']=json_obj['fully_qualified_digest']
            image_info['digest']=json_obj['digest']
        except:
            return None
    else:
        image_info['fully_qualified_digest']=f"{json_obj['path']}@{json_obj['digest']}"


    return image_info

def upload_attestation(attestor_info,attestation_payload):

    url =f"https://containeranalysis.googleapis.com/v1/projects/{attestor_info['project_id']}/occurrences/"
    headers = {
        "x-goog-user-project": f"{attestor_info['project_id']}"
    }
    data=attestation_payload
    response = send_request(url=url, method="POST", headers=headers, data=data)
    if response['data']:
        return response['data']
    elif response['error'] and "Conflict for url" in response['error']:
        logging.critical("Attestation not uploaded: Conflict for the attestation url, are you trying to upload the same attestation twice?")
    return None

def get_env():
    env_variables = {
        "IMAGE_PATH": {"default":None,"info":"us-docker.pkg.dev/<projectid>/<repositoryname>/[image_name@sha256<image_digest> or image_name:<image_tag>]","required":True,"value":None},
        "ATTESTOR_PROJECT_NAME": {"default":None,"info":"<projectid>","required":False,"value":None},
        "ATTESTOR_NAME": {"default":"tag-attestor","info":"<tag-attestor-name>","required":False,"value":None},
        "ATTESTOR_KEY_ID": {"default":None,"info":"'//cloudkms.googleapis.com/v1/projects/<projectid>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key>/cryptoKeyVersions/1","required":False,"value":None},
        "SIGNER_LOGGING_LEVEL": {"default":"INFO","info":"CRITICAL|FATAL|ERROR|WARNING|INFO|DEBUG","required":False,"value":None}
    }

    config = {}

    # Read environment variables and populate the config dictionary
    for var_name, env_var in env_variables.items():
        env_var['value']=os.environ.get(var_name)
        if env_var['required'] and not env_var['value']:
            logging.critical(f"Error: The following environment variables are missing: {var_name}. A value in the format of {env_var['info']} should be provided")
            sys.exit(1)
        if not env_var['required'] and not env_var['value']:
            logging.info(f"{var_name} has not being set, a default value will be assumed in the format {env_var['info']}")
            env_var['value']=env_var['default']

    return env_variables

def get_key_info(attestor_info):
    #this assumes attestors has only one key id to use. In case of multiple keyid for the same attestor we need to choose the correct one.
    key_id=attestor_info['key_id']
    parts=key_id.split("/")
    projects_index = parts.index('projects') + 1
    locations_index = parts.index('locations') + 1
    keyRings_index = parts.index('keyRings') + 1
    cryptoKeys_index = parts.index('cryptoKeys') + 1
    cryptoKeyVersions_index = parts.index('cryptoKeyVersions') + 1

    key_info={
        "project_id":parts[ parts.index('projects') + 1],
        "location": parts[ parts.index('locations') + 1],
        "keyring":parts[ parts.index('keyRings') + 1],
        "key":parts[ parts.index('cryptoKeys') + 1],
        "version":parts[ parts.index('cryptoKeyVersions') + 1]
    }

    return key_info



def main():
    env_variables=get_env()

    # Configure the logging system
    level=logging.INFO
    if env_variables['SIGNER_LOGGING_LEVEL'] == "CRITICAL":
        level = logging.CRITICAL
    elif env_variables['SIGNER_LOGGING_LEVEL'] == "ERROR":
        level = logging.ERROR
    elif env_variables['SIGNER_LOGGING_LEVEL'] == "WARNING":
        level = logging.WARNING
    elif env_variables['SIGNER_LOGGING_LEVEL'] == "INFO":
        level = logging.INFO
    elif env_variables['SIGNER_LOGGING_LEVEL'] == "DEBUG":
        level = logging.DEBUG

    logging.root.setLevel(level)


    #generate_payload_file

    key_info = None

    attestor_info={
        "project_id":env_variables['ATTESTOR_PROJECT_NAME']['value'],
        "attestor": env_variables['ATTESTOR_NAME']['value'],
        "key_id":None,
        "note_reference":None,
        "note_id":None
    }
    #------ Initial check ----------------------#
    current_user_email=get_current_user_email()
    if not current_user_email:
        logging.critical("No user currently logged in. Make sure you have an active user in gcloud")
        return
    logging.debug(f"Acting as {current_user_email}")

    #------ Retrieve initial image info ----------------------#
    logging.info("Retrieving docker image")
    image_info=retrieve_image_info(env_variables['IMAGE_PATH']['value'])
    if not image_info:
        logging.critical("it was not possible to retrieve image digest. Please make sure to set the image digest or the image tag in the parameters")
        return


    #------ Retrieve initial attestor key info ----------------------#
    logging.info("Retrieving attestor information")
    attestor_retrieved_info=retrieve_attestor_info(attestor_info=attestor_info,current_user_email=current_user_email)
    if not attestor_retrieved_info:
        logging.critical(f"attestor {attestor_info} not present")
        return
    if env_variables['ATTESTOR_KEY_ID']['value']:
        logging.info("Using user defined attetor key")
        attestor_info['key_id']=env_variables['ATTESTOR_KEY_ID']['value']
    else:
       logging.warning("Using first attestor key id found in attestor key")
       attestor_info['key_id']=attestor_retrieved_info['userOwnedGrafeasNote']['publicKeys'][0]['id']
    attestor_info['note_reference']=attestor_retrieved_info['userOwnedGrafeasNote']['noteReference']
    attestor_info['note_id']=attestor_info['note_reference'].split("/")[-1]

    key_info=get_key_info(attestor_info=attestor_info)

    #------ Generating payload ----------------------#
    logging.info("Generating json docker_image_descriptiion with image url@sha256-digest")
    image_description_payload=generate_image_description_payload(image_info)
    logging.info("Generating docker_image_description in base64 of the docker_image_description_sha256")
    image_description_payload_sha256=get_base64_encoded_hash(image_description_payload)
    #------ generate payload signature----------------------#
    logging.info("Calling google kms to sign image_description_payload sha256")
    image_payload_signature=generate_image_payload_signature(image_description_payload_sha256,key_info,attestor_info)
    if not image_payload_signature:
        return

    serialized_payload=base64.b64encode(image_description_payload).decode('utf-8')

    #------ Generate image attestation ----------------------#
    logging.info("Generate attestation to upload")
    attestation_payload=generate_attestation_payload(image_info=image_info,attestor_info=attestor_info,serialized_payload=serialized_payload,payload_signature=image_payload_signature)
    logging.info("Upload attestation")
    uploaded_attestation=upload_attestation(attestor_info=attestor_info,attestation_payload=attestation_payload)
    logging.info("Process completed")

if __name__ == "__main__":
    main()

