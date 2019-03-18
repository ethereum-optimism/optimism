============
Contributing
============
Welcome! A huge thank you for your interest in contributing to Plasma Group.
Plasma Group is an open source initiative developing a simple and well designed plasma_ implementation.
If you're looking to contribute to a Plasma Group project, you're in the right place!
It's contributors like you that make open source projects work, we really couldn't do it without you.

We don't just need people who can contribute code.
We need people who can run this code for themselves and break it.
We need people who can report bugs, request new features, and leave helpful comments.
**We need you!**

We're always available to answer your questions and to help you become a contributor!
You can reach out to any of the `members of Plasma Group`_ on GitHub, or send us an email at contributing@plasma.group.

Here at Plasma Group we're trying to foster an inclusive, welcoming, and accessible open source ecosystem.
The best open source projects are those that make contributing an easy and rewarding experience.
We're trying to follow those best practices by maintaining a series of resources for contributors to Plasma Group repositories.

If you're a new contributor to ``@pigi/core``, please read through the following information.
These resources will help you get started and will help you better understand what we're building.

Contributing Guide and Code of Conduct
======================================
Plasma Group follows a `Contributing Guide and Code of Conduct`_ adapted slightly from the `Contributor Covenant`_.
**All contributors are expected to read through this guide.**
We're here to cultivate a welcoming and inclusive contributing environment.
Every new contributor needs to do their part to uphold our community standards.

Getting Started as a Contributor
================================
Requirements and Setup
----------------------
Cloning the Repo
~~~~~~~~~~~~~~~~
Before you start working on a Plasma Group project, you'll need to clone our GitHub repository:

.. code::

    git clone git@github.com:plasma-group/pigi.git

Now, enter the repository.

.. code::

    cd pigi

Node.js
~~~~~~~
Most of the Plasma Group projects are `Node.js`_ applications.
You'll need to install ``Node.js`` for your system before continuing.
We've provided a `detailed explanation of now to install Node.js`_ on Windows, Mac, and Linux.

Yarn
~~~~
We're using a package manager called `Yarn`_.
You'll need to `install Yarn`_ before continuing.

Installing Dependencies
~~~~~~~~~~~~~~~~~~~~~~~
``@pigi`` projects make use of several external packages.

Install all required packages with:

.. code::

   yarn install


Building
========
``@pigi`` provides convenient tooling for building a package or set of packages.

Build all packages:

.. code::

    yarn run build

Build a specific package or set of packages:

.. code::

    PKGS=your,packages,here yarn run build

Linting
=======
Clean code is the best code, so we've provided tools to automatically lint your projects.

Lint all packages:

.. code::

    yarn run lint

Lint a specific package or set of packages:

.. code::

    PKGS=your,packages,here yarn run lint

Automatically Fixing Linting Issues
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
We've also provided tools to make it possible to automatically fix any linting issues.
It's much easier than trying to fix issues manually.

Fix all packages:

.. code::

    yarn run fix

Fix a specific package or set of packages:

.. code::

    PKGS=your,packages,here yarn run fix

Running Tests
=============
``@pigi`` projects usually makes use of a combination of `Mocha`_ (a testing framework) and `Chai`_ (an assertion library) for testing.

Run all tests:

.. code::
    
    yarn test

Run tests for a specific package or set of packages:

.. code::

    PKGS=your,packages,here yarn test

**Contributors: remember to run tests before submitting a pull request!**
Code with passing tests makes life easier for everyone and means your contribution can get pulled into this project faster.

.. _`plasma`: https://plasma.io
.. _`Contributing Guide and Code of Conduct`: https://github.com/plasma-group/pigi/blob/master/.github/CONTRIBUTING.md
.. _`Contributor Covenant`: https://www.contributor-covenant.org/version/1/4/code-of-conduct.html
.. _`Architecture`: architecture.html
.. _`members of Plasma Group`: https://github.com/orgs/plasma-group/people
.. _`Node.js`: https://nodejs.org/en/
.. _`Mocha`: https://mochajs.org/
.. _`Chai`: https://www.chaijs.com/
.. _`detailed explanation of now to install Node.js`: https://plasma-core.readthedocs.io/en/latest/reference.html#installing-node-js
.. _`Yarn`: https://yarnpkg.com/en/
.. _`install Yarn`: https://yarnpkg.com/en/docs/install
