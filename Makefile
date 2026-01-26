.PHONY: help
help: ## Help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
    awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run Go test
	@go test -v ./... -covermode=count -coverprofile=coverage.out fmt
	@go tool cover -func=coverage.out -o=coverage.out
	@cat coverage.out

.PHONY: get
get: ## Run Go get
	@go get -v -d ./...

.PHONY: lint