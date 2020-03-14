Example of a Consul config for AWS

```json
{
    "bootstrap_expect": 5,
    "server": true,
    "datacenter": "dc1",
    "data_dir": "/opt/consult/data",
    "log_level": "info",
    "disable_remote_exec": true,
    "encrypt": "dfsdfsd",
    "retry_join": [
        "provider=aws tag_key=ConsulDataCenter tag_value=foobar"
    ],
    "rejoin_after_leave": true,
    "cert_file": "/home/immutability/vault.crt",
    "ca_file": "/home/immutability/root.crt",
    "key_file": "/home/immutability/vault.key",
    "verify_incoming": true,
    "verify_outgoing": true,
    "recursors": ["localhost:8600"],
    "dns_config": {
        "allow_stale": true
    },
    "enable_local_script_checks": true
}
```