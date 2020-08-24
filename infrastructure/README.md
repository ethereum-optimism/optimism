# GCP and Vault Infrastructure

This directory holds the Terraform scripts and the Helm chart overrides to deploy the cloud infrastructure and Kubernetes resources into a Google Cloud project.

## GCloud Authentication

You have to authenticate with the `gcloud` command line tool (download if not already installed). If you already have credentials locally for the project, you can simply set `GOOGLE_APPLICATION_CREDENTIALS` to the local file path, and Terraform will utilize those credentials.

Otherwise, you can run `gcloud init` to create a fresh GCP project configuration locally, or just run:

```bash
$ gcloud config set project $PROJECT_ID # <-- important for subsequent commands
$ gcloud auth login
```

This will open up a browser window for you to login and authorize access to the target project for `gcloud` to provider credentials for. This session with `gcloud` can later be revoked by running:

```bash
$ gcloud auth revoke
```

> Note:
> 
> Alternatively, if you have a service account designated for IaC deployments, you can activate those credentials with:
> ```bash
> $ gcloud auth activate-service-account --key-file <path_to_key_file>
> ```

## Terraform

[Source](./terraform)

The Terraform scripts in this directory are used for standing up the cloud infrastructure in Google Cloud necessary for running and connecting to the Vault cluster deployed the GKE.

### Resources

- [VPC](./terraform/vpc.tf)
- [Firewall Rules](./terraform/firewall.tf)
  - Allows:
    - VPC peering for client connections
    - Datadog egress for telemetry
    - Infura egress for Ethereum API connection
- [Container Registry](./terraform/gcr.tf)
  - Creates a new service account for GCR read/write permissions
  - Dumps the service account credential file into `./terraform/gcr_account.key.json`
- [Kubernetes Engine](./terraform/gke.tf)
- [KMS](./terraform/kms.tf)
  - Creates a new service account for KMS read/write permissions
  - Dumps the service account credential file into `./terraform/kms_account.key.json`
  - Creates a new key ring and crypto key used for auto-unsealing Vault nodes

### Deployment

With a Google Cloud project already created, first you must enable the necessary Google API services in order to successfully run the Terraform scripts to deploy the resources. There is a script to enable (and disable for teardown) the APIs for you:

```bash
$ ./scripts/gcp_services.sh # [-d: disable]
```

This enables:
  - `compute.googleapis.com`
  - `cloudkms.googleapis.com`
  - `containerregistry.googleapis.com`
  - `iap.googleapis.com`
  - `iam.googleapis.com`
  - `container.googleapis.com`

Without doing this first, the deployment will not succeed.

Now that you're authenticated locally to your GCP project, you can now deploy the scripts. Ensure that you have all of the necessary Terraform variables set and run:

```bash
$ cd ./terraform
$ terraform init
$ terraform plan
$ terraform apply
```

The deployment could take some time because of spinning up a new GKE cluster, but once complete the two new service account credential key files (GCR and KMS service account) will be created within the `./terraform` directory.

## Kubernetes

### Credentials

[Source](./k8s)

Now that you have your GKE cluster running in a configured GCP project, you can setup your local Kubernetes configuration to point to the GKE cluster you've created! Assuming you are still authenticated to your project with the `gcloud` CLI, you can run:

```bash
$ export GKE_CLUSTER_NAME=$(gcloud container clusters list --format=json | jq -r '.[].name')
$ gcloud container clusters get-credentials $GKE_CLUSTER_NAME --region $GCP_REGION
```

This will fetch and write new credentials for your GKE cluster into your local `~/.kube/config` file that you can inspect to verify the population. To verify the credentials assignment was ok, run:

```bash
$ kubectl config view
```

And validate that there is a cluster context for GKE with the format `gke_${PROJECT}_${REGION}_${CLUSTER}`, and that its set to your `current-context`. If it exists but is not your current context, you can enable it with:

```bash
$ kubectl config use-context $GKE_CONTEXT_NAME
```

And later _delete_ it (if necessary with):

```bash
$ kubectl config delete-context/delete-cluster $GKE_CONTEXT_NAME
$ kubectl config unset users.$GKE_CONTEXT_NAME
$ kubectl config unset current-context
```

### KMS

Due to the lifecycle restrictions on KMS resources by Google, this had to be removed from the Terraform as a separate step. A new KMS symmetric key must be created and injected into the Kubernetes cluster's secrets to be able to be pulled and read by the Vault nodes for auto-unseal.

Assuming you have already run the Terraform and have the `kms_account.key.json` file generated for the service account under your `./terraform` directory, you can now run:

```bash
$ ./scripts/kms.sh -c ./terraform/kms_account.key.json -r $GCP_REGION
```

> Note:
>
> Before running the KMS script, ensure you have the current Kubernetes context and credentials active for the GKE cluster.

This script will activate the KMS service account in the `gcloud` tool using the generate credential file path provided and create a new KMS key ring and symmetric unsealing key within that ring for you (if one or both already exist, these steps will be skipped). Once the key ring and unsealer key have been created within your GCP project, the script [injects the service account credential file into cluster secrets to be mounted into the nodes for unsealing](https://www.vaultproject.io/docs/platform/k8s/helm/run#google-kms-auto-unseal) before revoke your `gcloud` authentication session.

### Helm / Deployment

<!-- TODO: -->
