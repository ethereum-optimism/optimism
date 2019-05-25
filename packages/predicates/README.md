# plasma-utils
[![Coverage Status](https://coveralls.io/repos/github/plasma-group/plasma-utils/badge.svg?branch=master)](https://coveralls.io/github/plasma-group/plasma-utils?branch=master) [![Build Status](https://travis-ci.org/plasma-group/plasma-utils.svg?branch=master)](https://travis-ci.org/plasma-group/plasma-utils) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/deb13b3afcc44244ad3faa8b9be39585)](https://www.codacy.com/app/kfichter/plasma-utils?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=plasma-group/plasma-utils&amp;utm_campaign=Badge_Grade)

`plasma-utils` is the set of core utilities for the Plasma Group series of projects.
These utilities can be imported into other projects when necessary or convenient.

## Documentation
Detailed documentation for `plasma-utils` is available on ReadTheDocs: https://docs.plasma.group/projects/utils/en/latest/.

## Installation
There are several easy ways to start using `plasma-utils`! 

### Node.js
If you're developing a `Node.js` application, you can simply install `plasma-utils` via `npm`:

```
$ npm install --save plasma-utils
```

### Browser
If you're developing a browser application, we provide a compressed and minified version of `plasma-utils` that you can include in a `<script>` tag.

```
<script src="https://raw.githubusercontent.com/plasma-group/plasma-utils/master/dist/plasma-utils.min.js" type="text/javascript"></script>
```

## Contributing
Welcome! If you're looking to contribute to `plasma-utils`, you're in the right place.

### Contributing Guide and CoC
Plasma Group follows a [Contributing Guide and Code of Conduct](https://github.com/plasma-group/plasma-utils/blob/master/.github/CONTRIBUTING.md) adapted slightly from the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct.html).
All contributors are expected to read through this guide.
We're here to cultivate a welcoming and inclusive contributing environment, and every new contributor needs to do their part to uphold our community standards.

### Requirements and Setup
#### Cloning the Repo
Before you start working on `plasma-utils`, you'll need to clone our GitHub repository:

```
git clone git@github.com:plasma-group/plasma-utils.git
```

Now, enter the repository.

```
cd plasma-utils
```

#### Node.js
`plasma-utils` is tested and built with [`Node.js`](https://nodejs.org/en/).
Although you **do not need [`Node.js`] to use this library in your application**, you'll need to install `Node.js` (and it's corresponding package manager, `npm`) for your system before contributing.

We've provided a [detailed explanation of now to install `Node.js`](https://docs.plasma.group/en/latest/src/pigi/reference.html#installing-node-js) on Windows, Mac, and Linux.

`plasma-utils` has been tested on the following versions of Node:

- v8
- v9
- v10

If you're having trouble getting a component of `plasma-utils` running, please try installing one of the above versions of `Node.js` and try again.
It's pretty easy to switch `Node.js` versions using `n`.
First, install `n` globally.

```
npm install -g n
```

Next, install your desired verson of `Node.js`, say `v10`:

```
n 10
```

#### Packages
`plasma-utils` makes use of several `npm` packages.

Install all required packages with:

```
$ npm install
```

### Running Tests
`plasma-utils` makes use of a combination of [`Mocha`](https://mochajs.org/) (a testing framework) and [`Chai`](https://www.chaijs.com/) (an assertion library) for testing.

Run all tests with:

```
$ npm test
```

### Building
We're using `gulp` to provide a process to build `plasma-utils` for in-browser usage.

If you'd like to build `plasma-utils` yourself, simply run:

```
$ npm run build
```
