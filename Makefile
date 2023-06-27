.PHONY: clean

REPO= github.com/ca-gip/kotary
IMAGE= kotary
DOCKER_REPO= cagip

dependency:
	go mod vendor

codegen: dependency
	bash hack/update-codegen.sh

test:
	go mod tidy
	go mod vendor
	GOARCH=amd64 go test ./internal/controller -coverprofile coverage.out
	GOARCH=amd64 go tool cover -func coverage.out
	GOARCH=amd64 go tool cover -html=coverage.out -o coverage.html

build:
	GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -v -o ./build/$(IMAGE) $(GOPATH)/src/$(REPO)/cmd/main.go
