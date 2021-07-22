pid_file = "/vault/pidfile"

auto_auth {
    method "kubernetes" {
        mount_path = "auth/kubernetes"
        config = {
            role = "authority"
        }
    }

    sink "file" {
        config = {
            path = "/vault/.vault-token"
        }
    }
}
