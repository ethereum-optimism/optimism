# @eth-optimism/ci-builder

## 0.3.6

### Patch Changes

- 011acf411: Add echidna to ci-builder

## 0.3.5

### Patch Changes

- c44ff357f: Update foundry in ci-builder

## 0.3.4

### Patch Changes

- 8e22c28f: Update geth to 1.10.25

## 0.3.3

### Patch Changes

- 3f485627: Pin slither version to 0.9.0

## 0.3.2

### Patch Changes

- fcfcf6e7: Remove ugly shell hack
- 009939e0: Fix codecov download step

## 0.3.1

### Patch Changes

- 7375a949: Download and verify codecov uploader binary in the ci-builder image

## 0.3.0

### Minor Changes

- 25c564bc: Automate foundry build

## 0.2.4

### Patch Changes

- c6fab69f: Update foundry to fix a bug in coverage generation
- f7323e0b: Upgrade foundry to support consistent storage layouts

## 0.2.3

### Patch Changes

- 9ac88806: Update golang, geth and golangci-lint

## 0.2.2

### Patch Changes

- c666fedc: Upgrade to Debian 11

## 0.2.1

### Patch Changes

- 9bb6a152: Trigger release to update foundry version

## 0.2.0

### Minor Changes

- e8909be0: Fix unbound variable in check_changed script

  This now uses -z to check if a variable is unbound instead of -n.
  This should fix the error when the script is being ran on develop.

## 0.1.2

### Patch Changes

- 184f13b6: Retrigger release of ci-builder

## 0.1.1

### Patch Changes

- 7bf30513: Fix publishing
- a60502f9: Install new version of bash

## 0.1.0

### Minor Changes

- 8c121ece: Update foundry in ci builder

### Patch Changes

- 445efe9d: Use ethereumoptimism/foundry:latest
