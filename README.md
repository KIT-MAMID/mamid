| Overall | Test | Format | Vet | Release |
| --- | --- | --- | --- | --- | --- |
| [![Overall](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid)](https://jenkins.dogcraft.de/job/mamid/) | [![Test](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=test)](https://jenkins.dogcraft.de/job/mamid/target=test) | [![Format](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=check-format)](https://jenkins.dogcraft.de/job/mamid/target=check-format) | [![Vet](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=vet)](https://jenkins.dogcraft.de/job/mamid/target=vet) | [![Release](https://jenkins.dogcraft.de/buildStatus/icon?job=mamid/target=release)](https://jenkins.dogcraft.de/job/mamid/target=release) |

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

## Pre-Commit Checklist 

To make your life simpler, 

* assert that your changes do not introduce bad formatting
  * `make check-format` lists up to 10 changes proposed by `gofmt`
  * `make format` applies `gofmt` to all unvendored go files
* assert that your changes do not cause `make vet` to fail
* build before commit

# Producing a release

```
cd $GOPATH/src/github.com/KIT-MAMID/mamid
./makeRelease.bash
# binaries for all paltforms are located in
# $GOPATH/src/github.com/KIT-MAMID/mamid/build 
```

