/*
 * Helm Release - https://www.terraform.io/docs/providers/helm/r/release.html
 * Installs the Consul Helm chart with value overrides
 * This depends on the Consul gossip key existing in K8S secrets
 * prior to attempting to install the Helm chart
 */
resource "helm_release" "consul_chart" {
  name      = "omisego-consul"
  chart     = "../helm/consul"
  namespace = var.k8s_namespace

  atomic          = true
  cleanup_on_fail = true

  set {
    name  = "global.certificatesSecretNamePrefix"
    value = var.k8s_certificates_secret_name_prefix
  }

  set {
    name  = "global.image"
    value = local.consul_img
  }

  set {
    name  = "global.imageK8S"
    value = local.consul_k8s_img
  }

  set {
    name  = "global.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "global.gossipKey"
    value = data.vault_generic_secret.consul_gossip_key.data["value"]
  }

  set {
    name  = "server.replicas"
    value = var.consul_replicas
  }

  set {
    name  = "server.bootstrapExpect"
    value = var.consul_bootstrap_expect
  }
}

/*
 * Helm Release - https://www.terraform.io/docs/providers/helm/r/release.html
 * Installs the Vault Helm chart with value overrides
 * This depends on the Consul Helm chart being installed already
 */
resource "helm_release" "vault_chart" {
  depends_on = [helm_release.consul_chart]
  name       = "omisego-vault"
  chart      = "../helm/vault"
  namespace  = var.k8s_namespace

  atomic          = true
  cleanup_on_fail = true

  set {
    name  = "global.certificatesSecretNamePrefix"
    value = var.k8s_certificates_secret_name_prefix
  }

  /*
   * https://cloud.google.com/kubernetes-engine/docs/tutorials/http-balancer
   * If the cluster context is not local/minikube, then the Vault service will be set
   * to be of type `LoadBalancer` which will trigger an automatic GCP load balancer to
   * be created to manage the inboudn traffic to the specified service in the manifest
   */
  set {
    name  = "global.loadBalancer"
    value = var.k8s_context_cluster != "minikube"
  }

  set {
    name  = "server.image"
    value = local.vault_img
  }

  set {
    name  = "server.acl.token"
    value = data.kubernetes_secret.vault_acl_token.data.token
  }

  set {
    name  = "server.replicas"
    value = var.vault_replicas
  }

  set {
    name  = "server.mlockDisabled"
    value = var.mlock_disabled
  }

  set {
    name  = "server.unseal.address"
    value = var.unsealer_vault_addr
  }

  set {
    name  = "server.unseal.token"
    value = data.vault_generic_secret.unseal_token.data["value"]
  }

  set {
    name  = "consul.image"
    value = local.consul_img
  }

  set {
    name  = "consul.acl.token"
    value = data.kubernetes_secret.client_acl_token.data.token
  }

  set {
    name  = "consul.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "consul.gossipKey"
    value = data.vault_generic_secret.consul_gossip_key.data["value"]
  }
}

/*
 * Null Resource - https://www.terraform.io/docs/providers/null/resource.html
 * Uses as a work around for the external reference deprecation for destroy-time provisioners
 * because of variables are considered "external references". On creation this resource is created
 * but has no side-effects or deployed artifacts, however, on destroy it is responsible for cleaning
 * out the Kubernetes secrets that Helm uninstallation doesn't handle.
 */
resource "null_resource" "local" {
  depends_on = [helm_release.consul_chart]

  triggers = {
    secrets_list = "${var.k8s_consul_bootstrap_acl_token_name} ${var.k8s_consul_client_acl_token_name} ${var.k8s_consul_vault_acl_token_name}"
  }

  lifecycle {
    ignore_changes = [triggers.secrets_list]
  }

  /*
   * https://www.terraform.io/docs/provisioners/local-exec.html
   * On `destroy`, the provisioner will utilize `kubectl` to delete the Kubernetes
   * secrets that were created during the installation process which are not in the Helm
   * chart templates for auto-deletion during uninstall.
   */
  provisioner "local-exec" {
    when       = destroy
    command    = "kubectl delete secret ${self.triggers.secrets_list}"
    on_failure = fail
  }
}
