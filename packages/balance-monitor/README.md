# Minimum Balance Agent

## Description

A forta agent which detects when a specified account balance is below 0.5 ETH

## Running

1. Copy `.env.example` into `.env` and set the values as desired.
2. `yarn run start:prod`


## Alerts

- `OPTIMISM-BALANCE-WARNING-[ACCOUNT_NAME]`
  - `ACCOUNT_NAME` is either `SEQUENCER` or `PROPOSER`
  - Fired when the specified account balance is below the configured WARNING threshold
  - Severity is always set to "info"
  - Type is always set to "info"
  - Metadata "balance" field contains amount of wei in account

- `OPTIMISM-BALANCE-DANGER-[ACCOUNT_NAME]`
  - Fired when the specified account balance is below the configured DANGER threshold
  - Severity is always set to "high"
  - Type is always set to "info"
  - Metadata "balance" field contains amount of wei in account
