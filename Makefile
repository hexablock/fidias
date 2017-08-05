NAME = fidias

COMMIT = $(shell git rev-parse --short HEAD)

VERSION = $(shell git describe 2> /dev/null || echo "0.0.0-$(COMMIT)")
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

BUILD_CMD = CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo
LD_OPTS = -ldflags="-X main.version=$(VERSION) -X main.buildtime=$(BUILDTIME) -w"
SRC_FILES = ./cmd/*.go

clean:
	go clean -i ./...
	rm -f $(NAME)d
	rm -rf dist

# Local platform build
$(NAME)d:
	go build $(LD_OPTS) -o $(NAME)d $(SRC_FILES)

deps:
	go get -d ./...

test:
	go test -v -cover ./...

# Build all
dist: dist/$(NAME)d-windows.zip
	for os in linux darwin; do \
		GOOS=$${os} $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME)d $(SRC_FILES) && \
		tar -C dist -czf dist/$(NAME)d-$${os}.tgz $(NAME)d; rm -f dist/$(NAME)d; \
	done;

# Build windows
dist/$(NAME)d-windows.zip:
	GOOS=windows $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME)d.exe $(SRC_FILES) && \
	cd dist && zip $(NAME)d-windows.zip $(NAME)d.exe; rm -f dist/$(NAME)d.exe

protoc:
	protoc rpc.proto -I ./ -I ../../../ --go_out=plugins=grpc:.

all: $(NAME)d
