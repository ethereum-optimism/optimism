=========
Reference
=========
This page provides a series of miscellaneous reference articles that can be helpful when installing Plasma Group components.

Running a Terminal
==================
Before you keep going, it's probably good to become familiar with using the terminal on your computer.
Here are some resources for getting started:

- Windows: `Command Prompt: What It Is and How to Use It`_
- MacOS: `Introduction to the Mac OS X Command Line`_
- Linux: `How to Start Using the Linux Terminal`_

Installing Git
==============
``git`` is an open source version control system.
You don't really need to know how it works, but you *will* need it in order to install most Plasma Group components.

Windows
-------
Atlassian has a `good tutorial`_ on installing ``git`` on Windows.
It's basically just installing an ``.exe`` and running a setup wizard.

MacOS
-----
Installing ``git`` on a Mac is `pretty easy`_.
You basically just need to type ``git`` into your terminal.
If you have ``git`` installed, you'll see a bunch of output.
Otherwise, you'll get a pop-up asking you to install some command-line tools (including ``git``).

Linux
-----
Installing ``git`` on Linux is also pretty easy.
However, the exact install process depends on your distribution.
`Here's a guide`_ for installing ``git`` on some popular distributions.

Installing Node.js
==================
Most of the Plasma Group apps are built in JavaScript and make use of a tool called Node.js_.
In order to run our tools, you'll need to make sure that you’ve got ``Node.js`` installed.

Here's a list of ways to install ``Node.js`` on different operating systems:

Windows
-------
If you're on a windows computer, you can download the latest Long-term Support (LTS) version of ``Node.js`` `here`_.
You'll just need to install the ``.msi`` file that ``Node.js`` provides and restart your computer.

MacOS
-----
You have some options if you want to install ``Node.js`` on a Mac.
The simplest way is to download the ``.pkg`` file from the ``Node.js`` `downloads page`_.
Once you've installed the ``.pkg`` file, run this command on your terminal to make sure everything is working properly:

.. code-block:: console

    node -v

If everything is working, you should see a version number pop up that looks something like this:

.. code-block:: console

    v10.15.1

Homebrew
~~~~~~~~
**Note**: If you've already installed ``Node.js`` with the above steps, you can skip this section!

You can also install ``Node.js`` using Homebrew_.
First, make sure Homebrew is up to date:

.. code-block:: console

    brew update

Now just install ``Node.js``:

.. code-block:: console

    brew install node

Linux
-----
There are different ways to install ``Node.js`` depending on your Linux distribution.
`Here's an article`_ that goes through installing ``Node.js`` on different distributions.

.. _Command Prompt\: What It Is and How to Use It: https://www.lifewire.com/command-prompt-2625840
.. _Introduction to the Mac OS X Command Line: https://blog.teamtreehouse.com/introduction-to-the-mac-os-x-command-line
.. _How to Start Using the Linux Terminal: https://www.howtogeek.com/140679/beginner-geek-how-to-start-using-the-linux-terminal/
.. _good tutorial: https://www.atlassian.com/git/tutorials/install-git#windows
.. _pretty easy: https://git-scm.com/book/en/v2/Getting-Started-Installing-Git
.. _Here\'s a guide: https://gist.github.com/derhuerst/1b15ff4652a867391f03#file-linux-md
.. _Node.js: https://nodejs.org/en/
.. _here: https://nodejs.org/en/download/
.. _downloads page: https://nodejs.org/en/download/
.. _Homebrew: https://brew.sh/
.. _Here\'s an article: https://nodejs.org/en/download/package-manager/
