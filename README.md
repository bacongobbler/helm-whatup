# Helm Whatup

[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Maintainability](https://api.codeclimate.com/v1/badges/ec4254803f465c1c8a58/maintainability)](https://codeclimate.com/github/fabmation-gmbh/helm-whatup/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/ec4254803f465c1c8a58/test_coverage)](https://codeclimate.com/github/fabmation-gmbh/helm-whatup/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/fabmation-gmbh/helm-whatup)](https://goreportcard.com/report/github.com/fabmation-gmbh/helm-whatup)


This Repo is a fork of [bacongobbler/helm-whatup][], because the original project is no longer actively further developed.

This is a Helm plugin to help users determine if there's an update available for their installed charts.
It works by reading your locally cached index files from the chart repositories (via `helm repo update`) and checking
the version against the latest deployed version of your charts in the Kubernetes cluster.


## Usage

```
helm repo update
helm whatup
```


## Install

```
$ helm plugin install https://github.com/fabmation-gmbh/helm-whatup
```

The above will fetch the latest binary release of `helm whatup` and install it.


### Developer (From Source) Install

If you would like to handle the build yourself, instead of fetching a binary, this is how recommend doing it.

First, set up your environment:

- You need to have [Go](http://golang.org) installed. Make sure to set `$GOPATH`
- You need to have [Glide](http://glide.sh) installed.

Clone this repo into your `$GOPATH` using git.

```
mkdir $GOPATH/src/github.com/fabmation-gmbh
cd $GOPATH/src/github.com/fabmation-gmbh
git clone https://github.com/fabmation-gmbh/helm-whatup
```

Then run the following to get running.

```
cd helm-whatup
make bootstrap build
SKIP_BIN_INSTALL=1 helm plugin install $GOPATH/src/github.com/fabmation-gmbh/helm-whatup
```

That last command will skip fetching the binary install and use the one you
built.




<!-- LINKS -->
[bacongobbler/helm-whatup]: https://github.com/bacongobbler/helm-whatup
