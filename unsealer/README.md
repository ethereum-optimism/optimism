# Installation Guideline

In an attempt to make the process of installing all the components that comprise the OmiseGo Vault ecosystem simple and repeatable, follows is a guideline that will walk each administrative actor through each installation workflow.

## The Unsealer Vault

The Unsealer Vault resides on a laptop that will be airgapped from the network most of the time. The Unsealer Vault contains the master encryption keys for the Plasma Authority Vault Cluster. It also manages other credentials such as the Consul gossip encryption key, the Consul ACL tokens and the (long-lived) token used by the Plasma Authority Vault Cluster for authenticated unsealing.

The Unsealer Vault will also be used as the control point for installing, upgrading and backing up the Vault infrastructure. The laptop will be used to run Terraform, helm and to save and restore (if necessary) Consul snapshots. 

Note: The vault data, Terraform state and Consul snapshots will need to be backed up onto non-volatile storage and secured offsite. The same approach (off-site, non-volatile storage) will need to be used to store the Shamir key shards used to construct the master encryption key.

### Approach

The installation will be in phases in order not to obfuscate important aspects involved in configured the Unsealer Vault as well as installing, securing and configuring the network and Plasma Authority Vault Cluster. Certain keys will be created as a by-product of installation and these keys will be:

1. Stored in the Unsealer Vault; and,
2. Deleted from the GCP/GKE configurations (for security).

Since the keys will be deleted from the GCP/GKE configurations, it is very important that there is complete awareness of where these keys are and how they are used.

### Tooling Expectations 

The following has been tested on a factory-fresh Mac laptop. There is no requirement that this is a Mac; however, in order to avoid unnecessary complexity (there is enough complexity as it is) we will bias the discussion towards a Mac-based laptop.

