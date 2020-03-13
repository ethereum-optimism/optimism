# Install the Vault Helm chart with value overrides
# Depends on the Consul helm chart being install
# prior to installing the Vault chart
resource "helm_release" "vault" {
  depends_on = [helm_release.consul_base]

  name  = "vault"
  chart = "../helm/vault"

  cleanup_on_fail = true

  # FIXME:
  # set {
  #   name  = "global.tlsDisabled"
  #   value = false
  # }

  set {
    name  = "injector.enabled"
    value = false
  }

  # FIXME:
  # set {
  #   name  = "server.image.repository"
  #   value = "omisego/immutability-vault-ethereum"
  # }

  set {
    name  = "server.image.tag"
    value = "latest"
  }

  set {
    name  = "server.standalone.enabled"
    value = false
  }

  set {
    name  = "server.ha.enabled"
    value = true
  }

  # TODO: convert file paths and others to TF variables
  set {
    name  = "server.ha.config"
    value = <<EOF
      disable_mlock = true

      storage "file" {
        path = "/home/immutability/data"
      }

      listener "tcp" {
        tls_disable = 1
        address     = "0.0.0.0:8200"
        tls_cert_file = "/home/immutability/vault.crt"
        tls_client_ca_file = "/home/immutability/root.crt"
        tls_key_file = "/home/immutability/vault.key"
      }

      seal "transit" {
        address = "https://10.8.0.2:8200"
        token = "s.CvWxYRt6QRd8QRbGrBr5eTup"
        tls_ca_cert = "/home/immutability/root.crt"
        disable_renewal = "true"
        key_name = "autounseal"
        mount_path = "transit/"
      }
    EOF
  }

  set {
    name  = "ui.enabled"
    value = false
  }
}
