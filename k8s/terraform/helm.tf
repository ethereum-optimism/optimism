# Installs the Consul Helm chart with value overrides
#
# This depends on the Consul gossip key existing in K8S secrets
# prior to attempting to install the Helm chart
resource "helm_release" "consul_chart" {
  depends_on = [kubernetes_secret.consul_gossip_key]

  name  = "omisego-consul"
  chart = "../helm/consul"

  cleanup_on_fail = true

  set {
    name  = "global.tlsEnabled"
    value = var.tls_enabled
  }

  set {
    name  = "global.certificatesSecretNamePrefix"
    value = var.k8s_certificates_secret_name_prefix
  }

  set {
    name  = "global.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "global.gossipEncryption.secretName"
    value = var.k8s_consul_gossip_key_name
  }

  set {
    name  = "global.gossipEncryption.secretKey"
    value = "key"
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

# Installs the Vault Helm chart with value overrides
# This depends on the Consul Helm chart being installed already
resource "helm_release" "vault_chart" {
  depends_on = [data.kubernetes_secret.vault_acl_token]

  name  = "omisego-vault"
  chart = "../helm/vault"

  cleanup_on_fail = true

  set {
    name  = "global.tlsEnabled"
    value = var.tls_enabled
  }

  set {
    name  = "global.certificatesSecretNamePrefix"
    value = var.k8s_certificates_secret_name_prefix
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
    name  = "server.unseal.address"
    value = "https://192.168.64.1:8200"
  }

  set {
    name  = "server.unseal.token"
    value = data.vault_generic_secret.unseal_token.data["value"]
  }

  set {
    name  = "consul.acl.token"
    value = var.k8s_consul_client_acl_token_name
  }

  set {
    name  = "consul.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "consul.replicas"
    value = var.consul_replicas
  }

  set {
    name  = "consul.gossipEncryption.secretName"
    value = var.k8s_consul_gossip_key_name
  }

  set {
    name  = "consul.gossipEncryption.secretKey"
    value = "key"
  }
}
