.PHONY: lint
lint: ## Apply go lint check
	@golangci-lint run --timeout 10m ./...

.PHONY: swagger
swagger:
	swag fmt && swag init --parseInternal --pd -p snakecase 
