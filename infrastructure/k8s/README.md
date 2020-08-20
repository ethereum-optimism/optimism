# Launch vault via helm in Minikube

## Start Minikube

Execute:

```
minikube start
```

## Generate Credentials to GCP KMS

First, you need to enable the KMS endpoint in your project by visiting this [GCP](https://console.developers.google.com/apis/library/cloudkms.googleapis.com). Select your project from the dropdown at the top, then click Enable.

Next, go to the [Credentials](https://console.developers.google.com/apis/credentials) page. Select your project from the dropdown at the top, then click _+ CREATE CREDENTIALS_ -> _Service account_. Pick any name you want for the _Service account name_, then click _CREATE_.

Once you have a service account, go to it and click _ADD KEY_ -> _Create new key_. Choose _JSON_ -> _CREATE_ and download the file. Move the file to the current directory and call the file credentials.txt.

Execute:

```
kubectl create secret generic kms-creds --from-file=credentials.json
```

## Create a KMS Key

Visit [Cryptographic Keys](https://console.cloud.google.com/security/kms). Select your project from the dropdown at the top, then click _+ CREATE KEY RING_. Give it whatever name you want, but remember it because you'll need it down below.

Click the newly created keyring and then click _+ CREATE KEY_. Give it a name (again, you'll need it down below) and the rest of the defaults are okay. Click _CREATE_.

## Update value overrides

Edit _vault-overrides.yaml_. Be sure to change the values for:

* ClusterName
* Project
* KeyRing
* Key

## Start the Pods

Execute:

```
helm install vault ./vault --values vault-overrides.yaml
```

## Install the Vault Helm Chart

Execute:

```
kubectl exec --stdin --tty vault-0 -- /bin/sh
```

From here, you can interact with vault. For example:

```
vault status
vault operator init
vault status
```

## Remove the Vault Helm Chart

When you're done, execute:

```
helm uninstall vault
```
