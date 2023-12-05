.PHONY: test
test:
	@echo "Running tests..."
	@go test -count=1 ./...