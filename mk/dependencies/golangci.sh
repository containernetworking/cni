#!/bin/bash
set -e

GOLANGCI_LINT_VERSION="v1.54"

go install "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"