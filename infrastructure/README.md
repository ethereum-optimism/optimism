# GCP and Vault Infrastructure

This directory holds the Terraform scripts and the Helm chart overrides to deploy the cloud infrastructure and Kubernetes resources into two Google Cloud projects.

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
- [DNS](./terraform/dns.tf)
  - Creates a DNS zone for Vault
  - Adds an A record for the chosen ingress IP
  - Gives the cluster DNS admin permissions and dumps the service account credential file to `./terraform/dns_account.key.json`

### Deployment

You must enable the necessary Google API services, first, in order to successfully run the Terraform scripts to deploy the resources. There is a script to enable (and disable for teardown) the APIs for you:

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

> NOTE:
> <br />
> There are three variables that are networking CIDR blocks that must be set properly for a successful apply: `gke_pod_cidr`, `gke_service_cidr`, and `vault_subnet_cidr`. All three of there blocks must be different, but can whatever you'd like.
> <br />
> 
> - `vault_subnet_cidr`: the CIDR block for the VPC subnet that the GKE cluster nodes will be created and restricted to
> - `gke_service_cidr`: the CIDR block for all Kubernetes services and load balancer to be created within
> - `gke_pod_cidr`: the CIDR block for the Kubernetes pods to be restricted to (must be a `/21` or lower mask per GCP restrictions)
> <br />
> 
> Once created, each of these CIDR blocks will be automatically attached to the Vault subnet in the VPC to allow VPC peering to all three of the CIDR blocks. 

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

> Note:
>
> Before running the KMS script, ensure you have the current Kubernetes context and credentials active for the GKE cluster.

```bash
./scripts/kms.sh -c ./terraform/kms_account.key.json -r $GCP_REGION
```

