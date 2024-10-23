# deployment-utils

This image provides a minimal set of Foundry and Bash tools for use with builder images like Kurtosis. It contains the
following packages:

- The Foundry suite (`forge`, `cast`, `anvil`)
- [`Dasel`](https://github.com/TomWright/dasel), for TOML/YAML manipulation.
- `jq` for JSON manipulation.
- `curl`, for when you need to cURLs.
- A default `bash` shell.

## Image Size

According to `dive`, this image is 255MB in size including the base Debian image. Most of the additional size comes from
the tools themselves. I'd like to keep it this way. This image should not contain toolchains, libraries, etc. - it is
designed to run prebuilt software and manipulate configuration files. Use the CI builder for everything else.