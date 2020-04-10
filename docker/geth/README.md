# Optimism Geth Container
This is a Docker container that builds on the default Geth image, adding configuration specific to the uses within Optimstic Rollup.

# Building
You can build this image by running
`docker build -t optimism/geth .`

# Running
You can run this image by running the base `docker-compose.yml` from the base dir
```
cd ../..                    # Go to root of this repo
docker compose up --build   # If no changes since last build, can omit `--build`
```

# Configuration
Config is handled entirely through environment variables. Below are some config variable names, whether or not they're optional, and what they do:

Data:
* `CLEAR_DATA_KEY` - (optional) Set to clear all persisted data in the geth node. Data is only cleared on startup when this variable is set and is different from last startup (e.g. last start up it wasn't set, this time it is or last start up it was set to a different value than it is this start up).
* `VOLUME_PATH` - (required) The base filesystem path to use to store persisted data from geth

Node:
* `HOSTNAME` - (required) The hostname to use for this geth instance. It should almost always be `0.0.0.0`.
* `PORT` - (required) The port to expose this node through. This should be set to 9545 if there is not a good reason to set it to something else.
* `NETWORK_ID` - (required) The network ID for this node. This should be 108 for Optimistic Rollup L2 Node.
* `PRIVATE_KEY` - (optional) Set to provide a PK to seed with a balance. If no private key is specified, one will be created.

Populated During Node Setup (run if no node data is present):
* `PRIVATE_KEY_PATH_SUFFIX` - (optional) The suffix of the path (assumed to be prepended with `VOLUME_PATH`) where the private key used with this geth instance will be stored during initial setup.
* `ADDRESS_PATH_SUFFIX` - (required) The suffix of the path (assumed to be prepended with `VOLUME_PATH`) where the address of the PK will be stored during initial setup.
* `SEALER_PRIVATE_KEY_PATH_SUFFIX`- (required) The suffix of the path (assumed to be prepended with `VOLUME_PATH`) where the sealer private key is stored after initial setup (or where the sealer private key will be stored during geth setup).
* `SEALER_ADDRESS_PATH_SUFFIX` - (required) The suffix of the path (assumed to be prepended with `VOLUME_PATH`) where the sealer address will be stored during initial setup.
* `INITIAL_BALANCE` - (required) The initial balance to seed the provided address with during setup (should be a hex number).
* `GENESIS_PATH` - (required) The path to the genesis file used to bootstrap this node. Should be set to `etc/rollup-fullnode.json` based on Dockerfile at the time of this writing.
* `SETUP_RUN_PATH_SUFFIX` - (required) The suffix of the path (assumed to be prepended with `VOLUME_PATH`) where the file to be saved indicating whether or not setup has already run should be stored.


# Publishing to AWS ECR:
Make sure the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) is installed and [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#cli-quick-configuration)

1. Make sure you're authenticated: 
    ```
    aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth
    ```
2. Build and tag latest: 
    ```
    docker build -t optimism/geth .
    ```
3. Tag the build: 
    ```
    docker tag optimism/geth:latest <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:latest
    ```
4. Push tag to ECR:
    ```
    docker push <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:latest
    ``` 
