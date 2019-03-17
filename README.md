<img src="https://github.com/plasma-group/branding/blob/master/logos/pg-logo-red.png" width="150px" >

---

[Plasma Group](https://plasma.group/) is an independent non-profit organization that's developing standards for [plasma](https://plasma.io) and beyond.
PG is dedicated to the creation of an open plasma implementation that supports well designed standard data formats and interfaces for the greater Ethereum community.

`@pigi` is the Plasma Group monorepo.
All of the core plasma group projects are hosted inside of the [packages](https://github.com/plasma-group/pigi/tree/master/packages) folder of this repository.

## Packages

| Package                                                             | Version                                                                                                                                     | Description                                                                                                                                            |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| [`@pigi/contracts`](/packages/contracts)                            | [![npm](https://img.shields.io/npm/v/@pigi/contracts.svg)](https://www.npmjs.com/package/@pigi/contracts)                                       | Core Vyper contracts used for the PG plasma chain.
| [`@pigi/core`](/packages/core)                                      | [![npm](https://img.shields.io/npm/v/@pigi/core.svg)](https://www.npmjs.com/package/@pigi/core)                                                  | Core PG plasma chain client modules.
| [`@pigi/plasma.js`](/packages/plasma.js)                            | [![npm](https://img.shields.io/npm/v/@pigi/plasma.js.svg)](https://www.npmjs.com/package/@pigi/plasma.js)                                       | JS client library for interacting with PG plasma chains.
| [`@pigi/utils`](/packages/utils)                                    | [![npm](https://img.shields.io/npm/v/@pigi/utils.svg)](https://www.npmjs.com/package/@pigi/utils)                                           | Utilities used in many PG projects.
| [`@pigi/verifier`](/packages/verifier)                              | [![npm](https://img.shields.io/npm/v/@pigi/verifier.svg)](https://www.npmjs.com/package/@pigi/verifier)                                        | State transition execution library for PG plasma chains.
| [`@pigi/vyper-js`](/packages/vyper-js)                              | [![npm](https://img.shields.io/npm/v/@pigi/vyper-js.svg)](https://www.npmjs.com/package/@pigi/vyper-js)                                        | JavaScript bindings for the Vyper compiler.


## Contributing
Welcome! If you're looking to contribute to the future of plasma, you're in the right place.

### Contributing Guide and Code of Conduct
Plasma Group follows a [Contributing Guide and Code of Conduct](https://github.com/plasma-group/pigi/blob/master/.github/CONTRIBUTING.md) adapted slightly from the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct.html).
All contributors **must** read through this guide before contributing.
We're here to cultivate a welcoming and inclusive contributing environment, and every new contributor needs to do their part to uphold our community standards.

### Requirements and Setup
#### Cloning the Repo
Before you start working on a Plasma Group project, you'll need to clone our GitHub repository:

```
git clone git@github.com:plasma-group/pigi.git
```

Now, enter the repository.

```
cd pigi
```

#### Node.js
Most of the projects in `@pigi` are [`Node.js`](https://nodejs.org/en/) projects.
You'll need to install `Node.js` for your system before continuing.
We've provided a [detailed explanation of now to install `Node.js`](https://plasma-core.readthedocs.io/en/latest/reference.html#installing-node-js) on Windows, Mac, and Linux.

#### Yarn
We're using a package manager called [Yarn](https://yarnpkg.com/en/).
You'll need to [install Yarn](https://yarnpkg.com/en/docs/install) before continuing.

#### Installing Dependencies
`@pigi` projects make use of several external packages.

Install all required packages with:

```
yarn install
```

### Building
`@pigi` provides convenient tooling for building a package or set of packages.

Build all packages:

```
yarn run build
```

Build a specific package or set of packages:

```
PKGS=your,packages,here yarn run build
```

### Linting
Clean code is the best code, so we've provided tools to automatically lint your projects.

Lint all packages:

```
yarn run lint
```

Lint a specific package or set of packages:

```
PKGS=your,packages,here yarn run lint
```

#### Automatic Fixing Linting Issues
We've also provided tools to make it possible to automatically fix any linting issues.
It's much easier than trying to fix issues manually.

Fix all packages:

```
yarn run fix
```

Fix a specific package or set of packages:

```
PKGS=your,packages,here yarn run fix
```

### Running Tests
`@pigi` projects usually makes use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Run all tests:

```
yarn test
```

Run tests for a specific package or set of packages:

```
PKGS=your,packages,here yarn test
```

**Contributors: remember to run tests before submitting a pull request!**
Code with passing tests makes life easier for everyone and means your contribution can get pulled into this project faster.

## Credit where credit is due!
We'd like to give a big shoutout to [0x](https://0x.org/) and [Nest.js](https://nestjs.com/) for inspiration about the best ways to design this monorepo.
Please check out their respective repos ([0x](https://github.com/0xProject/0x-monorepo) and [Nest.js](https://github.com/nestjs/nest)) if you're looking for other cool projects to work on :blush:!
