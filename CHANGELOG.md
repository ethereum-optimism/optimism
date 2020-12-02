## 0.0.7 (October 6, 2020)

NEW FEATURES:

* Backup / Restore scripts created for Vault Raft Data
* Creation of the gen_overrides.sh script

REFACTOR:

* updated `VERSION` file to `0.0.7`
* Regionalized SSD Persistent Data Volumes for Vault Raft Data
* Vault Auditing is now enabled
* Fix Vault Raft Peering
* Replaced the Custom Vault Helm Chart with the officially supported Helm Chart from Hashicorp
* Nonce refactored to be passed in
* Only --build on test

## 0.0.6 (August 15, 2020)

NEW FEATURES:

* GCP KMS-based Auto Unseal
* Raft-based Vault Backend
* Enable GCR and KMS in the Vault GCP project with service accounts
* CircleCI config to push `omgnetwork/vault` images into GCR

REFACTOR:

* updated `VERSION` file to `0.0.6`
* removal of the unsealer Vault server
* clean Helm and Kubernetes from the infrastructure Terraform scripts
* Helm and GCP are now separate deployments
* cleaned firewall rules in the Vault infrastructure Terraform scripts
* use golang 1.14 as the builder
* stopped using `-dev` mode - use file backend to support snapshotting
  - [x] `unseal.json` holds the keys
  - [x] the Vault data is at `/vault/config/data` 
* Update to github.com/ethereum/go-ethereum v1.9.16
* Removed redundant types from array, slice or map composite literals.
  - [x] `&framework.Path`
  - [x] `&framework.FieldSchema`
* Remove `activateChildChain`
* Wallet Smoke Test 
  - [x] Remove test of `activateChildChain`.
* Re-generate Plasma bindings using v1.9.16 `abigen`
* Update to hashicorp/vault v1.5.2
* Use official hashicorp/vault helm chart
* Removed the local copy of the helm chart
* Standardize GCP resource names to be of the form omgnetwork-<resource>

BUG FIXES:

N/A
## 0.0.5 (January 18, 2020)

NEW FEATURES:

N/A

IMPROVEMENTS:

* Wallet Smoke Test 
  - [x] Execute test of `activateChildChain`.
* Re-generate Plasma bindings

BUG FIXES:

N/A

## 0.0.4 (November 17, 2019)

NEW FEATURES:

* Remove Export JSON Keystore
* Add k8s Example in examples/k8s Showing Integration of k8s Clients and Vault
  - [x] Uses minikube
  - [x] Integrates with existing testbed (`make run`)
  - [x] Shows steps needed to enable k8s auth in Vault

IMPROVEMENTS:

* Wallet Smoke Test 
  - [x] Remove test for Export JSON Keystore from Account
* Document Networking Recommendations
* Refine Plamsa Contract integration
  - [x] Remove Set Authority

BUG FIXES:

* Removed imports of `gitlab.com/shearline-gateway`

## 0.0.3 (November 10, 2019)

NEW FEATURES:

* Implement Plamsa Contract integration
  - [x] Submit Block
  - [x] Set Authority
  - [x] Submit Deposit Block
  - [x] Activate Child Chain
* Added Smoke Test for Plasma
  - [x] Truffle docker container
  - [x] Pull latest from OmiseGO plasma-contracts
  - [x] Builds and Deploys 
  - [x] Integrates with Ganache and Vault in `make run` for full integration test
* Added Docs
  - [x] Uses Sphinx and sphinx rtd theme
  - [x] Captured high level design Q & A
  - [x] Described Vault cluster architecture

IMPROVEMENTS:

* Separated Smoke Tests
* Wallet Smoke Test 
  - [x] Configure Mount
  - [x] Create Wallet (BIP44) Without Mnemonic
  - [x] Create Wallet (BIP44) With Mnemonic
  - [x] List Wallets
  - [x] Create New Account
  - [x] Check Account Balance
  - [x] Transfer ETH
  - [x] Sign Raw TX
  - [x] Sign Raw TX (Encoded)
  - [x] Export JSON Keystore from Account
* Smoke Test for Whitelisting
  - [x] Whitelist Address at an Account
  - [x] Whitelist Address at a Wallet
  - [x] Whitelist Address Globally
* Smoke Test for Blacklisting
  - [x] Blacklist Address at an Account
  - [x] Blacklist Address at a Wallet
  - [x] Blacklist Address Globally
* Smoke Test for ERC20
  - [x] Deploy Contract (FixedSupplyToken)
  - [x] Total Token Supply
  - [x] Token Balance
  - [x] Transfer Token
* Smoke Test for Plasma
  - [x] Submit Block
  - [x] Set Authority
  - [x] Submit Deposit Block
  - [x] Activate Child Chain

BUG FIXES:

* N/A

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
