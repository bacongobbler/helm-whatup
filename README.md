# Helm Whatup

This is a Helm plugin to help users determine if there's an update available for their installed charts. It works by reading your locally cached index files from the chart repositories (via `helm repo update`) and checking the version against the latest deployed version of your charts in the Kubernetes cluster.

## Usage

```
$ helm repo update
$ helm whatup
```

## Install

```
$ helm plugin install https://github.com/bacongobbler/helm-whatup
```

The above will fetch the latest binary release of `helm whatup` and install it.

### Developer (From Source) Install

If you would like to handle the build yourself, instead of fetching a binary,
this is how recommend doing it.

First, set up your environment:

- You need to have [Go](http://golang.org) installed. Make sure to set `$GOPATH`
- If you don't have [Glide](http://glide.sh) installed, this will install it into
  `$GOPATH/bin` for you.

Clone this repo into your `$GOPATH` using git.

```
cd $GOPATH/src/github.com/bacongobbler
git clone https://github.com/bacongobbler/helm-whatup
```

Then run the following to get running.

```
$ cd helm-whatup
$ make bootstrap build
$ SKIP_BIN_INSTALL=1 helm plugin install $GOPATH/src/github.com/technosophos/helm-template
```

That last command will skip fetching the binary install and use the one you
built.
