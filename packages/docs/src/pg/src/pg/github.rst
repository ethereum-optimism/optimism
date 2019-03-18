===================
GitHub Interactions
===================

Most of our interactions with community members happen through GitHub.
As a result, it's important that the key GitHub interactions are well planned out.
This section describes the most important interactions (Bug Reports, Feature Requests, and Pull Requests) and provides basic guidelines for how to best work with community members.

Bug Reports
===========
1. **If a potential active contributor submits an issue, give them the resources to become an active contributor.**

A lot of the time, users submitting an issue aren’t familiar with the underlying code.
They're probably submitting an issue because they can’t find an easy fix!
If you just tell the user that you’re "on it" or something similar, you’re likely making the situation worse.

First, thank the contributor for reporting the issue in a timely manner.
They’re taking time out of their day to report a problem and, if they’re like a lot of us, it’s probably something that’s caused them a headache.
A quick acknowledgment will let them know that you’re available, responsive, and are taking their issue seriously.

If the user hasn’t provided enough information, make sure to politely ask for information that you think might be relevant.
Always make sure the user provides steps to reproduce the problem so that it can be more easily solved.
Unless you know otherwise, never assume that the user is particularly technical.
Help them out by giving them the exact commands and relevant output that’ll help solve the issue.

.. todo::

    Create a basic diagnostics guide that helps users figure out what's going on with their projects.

Next is getting to the source of the issue.
Now this is where things get fun!
The first thing you’ll want to do is to assess the general difficulty of the underlying problem.
Using the steps to reproduce, try to locate the problematic areas of the software.
Is it an encoding issue?
Is the wrong thing being passed to another function?
Is something undefined when it shouldn’t be?
Is it a problem that’ll require fixes in ten different places?

Depending on the difficulty of the problem, you’ll want to take different next steps.
If the problem is relatively simple, great!
This is an awesome opportunity to convert the potential contributor into an active one.
If a problem is simple, unless the it's absolutely critical and needs an immediate fix, your top priority should be to give someone else the tools to solve the problem.
Remember, letting potential contributors get their hands into some code is the best way to convert them to active contributors.

Giving someone the tools to solve the issue for themselves is a multi-step process.
First, you’ll want to provide the user with all the necessary background information.
Explain the different relevant components and then explain what you think is probably the general cause of the issue.
Feel free to tag another contributor if you think they'd have a better understanding of the problem.
Your next steps depend on whether you know exactly what's causing the issue.

If you don't know exactly what's causing the issue, this is a good opportunity to ask a potential contributor if they'd like to step in and solve the issue!
This might be the person who submitted the issue or it might be another contributor that you tag.
If it's not the person who submitted the issue, try to think about which contributor would most benefit from working on the issue. 

If you already know what's causing the issue, try to explain the cause in a very detailed manner.
You can even go as far as pointing out specific lines that are causing problems or sketching out a potential solution.
You don't necessarily want to entirely solve the problem for someone else, but you do want to give enough leads if possible.
