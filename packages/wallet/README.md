# predicates
[![Coverage Status](https://coveralls.io/repos/github/plasma-group/plasma-utils/badge.svg?branch=master)](https://coveralls.io/github/plasma-group/plasma-utils?branch=master) [![Build Status](https://travis-ci.org/plasma-group/plasma-utils.svg?branch=master)](https://travis-ci.org/plasma-group/plasma-utils) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/deb13b3afcc44244ad3faa8b9be39585)](https://www.codacy.com/app/kfichter/plasma-utils?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=plasma-group/plasma-utils&amp;utm_campaign=Badge_Grade)

`@pigi/predicates` is a set of basic predicates which are critical for the basic functioning of a plasma chain.

## Documentation
Detailed documentation for `predicates` is available on ReadTheDocs: <https://docs.plasma.group/projects/utils/en/latest/.>

## Installation
There are several easy ways to start using `@pigi/predicates`! For now we just describe the node install.

### Node.js
If you're developing a `Node.js` application, you can simply install `@pigi/predicates` via `npm`:

```sh
npm install --save @pigi/predicates
```

## Running the Validator
### Configuration
The validator expects an `.env` file that looks like the `.env.example` in the same location. The idea is that there is some sensitive info there, so `.env` files are specifically ignored from git so that we never accidentally check in credentials

### Running
Make sure the project is built and run:
```sh
./runValidator
```
### Clearing Data
If you'd like to blow away data, just run `yarn clean && yarn build` and run the validator again. The DB for the validator is leveldb, which persists to files in the `/build` directory that get blown away when you `yarn clean`

## Contributing
Welcome! If you're looking to contribute to `@pigi/predicates`, you're in the right place.

### Contributing Guide and CoC
Plasma Group follows a [Contributing Guide and Code of Conduct](https://github.com/plasma-group/plasma-utils/blob/master/.github/CONTRIBUTING.md) adapted slightly from the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct.html).
All contributors are expected to read through this guide.
We're here to cultivate a welcoming and inclusive contributing environment, and every new contributor needs to do their part to uphold our community standards.

