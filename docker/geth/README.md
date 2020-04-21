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

# Publishing to AWS ECR:
Make sure the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) is installed and [configured](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html#cli-quick-configuration)

1. Make sure you're authenticated: 
    ```
    aws ecr get-login-password --region us-east-2 | docker login --username AWS --password-stdin <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth
    ```
2. Change the working directory to `docker/geth`:
    ```
    cd docker/geth
    ``` 
3. Build and tag latest: 
    ```
    docker build -t optimism/geth .
    ```
4. Tag the build: 
    ```
    docker tag optimism/geth:latest <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:latest
    ```
5. Push tag to ECR:
    ```
    docker push <aws_account_id>.dkr.ecr.us-east-2.amazonaws.com/optimism/geth:latest
    ``` 
