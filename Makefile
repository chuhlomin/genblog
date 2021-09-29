run:
	@go run .

vet:
	@go vet ./...

test: vet
	@go test ./...  -coverprofile cp.out

build:
	@go build .

build-static:
	@CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -installsuffix cgo -o genblog .

build-docker:
	@docker build --tag chuhlomin/genblog:latest ./;

push-docker:
	@docker push chuhlomin/genblog:latest
