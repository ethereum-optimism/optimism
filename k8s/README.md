# Omisego Helm Charts

## Charts

The charts contained within the [`helm` directory](./helm) are local copies of stock chart releases for Consul as a backend, Vault and applying base ACLs onto the clusters.

## Pod Deployments

The Helm operations have been abstracted away through Terraform for installation, updates and destroys. The [`terraform` directory](./terraform) contains the `.tf` files associated with performing the following:

1. Applying custom value overrides for stock Helm charts
2. Installing/deploying the charts to the targeted Kubernetes cluster
3. Performing deploying updates/upgrades

### Variables

|                  Name                  |  Type   |               Default                |
| :------------------------------------: | :-----: | :----------------------------------: |
|       `consul_bootstrap_expect`        | number  |                 `3`                  |
|          `consul_datacenter`           | string  |                `dc1`                 |
|           `consul_replicas`            | number  |                 `5`                  |
|         `docker_registry_addr`         | string  |         `192.168.64.1:5000`          |
| `k8s_certificates_secret_name_prefix`  | string  |        `omisego-certificates`        |
|           `k8s_config_path`            | string  |           `~/.kube/config`           |
|         `k8s_context_cluster`          | string  |              `minikube`              |
| `k8s_consul_bootstrap_acl_token__name` | string  | `omisego-consul-bootstrap-acl-token` |
|   `k8s_consul_client_acl_token_name`   | string  |  `omisego-consul-client-acl-token`   |
|   `k8s_consul_vault_acl_token_name`    | string  |   `omisego-consul-vault-acl-token`   |
|            `k8s_namespace`             | string  |              `default`               |
|        `local_certificates_dir`        | string  |                  -                   |
|               `recovery`               | boolean |               `false`                |
|         `unsealer_vault_addr`          | string  |     `https://192.168.64.1:8200`      |
|            `vault_replicas`            | number  |                 `3`                  |

### Execution

Prior to execution, you must have installed, initialized and configured the Unsealer Vault. Terraform will need to have access to keys that are stored in the Unsealer Vault, so the following environment variables must be set:

```sh
export VAULT_CACERT=$HOME/etc/vault.unsealer/ca.pem
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

> None


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
