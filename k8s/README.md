# Omisego Helm Charts

## Charts

The charts contained within the [`helm` directory](./helm) are local copies of stock chart releases for Consul as a backend, Vault and applying base ACLs onto the clusters.

## Pod Deployments

The Helm operations have been abstracted away through Terraform for installation, updates and destroys. The [`terraform` directory](./terraform) contains the `.tf` files associated with performing the following:

1. Applying custom value overrides for stock Helm charts
2. Installing/deploying the charts to the targeted Kubernetes cluster
3. Performing deploying updates/upgrades

### Execution

Prior to execution, you must:
1.  installed, initialized and configured the Unsealer Vault
2.  your `gcloud` CLI configured for your target deployment project
3.  run `./scripts/gcr_docker_tags.sh -p <gcp_project>` to tag and push the Docker images to the existing GCR repository

Terraform will need to have access to keys that are stored in the Unsealer Vault, so the following environment variables must be set:

```sh
$ export VAULT_CACERT=$HOME/etc/vault.unsealer/ca.pem
$ export VAULT_TOKEN=$(cat keybase.name-of-admin.root.b64 | base64 --decode | keybase pgp decrypt)
$ export VAULT_ADDR=https://localhost:8200
```

Make sure that your `kubeconfig` entry is updated for the GKE cluster that was created via the scripts in [infrastructure](../../infrastructure):

```sh
$ export GKE_CLUSTER_NAME=$(gcloud container clusters list --project=<gcp_project> --format=json | jq -r '.[].name')
$ gcloud container clusters get-credentials $GKE_CLUSTER_NAME --project=<gcp_project> --region <gcp_region>
```

This is automatically populate your `$HOME/.kube/config` file with an entry for the GKE cluster. The value of the GKE cluster will follow the format `gke_<gcp_project>_<gcp_region>_$GKE_CLUSTER_NAME` and is constructed for you within the Terraform scripts based on the variables provided. If you have more than one context entry in your `kubeconfig`, you can see the list of them by running:

```sh
$ kubectl config get-contexts
```

### Variables

|                  Name                  |  Type   |               Default                |
| :------------------------------------: | :-----: | :----------------------------------: |
|       `consul_bootstrap_expect`        | number  |                 `3`                  |
|          `consul_datacenter`           | string  |                `dc1`                 |
|           `consul_replicas`            | number  |                 `5`                  |
|         `docker_registry_host`         | string  |               `gcr.io`               |
|             `gcp_project`              | string  |                  -                   |
|              `gcp_region`              | string  |                  -                   |
|           `gke_cluster_name`           | string  |                  -                   |
| `k8s_certificates_secret_name_prefix`  | string  |        `omisego-certificates`        |
|           `k8s_config_path`            | string  |           `~/.kube/config`           |
| `k8s_consul_bootstrap_acl_token__name` | string  | `omisego-consul-bootstrap-acl-token` |
|   `k8s_consul_client_acl_token_name`   | string  |  `omisego-consul-client-acl-token`   |
|   `k8s_consul_vault_acl_token_name`    | string  |   `omisego-consul-vault-acl-token`   |
|            `k8s_namespace`             | string  |              `default`               |
|        `local_certificates_dir`        | string  |                  -                   |
|               `recovery`               | boolean |               `false`                |
|         `unsealer_vault_addr`          | string  |       `https://10.8.0.2:8200`        |
|            `vault_replicas`            | number  |                 `3`                  |

### Outputs

- `helm_consul_status`
- `helm_vault_status`
- `disaster_recovery_steps`


## Verify ACL and Snapshot
TODO: Fix this section of docs
```sh
$ export ACL_TOKEN="*****-****-****-*****"
# Received from the `terraform apply` output
# Alternatively you can fetch from K8s secret manually
# $ export ACL_TOKEN=$(kubectl get secret consul-backend-consul-bootstrap-acl-token -o json | jq -r .data.token | base64 --decode)
$ kubectl exec pod/consul-backend-consul-server-0 \
    -- consul snapshot save -token=$ACL_TOKEN backup.snap
```

## Recovery

In the event of disaster recovery, perform the followings steps:
1. Delete the existing local Terraform state file:
      
    `rm *.tfstate*`

2. Destroy the Vault and Consul Kubernetes resources:
      
    `kubectl -n ${var.k8s_namespace} delete pod,svc,deployment,statefulset,secret --all`

3. Change the Terraform variable `recovery` to true in your `.tfvars` file
4. Re-apply the Terraform script:

    ```
    $ terraform plan
    $ terraform apply --var "recovery=true"
    ```
