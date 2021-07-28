# OMGX ERC721 Example

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


```


    "deploy": "hardhat deploy",
    "deploy:ovm": "hardhat deploy --network optimism",
    "deploy:omgx": "hardhat deploy --network omgx_rinkeby"