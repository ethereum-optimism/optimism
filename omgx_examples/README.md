# OMGX Examples

- [OMGX First Steps](#omgx-first-steps)
  * [1. To spin up a local L1/L2 system](#1-to-spin-up-a-local-l1-l2-system)
  * [2. Chose an example, compile it, deploy it, and test it](#2-chose-an-example--compile-it--deploy-it--and-test-it)

## 1. To spin up a local L1/L2 system 

First, install needed modules:

```bash
$ cd optimism
$ yarn clean
$ yarn
$ yarn build
```

Then, navigate to the `/ops` folder and start the system. Make sure you have the docker app running!

```bash
$ cd ops
$ docker-compose down #only needed if you are currently running a local system
$ docker-compose build
$ docker-compose up -V
```

## 2. Chose an example, compile it, deploy it, and test it

For example, if you would like to deploy a simple ERC20 token using hardhat, navigate to the `hardhat` folder and compile the contract:

```bash
$ cd /omgx_examples/hardhat
$ yarn
$ yarn compile      #compile the examples for the local L1
$ yarn compile:ovm  #compile the examples for the local L2
$ yarn compile:omgx #compile the examples for the OMGX Rinkeby L2
```

Now, see it all work. The `ovm` suffix denotes deploying and running contracts on the L2.

```bash
$ yarn test:integration      #test the examples on the local L1
$ yarn test:integration:ovm  #test the examples on the local L2
$ yarn test:integration:omgx #test the examples on the OMGX Rinkeby L2
```

To deploy the contracts to the various chains,

```bash
$ yarn deploy      #test the examples on the local L1
$ yarn deploy:ovm  #test the examples on the local L2
$ yarn deploy:omgx #test the examples on the OMGX Rinkeby L2
```

Note that not all commands are availible for all the different examples, and the commands will be slightly different fro `waffle` and so forth. See the `package.json` files for example-specific syntax, or the Readme.md in the examples.