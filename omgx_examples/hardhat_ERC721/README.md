# OMGX ERC20 Example

## 1. Compiling the contracts

First, spin up a local system as usual (see the top-level `Readme.md`). Then, run:

```bash
$ yarn
$ yarn compile       #for your local L1
$ yarn compile:ovm   #for your local L2
$ yarn compile:omgx  #for OMGX Rinkeby
```

## 2. Testing

```bash
$ yarn test:integration:ovm  #for your local L1/L2
$ yarn test:integration:omgx #for OMGX Rinkeby
```

```bash
% yarn test:integration:omgx
yarn run v1.22.10
$ hardhat test --network omgx_rinkeby


  ERC20
    ✓ should have a name (1395ms)
    ✓ should have a total supply equal to the initial supply (1403ms)
    ✓ should give the initial supply to the creator's address (1733ms)
    transfer(...)
      ✓ should revert when the sender does not have enough balance (1970ms)
      ✓ should succeed when the sender has enough balance (5681ms)
    transferFrom(...)
      ✓ should revert when the sender does not have enough of an allowance (1974ms)
      ✓ should succeed when the owner has enough balance and the sender has a large enough allowance (9559ms)


  7 passing (46s)

✨  Done in 48.41s.

```