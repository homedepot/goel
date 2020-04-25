.PHONY: fuzz

fuzz: fuzz-build fuzz-run

fuzz-build:
	CGO_ENABLED=1 go-fuzz-build github.com/homedepot/goel/fuzz

fuzz-run:
	CGO_ENABLED=1 go-fuzz -bin=./fuzz-fuzz.zip -workdir=$(shell pwd)

install:
	CGO_ENABLED=1 GO111MODULE=off go get -u github.com/dvyukov/go-fuzz/...
