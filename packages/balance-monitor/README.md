# Minimum Balance Agent

## Description

A forta agent which detects when a specified account balance is below the
specified threshold.

## Installing and building

`yarn && yarn build`

## Running

1. Copy `.env.example` into `.env` and set the appropriate values.
2. Copy `forta.config.example.json` into `forta.config.json`, and set the RPC endpoint (yes, this is
   duplicated in the .env file).
2. `yarn run start:prod`

## Alerts

- `OPTIMISM-BALANCE-DANGER-[ACCOUNT_NAME]`
  - Fired when the specified account balance is below the configured DANGER threshold
  - Severity is always set to "high"
  - Type is always set to "info"
  - Metadata "balance" field contains amount of wei in account