This script will activate the KMS service account in the `gcloud` tool using the generate credential file path provided and create a new KMS key ring and symmetric unsealing key within that ring for you (if one or both already exist, these steps will be skipped). Once the key ring and unsealer key have been created within your GCP project, the script [injects the service account credential file into cluster secrets to be mounted into the nodes for unsealing](https://www.vaultproject.io/docs/platform/k8s/helm/run#google-kms-auto-unseal) before revoke your `gcloud` authentication session.

### DNS

Run:

```bash
./scripts/dns.sh -c ./terraform/dns_account.key.json
```

This will grant the cluster permissions to write to DNS for certificate verification.

If DNS for the domain is managed by another provider than Google's Cloud DNS (e.g. Cloudflare), an NS record can be added so that queries for a Vault subdomain are delegated to Google's nameservers.

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

##### Install cert-manager

If using ingress and an external certificate to access the Vault cluster, define this in k8s/cert-manager-issuers/values.yaml

To install cert-manager to generate SSL certificates for Vault:

```bash
helm dep up ./k8s/cert-manager
helm install cert-manager ./k8s/cert-manager
helm install cert-manager-issuers ./k8s/cert-manager-issuers
```

##### Install traefik

To install traefik for HTTPS ingress

```bash
helm dep up ./k8s/traefik
helm install traefik ./k8s/traefik
```


##### Configuring the vault chart

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

This updates the values.yaml file to correspond to your cluster.

##### Start the Pods using the Helm Chart

In `k8s`, execute:

Execute:

```bash

helm dep up ./k8s/vault
helm install vault ./k8s/vault
```

#### Sanity check

At this point, you should see the following:

```
% k get pods                               (vault-dev/default)
NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-56d6bbcb86-cnp96              1/1     Running   0          42m
cert-manager-cainjector-6dd56cf757-2927c   1/1     Running   0          42m
cert-manager-webhook-658654fddb-99nzp      1/1     Running   0          42m
datadog-57n6w                              2/2     Running   0          10m
datadog-695cp                              2/2     Running   0          10m
datadog-7vntp                              2/2     Running   0          10m
datadog-9wp8j                              2/2     Running   0          10m
datadog-lcdbc                              2/2     Running   0          10m
datadog-rgfhg                              2/2     Running   0          10m
datadog-vnz99                              2/2     Running   0          10m
datadog-wnfb6                              2/2     Running   0          10m
datadog-xnqr2                              2/2     Running   0          10m
traefik-84b6c7b79b-djkp9                   1/1     Running   0          7m48s
traefik-84b6c7b79b-t556r                   1/1     Running   0          7m48s
traefik-84b6c7b79b-wbdm2                   1/1     Running   0          7m48s
vault-0                                    0/1     Running   0          3m15s
vault-1                                    0/1     Running   0          3m15s
vault-2                                    0/1     Running   0          3m15s
vault-3                                    0/1     Running   0          3m15s
vault-4                                    0/1     Running   0          3m15s
```

```
% k get svc                                (vault-dev/default)
NAME                   TYPE           CLUSTER-IP     EXTERNAL-IP   PORT(S)                      AGE
cert-manager           ClusterIP      10.4.173.28    <none>        9402/TCP                     42m
cert-manager-webhook   ClusterIP      10.4.205.180   <none>        443/TCP                      42m
kubernetes             ClusterIP      10.4.0.1       <none>        443/TCP                      105m
traefik                LoadBalancer   10.4.155.10    10.5.0.100    80:30110/TCP,443:31466/TCP   8m17s
vault                  ClusterIP      None           <none>        8200/TCP,8201/TCP            3m45s
vault-active           ClusterIP      None           <none>        8200/TCP,8201/TCP            3m45s
vault-internal         ClusterIP      None           <none>        8200/TCP,8201/TCP            3m45s
vault-standby          ClusterIP      None           <none>        8200/TCP,8201/TCP            3m45s
```

```
% k get ing                                (vault-dev/default)
NAME    HOSTS                       ADDRESS      PORTS     AGE
vault   dev.vault-dev.omg.network   10.5.0.100   80, 443   4m18s
```

```
% k get certificate                        (vault-dev/default)
NAME                         READY   SECRET                       AGE
vault-ingress-certificate    True    vault-ingress-certificate    4m32s
vault-internal-certificate   True    vault-internal-certificate   4m32s
```

### Interact with Vault

The easiest way to interact with Vault is to install Vault locally and use port forwarding:

```
kubectl port-forward service/vault 8200:8200
export VAULT_ADDR=https://127.0.0.1:8200
```

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
vault status -tls-skip-verify
```

The status command may not work yet if the Vault Server isn't initialized.

#### Initialize Vault

```bash
vault operator init -format=json > cluster-keys.json -tls-skip-verify
vault status -tls-skip-verify
```

At this point, Vault should be up and the vault-active service should have a backend. From now on, use:

```kubectl port-forward service/vault-active 8200:8200```

#### Load the Immutability plugin

```
PLUGIN_NAME="immutability-eth-plugin"
INTERNAL_CERT_DIR="/vault/userconfig/vault-internal-certificate"
CA_CERT="$INTERNAL_CERT_DIR/ca.crt"
TLS_CERT="$INTERNAL_CERT_DIR/tls.crt"
TLS_KEY="$INTERNAL_CERT_DIR/tls.key"
SHA256SUM=$(kubectl exec vault-0 -- cat /vault/plugins/SHA256SUMS | cut -d' ' -f1)

vault write -tls-skip-verify "sys/plugins/catalog/secret/$PLUGIN_NAME" \
    sha_256="$SHA256SUM" \
    command="$PLUGIN_NAME --ca-cert=$CA_CERT --client-cert=$TLS_CERT --client-key=$TLS_KEY"


vault write "sys/plugins/catalog/secret/$PLUGIN_NAME" -tls-skip-verify \
    sha_256="$SHA256SUM" \
    command="$PLUGIN_NAME -tls-skip-verify"

vault secrets enable -tls-skip-verify -path="$PLUGIN_NAME" -plugin-name="$PLUGIN_NAME" plugin
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

*Rotational strategy*. Maintain a most-recent set of snapshots. This is implemented in a script and can be used as follows:

In `infrastructure`, execute:

```bash
./scripts/vault_backup.sh -d <dest-dir> [-p <file-prefix>] [-m <max-backups>] [--help]
``

#### Restore Vault RAFT Data from a Snapshot File

When you need to restore your Vault cluster back to a known-good state, identify the snapshot-file you want to restore and execute this command:

```bash
vault operator raft snapshot restore snapshot-file.raft
````

If using the *Rotational strategy*, this is implemented in a script and can be used as follows:

In `infrastructure`, execute:

```bash
./scripts/vault_restore.sh -s <src-dir> [-p <file-prefix>] [-b <backup-number>] [--help]
```

---

### Uninstalling Vault

When you're done, you can uninstall vault.

```bash
helm uninstall vault
```
