# Minimum Balance Agent

## Description

A forta agent which detects when a specified account balance is below the
specified threshold.

## Running

1. Copy `.env.example` into `.env` and set the values as desired.
2. `yarn run start:prod`


## Alerts

- `OPTIMISM-BALANCE-DANGER-[ACCOUNT_NAME]`
  - Fired when the specified account balance is below the configured DANGER threshold
  - Severity is always set to "high"
  - Type is always set to "info"
  - Metadata "balance" field contains amount of wei in account
