# Artifacts Locators

OP Deployer calls into precompiled contract artifacts. To make this work, OP Deployer uses artifacts locators to
point to the location of contract artifacts. While locators are nothing more than URLs, they do encode some
additional behaviors which are described here.

## Locator Types

Locators can be one of three types:

- `tag://` locators, which point to a versioned contracts release. These resolve to a known URL. Artifacts
  downloaded using a tagged locator are validated against a hardcoded checksum in the OP Deployer implementation.
  This prevents tampering with the contract artifacts once they have been tagged. Additionally, tagged locators are
  cached on disk to avoid repeated downloads.
  - ex: `tag://op-contracts/v1.8.0-rc.4`
- `https://` locators, which point to a tarball of contract artifacts somewhere on the web. HTTP locators are cached
  just like tagged locators are, but they are not validated against a checksum.
  - ex: `https://<url-to-remote-forge-artifacts>`
- `file://` locators, which point to a directory on local disk containing the artifacts.
  - ex: `file://<path-to-repo-root>/packages/contracts-bedrock/forge-artifacts`