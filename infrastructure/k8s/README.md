# Launch Vault via helm in Minikube

## Establish hashicorp registry

Execute:

```
helm repo add hashicorp http://helm.releases.hashicorp.com
```

## Start Minikube

Execute:

```
minikube start --mount=true
```

## GCP Project

You will need a project to create all of your resources in. It's easiest if you call it `omgnetwork-vault`, but you can call it whatever you want.

## GCP Service Account

You will need a service account. Go to https://console.cloud.google.com/iam-admin/serviceaccounts?authuser=4&project=omgnetwork-vault

Go to the [Credentials](https://console.developers.google.com/apis/credentials) page. Select `omgnetwork-vault` from the dropdown at the top, then click _+ CREATE CREDENTIALS_ -> _Service account_. Pick any name you want for the _Service account name_, then click _CREATE_.

You need to add the following roles to this service account:

* Owner
* Cloud KMS Admin
* Cloud KMS CryptoKey Encrypter/Decrypter
* Compute Admin
* Kubernetes Engine Admin
* Storage Admin

Click your new service account and then click _ADD KEY_ -> _Create new key_. Choose _JSON_ -> _CREATE_ and download the file. Move the file to the current directory and call the file credentials.txt.

Execute:

```
kubectl create secret generic kms-creds --from-file=credentials.json
```

## Create a KMS Key

Visit [Cryptographic Keys](https://console.cloud.google.com/security/kms). Select your project from the dropdown at the top, then click _+ CREATE KEY RING_. Give it whatever name you want, but remember it because you'll need it down below.

Click the newly created keyring and then click _+ CREATE KEY_. Give it a name (again, you'll need it down below) and the rest of the defaults are okay. Click _CREATE_.

## Deploy the Infrastructure

Set the environment variable:

```
export GOOGLE_APPLICATION_CREDENTIALS=<path-to>/credentials.json
```

Over in infrastructure/terraform directory, execute:

```
terraform apply
```

You may get jillions of failures here, but follow the instructions for what they say and you should be fine. It's basically just enabling services in GCP. You can also try to get ahead of the game by going to infrastructure/scripts and executing:

```
./gcp_services.sh -p omgnetwork-vault
```

## Update value overrides

Back in infrastructure/k8s, edit _vault-overrides.yaml_ and verify all the values are correct (hint: they _should_ be fine unless you renamed things or something).

## Start the Pods

Execute:

```
helm install vault hashicorp/vault —-values vault-overrides.yaml
```

## Interact with Vault

### Logs

You can see the vault logs by executing:

```
kubectl logs vault-0
```

Before you initialize vault, you'll soee errors like this:

```
2020-08-21T03:26:41.607Z [INFO]  core: stored unseal keys supported, attempting fetch
2020-08-21T03:26:41.684Z [WARN]  failed to unseal core: error="fetching stored unseal keys failed: failed to decrypt encrypted stored keys: failed to decrypt envelope: rpc error: code = InvalidArgument desc = Decryption failed: verify that 'name' refers to the correct CryptoKey."
```

No worries, just go initialize vault.

### Connect to the Pods

Note that you can connect to vault-0, vault-1, or vault-2. Execute:

```
kubectl exec --stdin --tty vault-0 -- /bin/sh
```

and then initialize vault:

```
mkdir -p /vault/init
vault operator init > /vault/init/stdout 2> /vault/init/stderr
vault status
```

## Uninstalling Vault

When you're done, you can uninstall vault.

```
helm uninstall vault
```

## Ending Minikube

To stop the minikube VM:

```
minikube stop
```

To delete the minikube VM:

```
minikube delete
```
