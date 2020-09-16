# GCP and Vault Infrastructure

This directory holds the Terraform scripts and the Helm chart overrides to deploy the cloud infrastructure and Kubernetes resources into a Google Cloud project.

## GCloud Authentication

You have to authenticate with the `gcloud` command line tool (download if not already installed). If you already have credentials locally for the project, you can simply set `GOOGLE_APPLICATION_CREDENTIALS` to the local file path, and Terraform will utilize those credentials.

Otherwise, you can run `gcloud init` to create a fresh GCP project configuration locally, or just run:

```bash
$ gcloud auth login
$ gcloud config set project $PROJECT_ID # <-- important for subsequent commands
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
./scripts/gcp_services.sh # [-d: disable]
```

This enables:

- `compute.googleapis.com`
- `cloudkms.googleapis.com`
- `containerregistry.googleapis.com`
- `iap.googleapis.com`
- `iam.googleapis.com`
- `container.googleapis.com`

Without doing this first, the deployment will not succeed.

Now that you're authenticated locally to your GCP project, you can now deploy the scripts. Ensure that you have all of the necessary Terraform variables set in the `terraform.tfvars` file and run:

```bash
cd ./terraform
terraform init
terraform plan
terraform apply
```

The deployment could take some time because of spinning up a new GKE cluster, but once complete the two new service account credential key files (GCR and KMS service account) will be created within the `./terraform` directory.

## Kubernetes

### Credentials

[Source](./k8s)

Now that you have your GKE cluster running in a configured GCP project, you can setup your local Kubernetes configuration to point to the GKE cluster you've created. Assuming you are still authenticated to your project with the `gcloud` CLI, you can run:

```bash
export GKE_CLUSTER_NAME=$(gcloud container clusters list --format=json | jq -r '.[].name')
gcloud container clusters get-credentials $GKE_CLUSTER_NAME --region $GCP_REGION
```

This will fetch and write new credentials for your GKE cluster into your local `~/.kube/config` file that you can inspect to verify the population. To verify the credentials assignment was ok, run:

```bash
kubectl config view
```

And validate that there is a cluster context for GKE with the format `gke_${PROJECT}_${REGION}_${CLUSTER}`, and that its set to your `current-context`. If it exists but is not your current context, you can enable it with:

```bash
kubectl config use-context $GKE_CONTEXT_NAME
```

And later _delete_ it (if necessary with):

```bash
kubectl config delete-context $GKE_CONTEXT_NAME
kubectl config delete-cluster $GKE_CONTEXT_NAME
kubectl config unset users.$GKE_CONTEXT_NAME
kubectl config unset current-context
```

### KMS

Due to the lifecycle restrictions on KMS resources by Google, this had to be removed from the Terraform as a separate step. A new KMS symmetric key must be created and injected into the Kubernetes cluster's secrets to be able to be pulled and read by the Vault nodes for auto-unseal.

Assuming you have already run the Terraform and have the `kms_account.key.json` file generated for the service account under your `./terraform` directory, you can now run:

```bash
./scripts/kms.sh -c ./terraform/kms_account.key.json -r $GCP_REGION
```

> Note:
>
> Before running the KMS script, ensure you have the current Kubernetes context and credentials active for the GKE cluster.

