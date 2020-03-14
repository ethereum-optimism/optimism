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
  # Suppose the local Vault IP is 10.1.2.3 and the VPN is at 10.8.0.2
  #
  # Notice that the local Vault IP maps to:
  #   storage.advertise_addr  
  #   storage.redirect_addr
  #   api_addr
  #   listener.address
  # 
  # Notice that the VPN IP maps to:
  #   seal.address
  #
  # We need to push TLS keys/certs to a config map. These will
  # get mapped to:
  #   listener.tls_cert_file
  #   listener.tls_client_ca_file
  #   listener.tls_key_file
  #   seal.tls_ca_cert
  #
  # We need to set the transit Vault token as a TF Var
  #   seal.token
  #
  set {
    name  = "server.ha.config"
    value = <<EOF
      storage "consul" {
        address = "127.0.0.1:8500"
        path    = "vault/"
        advertise_addr =  "https://10.1.2.3:8200"
        redirect_addr =  "https://10.1.2.3:8200"
      }

      api_addr =  "https://10.1.2.3:8200"

      listener "tcp" {
        address     = "https://10.1.2.3:8200"
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

      plugin_directory = "/home/vault/plugins"
    EOF
  }

  set {
    name  = "ui.enabled"
    value = false
  }
}
