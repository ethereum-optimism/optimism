# @eth-optimism/ci-builder

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
