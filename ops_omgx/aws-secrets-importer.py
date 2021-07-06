#!/usr/bin/python3

import json
import re
import subprocess
import sys, getopt
from ruamel.yaml import YAML

def main(argv):
    inputfile = ''
    description = "Create Secret Manager"
    secret_name = ''
    region = 'us-east-1'
    profile = 'default'
    if (len(sys.argv) <= 1 ) or (len(sys.argv) > 11):
        print('aws-secrets-importer.py -i <inputfile> -d <description> -n <external secret name> -r <region> -p <profile> ')
        sys.exit()
    try:
        opts, args = getopt.getopt(argv,"hi:d:n:r:p:",["ifile=","name=","description=","region=","profile="])
        print(opts)
    except getopt.GetoptError:
        print('aws-secrets-importer.py -i <inputfile> -d <description> -n <external secret name> -r <region> -p <profile> ')
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            print('aws-secrets-importer.py -i <inputfile> -d <description> -n <external secret name> -r <region> -p <profile> ')
            sys.exit()
        elif opt in ("-i", "--ifile"):
            inputfile = arg
        elif opt in ("-d", "--description"):
            description = arg
        elif opt in ("-n", "--name"):
            secret_name = arg
        elif opt in ("-r", "--region"):
            region = arg
        elif opt in ("-p", "--profile"):
            profile = arg
        else:
            print("Else")
            print('aws-secrets-importer.py -i <inputfile> -d <description> -n <external secret name> -r <region> -p <profile> ')
            sys.exit()
    with open(inputfile) as vars_file:
        pattern = "="
        secrets = {}
        for line in vars_file:
            if re.search(pattern, line):
                variables = line.strip()
                clean = re.sub(r"^- ", "", variables)
                key, value = clean.split('=')
                secrets[key] = value

    cmd = ['aws', 'secretsmanager', 'create-secret', '--region', region, '--profile', profile, '--description', description, '--name', secret_name, '--secret-string', json.dumps(secrets)]
    result  = subprocess.run(cmd)
    print(result)

if __name__ == "__main__":
   main(sys.argv[1:])
