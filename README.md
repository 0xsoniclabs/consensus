[![codecov](https://codecov.io/gh/0xsoniclabs/consensus/graph/badge.svg?token=4YGWLYH1QX)](https://codecov.io/gh/0xsoniclabs/consensus)

# Sonic Consensus
A base library that defines interfaces and modules of the consensus protocol.

## Build Details
To build testing tooling, type `make all`. Consult the Makefile for individual build options.

## Documentation
commands used and replication instructions are as follows:
- install graphvis with `go install github.com/ofabry/go-callvis@latest` (link is https://github.com/ondrajz/go-callvis)
- using your package manager: brew, apt, etc.
- please also install graphviz.
- example of how to do this with apt is `sudo apt-get install graphviz` ; for homebrew on macs you can use `brew install graphviz`
- run `go-callvis -graphviz -focus=consensusengine ./consensus/consensusengine` - where focus denotes the package name and the last argument is the path to the package relative to go.mod . go.mod should be in the same directory as your current working directory from where you should run the go-callvis command, that is, when you run go-callvis your path should be something like the following: the top level of the repo.

### Testing
To test, type `make test'. Consult the Makefile for individual options.
