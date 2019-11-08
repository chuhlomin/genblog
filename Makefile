run:
	@cd cmd/utterson; \
	go run .

vet:
	@go vet ./...

test: vet
	@go test ./...  -coverprofile cp.out

build:
	@cd cmd/utterson; \
	go build .

build-static:
	@cd ./cmd/utterson; \
	CGO_ENABLED=0 GOOS=linux go build -mod=readonly -a -installsuffix cgo -o utterson .

build-docker:
	@docker build --tag utterson:latest ./cmd/utterson;
