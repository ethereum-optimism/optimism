# Omisego Helm Charts

## Charts

The charts contained within the [`helm` directory](./helm) are local copies of stock chart releases for Consul as a backend, Vault and applying base ACLs onto the clusters.

## Pod Deployments

The Helm operations have been abstracted away through Terraform for installation, updates and destroys. The [`terraform` directory](./terraform) contains the `.tf` files associated with performing the following:

1. Applying custom value overrides for stock Helm charts
2. Installing/deploying the charts to the targeted Kubernetes cluster
3. Performing deploying updates/upgrades

### Variables

|           Name            |  Type  |            Default             |
| :-----------------------: | :----: | :----------------------------: |
|     `k8s_config_path`     | string |        `~/.kube/config`        |
|   `k8s_context_cluster`   | string |           `minikube`           |
| `consul_gossip_key_name`  | string | `consul-gossip-encryption-key` |
|    `consul_datacenter`    | string |             `dc1`              |
|     `consul_replicas`     | number |              `5`               |
| `consul_bootstrap_expect` | number |              `3`               |
|       `vault_addr`        | string |    `https://localhost:8200`    |

### Execution

Prior to execution, you must have installed, initialized and configured the Unsealer Vault. Terraform will need to have access to keys that are stored in the Unsealer Vault, so the following environment variables must be set:

```sh
export VAULT_CACERT=$HOME/etc/vault.unsealer/root.crt
export VAULT_TOKEN=$(cat keybase.name-of-admin.root.b64 | base64 --decode | keybase pgp decrypt)
export VAULT_ADDR=https://localhost:8200
```

```sh
$ cd terraform
$ terraform init
$ terraform plan
$ terraform apply
```

### Outputs

* `consul_bootstrap_acl_token`: the decoded bootstrap ACL token created and stored as a Kubernetes secret


## Verify ACL and Snapshot

```sh
$ export ACL_TOKEN="*****-****-****-*****"
# Received from the `terraform apply` output
# Alternatively you can fetch from K8s secret manually
# $ export ACL_TOKEN=$(kubectl get secret consul-backend-consul-bootstrap-acl-token -o json | jq -r .data.token | base64 --decode)
$ kubectl exec pod/consul-backend-consul-server-0 \
    -- consul snapshot save -token=$ACL_TOKEN backup.snap
```
