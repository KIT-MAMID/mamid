# Development Workflow

## Git Branches

The `master` branch must work. This invariant holds true by adhering to the following policy:

* `master` is protected, i.e. cannot be pushed to from developer machines
* development happens in sensibly named feature-branches
* to merge a feature-branch into `master`, HEAD of the feature-branch must
  * be accepted by the [CI server](https://jenkins.dogcraft.de)
    * `make check-format vet test`
    * `make release` builds without errors 
  * be `fast-forward` mergeable (*rebase before merge*)

This workflow is enforced through GitHub repository settings.

```
# Create a new branch before you start working on a feature
git checkout -b myfeature
# Work and commit frequently...
```

## Pre-Commit Checklist 

To make your life simpler, 

* assert that your changes do not introduce bad formatting
  * `make check-format` lists up to 10 changes proposed by `gofmt`
  * `make format` applies `gofmt` to all unvendored go files
* assert that your changes do not cause `make vet` to fail
* build before commit

## Pull Requests & Peer Review

Once you verified all local tests pass, create a pull request, allowing other project members to review your changes.

Remember the rule **rebase before merge/pull-request**!

```
# Assert you are on your feature branch. The following command should print `myfeature`
git branch
# Fetch the latest changes
git fetch
# Rebase onto the latest changes in master
# If your branch and master are out of sync, you may have to resolve merge conflicts.
# Ask your team members if you are unsure about how to merge changes properly.
git rebase origin/master
git push -u origin myfeature:myfeature 
```

Once you pull request is reviewed, merge your changes fast-forward into master.

GitHub does not provide functionality for this in the GUI, so use the command line.

Given all tests on your PR passed, branch-protection will allow your commits on master.

```
git checkout master
git merge --ff-only myfeature
git push
```

## Local Staging Environment

You can spawn a local staging environment using **docker** and **sudo**.
Assert that the Docker daemon is running on your system.

```
make testbed_up
```
Checkout the `Makefile` for details.

# Producing a release

```
cd $GOPATH/src/github.com/KIT-MAMID/mamid
./makeRelease.bash
# binaries for all paltforms are located in
# $GOPATH/src/github.com/KIT-MAMID/mamid/build 
```

