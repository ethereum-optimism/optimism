============
Contributing
============
Welcome! A huge thank you for your interest in contributing to Plasma Group.
Plasma Group is an open source initiative developing a simple and well designed plasma_ implementation.
If you're looking to contribute to ``plasma-core``, you're in the right place!
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

If you're a new contributor to ``plasma-core``, please read through the following information.
These resources will help you get started and will help you better understand what we're building.

Contributing Guide and Code of Conduct
======================================
Plasma Group follows a `Contributing Guide and Code of Conduct`_ adapted slightly from the `Contributor Covenant`_.
**All contributors are expected to read through this guide.**
We're here to cultivate a welcoming and inclusive contributing environment.
Every new contributor needs to do their part to uphold our community standards.

Getting Started as a Contributor
================================
Design and Architecture
-----------------------
Before you start contributing, please read through our `Architecture`_ document.
This will give you a high-level understanding of what ``plasma-core`` is and what ``plasma-core`` isn't.

Requirements and Setup
----------------------
Node.js
~~~~~~~
``plasma-core`` is a `Node.js`_ application.
You'll need to install ``Node.js`` (and it's corresponding package manager, ``npm``) for your system before continuing.

``plasma-core`` has been tested on the following versions of Node:

- 10.14.2

If you're having trouble getting a component of ``plasma-core`` running, please try running one of the above versions.

Packages
~~~~~~~~
``plasma-core`` makes use of several ``npm`` packages.

Install all required packages with:

.. code::
   $ npm install

Running Tests
-------------
``plasma-core`` makes use of a combination of Mocha_ (a testing framework) and Chai_ (an assertion library) for testing.

Run all tests with:

.. code::
    $ npm test

**Contributors: remember to run tests before submitting a pull request!**
Code with passing tests makes life easier for everyone and means your contribution can get pulled into this project faster.

.. _plasma: https://plasma.io
.. _Contributing Guide and Code of Conduct: https://github.com/plasma-group/plasma-core/blob/master/.github/CONTRIBUTING.md
.. _Contributor Covenant: https://www.contributor-covenant.org/version/1/4/code-of-conduct.html
.. _Architecture: architecture.html
.. _members of Plasma Group: https://github.com/orgs/plasma-group/people
.. _Node.js: https://nodejs.org/en/
.. _Mocha: https://mochajs.org/
.. _Chai: https://www.chaijs.com/
