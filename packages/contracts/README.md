# plasma-contracts
[![Build Status](https://travis-ci.org/plasma-group/plasma-contracts.svg?branch=master)](https://travis-ci.org/plasma-group/plasma-contracts)

`plasma-contracts` is the set of smart contracts written in Vyper for the Plasma Group series of projects. It includes an implementation of a Plasma Cash variant, and a registry contract for discovering Plasma chains. This repo is used for compiling those contracts--If you don't need any modifications, you can spin up your own with [`plasma-chain-operator`](https://github.com/plasma-group/plasma-chain-operator) or try out other chains with [`plasma-js-lib`](https://github.com/plasma-group/plasma-js-lib).

## Contributing
If you're looking to contribute to `plasma-contracts`, you're in the right place. Welcome!

### Contributing Guide and CoC
Plasma Group follows a [Contributing Guide and Code of Conduct](https://github.com/plasma-group/plasma-utils/blob/master/.github/CONTRIBUTING.md) adapted slightly from the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct.html). All contributors are expected to read through this guide. We're here to cultivate a welcoming and inclusive contributing environment, and every new contributor needs to do their part to uphold our community standards.

### Requirements and Setup
The first step is cloning this repo.  Via https:
```
$ git clone https://github.com/plasma-group/plasma-contracts.git
```
or ssh:
```
$ git clone git@github.com:plasma-group/plasma-contracts.git
```

#### Node.js
`plasma-contracts` is tested with [`Node.js`](https://nodejs.org/en/) and has been tested on the following versions of Node:

- 11.6.0

If you're having trouble getting `plasma-contracts` tests running, please make sure you have one of the above `Node.js` versions installed.

#### Packages
`plasma-contracts` makes use of several `npm` packages.

Install all required packages with:

```
$ npm install
```
#### Python and Vyper
`plasma-contracts` is written in Vyper, a pythonic Ethereum smart contract language. You'll need [Python 3.6 or above](https://www.python.org/downloads/) to install Vyper.

We reccomend setting up a [virtual environment](https://cewing.github.io/training.python_web/html/presentations/venv_intro.html) instead of installing globally:
```
python3 -m venv venv
```
To activate:
```
$ source venv/bin/activate
```
Install Vyper:
```
pip3 install vyper
```
Your `venv` must be activated whenever testing or otherwise using Vyper, but it will break the `npm install`, so be sure to `$ deactivate` if you still need to do that and reactivate afterwards.

### Running Tests
`plasma-contracts` makes use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Run all tests with:

```
$ npm test
```
So that Python and Vyper aren't requirements for our other components, we do include a `compiled-contracts` folder which contains JS exports of the bytecode and ABI. Compilation is done automatically before testing.
