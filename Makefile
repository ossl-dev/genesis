.PHONY: build test race vet check docs docs-serve clean release

build:
	go build -trimpath -ldflags='-s -w' -o bin/genesis ./apps/cli

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

check: vet test
	go run ./scripts/check

docs:
	go run ./apps/docs -output dist/docs

docs-serve:
	go run ./apps/docs -serve localhost:8080

release:
	go run ./scripts/release

clean:
	rm -rf bin dist
