/*
 * Null Resource - https://www.terraform.io/docs/providers/null/resource.html
 * Uses as a work around for the external reference deprecation for destroy-time provisioners
 * because of variables are considered "external references". On creation this resource is created
 * but has no side-effects or deployed artifacts, however, on destroy it is responsible for cleaning
 * out the Kubernetes secrets that Helm uninstallation doesn't handle.
 */
resource "null_resource" "kubectl_delete" {
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
    command    = "kubectl delete secret ${self.triggers.secrets_list}"
    on_failure = fail
  }
}
