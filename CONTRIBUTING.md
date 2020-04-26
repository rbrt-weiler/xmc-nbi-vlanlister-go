# Contributing to XMC NBI VlanLister (Go)

Everyone is welcome to contribute to the project. Contributions may be anything that drives the project forward, with

* bug reports,
* suggestions and
* updating code

being the basics.

## Bug Reports

_TL;DR: Is it really a bug? --> Has nobody else already reported it? --> Does it still exist in the most current version? --> File a helpful issue._

If you encouter something that classifies as a bug for you, please ensure that it really is a bug: A currently implemented function, suggested by the accompanying documentation or the usage message, that is not working as defined.

Should your finding indeed be a bug as by above definition, please have a look at the [open issues](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/issues) and check if someone else has already filed an issue regarding the bug you have encountered. If an issue already exists, feel free to comment on it, but do not file a new issue.

If no issue exists that describes the bug you have encountered, please ensure that you are using the most current version of the software that is available. Try the [latest stable version](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/tree/stable), followed by - if the bug still exists in the latest stable version - the [latest development version](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/tree/master) from the master branch.

In case that the latest working version of the software still cotains the bug you have found, please [file an issue](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/issues/new). When creating the new issue, adhere to the following guidelines:

* The title SHOULD already give a hint on what functionality is broken.
* The description MUST contain a description of what you have tried to accomplish and how you wanted to accomplish it.
* That description MUST be extensive and precise enough to reproduce your activities.
* The description MUST contain all relevant version numbers, at least:
  * Version of the software that showed the buggy behaviour.
  * Version of the XMC installation that was interacted with.
* The description SHOULD cotain a statement that you have followed above test instructions.
  * For absolute clarity, please include Go version and commit IDs.

After the bug report has been filed, it will be reviewed by the code owner(s). All further communication will be handled via the issue comments.

## Suggestions

_TL;DR: Is the functionality missing from the latest development version? --> Has nobody else already suggested it? --> File a helpful issue._

If some specific functionality is missing from the [latest development version](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/tree/master) of the software that you would like to see implemented, head over to the issues and review the [issues labeled Idea](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/issues?scope=all&state=all&label_name[]=Idea).

In case someone else has already suggested the functionality you are looking for, feel free to comment on it, but do not file a new issue.

Should there be no issue suggesting the functionality you are looking for, go ahead and [file an issue](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/issues/new). When creating the new issue, adhere to the following guidelines:

* The title SHOULD already give a hint on what functionality shall be implemented.
* The description MUST contain a description of what functionality exactly you are looking for.
  * Feel free to go into implementation details like CLI arguments and expected output.
* That description MUST be extensive and precise enough for a regular user to understand your intentions and the outcome.

After the suggestion has been filed, it will be reviewed by the code owner(s). All further communication will be handled via the issue comments.

## Updating Code

_TL;DR: Fork the repository. --> Develop bugfix/feature in own branch. --> Send merge request against master._

Whether it is a bug or a feature, if you are able to satisfy your own needs by coding you are welcome to directly contribute your code to the project. Start by [forking the repository](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/forks/new). Once you have your own fork, develop your bugfix/feature in there and finish by sending a merge request.

Here are some general guidelines for contributing code to the project:

* The master branch is where development starts and ends.
* Your master branch should always be up-to-date with the upstream master branch.
* Every bugfix/feature should be developed in its own branch to simplify merge requests.
* Merge requests must be filed against the master branch; merge requests that target other branches will be dismissed.
* The preferred way to develop bugfixes/features is by using [Visual Studio Code Remote Development Containers](https://code.visualstudio.com/docs/remote/containers); a config is included with the project.

Please keep in mind that every line of code contributed to the project will be licensed under [the project's license](https://gitlab.com/rbrt-weiler/xmc-nbi-vlanlister-go/-/blob/master/LICENSE). After receiving the merge request, the code owner(s) will review it. All further communication will be handled via the merge request comments.