The installation does not require anything more than a MacAir (this https://www.amazon.com/Apple-MacBook-1-8GHz-dual-core-Intel/dp/B07211W6X2 is sufficient.) The laptop will be airgapped for most of its existence, but it will need to have the following tools installed. This will mean that it will likely be on the network to download the basic kit required. It is incumbent upon the person doing this to check the signatures of the various distros used.

A script to install the HashiCorp tools, along with the HashiCorp PGP key is provided. To install Vault, Terraform and Consul using this script, run:

```sh
$ cd $GOPATH/src/github.com/omisego/immutability-eth-plugin/unsealer
$ ./install.sh vault 1.3.2
$ ./install.sh terraform 0.12.21
$ ./install.sh consul 1.7.1

```
**Required tools**

(Assumes /usr/local/bin is in your PATH.)

1. Vault:
```sh
$ vault --version
Vault v1.3.2
```

2. Terraform:
```sh
$ terraform --version
Terraform v0.12.21
```

3. Consul:
```sh
$ consul --version
Consul v1.7.1
Protocol 2 spoken by default, understands 2 to 3 (agent will automatically use protocol >2 when speaking to compatible agents)
```

4. Gcloud:
```sh
$ gcloud version
Google Cloud SDK 281.0.0
bq 2.0.53
core 2020.02.14
gsutil 4.47
```

5. Helm:
```sh
$ helm version
version.BuildInfo{Version:"v3.1.1", GitCommit:"afe70585407b420d0097d07b21c47dc511525ac8", GitTreeState:"clean", GoVersion:"go1.13.8"}
```

6. jq (https://github.com/stedolan/jq):
```sh
$ jq --version
jq-1.6
```

7. yq (https://github.com/mikefarah/yq):
```sh
$ yq -V
yq version 3.1.2
```

8. Keybase:
```sh
$ keybase version
Client:  5.2.1-20200225174716+9845113a89
Service: 5.2.1-20200225174716+9845113a89
```

## Initialize the Unsealer Vault

Initializing the Unsealer Vault creates the self-signed CA certificate and TLS material used to encrypt the transport between the Plasma Authority Vault Cluster and the Unsealer Vault, generating the keyshards used to construct the master encryption key, generating the root token, and unsealing the Unsealer Vault.

We use Keybase to encrypt the keyshards as described here - https://www.vaultproject.io/docs/concepts/pgp-gpg-keybase/. This script requires as input:

* Keybase Identity of the Unsealer Admin. The Vault Root Token will be encrypted using this identity's PGP key.
* Keybase Identities (exactly 5) of the Keyshard Holders.  These identities will be used to encrypt the keyshards that form the master unseal key for the Unsealer Vault.

Example usage:
```sh
$ ./initialize.sh "keybase:cypherhat" "keybase:immutability,keybase:cypherhat,keybase:zambien,keybase:webjuan,keybase:tajobe"
```

### Unsealing the Unsealer

The Unsealer Vault should be running now. You can verify this by executing:
```sh
$ lsof -i:8200
COMMAND   PID         USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
vault   18563 immutability    5u  IPv4 0x8d4594ade6228ef3      0t0  TCP localhost:trivnet1 (LISTEN)
```

The unseal keyshards and root token have been written to files and encrypted using the Keybase identities supplied. For example:
```sh
$ la keybase.*
-rw-r--r--  1 immutability  staff   889B Mar  1 07:47 keybase.cypherhat.b64
-rw-r--r--  1 immutability  staff   837B Mar  1 07:47 keybase.cypherhat.root.b64
-rw-r--r--  1 immutability  staff   545B Mar  1 07:47 keybase.immutability.b64
-rw-r--r--  1 immutability  staff   889B Mar  1 07:47 keybase.tajobe.b64
-rw-r--r--  1 immutability  staff   545B Mar  1 07:47 keybase.webjuan.b64
-rw-r--r--  1 immutability  staff   545B Mar  1 07:47 keybase.zambien.b64
```

Each Keyshard should be distributed to the owners of the Keybase identities and secured in persistant storage offline.

The keyshards and root token can be decrypted as follows:
```sh
$ cat keybase.cypherhat.b64 | base64 --decode | keybase pgp decrypt
0194de4d8bb6e380a92d215bc6c66cadcb0ab2df97b90096d07dc4d1a552d43dcd%                                          
$ cat keybase.cypherhat.root.b64 | base64 --decode | keybase pgp decrypt
s.x8rE0YEf5QZ7UGoGHdlkwhvO%
```

NOTE: The Keybase user has to be logged into Keybase in order to decrypt. If not, the following message will appear:
```sh
$ cat keybase.tajobe.b64 | base64 --decode | keybase pgp decrypt
▶ ERROR decrypt error: unable to find a PGP decryption key for this message
```

### Recommended Approach for Unsealing

We recommend that at least 3 of 5 of the Keybase identities login to Keybase on the unsealer laptop to engage with the unsealing ceremony. After logging in to Keybase, each keyshard holders must execute the following command:
```sh
$ cat keybase.immutability.b64 | base64 --decode | keybase pgp decrypt | xargs vault operator unseal
Key                Value
---                -----
Seal Type          shamir
Initialized        true
Sealed             true
Total Shares       5
Threshold          3
Unseal Progress    1/3
Unseal Nonce       3fef5fc2-0f99-971d-7f62-932b5eb7969f
Version            1.3.2
HA Enabled         false
```

Once 3 of 5 have unsealed, you can confirm this by executing:
```sh
$ vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    5
Threshold       3
Version         1.3.2
Cluster Name    vault-cluster-0e93abec
Cluster ID      9c4c1b44-8b3b-53f6-5cf6-b0e61bee70cc
HA Enabled      false
```

The last Keybase user should logout of Keybase after unsealing.

## Configure the Unsealer Vault

Configuring the Unsealer Vault amounts to enabling the backends necessary for infrastructure key storage and Vault unsealing and establishing the policies necessary for the Plasma Authority Vault Cluster to unseal itself.

To configure the Unsealer Vault, an environment variable named VAULT_TOKEN must be set. For simplicity's sake, we will use the root token acquired in the previous step:
```sh
$ export VAULT_TOKEN=$(cat keybase.cypherhat.root.b64 | base64 --decode | keybase pgp decrypt)
$ ./configure.sh
```

We are now ready to begin provisioning the Vault cluster.

## Terraform to Create Network

## Terraform to Install Vault Cluster

## Post Install

When we installed the Consul Helm chart, a powerful ACL token was injected into a Kubernetes secret. In order to protect this token, we will remove it from the Kubernetes secret store and save it in the Unsealer Vault:

```sh
$ ./k8sclean.sh
Success! Data written to: kv/consul-backend-consul-bootstrap-acl-token
secret "consul-backend-consul-bootstrap-acl-token" deleted
```