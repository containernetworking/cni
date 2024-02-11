#!/bin/bash
set -e

GOLANGCI_LINT_VERSION="v1.57.1"

go install "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
