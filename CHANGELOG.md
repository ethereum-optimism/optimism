## 0.0.1 (October 20, 2019)

NEW FEATURES:

* BIP44 implementation: Wallets are constructed from supplied or generated mnemonic. Accounts are derived using index: "m/44'/60'/0'/0/%d"
* Whitelists/Blacklists can be scoped at 3 levels: Global (config), Wallet and/or Account.
* Export JSON keystore using supplied or generated passphrase.
* Gas estimation for contract deployment.
* Golang unit tests
* Smoketest does integration testing against Ganache:
  - [x] plugin config
  - [x] wallet create/update/read/list
  - [x] account create/update/read/list
  - [x] account debits
  - [x] whitelist/blacklist testing at all levels
* Smoketest will print curl examples for all tests to aid with documentation
* Dockerfile builds plugin and vault image with plugin pre-packaged.
  - multistage build reduces image size and attack surface 
  - plugin built natively for Alpine using musl
  - Runs as non-root `vault` user (CIS Docker Benchmark 1.20 -  4.1 Ensure that a user for the container has been created).
* makefile with `docker-build`, `run`, and `all` targets.
* Use docker-compose to build ganache-based development environment for testing

IMPROVEMENTS:

* N/A

BUG FIXES:

* N/A
