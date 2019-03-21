===================
GitHub Interactions
===================

Most of our interactions with community members happen through GitHub.
As a result, it's important that the key GitHub interactions are well planned out.
This section describes the most important interactions (Bug Reports, Feature Requests, and Pull Requests) and provides basic guidelines for how to best work with community members.

Remember that everyone who contributes to our projects is taking time out of their day.
Always thank people for helping out, even if you're closing out an issue for being a duplicate of something else!
The golden rule is the best rule :-).

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

2. **Get to the source of the issue.**

Next is getting to the source of the issue.
This is where things get fun!
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

3. **Try to give people the tools to solve problems themselves.**

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

Feature Requests
================
1. **Clarify the exact parameters of the feature request.**

Sometimes contributors are already very familiar with our codebase/functionality and have a clear idea of what new features they'd like.
Other times contributors have a general idea of what they'd like, but we still need to figure out exactly how the feature will work.
It's important to start off by clarifying exactly how the new feature will work.
Clear feature requests are the best way to make sure that we're implementing exactly what was requested.
We definitely don't want to spend a lot of time building something out that isn't useful!

2. **Figure out what changes would need to be made for the new feature.**

Understanding the scope and impact of any changes is necessary for figuring out a timeline.
Once you've clarified exactly what feature is being requested, you can start figuring out what needs to be changed in order to add the feature.
If you're familiar with the codebase, this is a great time to point to the files that would need to be changed.
Otherwise, feel free to tag any other contributors who might have a better idea about the problem!

3. **Figure out a timeline for the new feature.**

We want to make sure that contributors who request new features get a timeline so they know how long they'll need to wait.
Contributors might otherwise turn to another project with a more explicit roadmap.
Try to guesstimate the amount of time that a feature will take to complete.
Also think about where the feature fits in with other features because it might make more sense to release several updates simultaneously.
Definitely discuss with other contributors if you're not sure on an exact timeline.

4. **Keep people updated with the status of a new feature.**

Figuring out an exact timeline is more of an art than a science.
There are always unexpected things that might speed up (or slow down!) the addition of a new feature.
It's important that we keep a feature request thread updated with the latest work on a feature.

Duplicate Requests
==================
1. **Make sure the request is actually a duplicate.**

Sometimes we get duplicate bug reports or feature requests.
This means that someone has created an issue that's already been created before.
Duplicates are easy to handle well but they're also easy to handle poorly.

The very first thing you should do is make sure that the issue is actually a duplicate!
Sometimes people submit very similar issues that have subtle differences. 
These subtle differences can actually have a huge impact on the required fixes.

2. **Be nice about it!**

Contributors who submit duplicates still did the same amount of work as the contributor who submitted the original issue.
It's disappointing realizing that you did duplicate work, so make sure that they understand we still appreciate their work immensely.
Maintainers too often treat duplicates like throwaway issues and close them without much interaction.
Thank contributors for their work and direct them to the new thread so they can continue to help out.

3. **Close the duplicate issue and label it.**

This is the last step!
Make sure to close the duplicate (but not before being nice) and label it as a duplicate for the future.
