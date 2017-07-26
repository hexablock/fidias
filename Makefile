
NAME = fidias
VERSION = 0.0.0
COMMIT = $(shell git rev-parse --short HEAD)
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

BUILD_CMD = CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo
LD_OPTS = -ldflags="-X main.version=$(VERSION)-$(COMMIT) -X main.buildtime=$(BUILDTIME) -w"

clean:
	go clean -i ./...
	rm -f $(NAME)d

$(NAME)d:
	go build $(LD_OPTS) -o $(NAME)d ./cmd/*.go

all: $(NAME)d
