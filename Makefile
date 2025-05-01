makeGOSRC:=${GOPATH}/src

GIT_TAG:=$$(git describe --tags --always)
BUILD_TIME:=$$(date +%FT%T%z)

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	go build -tags '-trimpath' -ldflags "-s -w -extldflags '-static' -X main.version=$GIT_TAG -X main.build=$BUILD_TIME" -o tn-mermaid cmd/mermaid/*

.PHONY: lint-docker
lint-docker:
	docker run -t --rm \
		-e GONOSUMDB=gitlаb.tn.ru -e GOPRIVATE=gitlab.tn.ru \
		-v ~/.netrc:/root/.netrc \
		-v ~/.cache/golangci-lint/v2.1.2:/root/.cache \
		-v .:/app -w /app golangci/golangci-lint:v2.1.2 \
		golangci-lint run -v --config ./.golangci.yml

