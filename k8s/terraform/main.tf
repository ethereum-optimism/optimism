terraform {
  required_version = ">= 0.12"
}

provider "kubernetes" {
  config_path            = var.k8s_config_path
  config_context_cluster = var.k8s_context_cluster
}

provider "helm" {
  kubernetes {
    config_path    = var.k8s_config_path
    config_context = var.k8s_context_cluster
  }
}

provider "vault" {
  address = var.vault_addr
}

data "vault_generic_secret" "consul_gossip_key" {
  path = "kv/consul_gossip_key"
}

data "vault_generic_secret" "unseal_token" {
  path = "kv/unseal_token"
}

data "kubernetes_secret" "bootstrap_acl_token" {
  depends_on = [helm_release.consul_base]
  metadata {
    name = "consul-backend-consul-bootstrap-acl-token"
  }
}

# Set the Consul gossip key from variables in K8s as a generic secret
resource "kubernetes_secret" "consul_gossip_key" {
  metadata {
    name = var.consul_gossip_key_name
  }

  data = {
    key = "${data.vault_generic_secret.consul_gossip_key.data["value"]}"
  }

  type = "generic"
}

# Install the Consul Helm chart with value overrides
# This depends on the Consul gossip key existing in K8s secrets
# prior to attempting to install the Helm chart
resource "helm_release" "consul_base" {
  depends_on = [kubernetes_secret.consul_gossip_key]

  name  = "consul-backend"
  chart = "../helm/consul-backend"

  cleanup_on_fail = true

  set {
    name  = "global.enabled"
    value = false
  }

  set {
    name  = "global.datacenter"
    value = var.consul_datacenter
  }

  set {
    name  = "global.gossipEncryption.secretName"
    value = var.consul_gossip_key_name
  }

  set {
    name  = "global.gossipEncryption.secretKey"
    value = "key"
  }

  set {
    name  = "global.bootstrapACLs"
    value = true
  }

  set {
    name  = "global.tls.enabled"
    value = true
  }

  set {
    name  = "global.tls.verify"
    value = true
  }

  set {
    name  = "consul.server.enabled"
    value = true
  }

  set {
    name  = "consul.server.replicas"
    value = var.consul_replicas
  }

  set {
    name  = "consul.server.bootstrapExpect"
    value = var.consul_bootstrap_expect
  }

  set {
    name  = "consul.server.connect"
    value = false
  }

  set {
    name  = "consul.server.affinity"
    value = <<EOF
      podAntiAffinity:
        preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 1
          podAffinityTerm:
            topologyKey: kubernetes.io/hostname
            labelSelector:
              matchExpressions:
              - key: component
                operator: In
                values:
                - "{{ .Release.Name }}-{{ .Values.Component }}"
    EOF
  }

  set {
    name  = "consul.client.enabled"
    value = true
  }

  provisioner "local-exec" {
    command = "kubectl delete pods,jobs.batch -l component=consul-vault-acl-init"
  }
}
