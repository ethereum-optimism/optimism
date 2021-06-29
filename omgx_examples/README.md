# OMGX First Steps

## 1. To deploy on a LOCAL system

First, spin up a local system as usual:

```bash
$ cd optimism
$ yarn clean
$ yarn
$ yarn build
$ cd ops
```
Make sure you have the docker app running!

```bash
$ docker-compose down -v
$ docker-compose build
$ docker-compose up
```

Second, in the example folder run:

```bash
$ yarn
$ yarn compile
$ yarn compile:ovm
```

Now, see if it works

```bash
$ yarn test:integration:ovm
```

## 2. To deploy on RINKEBY

First, run:

```bash
$ yarn compile:omgx
```

This will compile the contract, Now, see if it works

```bash
$ yarn test:integration:omgx
```

For tests that can deploy the contract:
```bash
/hardhat
```
You can run:
```bash
$ yarn deploy:ovm
$ yarn deploy:omgx
```
To deploy on local and Rinkeby respectively.