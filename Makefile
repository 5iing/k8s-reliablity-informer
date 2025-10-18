.PHONY: build test clean run docker-build docker-run

BINARY_NAME=k3s-health-checker
DOCKER_IMAGE=k3s-health-checker
CONFIG_FILE=pkg/config/config.yaml

build:
	go build -o $(BINARY_NAME) .

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME) -config $(CONFIG_FILE)

docker-build:
	docker build -t $(DOCKER_IMAGE):latest .

docker-run:
	docker run --rm -v ~/.kube/config:/root/.kube/config $(DOCKER_IMAGE):latest

install:
	go install

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

all: clean lint test build
