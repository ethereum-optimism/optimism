/*
 * Kubernetes Secret - https://www.terraform.io/docs/providers/kubernetes/d/secret.html
 * Loads the Consul bootstrap ACL token from the K8S cluster's
 * secrets AFTER the Consul Helm chart has been successfully installed
 */
data "kubernetes_secret" "bootstrap_acl_token" {
  count      = var.recovery ? 0 : 1
  depends_on = [helm_release.consul_chart]
  metadata {
    name = var.k8s_consul_bootstrap_acl_token_name
  }
}

/*
 * Kubernetes Secret - https://www.terraform.io/docs/providers/kubernetes/d/secret.html
 * Loads the Consul client ACL token from the K8S cluster's
 * secrets AFTER the Consul Helm chart has been successfully installed
 */
data "kubernetes_secret" "client_acl_token" {
  count      = var.recovery ? 0 : 1
  depends_on = [helm_release.consul_chart]
  metadata {
    name = var.k8s_consul_client_acl_token_name
  }
}

/*
 * Kubernetes Secret - https://www.terraform.io/docs/providers/kubernetes/d/secret.html
 * Loads the Consul Vault policy ACL token from the K8S cluster's
 * secrets AFTER the Consul Helm chart has been successfully installed
 */
data "kubernetes_secret" "vault_acl_token" {
  count      = var.recovery ? 0 : 1
  depends_on = [helm_release.consul_chart]
  metadata {
    name = var.k8s_consul_vault_acl_token_name
  }
}

/*
 * Kubernetes Secret - https://www.terraform.io/docs/providers/kubernetes/r/secret.html
 * Injects the CA certificate and key file into Kubernetes secrets
 * for the service pods to use for TLS
 */
resource "kubernetes_secret" "ca_certificates" {
  metadata {
    name = "${var.k8s_certificates_secret_name_prefix}-ca"
  }

  data = {
    "tls.crt" = file("${var.local_certificates_dir}/ca.pem")
    "tls.key" = file("${var.local_certificates_dir}/ca-key.pem")
  }

  type = "kubernetes.io/tls"
}

/*
 * Kubernetes Secret - https://www.terraform.io/docs/providers/kubernetes/r/secret.html
 * Injects the services' certificate and key file into Kubernetes secrets
 * for the service pods to use for TLS
 */
resource "kubernetes_secret" "services_certificates" {
  metadata {
    name = "${var.k8s_certificates_secret_name_prefix}-services"
  }

  data = {
    "tls.crt" = file("${var.local_certificates_dir}/services.pem")
    "tls.key" = file("${var.local_certificates_dir}/services-key.pem")
  }

  type = "kubernetes.io/tls"
}
