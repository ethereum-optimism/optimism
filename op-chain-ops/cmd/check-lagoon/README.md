# check-lagoon

Smoke tests for interop cross-chain messaging and the op-interop-filter failsafe.

## Commands

### roundtrip

Bridges ETH A→B and B→A via `SuperchainETHBridge`, relaying each message, for N iterations.

```bash
go run . roundtrip --config <config.toml>
```

### failsafe

Full failsafe lifecycle test:
1. Bridge A↔B (expect success)
2. Enable failsafe on all configured filter instances
3. Attempt relay in both directions (expect rejection, 3 attempts each)
4. Disable failsafe
5. Bridge A↔B again (expect success)

```bash
go run . failsafe --config <config.toml>
```

## Config

Copy `config.example.toml` to a local file, fill in your values, and pass with
`--config`. The config holds live secrets (the `account` private key and filter
`jwt-secret`s), so every `*.toml` in this directory is gitignored except
`config.example.toml` — your copy can use any name and won't be committed:

```toml
l2-a = "https://your-chain-a-rpc"
l2-b = "https://your-chain-b-rpc"
account = "<hex-private-key-no-0x>"
relay-timeout = "2m"
iterations = 3          # roundtrip only
propagation-wait = "6s" # failsafe only

[filter]                # failsafe only
admin-rpc  = ["http://filter-host:8420"]
jwt-secret = ["0x<32-byte-hex-secret>"]
```

The `account` key uses the foundry dev key on devnets with `fund_dev_accounts: true`:
```
ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
```

CLI flags and `CHECK_LAGOON_*` env vars override values in the config file.
