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
  depends_on = [helm_release.consul_chart]
  metadata {
    name = var.k8s_consul_vault_acl_token_name
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

# Injects the CA certificate and key file into Kubernetes secrets
# for the service pods to use for TLS
resource "kubernetes_secret" "ca_certificates" {
  count = var.tls_enabled ? 1 : 0

  metadata {
    name = "${var.k8s_certificates_secret_name_prefix}-ca"
  }

  data = {
    "tls.crt" = file("${var.local_certificates_dir}/ca.pem")
    "tls.key" = file("${var.local_certificates_dir}/ca-key.pem")
  }

  type = "kubernetes.io/tls"
}

# Injects the services' certificate and key file into Kubernetes secrets
# for the service pods to use for TLS
resource "kubernetes_secret" "services_certificates" {
  count = var.tls_enabled ? 1 : 0

  metadata {
    name = "${var.k8s_certificates_secret_name_prefix}-services"
  }

  data = {
    "tls.crt" = file("${var.local_certificates_dir}/services.pem")
    "tls.key" = file("${var.local_certificates_dir}/services-key.pem")
  }

  type = "kubernetes.io/tls"
}
