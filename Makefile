NAME := "bin/cli-test"

.PHONY: init
init:
	GO111MODULE=on go mod init

.PHONY: test
test:
	GO111MODULE=on go test -v ./...

.PHOMY: lint
lint:
	GO111MODULE=on golint ./...

.PHONY: build
build: lint test $(NAME)

$(NAME): main.go $(shell find . -type f -name "*.go")
	GO111MODULE=on go build -o $(NAME)
