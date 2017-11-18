NAME = fid

COMMIT = $(shell git rev-parse --short HEAD)

VERSION = $(shell git describe 2> /dev/null || echo "0.0.0-$(COMMIT)")
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

BUILD_CMD = CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo
LD_OPTS = -ldflags="-X main.version=$(VERSION) -X main.buildtime=$(BUILDTIME) -w"
SRC_FILES = ./cmd/*.go

clean:
	go clean -i ./...
	rm -f $(NAME)
	rm -rf dist
	rm -rf tmp/*

# Local platform build
$(NAME):
	go build $(LD_OPTS) -o $(NAME) $(SRC_FILES)

deps:
	go get -d ./...

test:
	go test -cover $(shell go list ./... | grep -v /vendor/)

show-version:
	@echo $(VERSION)

# Build all
dist: dist/$(NAME)-windows.zip
	for os in linux darwin; do \
		GOOS=$${os} $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME) $(SRC_FILES) && \
		tar -C dist -czf dist/$(NAME)-$${os}.tgz $(NAME); rm -f dist/$(NAME); \
	done;

# Build windows
dist/$(NAME)-windows.zip:
	GOOS=windows $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME).exe $(SRC_FILES) && \
	cd dist && zip $(NAME)-windows.zip $(NAME).exe; rm -f dist/$(NAME).exe

dist/$(NAME)-linux.tgz:
	GOOS=linux $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME) $(SRC_FILES) && \
	tar -C dist -czf dist/$(NAME)-linux.tgz $(NAME); rm -f dist/$(NAME);

protoc:
	protoc rpc.proto -I ./ -I ../../../ --go_out=plugins=grpc:.


all: $(NAME)
