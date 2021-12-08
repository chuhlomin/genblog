.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: run
## run: runs the Go program
run:
	@go run .

.PHONY: vet
## vet: runs `go vet`
vet:
	@go vet ./...

.PHONY: test
## test: runs `go test`
test: vet
	@go test ./...  -coverprofile cp.out

.PHONY: build
## build: runs `go build`
build:
	@go build .

.PHONY: build-static
## build-static: builds the static binary for linux
build-static:
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -installsuffix cgo -o genblog .

.PHONY: build-docker
## build-docker: builds the container image with Docker
build-docker:
	@docker build --tag chuhlomin/genblog:latest ./;

.PHONY: build-podman
## build-podman: builds the container image with Podman
build-podman:
	@podman build --tag chuhlomin/genblog:latest ./;

.PHONY: push-docker
## push-docker: pushes the docker image to Docker Hub
push-docker:
	@docker push chuhlomin/genblog:latest

.PHONY: push-podman
## push-podman: pushes the docker image to Docker Hub with Podman
push-podman:
	@podman push chuhlomin/genblog:latest
