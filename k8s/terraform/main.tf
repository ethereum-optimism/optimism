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
    key = data.vault_generic_secret.consul_gossip_key.data["value"]
  }

  type = "generic"
}
