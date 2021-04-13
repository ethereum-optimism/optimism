# Changelog

## 0.1.9

### Patch Changes

- 3b00b7c: bump private package versions to try triggering a tag

## 0.1.8

### Patch Changes

- 6cbc54d: allow injecting L2 transaction and block context via core-utils (this removes the need to import the now deprecated @eth-optimism/provider package)
- Updated dependencies [6cbc54d]
  - @eth-optimism/core-utils@0.2.0
  - @eth-optimism/contracts@0.2.2

## v0.1.3

- Add tx resubmission logic
- Log when the batch submitter runs low on ETH

## v0.1.2

Adds mnemonic config parsing

## v0.1.1

Final fixes before minnet release.

- Add batch submission timeout
- Log sequencer address
- remove ssh

## v0.1.0

The inital release
