BUILD_FLAGS     :=
GOBIN           := $(shell go env GOPATH)/bin

all: lint build

docker-build:
	@echo "Building from Docker Compose file..."
	@docker-compose build

docker-run: docker-build
	docker-compose up -d

docker-down:
	@echo "Stopping Docker container..."
	@docker-compose down

down:
	@make migration-down
	@docker-compose down

deps:
	go mod tidy
	go mod download

clean:
	rm -f $(BINARY_NAME)