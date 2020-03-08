# Validating the infrastructure provisioned

Prereq: You'll need 2 separate projects. A project for vault where the vault_vpc and infrastructure is to be provisioned and a project for the infrastructure that's going to represent the omisego network/vpc.

## Making sure VPN connection is working

1. Execute `terraform apply` on the infrastructure directory. This will create core networking resources. The VPN OpenVPN install script on the VPN instance generates an `.ovpn` VPN file to be used in the unsealer and places it on a bucket.
2. Get the ovpn file by executing: `gsutil cp gs://${BUCKET_NAME}/unsealer.ovpn`.
3. Once downloaded the file can be safely removed form the bucket: `gsutil rm gs://${BUCKET_NAME}/unsealer.ovpn`.
4. Access VPN from laptop. Using *Tunnelblick*, click on "**VPN Details**" and drag/drop the `unsealer.ovpn` into the "**Configurations**" drop down, then click the "**Connect**" button.
5. Once connected check your public IP by running: `curl 'https://api.ipify.org?format=json'` The returned value should match the value of the IP of the `vpn_public_instance_ip` terraform output.

## Validating connection from test instance in Vault VPC to Unsealer laptop

1. On the laptop, run a test vault server: `vault server -dev -dev-listen-address="0.0.0.0:8200"`.
2. In the `local_testing/vault_vpc` directory, create `terraform.tfvars` file with values required by the `variables.tf` file.
3. Execute `terraform apply`.
4. For future reference, export the IP given in the output: `export VAULT_IP=192.168.10.3`.
5. SSH to the test instance created by running the command specified in the `vault_vpc_test_instance_ssh_command` terraform output. For example: `gcloud beta compute ssh --zone us-central1-a test --tunnel-through-iap --project omsisego`.
6. Check connection from instance to the unsealer laptop by running: `curl http://10.8.0.2:8200/v1/sys/health`.

Note: don't delete the instance yet as we'll keep using it for further testing.

## Validating connection from test instance to datadog

1. The instance should have sent metrics to datadog. Check them by going to the datadog UI.

## Validating connection from Omisego VPC to the Vault VPC.

1. SSH to the test instance created by running the command specified in the `vault_vpc_test_instance_ssh_command` terrfaform output. For example: `gcloud beta compute ssh --zone us-central1-a test --tunnel-through-iap --project omisego`.
2. The startup script of the test instance starts a Vault dev server. Make sure vault server is running: `curl http://127.0.0.1:8200/v1/sys/health`.
3. In the local_testing/omisego_vpc directory, create terraform.tfvars file with values required by the variables.tf file.
4. Execute `terraform apply`.
5. SSH to the test instance created by running the command specified in the `omisego_vpc_test_instance_ssh_command` terrfaform output. For example: `gcloud beta compute ssh --zone us-central1-a test --tunnel-through-iap --project omisego`.
6. Validate connection to vault is working by executing: `curl http://${VAULT_IP}:8200/v1/sys/health`.
