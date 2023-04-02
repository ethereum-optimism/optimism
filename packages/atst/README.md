<div align="center">
  <br />
  <br />
  <a href="https://optimism.io"><img alt="Optimism" src="https://raw.githubusercontent.com/ethereum-optimism/brand-kit/main/assets/svg/OPTIMISM-R.svg" width=600></a>
  <br />
  <h3>@eth-optimism/atst</h3> The official SDK and cli for Optimism's attestation Station
  <br />
</div>

<p align="center">

<p>
<a href="https://www.npmjs.com/package/@eth-optimism/atst" target="\_parent">
<img alt="" src="https://img.shields.io/npm/dm/@eth-optimism/atst.svg" />
</a>

# atst

atst is a typescript / javascript sdk and cli around AttestationStation

**Visit [Docs](https://community.optimism.io/docs/governance/attestation-station/) for general documentation on AttestationStation.**

## Getting started

Install

```bash
npm install @eth-optimism/atst wagmi @wagmi/core ethers@5.7.0
```

## atst typescript/javascript sdk

The typescript sdk provides a clean [wagmi](https://wagmi.sh/) based interface for reading and writing to AttestationStation.

**See [sdk docs](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/docs/sdk.md) for usage instructions.**

## atst cli

The cli provides a convenient cli for interacting with the AttestationStation contract

![preview](./assets/preview.gif)

**See [cli docs](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/docs/cli.md) for usage instructions.**

## React API

For react hooks we recomend using the [wagmi cli](https://wagmi.sh/cli/getting-started) with the [etherscan plugin](https://wagmi.sh/cli/plugins/etherscan) and [react plugin](https://wagmi.sh/cli/plugins/react) to automatically generate react hooks around AttestationStation.

Use `createKey` and `createValue` to convert your raw keys and values into bytes that can be used in AttestationStation contract calls

Use `parseString`, `parseBool`, `parseAddress` and `parseNumber` to convert values returned by AttestationStation to their correct data type.

For convenience we also [export the hooks here](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/src/index.ts):
- `useAttestationStationAttestation` - Reads attestations with useContractRead
- `useAttestationStationVersion` - Reads attestation version
- `useAttestationStationAttest` - Wraps useContractWrite with AttestationStation abi calling attest
- `usePrepareAttestationStationAttest` - Wraps usePrepare with AttestationStation abi calling attest
- `useAttestationStationAttestationCreatedEvent` - Wraps useContractEvents for Created events

Also some more hooks exported by the cli but these are likely the only ones you need.

## Contributing

Please see our [contributing.md](https://github.com/ethereum-optimism/optimism/blob/develop/CONTRIBUTING.md). No contribution is too small.

Having your contribution denied feels bad. 
Please consider [opening an issue](https://github.com/ethereum-optimism/optimism/issues) before adding any new features or apis.


## Getting help

If you have any problems, these resources could help you:

- [sdk documentation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/docs/sdk.md)
- [cli documentation](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/docs/cli.md)
- [Optimism Discord](https://discord-gateway.optimism.io/)
- [Telegram group](https://t.me/+zwpJ8Ohqgl8yNjNh) 
