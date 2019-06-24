<img src="https://github.com/plasma-group/branding/blob/master/logos/pg-logo-red.png" width="150px" >

---

[Plasma Group](https://plasma.group/) is an independent non-profit organization that's developing standards for [plasma](https://plasma.io) and beyond.
PG is dedicated to the creation of an open plasma implementation that supports well designed standard data formats and interfaces for the greater Ethereum community.

`@pigi` is the Plasma Group monorepo.
All of the core plasma group projects are hosted inside of the [packages](https://github.com/plasma-group/pigi/tree/master/packages) folder of this repository.

## Packages

| Package                                                             | Version                                                                                                                                     | Description                                                                                                                                            |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------|
| [`@pigi/contracts`](/packages/contracts)                            | [![npm](https://img.shields.io/npm/v/@pigi/contracts.svg)](https://www.npmjs.com/package/@pigi/contracts)                                   | Core Vyper contracts used for the PG plasma chain.                                            |
| [`@pigi/core`](/packages/core)                                      | [![npm](https://img.shields.io/npm/v/@pigi/core.svg)](https://www.npmjs.com/package/@pigi/core)                                             | Core PG plasma chain client modules.                                                          |
| [`@pigi/plasma-js`](/packages/plasma-js)                            | [![npm](https://img.shields.io/npm/v/@pigi/plasma-js.svg)](https://www.npmjs.com/package/@pigi/plasma-js)                                   | JS client library for interacting with PG plasma chains.                                      |
| [`@pigi/predicates`](/packages/predicates)                              | [![npm](https://img.shields.io/npm/v/@pigi/predicates.svg)](https://www.npmjs.com/package/@pigi/predicates)                                     | Predicate contracts & plugins..                                                   |

## Repo Status
[![Build Status](https://travis-ci.org/plasma-group/pigi.svg?branch=master)](https://travis-ci.org/plasma-group/pigi) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/a822ee0425164be586235be45100f7d6)](https://www.codacy.com/app/kfichter/pigi?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=plasma-group/pigi&amp;utm_campaign=Badge_Grade)

## Contributing
Welcome! If you're looking to contribute to the future of plasma, you're in the right place.

### Contributing Guide and Code of Conduct
Plasma Group follows a [Contributing Guide and Code of Conduct](https://github.com/plasma-group/pigi/blob/master/.github/CONTRIBUTING.md) adapted slightly from the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct.html).
All contributors **must** read through this guide before contributing.
We're here to cultivate a welcoming and inclusive contributing environment, and every new contributor needs to do their part to uphold our community standards.

### Requirements and Setup
#### Cloning the Repo
Before you start working on a Plasma Group project, you'll need to clone our GitHub repository:

```sh
git clone git@github.com:plasma-group/pigi.git
```

Now, enter the repository.

```sh
cd pigi
```

#### Node.js
Most of the projects in `@pigi` are [`Node.js`](https://nodejs.org/en/) projects.
You'll need to install `Node.js` for your system before continuing.
We've provided a [detailed explanation of now to install `Node.js`](https://github.com/plasma-group/pigi/blob/c1c70a9ac6fe741fd937b9ca13ee7c1f6f9f4061/packages/docs/src/pg/src/reference/misc.rst#installing-node-js) on Windows, Mac, and Linux.

**Note**: This is confirmed to work on `Node.js v11.6`, but there may be issues on other versions. If you have trouble, please peg your Node.js version to 11.6.

#### Yarn
We're using a package manager called [Yarn](https://yarnpkg.com/en/).
You'll need to [install Yarn](https://yarnpkg.com/en/docs/install) before continuing.

#### Installing Dependencies
`@pigi` projects make use of several external packages.

Install all required packages with:

```sh
yarn install
```

### Building
`@pigi` provides convenient tooling for building a package or set of packages.

Build all packages:

```sh
yarn run build
```

Build a specific package or set of packages:

```sh
PKGS=your,packages,here yarn run build
```

### Linting
Clean code is the best code, so we've provided tools to automatically lint your projects.

Lint all packages:

```sh
yarn run lint
```

Lint a specific package or set of packages:

```sh
PKGS=your,packages,here yarn run lint
```

#### Automatically Fixing Linting Issues
We've also provided tools to make it possible to automatically fix any linting issues.
It's much easier than trying to fix issues manually.

Fix all packages:

```sh
yarn run fix
```

Fix a specific package or set of packages:

```sh
PKGS=your,packages,here yarn run fix
```

### Running Tests
`@pigi` projects usually makes use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Run all tests:

```sh
yarn test
```

Run tests for a specific package or set of packages:

```sh
PKGS=your,packages,here yarn test
```

**Contributors: remember to run tests before submitting a pull request!**
Code with passing tests makes life easier for everyone and means your contribution can get pulled into this project faster.

## Acknowledgements
We'd like to give a big shoutout to [0x](https://0x.org/) and [Nest.js](https://nestjs.com/) for inspiration about the best ways to design this monorepo.
Please check out their respective repos ([0x](https://github.com/0xProject/0x-monorepo) and [Nest.js](https://github.com/nestjs/nest)) if you're looking for other cool projects to work on :blush:!
