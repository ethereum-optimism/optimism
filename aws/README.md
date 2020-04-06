# Creating an AWS ECS Environment
The contents of this directory can be used to deploy a fully-functional Full Node to AWS.

Below are some instructions on how to do so. For more info, the instructions below loosely follow [this tutorial](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs-cli-tutorial-ec2.html).

## Prerequisites
AWS:
* Set up an AWS Account, Access Key & Secret, and keypair
* Install the [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)
* Install the [AWS ECS CLI](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_CLI_installation.html)

Other:
* Install [Docker](https://docs.docker.com/docker-for-mac/install/)
## Steps

### 1) Configure the Amazon ECS CLI
1. Create a cluster configuration:
    ```
    ecs-cli configure --cluster dev-full-node --default-launch-type EC2 --config-name dev-full-node-config --region us-east-2
    ```
    Note: choose an appropriate cluster name and have the config name derived from it.

2. Create a profile to use to create the environment
    ```
    ecs-cli configure profile --access-key <your access key here> --secret-key <your secret here> --profile-name dev-full-node-profile
    ```

### 2) Create the Cluster
```
ecs-cli up --keypair <your keypair name> --capability-iam --size 1 --instance-type t3.micro --cluster-config dev-full-node-config --ecs-profile dev-full-node-profile --port 8545 --security-group <Security Group ID for the instance> --vpc <VPC ID for the instance> --subnets <comma-separated list of subnet IDs>
```
Note:
* `size` is the number of ECS instances to create
* `instance-type` can be bumped up for a more powerful environment. More info [here](https://aws.amazon.com/ec2/instance-types/)
* If you haven't already, you'll need to create a Security Group. This is not individual to you, so see if your organization already has a suitable one.


This may take a few minutes to finish. The result will be a fully provisioned EC2 instance on which your service/task will be deployed.

### 3) Choose the appropriate `docker-compose.yml` and `ecs-params.yml`
For the rest of the commands, you'll need to be in a directory with a `docker-compose.yml` and an `ecs-params.yml`. These will define the service(s) you are going to create. Change to the appropriate directory or create one of your own basing them off of an existing one.

### 4) Deploy Tasks to Cluster 
```
ecs-cli compose up --create-log-groups --cluster-config dev-full-node-config --ecs-profile dev-full-node-profile
```

This will just start up the task(s) under the appropriate cluster. Ultimately we want a service to manage our task(s), but we don't want to do that until we know our tasks work. Make sure your tasks are functioning properly by checking their status and possibly even logging into CloudWatch and looking at the logs.

To check the status of your task(s), run:
```
ecs-cli ps --cluster-config dev-full-node-config --ecs-profile dev-full-node-profile
```

### 5) Create the ECS Service
First, kill the PoC tasks:
```
ecs-cli compose down --cluster-config dev-full-node-config --ecs-profile dev-full-node-profile
```

Now create the service:
```
ecs-cli compose service up --cluster-config dev-full-node-config --ecs-profile dev-full-node-profile
```

## Volumes
Right now volumes needed by the various containers are configured to be locally stored on the EC2 instance on which the containers are run and automatically created if not present. Eventually we will want to move to a more redundant form of storage (like EBS mounts), but this is fine for now.

For now, if you want to modify/delete the data in an environment you will need to
* `ssh` into the EC2 instance running the containers
* Find the volume(s) you would like to modify/delete (they are located at `/var/lib/docker/volumes`)
* Modify/Delete them as necessary
  * IMPORTANT: This will likely mess up any running tasks, so make sure to kill the task before
  * Also note that the Service will auto-replace your tasks, so if you can't do this quickly, disable that feature during your maintenance.  