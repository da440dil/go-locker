# go-locker

[![Build Status](https://travis-ci.com/da440dil/go-locker.svg?branch=master)](https://travis-ci.com/da440dil/go-locker)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-locker/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-locker?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-locker?status.svg)](https://godoc.org/github.com/da440dil/go-locker)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-locker)](https://goreportcard.com/report/github.com/da440dil/go-locker)


Distributed locking with pluggable storage to store a lock state.

## Usage

- [example](./examples/locker-gateway-default/main.go) usage with default gateway.
- [example](./examples/locker-gateway-memory/main.go) usage with memory gateway.
- [example](./examples/locker-gateway-redis/main.go) usage with Redis gateway.