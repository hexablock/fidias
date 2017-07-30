NAME = fidias
VERSION = 0.0.0
COMMIT = $(shell git rev-parse --short HEAD)
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

BUILD_CMD = CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo
LD_OPTS = -ldflags="-X main.version=$(VERSION)-$(COMMIT) -X main.buildtime=$(BUILDTIME) -w"
SRC_FILES = ./cmd/*.go

clean:
	go clean -i ./...
	rm -f $(NAME)d
	rm -rf dist

# Local platform build
$(NAME)d:
	go build -o $(NAME)d $(SRC_FILES)

deps:
	go get -d -v ./...

test:
	go test -v -cover ./...

dist:
	[ -d dist ] || mkdir dist
	for os in linux darwin windows; do \
		GOOS=$${os} $(BUILD_CMD) $(LD_OPTS) -o dist/$(NAME)-$${os} $(SRC_FILES); \
		tar -C dist -czf dist/$(NAME)-$${os}.tgz $(NAME)-$${os}; \
	done;

all: $(NAME)d
