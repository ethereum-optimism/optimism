# Consul Helm Chart

This repository contains the official HashiCorp Helm chart for installing
and configuring Consul on Kubernetes. This chart supports multiple use
cases of Consul on Kubernetes depending on the values provided.

For full documentation on this Helm chart along with all the ways you can
use Consul with Kubernetes, please see the
[Consul and Kubernetes documentation](https://www.consul.io/docs/platform/k8s/index.html).

## Prerequisites

To use the charts here, [Helm](https://helm.sh/) must be installed in your
Kubernetes cluster. Setting up Kubernetes and Helm and is outside the scope
of this README. Please refer to the Kubernetes and Helm documentation.

The versions required are:

  * **Helm 2.10+** - This is the earliest version of Helm tested. It is possible
    it works with earlier versions but this chart is untested for those versions.
  * **Kubernetes 1.9+** - This is the earliest version of Kubernetes tested.
    It is possible that this chart works with earlier versions but it is
    untested.

## Usage

For now, we do not host a chart repository. To use the charts, you must
download this repository and unpack it into a directory. Either
[download a tagged release](https://github.com/hashicorp/consul-helm/releases) or
use `git checkout` to a tagged release.
Assuming this repository was unpacked into the directory `consul-helm`, the chart can
then be installed directly:

    helm install ./consul-helm

Please see the many options supported in the `values.yaml`
file. These are also fully documented directly on the
[Consul website](https://www.consul.io/docs/platform/k8s/helm.html).
