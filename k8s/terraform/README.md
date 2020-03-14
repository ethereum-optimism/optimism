# Mapping Configuration

Consider the following:

```
  set {
    name  = "server.ha.config"
    value = <<EOF
      disable_mlock = false

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
```

