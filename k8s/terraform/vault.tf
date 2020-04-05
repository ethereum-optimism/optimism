/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/d/generic_secret.html
 * Loads the Consul gossip encryption key that should be
 * preloaded into the unsealer Vault instance prior to
 * executing this Terraform
 */
data "vault_generic_secret" "consul_gossip_key" {
  path = "kv/consul_gossip_key"
}

/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/d/generic_secret.html
 * Loads the Consul acl token that Vault uses for storage
 */
data "vault_generic_secret" "consul_vault_token" {
  path = "kv/consul_vault_token"
}


/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/d/generic_secret.html
 * Loads the unseal Vault token from the running Vault node
 * to be injected into the K8S Vault configuration to unseal
 * themselves once them come online and are ready
 */
data "vault_generic_secret" "unseal_token" {
  path = "kv/unseal_token"
}

/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/r/generic_secret.html
 * Writes the Consul bootstrap ACL token into a new kv/ Vault path
 * after the Consul chart is installed and the K8S secret containing the
 * token value is able to be read
 */
resource "vault_generic_secret" "consul_bootstrap_token" {
  path      = "kv/consul_bootstrap_token"
  data_json = jsonencode({ "value" = data.kubernetes_secret.bootstrap_acl_token.data.token })
}

/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/r/generic_secret.html
 * Writes the Consul client ACL token into a new kv/ Vault path
 * after the Consul chart is installed and the K8S secret containing the
 * token value is able to be read
 */
resource "vault_generic_secret" "consul_client_token" {
  path      = "kv/consul_client_token"
  data_json = jsonencode({ "value" = data.kubernetes_secret.client_acl_token.data.token })
}

/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/r/generic_secret.html
 * Writes the Consul Vault policy ACL token into a new kv/ Vault path
 * after the Consul chart is installed and the K8S secret containing the
 * token value is able to be read
 */
resource "vault_generic_secret" "consul_vault_token" {
  path      = "kv/consul_vault_token"
  data_json = jsonencode({ "value" = var.recovery ? data.vault_generic_secret.consul_vault_token.data["value"] : data.kubernetes_secret.vault_acl_token.data.token })
}
