# Deploying L2 Geth node to Synthetix UAT

## Prerequisites
See prerequisites from parent AWS directory.

## Steps

### 1) Configure the Amazon ECS CLI
1. Create a cluster configuration:
    ```
    ecs-cli configure --cluster synthetix-uat-geth --default-launch-type EC2 --config-name synthetix-uat-geth-config --region us-east-2
    ```

2. Create a profile to use to create the environment
    ```
    ecs-cli configure profile --access-key <your access key here> --secret-key <your secret here> --profile-name synthetix-uat-geth-profile
    ```

### 2) Create the Cluster
    ```
    ecs-cli up --keypair synthetix-uat --capability-iam --size 1 --instance-type c5.4xlarge --cluster-config synthetix-uat-geth-config --ecs-profile synthetix-uat-geth-profile --port 9545 --security-group <security group ID> --vpc <vpc ID> --subnets <comma-separated subnet IDs>
    ```

This may take a few minutes to finish. The result will be a fully provisioned EC2 instance on which your service/task will be deployed.

### 3) Choose the appropriate `docker-compose.yml` and `ecs-params.yml`
For the rest of the commands, you'll need to be in this directory to use the `docker-compose.yml` and an `ecs-params.yml`.
Make any necessary changes now.

### 4) Deploy Service & Task to Cluster & register service discovery. 
    ```
    ecs-cli compose --project-name synthetix-uat-geth service up --private-dns-namespace synthetix-uat --vpc <vpc ID> --enable-service-discovery --cluster-config synthetix-uat-geth-config --ecs-profile synthetix-uat-geth-profile --create-log-groups
    ```
