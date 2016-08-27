| Overall | Test | Format | Vet | Build |
| --- | --- | --- | --- | --- | --- |
| [![Overall](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid)](https://jenkins.dogcraft.de/job/mamid/) | [![Test](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=test)](https://jenkins.dogcraft.de/job/mamid/target=test) | [![Format](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=check-format)](https://jenkins.dogcraft.de/job/mamid/target=check-format) | [![Vet](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=vet)](https://jenkins.dogcraft.de/job/mamid/target=vet) | [![Release](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=build)](https://jenkins.dogcraft.de/job/mamid/target=build) |

# Development Setup

* Assert `go version` >= 1.6
* Assert `go env`
  * contains `GO15VENDOREXPERIMENT="1"` (dependencies are tracked using [vendoring](https://golang.org/cmd/go/#hdr-Vendor_Directories))
* Assert `$GOPATH` environment variable is set to your GOPATH

  ```
  # Setup the $GOPATH environment variable
  # Checkout this repository to $GOPATH/src/github.com/KIT-MAMID/mamid
  cd $GOPATH/src/github.com/KIT-MAMID/mamid
  git submodule update --fetch
  make
  ```

* Running tests requires a PostgreSQL instance with permission to `CREATE DATABASE` and `DESTROY DATABASE`
  * Running the test instance in a docker container different than the one used for `make testbed_*` builds is recommended

    ```
    docker run --name mamid-postgres-tests -e POSTGRES_PASSWORD=mysecretpassword -d postgres
    # You should now be able to connect to the container using the password above
    psql postgres
    # You can run tests by setting the appropriate DSN environment variable
    MAMID_TESTDB_DSN="host=localhost sslmode=disable dbname=postgres" make test-verbose
    ```

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

