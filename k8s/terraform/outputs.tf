output "consul_bootstrap_acl_token" {
  value = data.kubernetes_secret.bootstrap_acl_token.data.token
}
