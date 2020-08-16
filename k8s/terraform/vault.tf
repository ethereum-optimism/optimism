/*
 * Vault Secret - https://www.terraform.io/docs/providers/vault/d/generic_secret.html
 * Loads the unseal Vault token from the running Vault node
 * to be injected into the K8S Vault configuration to unseal
 * themselves once them come online and are ready
 */
data "vault_generic_secret" "unseal_token" {
  path = "kv/unseal_token"
}
