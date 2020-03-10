# Vault Helm Chart

------
## WIP - forked from vault-Helm and under heavy development
------

This repository contains the official HashiCorp Helm chart for installing
and configuring Vault on Kubernetes. This chart supports multiple use
cases of Vault on Kubernetes depending on the values provided.

[//]: # (These docs don't exist yet)
[//]: # (For full documentation on this Helm chart along with all the ways you can)
[//]: # (use Vault with Kubernetes, please see the)
[//]: # ([Vault and Kubernetes documentation](https://www.vault.io/docs/platform/k8s/index.html).)

## Prerequisites

To use the charts here, [Helm](https://helm.sh/) must be installed in your
Kubernetes cluster. Setting up Kubernetes and Helm and is outside the scope
of this README. Please refer to the Kubernetes and Helm documentation.

The versions required are:

  * **Helm 2.10+** - This is the earliest version of Helm tested. It is possible
    it works with earlier versions but this chart is untested for those versions.
  * **Kubernetes 1.9+** - This is the earliest version of Kubernetes tested.
    It is possible that this chart works with earlier versions but it is
    untested. Other versions verified are Kubernetes 1.10, 1.11.

## Usage

For now, we do not host a chart repository. To use the charts, you must
download this repository and unpack it into a directory. Either
[download a tagged release](https://github.com/hashicorp/vault-helm/releases) or
use `git checkout` to a tagged release.
Assuming this repository was unpacked into the directory `vault-helm`, the chart can
then be installed directly:

    helm install ./vault-helm

[//]: # (These docs don't exist yet)
[//]: # (Please see the many options supported in the `values.yaml`)
[//]: # (file. These are also fully documented directly on the)
[//]: # ([Vault website](https://www.vault.io/docs/platform/k8s/helm.html).)

### Using auto-unseal

Starting with Vault 1.0-beta, auto-unseal features are now included in the open source 
version on Vault.  In order to use these features, users must ensure that 
the Vault configuration is provided with the appropriate credentials and authorizations 
to access the APIs needed for the given key provider, as well as the necessary keys 
created before hand. 

## Testing

The Helm chart ships with both unit and acceptance tests.

The unit tests don't require any active Kubernetes cluster and complete
very quickly. These should be used for fast feedback during development.
The acceptance tests require a Kubernetes cluster with a configured `kubectl`.
Both require [Bats](https://github.com/bats-core/bats-core) and `helm` to be
installed and available on the CLI. The unit tests also require the correct
version of [yq](https://pypi.org/project/yq/) if running locally.

To run the unit tests:

    bats ./test/unit

To run the acceptance tests:

    bats ./test/acceptance

If the acceptance tests fail, deployed resources in the Kubernetes cluster
may not be properly cleaned up. We recommend recycling the Kubernetes cluster to
start from a clean slate.

**Note:** There is a Terraform configuration in the
[test/terraform/ directory](https://github.com/hashicorp/vault-helm/tree/master/test/terraform)
that can be used to quickly bring up a GKE cluster and configure
`kubectl` and `helm` locally. This can be used to quickly spin up a test
cluster for acceptance tests. Unit tests _do not_ require a running Kubernetes
cluster.
