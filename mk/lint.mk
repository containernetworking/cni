.PHONY: lint
lint: golangci/install golangci/lint

.PHONY: golangci/install
golangci/install:
	./mk/dependencies/golangci.sh

.PHONY: golangci/lint
golangci/lint:
	golangci-lint run --verbose

.PHONY: golangci/fix
golangci/fix:
	golangci-lint run --verbose --fix