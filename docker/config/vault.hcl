default_lease_ttl = "168h"
disable_mlock = "true"
max_lease_ttl = "720h"

backend "file" {
    path = "/home/vault/config/data"
}

ui = "false"

api_addr = "https://localhost:8900"
plugin_directory = "/home/vault/plugins"
listener "tcp" {
    address = "0.0.0.0:8900"
    tls_cert_file = "/home/vault/ca/certs/my-service.crt"
    tls_client_ca_file = "/Users/immutability/etc/vault.d/root.crt"
    tls_key_file = "/home/vault/ca/private/my-service.key"
    tls_require_and_verify_client_cert = "false"
}
