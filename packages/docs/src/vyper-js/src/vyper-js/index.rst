========
Vyper-JS
========

It is a high-level API giving a uniform interface to all compiler versions. The high-level API consists of a single 
function; compile, which expects the Compiler Standard Input and Output JSON.

It also accepts an optional callback function to resolve unmet dependencies. This callback receives a path and must 
synchronously return either an error or the content of the dependency as a string. It cannot be used together with 
callback-based, asynchronous, filesystem access. A walkaround is to collect the names of dependencies, return an error, 
and keep rerunning the compiler until all of them are resolved. Instead of compiling vyper contracts using vyper we are 
now using a js library.

Vyper-JS is a NodeJS library and CLI tool that is used to compile vyper files. It doesn't use the vyper command-line 
compiler; instead it allows one to compile vyper files from inside a javascript app and return nicely formatted JS objects.

Vyper is the actual compiler and it’s written in Python. The python code is compiled to JavaScript using emscripten.

Installing Vyper
================

Prerequisites
-------------

Please refer to the Vyper docs for guidelines on installing the Vyper Compiler.

**Installation**

To Install VyperJS

*NodeJs*

``npm install --save @pigi/vyper-js``

*Usage*

``const vyperjs = require(“@pigi/vyper-js”);``

``const contract = await vyperjs.compile(“/path/to/contract.vy”);``

``console.log(contract);``

``output: {bytecode: “0x……….”, abi: [........., ……...]}``


NB: Installing VyperJS doesn’t install the actual Vyper compiler for you, you can install Vyper via the command

Being able to work with the vyper-js compiler is an important skill when developing for the Ethereum Platform, especially 
when you don’t want to depend on web-based IDEs like Remix. Understanding how to use vyper-js will help you to understand 
the toolchain for developing Smart Contracts.

``mkdir MyProject //create a project folder``

``cd MyProject //change directory``

``npm init //go through the initialization process``

Now the fun part can start! At the root of your project folder, create a JavaScript file called compile.js. This file will 
contain all the code you will need to compile your Vyper Smart Contracts.





