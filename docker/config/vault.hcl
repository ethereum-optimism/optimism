default_lease_ttl = "168h"
disable_mlock = "true"
max_lease_ttl = "720h"

backend "file" {
    path = "/vault/config/data"
}

ui = "false"

api_addr = "https://localhost:8900"
plugin_directory = "/vault/plugins"
listener "tcp" {
    address = "0.0.0.0:8900"
    tls_cert_file = "/vault/config/my-service.crt"
    tls_client_ca_file = "/vault/config/ca.crt"
    tls_key_file = "/vault/config/my-service.key"
    tls_require_and_verify_client_cert = "false"
}
