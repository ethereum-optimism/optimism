# Testing K8S Integration

**NOTE:** For the purpose of development, this runs Minikube as a
Kubernetes environment. If you already have a running Kubernetes environment  in a cloud, you can use that instead.

We use an environment variable, $DEVDIR, to point to the Immutability-OmiseGo repo. For example:

```sh
$ export DEVDIR=/Users/immutability/omisego-dev
$ la $DEVDIR/docker/ca/certs
total 24
-rwxrwxrwx@ 1 immutability  staff   1.2K Nov 14 15:38 ca.crt
-rwxrwxrwx@ 1 immutability  staff    41B Nov 14 15:38 ca.srl
-rwxrwxrwx@ 1 immutability  staff   1.3K Nov 14 15:38 my-service.crt
```

## Prerequisites

You need:

- [Minikube installed](https://kubernetes.io/docs/tasks/tools/install-minikube/)
- Make sure that Minikube has been started: `minikube start`. (It is assumed that you are using hyperkit for the VM.)
- Run Vault + OmiseGo Ethereum Plugin (`cd $DEVDIR && make run`)

## As a K8S Admin [Create K8S service account](./create-service-account.sh)

This service account will be used to gain access to the Authority account to submit blocks. *Care must be taken with access to this service account.*

```sh
$ ./create-service-account.sh
serviceaccount/omisego-service created
clusterrolebinding.rbac.authorization.k8s.io/role-tokenreview-binding configured
```

## As a Vault admin [Provision Access to Authority Account](./provision-access.sh)

We will create a policy that allows the OmiseGo service account to submit blocks, then enable and configure the K8S authentication method. 

**NOTE:** Configuration of the kubernetes authentication backend requires access to the credentials for the service account:

```sh
export VAULT_SA_NAME=$(kubectl get sa omisego-service -o jsonpath="{.secrets[*]['name']}")

export SA_JWT_TOKEN=$(kubectl get secret $VAULT_SA_NAME -o jsonpath="{.data.token}" | base64 --decode; echo)
```

It is assumed that the Vault admin and K8S admin have similar trust profiles within the firm. Access to the K8S service account credentials *can* give access to submit blocks. The `omisego/immutability-eth-plugin` does have the ability to whitelist client IPs which would prevent access from unauthorized environments. Ideally, the actors which have legitimate access to the service account would be in a restricted subnet and disable remote access.

```sh
$ ./provision-access.sh
path "immutability-eth-plugin/wallets/plasma-deployer/accounts/0x4BC91c7fA64017a94007B7452B75888cD82185F7/plasma/submitBlock" {
    capabilities = ["update", "create"]
}
Success! Uploaded policy: submit-blocks
Success! Enabled kubernetes auth method at: kubernetes/
Success! Data written to: auth/kubernetes/config
Success! Data written to: auth/kubernetes/role/authority
```

## As a K8S Admin [Configure Vault Agent](./configmaps.sh)

```sh
$ ./configmaps.sh
configmap/cacerts created
configmap/vault-agent-config created
```

## As a K8S Admin [Deploy Vault Agent](./deploy-agent.sh)

If you are using hyperkit for your VM, your host address should be: `192.168.64.1`. However, you can always run this command from your host to determine the actual host address:

```sh
$ minikube ssh "route -n | grep ^0.0.0.0 | awk '{ print \$2 }'"
192.168.64.1
```

If your host is different than what is shown above, you will need to edit [the K8S deployment config](k8s-agent-spec.yml) to provide a different setting for:

```yaml
      env:
        - name: VAULT_ADDR
          value: https://192.168.64.1:8200
```

**ALSO** if your host IP is different, you will have to [regenerate your certs](../../docker/config/gencerts.sh) with an IP SANS for that host:

```
# Alternative names are specified as IP.# and DNS.# for IPs and
# DNS accordingly.
[alt_names]
IP.1  = 127.0.0.1
IP.2  = 192.168.64.1
IP.3  = 192.168.122.1
DNS.1 = localhost

```

Deploy the vault-agent as a **normal** container. When you deploy it as a sidecar, you would deploy it as an `initContainers`. However, we want to get a shell to this container to demonstrate block submission.

```sh
$ ./deploy-agent.sh
pod/omisego-agent-sidecar created
```

Now, we get shell on the vault agent:

```sh
$ kubectl exec -it omisego-agent-sidecar --container vault-agent-auth sh
/ # export VAULT_TOKEN=$(cat /vault/.vault-token)
/ # export VAULT_CACERT=/certs/vault-cacert
/ # vault  write immutability-eth-plugin/wallets/plasma-deployer/accounts/0x4BC91c7fA64017a94007B7452B75888cD82185F7/plasma/submitBlock block_root=1234qweradgf1234qweradgf contract=0xd185aff7fb18d2045ba766287ca64992fdd79b1e
Key                   Value
---                   -----
contract              0xd185AFF7fB18d2045Ba766287cA64992fDd79B1e
from                  0x4BC91c7fA64017a94007B7452B75888cD82185F7
gas_limit             75932
gas_price             20000000000
nonce                 2
signed_transaction    0xf889028504a817c8008301289c94d185aff7fb18d2045ba766287ca64992fdd79b1e80a4baa4769431323334717765726164676631323334717765726164676600000000000000001ca04b43b927af8dd7f085eb07b7a5e6e41061e3292a98c5ac08fe226d20309e3c16a04ffbf5f431b4c379b455f893a974c87304df04225aea6ab014f999f61f479130
transaction_hash      0x45d1fb775a9c3d406becaa6bfbc9ec9777e49563508b6538962e04fa42b2e97e
```

