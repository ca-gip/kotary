.PHONY: clean

REPO= github.com/ca-gip/kotary
IMAGE= kotary
DOCKER_REPO= cagip

dependency:
	go mod vendor

codegen: dependency
	bash hack/update-codegen.sh

test: codegen
	GOARCH=amd64 go test ./internal/controller

build: test
	GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -a -ldflags '-extldflags "-static"' -v -o ./build/$(IMAGE) -i $(GOPATH)/src/$(REPO)/cmd/main.go
