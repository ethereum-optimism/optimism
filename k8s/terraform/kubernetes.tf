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
