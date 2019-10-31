## 0.0.2 (October 26, 2019)

NEW FEATURES:

* Demonstrate Smart Contract integration model
  - [x] Compile to ABI
  - [x] Generate bindings
* Added support for ERC20
  - [x] Deployed ERC20 implementation - FixedSupplyToken
  - [x] Implemented all methods in ERC20Interface

IMPROVEMENTS:

* Verified CIS Docker Hardening 1.20 for images where applicable to Dockerfile
  - [x] 4.1 Ensure that a user for the container has been created
  - [x] 4.2 Ensure that containers use only trusted base images 
        (HashiCorp Vault/Alpine)
  - [x] 4.3 Ensure that unnecessary packages are not installed in the container
  - [x] 4.4 Ensure images are scanned and rebuilt to include security patches 
        (apk update && apk upgrade added to Dockerfile)
  - [x] 4.5 - N/A  - Ensure Content trust for Docker is Enabled
  - [x] 4.6 Ensure that HEALTHCHECK instructions have been added to
        container images
  - [x] 4.7 Ensure update instructions are not use alone in the Dockerfile 
        - used epoch date for this in dockerfile/makefile
  - [x] 4.8 Ensure setuid and setgid permissions are removed 
        (vault user prevents this)
  - [x] 4.9 Ensure that COPY is used instead of ADD in Dockerfiles 
  - [x] 4.10 Ensure secrets are not stored in Dockerfiles
  - [x] 4.11 Ensure only verified packages are are installed 
        (using Alpine package manager)
* Smoke Test for transaction signing
* Smoke Test for ERC20
  - [x] Deploy Contract
  - [x] Read Token Supply
  - [x] Read Token Balance
  - [x] Transfer Token
  - [x] Approve Transfer

BUG FIXES:

* N/A

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
