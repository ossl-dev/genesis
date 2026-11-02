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
	cd apps/docs && bun run build

docs-serve:
	cd apps/docs && bun run dev

release:
	go run ./scripts/release

clean:
	rm -rf bin dist
