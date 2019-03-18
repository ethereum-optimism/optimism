===============
Getting Started
===============
Hello! If you're looking to build your first plasma chain application, you're in the right place.

``@pigi/plasma-js`` is a JavaScript library that makes it easy for you to interact with plasma chains.
This includes things like making transactions, querying balances, querying blocks, and a lot more.

Adding @pigi/plasma-js
======================
There are a few simple ways to add ``@pigi/plasma-js`` to your project.

npm
---
If you're working with a project that supports npm_ imports, you can install ``@pigi/plasma-js`` with ``npm``:

.. code::
   npm install --save @pigi/plasma-js

Then you'll be able to import ``Plasma`` in your project:

.. code:: javascript
   const PlasmaClient = require('@pigi/plasma-js')

Browser
-------
You can also import ``@pigi/plasma-js`` with a ``<script>`` tag:

.. code:: html
   <script src="https://raw.githubusercontent.com/plasma-group/@pigi/plasma-js/master/dist/@pigi/plasma-js.min.js" type="text/javascript"></script>

This will give you access to a window variable:

.. code:: javascript
   const PlasmaClient = window.PlasmaClient

.. _npm: https://www.npmjs.com/