This script will activate the KMS service account in the `gcloud` tool using the generate credential file path provided and create a new KMS key ring and symmetric unsealing key within that ring for you (if one or both already exist, these steps will be skipped). Once the key ring and unsealer key have been created within your GCP project, the script [injects the service account credential file into cluster secrets to be mounted into the nodes for unsealing](https://www.vaultproject.io/docs/platform/k8s/helm/run#google-kms-auto-unseal) before revoke your `gcloud` authentication session.

### Helm / Deployment

Deploying services to a Kubernetes cluster typically require the use of [helm](https://helm.sh) to manage the cluster configuration and dependencies. This guide shows how to use the official [Hashicorp](https://www.hashicorp.com) _helm chart_ to deploy a Vault cluster.

#### Installing Helm and Supporting Tools

If you are running on MacOS, you can install helm by executing:

```bash
brew install helm yq
```

If you are running on Linux or Windows, see the [Helm Download Page](https://github.com/helm/helm/releases/latest). You'll also want to install the yq utility.

#### Establish Remote Registries

In order to use the official Hashicorp and Datadog Helm repositories, we need to add it to the local helm registry by executing:

```bash
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo add datadog https://helm.datadoghq.com
helm repo add stable https://kubernetes-charts.storage.googleapis.com
helm repo update
```

#### Datadog

In the [Datadog overrides file](./k8s/datadog-overrides.yaml), insert your Datadog API and App key into the YAML file at `.datadog.apiKey` and `.datadog.appKey` respectively.

From the `infrastructure` folder, execute the following command to deploy the Datadog Helm chart:

```sh
helm upgrade --atomic --cleanup-on-fail --install --values ./k8s/datadog-overrides.yaml datadog datadog/datadog
```

With the existing overrides (in addition to your API and app keys), this Helm chart instantiates a DaemonSet for the Datadog agent pods that will be responsible for collecting and forwarding Vault server and audit logs into your Datadog dashboards.

#### Vault

##### Generate Self-Signed Certs

In `./infrastructure`, execute:

```bash
./scripts/gen_certs.sh -d <dns-domain>
```

For GKE clusters, use: `-d vault-internal.default.svc.cluster.local`
For Minikube, use: `-d vault-internal`

---

**NOTE**: The Root CA Certificate uses a 20 year TTL

**NOTE**: The TLS Key Material should be generated each time the Vault Cluster is upgraded. We currently suggest an upgrade cycle of 2-4 months, so the TLS Key Material should have a TTL of 6 months.

---

##### Create a kubernetes secret with the cert

The `gen-certs.sh` script updates `k8s/vault-overrides.yaml` with the name of the secret that was generated with the new certs material. To see the created secret, execute:

```bash
kubectl get secrets
```

and look for "omgnetwork-certs-"

##### Generate Storage Classes

In `infrastructure`, execute:

```bash
./scripts/gen_storage.sh
```

##### Update Value Overrides

In `infrastructure`, execute:

```bash
./scripts/gen_overrides.sh
```

You can supply the `--help` option to see what options are available. If you have the following environment variables set, it will use these values as defaults:

```bash
$GCP_REGION
$GCP_PROJECT
$GKE_CLUSTER_NAME
```

<<<<<<< HEAD
##### Start the Pods using the Helm Chart
=======
>>>>>>> 1526b2980e828e9057bfe4cbaf0a629887648fc5

Execute:

```bash
helm upgrade --atomic --cleanup-on-fail --install --wait --values vault-overrides.yaml vault hashicorp/vault
```

### Interact with Vault

#### Logs

You can see the vault logs by executing:

```bash
kubectl logs vault-0
```

Before you initialize vault, you'll see errors like this:

```
2020-08-21T03:26:41.607Z [INFO]  core: stored unseal keys supported, attempting fetch
2020-08-21T03:26:41.684Z [WARN]  failed to unseal core: error="fetching stored unseal keys failed: failed to decrypt encrypted stored keys: failed to decrypt envelope: rpc error: code = InvalidArgument desc = Decryption failed: verify that 'name' refers to the correct CryptoKey."
```

#### Access Vault using the CLI

In another terminal, execute:

```bash
export VAULT_ADDR=https://<load-balancer>:8200
export VAULT_CACERT=$K8S/certs/ca-chain.cert.pem

vault status
```

The status command may not work yet if the Vault Server isn't initialized.

#### Initialize Vault

```bash
vault operator init -format=json > cluster-keys.json
vault status
```

#### Enable Auditing

```bash
vault audit enable file file_path=/vault/audit/audit.log
```

#### Backup Vault RAFT Data to a Snapshot File

Determining how many backup files you want to keep is a business decision. There are different strategies for maintaining a set of backup snapshots that can be employed.

*Time-based strategy*. The snapshot filename is derived from the formatted timestamp. In this strategy, you'll have to determine how many snapshots to maintain and how to rotate them out when they're no longer appropriate.

```bash
vault operator raft snapshot save snapshot-$(date +%Y%m%d-%H%M%S).raft
```

*Rotational strategy*. In this example, a maximum of 5 snapshots are maintained at any given time.

```bash
rm -f snapshot-4.raft

for i in 3 2 1; do
  let NEXT=$i+1
  mv -f snapshot-${i}.raft snapshot-${NEXT}.raft 2> /dev/null
done

mv -f snapshot.raft snapshot-1.raft 2> /dev/null

vault operator raft snapshot save snapshot.raft
```

#### Restore Vault RAFT Data from a Snapshot File

When you need to restore your Vault cluster back to a known-good state, identify the snapshot-file you want to restore and execute this command:

```bash
vault operator raft snapshot restore snapshot-file.raft
```

### Uninstalling Vault

When you're done, you can uninstall vault.

```bash
helm uninstall vault
```
