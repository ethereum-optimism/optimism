terraform {
  required_version = ">= 0.12"
}

# Kubernetes provider configuration for specifying the
# cluster target context and interaction with the cluster
# secrets API
provider "kubernetes" {
  config_path            = var.k8s_config_path
  config_context_cluster = var.k8s_context_cluster
}

# Helm provider configuration for installing charts for
# Consul and Vault within the target K8S cluster
provider "helm" {
  kubernetes {
    config_path    = var.k8s_config_path
    config_context = var.k8s_context_cluster
  }
}

# Vault provider configuration for utilizing unsealer instance
# secrets and managing the lifecycle of K8S secrets storage
provider "vault" {
  address = var.unsealer_vault_addr
}

# Loads the Consul gossip encryption key that should be
# preloaded into the unsealer Vault instance prior to
#executing this Terraform
data "vault_generic_secret" "consul_gossip_key" {
  path = "kv/consul_gossip_key"
}

# Loads the unseal Vault token from the running Vault node
# to be injected into the K8S Vault configuration to unseal
# themselves once them come online and are ready
data "vault_generic_secret" "unseal_token" {
  path = "kv/unseal_token"
}

# Loads the Consul bootstrap ACL token from the K8S cluster's
# secrets AFTER the Consul Helm chart has been successfully installed
data "kubernetes_secret" "bootstrap_acl_token" {
  depends_on = [helm_release.vault_chart]
  metadata {
    name = var.k8s_consul_bootstrap_acl_token_name
  }
}

# Loads the Consul Vault policy ACL token from the K8S cluster's
# secrets AFTER the Consul Helm chart has been successfully installed
data "kubernetes_secret" "vault_acl_token" {
  depends_on = [helm_release.vault_chart]
  metadata {
    name = var.k8s_consul_vault_acl_token_name
  }
}

# Writes the Consul bootstrap ACL token into a new kv/ Vault path
# after the Consul chart is installed and the K8S secret containing the
# token value is able to be read
#
# Once completed, the provisioner deletes the bootstrap and client token
# secrets from the K8S cluster in order to clean up loose ends for secret management
resource "vault_generic_secret" "consul_bootstrap_token" {
  path      = "kv/consul_bootstrap_token"
  data_json = jsonencode({ "value" = data.kubernetes_secret.bootstrap_acl_token.data.token })

  provisioner "local-exec" {
    command = "kubectl delete secret ${var.k8s_consul_bootstrap_acl_token_name} ${var.k8s_consul_client_acl_token_name}"
  }
}

# Writes the Consul Vault policy ACL token into a new kv/ Vault path
# after the Consul chart is installed and the K8S secret containing the
# token value is able to be read
#
# Once completed, the provisioner deletes the Vault policy token secret
# from the K8S cluster in order to clean up loose ends for secret management
resource "vault_generic_secret" "consul_vault_token" {
  path      = "kv/consul_vault_token"
  data_json = jsonencode({ "value" = data.kubernetes_secret.vault_acl_token.data.token })

  provisioner "local-exec" {
    command = "kubectl delete secret ${var.k8s_consul_vault_acl_token_name}"
  }
}


# Injects the Consul gossip encryption key from the unsealer Vault
# into a K8S secret to be usable by the Consul agents running in
# the pods for initialization
resource "kubernetes_secret" "consul_gossip_key" {
  metadata {
    name = var.k8s_consul_gossip_key_name
  }

  data = {
    key = data.vault_generic_secret.consul_gossip_key.data["value"]
  }

  type = "Opaque"
}

resource "kubernetes_secret" "tls_certificates" {
  metadata {
    name = var.k8s_certificates_secret_name
  }

  data = {
    "services.pem"     = file("${var.local_certificates_dir}/services.pem")
    "services-key.pem" = file("${var.local_certificates_dir}/services-key.pem")
    "ca.pem"           = file("${var.local_certificates_dir}/ca.pem")
  }

  type = "Opaque"
}

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
    name  = "global.certificatesSecretName"
    value = var.k8s_certificates_secret_name
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
  depends_on = [helm_release.consul_chart]

  name  = "omisego-vault"
  chart = "../helm/vault"

  cleanup_on_fail = true

  set {
    name  = "global.tlsEnabled"
    value = var.tls_enabled
  }

  set {
    name  = "global.certificatesSecretName"
    value = var.k8s_certificates_secret_name
  }

  set {
    name  = "server.replicas"
    value = var.vault_replicas
  }

  set {
    name  = "server.unseal.token"
    value = data.vault_generic_secret.unseal_token.data["value"]
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
