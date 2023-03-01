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

atst is a typescript sdk and cli around the attestation station

### Visit [Docs](https://community.optimism.io/docs/governance/attestation-station/) for general documentation on the attestation station!

## Getting started

Install

```bash
npm install @eth-optimism/atst wagmi @wagmi/core ethers@5.7.0
```

## atst typescript sdk

The typescript sdk provides a clean [wagmi](https://wagmi.sh/) based interface for reading and writing to the attestation station

### See [sdk docs](https://github.com/ethereum-optimism/optimism/blob/develop/packages/atst/docs/sdk.md) for usage instructions

## atst cli

The cli provides a convenient cli for interacting with the attestation station contract

TODO put a gif here of using it

## React API

For react hooks we recomend using the [wagmi cli](https://wagmi.sh/cli/getting-started) with the [etherscan plugin](https://wagmi.sh/cli/plugins/etherscan) and [react plugin](https://wagmi.sh/cli/plugins/react) to automatically generate react hooks around the attestation station.

Use `parseAttestationBytes` and `stringifyAttestationBytes` to parse and stringify attestations before passing them into wagmi hooks.

For convenience we also export the hooks here.

`useAttestationStationAttestation` - Reads attestations with useContractRead

`useAttestationStationVersion` - Reads attestation version

`useAttestationStationAttest` - Wraps useContractWrite with attestation station abi calling attest

`usePrepareAttestationStationAttest` - Wraps usePrepare with attestation station abi calling attest

`useAttestationStationAttestationCreatedEvent` - Wraps useContractEvents for Created events

Also some more hooks exported by the cli but these are likely the only ones you need.

## Contributing

Please see our [contributing.md](docs/contributing.md). No contribution is too small.

Having your contribution denied feels bad. Please consider opening an issue before adding any new features or apis
