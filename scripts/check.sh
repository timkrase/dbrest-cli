#!/bin/sh
set -eu

fmt_out=$(gofmt -l .)
if [ -n "$fmt_out" ]; then
  echo "gofmt needed on:"
  echo "$fmt_out"
  exit 1
fi

go vet ./...
go test ./...

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "missing golangci-lint (install: https://golangci-lint.run/usage/install/)" >&2
  exit 1
fi

golangci-lint run
