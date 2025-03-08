![](./assets/EigenDA_TextLogo_White.svg)

# EigenDA powered Optimism-Fork

[![golang](https://github.com/Layr-Labs/optimism/actions/workflows/test-golang.yml/badge.svg)](https://github.com/Layr-Labs/optimism/actions/workflows/test-golang.yml)
[![kurtosis](https://github.com/Layr-Labs/optimism/actions/workflows/kurtosis-devnet.yml/badge.svg)](https://github.com/Layr-Labs/optimism/actions/workflows/kurtosis-devnet.yml)

This is repo contains our fork of [optimism](https://github.com/ethereum-optimism/optimism) to support EigenDA.

- [EigenDA powered Optimism-Fork](#eigenda-powered-optimism-fork)
  - [EigenDA Proxy](#eigenda-proxy)
  - [Fork Features](#fork-features)
    - [1. High Throughput (large parallel blobs)](#1-high-throughput-large-parallel-blobs)
    - [2. Failover (for Liveness)](#2-failover-for-liveness)
    - [3. Security (for Safety)](#3-security-for-safety)
  - [Testing](#testing)
    - [CI](#ci)
    - [Unit Tests](#unit-tests)
    - [op-e2e Tests](#op-e2e-tests)
    - [Kurtosis Devnet Tests](#kurtosis-devnet-tests)
  - [Releases and Branching Strategy](#releases-and-branching-strategy)

## EigenDA Proxy

OP's altda spec has both op-batcher and op-nodes interface with AltDA layers via a [da-server](https://specs.optimism.io/experimental/alt-da.html#da-server). EigenDA's implementation of the da-server is called the [EigenDA Proxy](https://github.com/Layr-Labs/eigenda-proxy). The proxy hides EigenDA's async grpc API behind a simple POST/GET sync (blocking) REST API.

## Fork Features

There are 3 important features for any rollup:
1. Performance
2. Liveness
3. Safety

The upstream code in optimism's repo currently does not support these features for altda rollups. The goal of our fork is to provide these for downstream altda rollups. We will try to upstream as many changes as possible, but the op-team has stopped being receptive to our PRs since the pectra upgrade.

We describe below the current feature-set of the upstream altda code. See release notes for the latest features.

### 1. High Throughput (large parallel blobs)

Because POSTs to the EigenDA Proxy are blocking (see [EigenDA Proxy](#eigenda-proxy) section), the throughput which a rollup can achieve is limited by the number and size of parallel blobs it can submit. The upstream code supports [parallel blobs submissions](https://github.com/ethereum-optimism/optimism/pull/11698) pre-holocene, but the [Holocene strict ordering rules](https://specs.optimism.io/protocol/holocene/derivation.html) have broken that implementation.

We will implement a new parallel blobs submission mechanism which is compatible with the Holocene strict ordering rules, and also enable submitting large blobs (EigenDA allows blobs up to 16MiB currently).

### 2. Failover (for Liveness)

The upstream altda code does not support failover. If the EigenDA network goes down, the rollup will be stuck.

We will implement a failover mechanism to allow the rollup to continue processing transactions even if the EigenDA network is down.

### 3. Security (for Safety)

The upstream derivation pipeline and challenger code does not currently support the EigenDA security model.

Because making altda fraud proofs secure is very involved, we have opted to first secure zk integrations like [op-succinct](https://github.com/succinctlabs/op-succinct) and [risc0-kailua](https://github.com/risc0/kailua) by using [op-rs](https://op-rs.github.io/kona/)'s stack. See our [Hokulea](https://github.com/Layr-Labs/hokulea) repo for the latest on this.

## Testing

### CI

OP uses circleci for CI, but we migrated to github actions for this fork. The unit and op-e2e tests are purely golang and so run as part of the [test-golang.yml](./.github/workflows/test-golang.yml) github workflow, whereas the kurtosis tests are run as part of the [test-kurtosis.yml](./.github/workflows/test-kurtosis.yml) workflow.

### Unit Tests

For each feature we add simple unit tests where fits.

### op-e2e Tests

We also add integration tests using op-e2e's framework. These tests are very useful as they are run purely in golang in a single process with very fast block times, but they are limited in that proxy is not spin up and the batcher available there is only a fake.

### Kurtosis Devnet Tests

For full e2e tests we leverage optimism's [kurtosis devnet](./kurtosis-devnet/README.md). See the config file to spin up a devnet with eigenda-proxy in [memstore](./kurtosis-devnet/eigenda-memstore.yaml) mode and [holesky](./kurtosis-devnet/eigenda-holesky.yaml) mode, as well as the available eigenda group commands in the [justfile](./kurtosis-devnet/justfile):
```sh
$ just --list
Available recipes:
    ...

    [eigenda]
    eigenda-holesky-devnet-clean
    eigenda-holesky-devnet-start                     # EigenDA devnet that uses eigenda-proxy connected to eigenda holesky testnet network
    eigenda-memstore-devnet-clean
    eigenda-memstore-devnet-configs
    eigenda-memstore-devnet-failback
    eigenda-memstore-devnet-failover                 # to failover to ethDA. Use `eigenda-memstore-devnet-failback` to revert.
    eigenda-memstore-devnet-grafana
    eigenda-memstore-devnet-restart-batcher          # Restart batcher with new flags or image.
    eigenda-memstore-devnet-start                    # EigenDA devnet that uses the eigenda-proxy in memstore mode (simulates an eigenda network but generates random certs)
    eigenda-memstore-devnet-sync-status
    eigenda-memstore-devnet-test
```

## Releases and Branching Strategy

We maintain an `eigenda-develop` branch which is our main development branch which keeps a linear history containing new feature work, fixes, as well as upstream merges. To make downstream integrations easier, we create release-specific branches which contain cleaned up history of commits on top of a specific upstream release. For example, the second eigenda-fork release in the below picture would be named `op-batcher/v1.11.2-eigenda.2`, and will consist of a cleaned-up history of commits (one per feature/service pair) on top of the upstream [op-batcher/v1.11.2](https://github.com/ethereum-optimism/optimism/releases/tag/op-batcher%2Fv1.11.2) release. We will strive to make our releases on top of op [production releases](https://github.com/ethereum-optimism/optimism?tab=readme-ov-file#production-releases), unless an urgent fix is needed.

![](./assets/fork-branching-and-releases.png)
