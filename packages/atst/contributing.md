# @eth-optimism/atst (WIP)

TODO

WIP sdk and cli tool for the attestation station. Currently in design phase. Any code here is purely experimental proof of concepts.

## Dev setup

To be able to run the cli commands with npx you must first link the node module so npm treats it like an installed package

```
npm link
```

Alternatively `yarn dev` will run the cli

```bash
# these are the same
npx atst --help
yarn dev --help
```

Example of how to read an attestation

```bash
npx atst read --key "optimist.base-uri" --about 0x2335022c740d17c2837f9C884Bfe4fFdbf0A95D5 --creator 0x60c5C9c98bcBd0b0F2fD89B24c16e533BaA8CdA3
```
